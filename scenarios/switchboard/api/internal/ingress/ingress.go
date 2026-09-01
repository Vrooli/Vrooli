package ingress

import (
	"fmt"
	"sync"

	"switchboard/internal/channels"
)

type Result string

const (
	Accepted    Result = "accepted"
	AlreadySeen Result = "already_seen"
	Refused     Result = "refused"
)

type Store struct {
	mu   sync.Mutex
	seen map[string]channels.Envelope
}

func New() *Store { return &Store{seen: map[string]channels.Envelope{}} }
func (s *Store) Accept(e channels.Envelope) (Result, error) {
	if e.ChannelID == "" || e.RemoteMessageID == "" {
		return Refused, fmt.Errorf("channel_id and remote_message_id are required")
	}
	key := e.ChannelID + "\x00" + e.RemoteMessageID
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[key]; ok {
		return AlreadySeen, nil
	}
	s.seen[key] = e
	return Accepted, nil
}
func (s *Store) Count() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.seen) }
