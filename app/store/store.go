package store

import (
	"sync"
	"time"
)

type entry struct {
	value  string
	expiry time.Time
}

type Store struct {
	mu      sync.Mutex
	data    map[string]entry
	lists   map[string][]string
	waiters map[string][]chan string
}

func NewStore() *Store {
	return &Store{
		data:    make(map[string]entry),
		lists:   make(map[string][]string),
		waiters: make(map[string][]chan string),
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
	return "none"
}
