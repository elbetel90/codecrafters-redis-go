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
	mu    sync.Mutex
	data  map[string]entry
	lists map[string][]string
}

func NewStore() *Store {
	return &Store{
		data:  make(map[string]entry),
		lists: make(map[string][]string),
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
	return len(s.lists[key]), nil
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

func (s *Store) LPop(key string, count int) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.lists[key]
	if !ok || len(v) == 0 {
		return "", false
	}
	if count > len(v) {
		count = len(v)
	}
	out := make([]string, count)
	copy(out, v[:count])
	item := v[0]
	if count == len(v) {
		delete(s.lists, key)
	} else {
		s.lists[key] = v[count:]
	}
	return item, true
}
