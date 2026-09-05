package filepreview

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// DefaultTTL is how long an issued preview id stays valid. Short by design:
// the browser resolves and loads a preview in one interaction, and a stale id
// forces a fresh resolve (which re-checks file metadata).
const DefaultTTL = 30 * time.Minute

// ErrPreviewNotFound is returned for an unknown, expired, or session-mismatched
// preview id. The blob handler maps it to 404 so a wrong session can't probe
// whether an id exists.
var ErrPreviewNotFound = errors.New("preview not found")

// Entry is the immutable record a preview id points at. Size/ModTimeUnixNano
// pin the file metadata captured at resolve time; the blob handler re-stats and
// rejects on mismatch so a swapped file is never served under a stale id.
type Entry struct {
	ID                   string
	SessionID            string
	ResolvedPath         string
	Basename             string
	MIMEType             string
	Kind                 Kind
	SizeBytes            int64
	ModTimeUnixNano      int64
	CanDownload          bool
	CanPreview           bool
	TextContentAvailable bool
	ListingAvailable     bool
	ExpiresAt            time.Time
}

// Store is an in-memory, session-bound, expiring preview-id store. It is safe
// for concurrent use. Lookups are constant-time map hits; expired entries are
// reaped lazily on access and by an optional background sweep.
type Store struct {
	mu   sync.Mutex
	byID map[string]*Entry
	ttl  time.Duration
	now  func() time.Time
	rand func([]byte) (int, error)
}

// NewStore constructs a Store with the given TTL (<=0 → DefaultTTL).
func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Store{
		byID: make(map[string]*Entry),
		ttl:  ttl,
		now:  time.Now,
		rand: rand.Read,
	}
}

// Issue stores a Target under a fresh opaque id bound to sessionID and returns
// the id and its expiry. The returned id is the only handle the browser ever
// sends back; the raw path never travels to the blob route.
func (s *Store) Issue(sessionID string, t *Target) (string, time.Time, error) {
	id, err := s.newID()
	if err != nil {
		return "", time.Time{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry := s.now().Add(s.ttl)
	s.byID[id] = &Entry{
		ID:                   id,
		SessionID:            sessionID,
		ResolvedPath:         t.ResolvedPath,
		Basename:             t.Basename,
		MIMEType:             t.MIMEType,
		Kind:                 t.Kind,
		SizeBytes:            t.SizeBytes,
		ModTimeUnixNano:      t.ModTimeUnixNano,
		CanDownload:          t.CanDownload,
		CanPreview:           t.CanPreview,
		TextContentAvailable: t.TextContentAvailable,
		ListingAvailable:     t.ListingAvailable,
		ExpiresAt:            expiry,
	}
	return id, expiry, nil
}

// Lookup returns the entry for id iff it exists, has not expired, and is bound
// to sessionID. All failure modes collapse to ErrPreviewNotFound.
func (s *Store) Lookup(sessionID, id string) (*Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return nil, ErrPreviewNotFound
	}
	if !s.now().Before(e.ExpiresAt) {
		delete(s.byID, id)
		return nil, ErrPreviewNotFound
	}
	if e.SessionID != sessionID {
		return nil, ErrPreviewNotFound
	}
	return e, nil
}

// Sweep removes expired entries. Safe to call periodically; a no-op when empty.
func (s *Store) Sweep() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	for id, e := range s.byID {
		if !now.Before(e.ExpiresAt) {
			delete(s.byID, id)
		}
	}
}

func (s *Store) newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := s.rand(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
