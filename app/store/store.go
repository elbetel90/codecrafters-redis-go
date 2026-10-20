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
