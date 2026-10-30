package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type entry struct {
	value  string
	expiry time.Time
}

type streamEntry struct {
	id     string
	fields []string
}

type Store struct {
	mu      sync.Mutex
	data    map[string]entry
	lists   map[string][]string
	waiters map[string][]chan string
	streams map[string][]streamEntry
}

func NewStore() *Store {
	return &Store{
		data:    make(map[string]entry),
		lists:   make(map[string][]string),
		waiters: make(map[string][]chan string),
		streams: make(map[string][]streamEntry),
	}
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[key]
	if !ok {
		return "", false
	}
	if !e.expiry.IsZero() && !time.Now().Before(e.expiry) {
		delete(s.data, key)
		return "", false
	}
	return e.value, true
}

func (s *Store) Set(key, value string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = entry{value: value, expiry: expiry}
}

func (s *Store) RPush(key string, value ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lists[key] = append(s.lists[key], value...)
	n := len(s.lists[key])
	s.serveWaiters(key)
	return n, nil
}

func (s *Store) LRange(key string, start, stop int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lists[key]
	if !ok {
		return []string{}
	}
	n := len(v)
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	if start < 0 {
		start = 0
	}
	if start >= n {
		return []string{}
	}
	if stop >= n {
		stop = n - 1
	}
	if start > stop {
		return []string{}
	}
	out := make([]string, stop-start+1)
	copy(out, v[start:stop+1])
	return out
}

func (s *Store) LPush(key string, value ...string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range value {
		s.lists[key] = append([]string{v}, s.lists[key]...)
	}
	s.serveWaiters(key)
	return len(s.lists[key]), nil
}

func (s *Store) LLen(key string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lists[key]
	if !ok {
		return 0
	}
	return len(v)
}

func (s *Store) serveWaiters(key string) {
	for {
		ws := s.waiters[key]
		list := s.lists[key]
		if len(ws) == 0 || len(list) == 0 {
			break
		}
		ch := ws[0]
		s.waiters[key] = ws[1:]
		if len(s.waiters[key]) == 0 {
			delete(s.waiters, key)
		}
		val := list[0]
		if len(list) == 1 {
			delete(s.lists, key)
		} else {
			s.lists[key] = list[1:]
		}
		ch <- val
	}
}

func (s *Store) removeWaiter(key string, ch chan string) bool {
	ws := s.waiters[key]
	for i, w := range ws {
		if w == ch {
			s.waiters[key] = append(ws[:i], ws[i+1:]...)
			if len(s.waiters[key]) == 0 {
				delete(s.waiters, key)
			}
			return true
		}
	}
	return false
}

func (s *Store) LPop(key string, count int) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lists[key]
	if !ok || len(v) == 0 {
		return nil, false
	}
	if count > len(v) {
		count = len(v)
	}
	out := make([]string, count)
	copy(out, v[:count])
	if count == len(v) {
		delete(s.lists, key)
	} else {
		s.lists[key] = v[count:]
	}
	return out, true
}

func (s *Store) BLpop(key string, timeout time.Duration) (string, bool) {
	s.mu.Lock()
	if list := s.lists[key]; len(list) > 0 {
		val := list[0]
		if len(list) == 1 {
			delete(s.lists, key)
		} else {
			s.lists[key] = list[1:]
		}
		s.mu.Unlock()
		return val, true
	}
	ch := make(chan string, 1)
	s.waiters[key] = append(s.waiters[key], ch)
	s.mu.Unlock()
	if timeout == 0 {
		return <-ch, true
	}
	select {
	case val := <-ch:
		return val, true
	case <-time.After(timeout):
		s.mu.Lock()
		stillWaiting := s.removeWaiter(key, ch)
		s.mu.Unlock()
		if stillWaiting {
			return "", false
		}
		return <-ch, true
	}
}

func (s *Store) Type(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.data[key]; ok {
		if !e.expiry.IsZero() && !time.Now().Before(e.expiry) {
			delete(s.data, key)
			return "none"
		}
		return "string"
	}
	if _, ok := s.lists[key]; ok {
		return "list"
	}
	if _, ok := s.streams[key]; ok {
		return "stream"
	}
	return "none"
}

func (s *Store) XAdd(key, id string, fields []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[key]; ok {
		return "", errors.New("wrong type error")
	}
	if _, ok := s.lists[key]; ok {
		return "", errors.New("wrong type")
	}
	ms, seq, autoSeq, err := parseStreamID(id)
	if err != nil {
		return "", err
	}
	stream := s.streams[key]
	if autoSeq {
		if len(stream) == 0 {
			if ms == 0 {
				seq = 1
			} else {
				seq = 0
			}
		} else {
			lastMs, lastSeq, _, err := parseStreamID(stream[len(stream)-1].id)
			if err != nil {
				return "", err
			}
			if ms < lastMs {
				return "", errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
			}
			if ms == lastMs {
				seq = lastSeq + 1
			} else {
				seq = 0
			}
		}
		id = fmt.Sprintf("%d-%d", ms, seq)
	}
	if ms == 0 && seq == 0 {
		return "", errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}
	if len(stream) > 0 {
		lastMs, lastSeq, _, err := parseStreamID(stream[len(stream)-1].id)
		if err != nil {
			return "", err
		}
		if ms < lastMs || (ms == lastMs && seq <= lastSeq) {
			return "", errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
	}
	s.streams[key] = append(s.streams[key], streamEntry{id: id, fields: fields})

	return id, nil
}

func parseStreamID(id string) (ms, seq int64, autoSeq bool, err error) {
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return 0, 0, false, errors.New("ERR Invalid stream ID specified as stream command argument")
	}
	ms, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false, errors.New("ERR Invalid stream ID specified as stream command argument")
	}
	if parts[1] == "*" {
		return ms, 0, true, nil
	} else {
		seq, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, false, errors.New("ERR Invalid stream ID specified as stream command argument")
		}
	}
	return ms, seq, false, nil
}
