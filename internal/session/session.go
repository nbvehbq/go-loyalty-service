package session

import (
	"context"
	"sync"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pkg/errors"
)

const (
	sessionTTL    = time.Hour * 1
	clearInterval = time.Minute * 10
)

type object struct {
	id      int64
	expires time.Time
}

type Session struct {
	mu      sync.RWMutex
	storage map[string]object
}

func NewSessionStorage(ctx context.Context) *Session {
	s := Session{
		mu:      sync.RWMutex{},
		storage: make(map[string]object),
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(clearInterval):
				s.reduceSessions()
			}
		}
	}()

	return &s
}

func (s *Session) Set(_ context.Context, id int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sid, err := gonanoid.New()
	if err != nil {
		return "", errors.Wrap(err, "generate sid")
	}
	s.storage[sid] = object{
		id:      id,
		expires: time.Now().Add(sessionTTL),
	}

	return sid, nil
}

func (s *Session) Get(_ context.Context, sid string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	o, ok := s.storage[sid]
	if !ok {
		return 0, false
	}

	return o.id, true
}

func (s *Session) reduceSessions() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for k, v := range s.storage {
		if now.After(v.expires) {
			delete(s.storage, k)
		}
	}
}
