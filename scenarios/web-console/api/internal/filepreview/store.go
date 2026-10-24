package filepreview

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
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
	SourceInfo           os.FileInfo
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
	// Data is immutable, bounded virtual member content. RootID ties its
	// session, lifetime, freshness and shared work budget to the original file.
	Data          []byte
	RootID        string
	Depth         int
	Sequence      uint64
	Budget        *archiveBudget
	activeReaders int // guarded by Store.mu
}

// Store is an in-memory, session-bound, expiring preview-id store. It is safe
// for concurrent use. Lookups are constant-time map hits; expired entries are
// reaped lazily on access and by an optional background sweep.
type Store struct {
	virtualBytes int64
	sequence     uint64
	archiveJobs  chan struct{}
	mu           sync.Mutex
	byID         map[string]*Entry
	ttl          time.Duration
	now          func() time.Time
	rand         func([]byte) (int, error)
}

// NewStore constructs a Store with the given TTL (<=0 → DefaultTTL).
func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Store{
		byID:        make(map[string]*Entry),
		archiveJobs: make(chan struct{}, 2),
		ttl:         ttl,
		now:         time.Now,
		rand:        rand.Read,
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
	s.sequence++
	expiry := s.now().Add(s.ttl)
	s.byID[id] = &Entry{
		ID:                   id,
		SourceInfo:           t.SourceInfo,
		Sequence:             s.sequence,
		Budget:               &archiveBudget{remaining: ArchiveWorkLimit},
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
		s.deleteLocked(id)
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
			s.deleteLocked(id)
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

// AcquireBlob pins virtual bytes while HTTP serves them. Active responses remain
// charged to the global quota, including after logical expiry, until released.
func (s *Store) AcquireBlob(sessionID, id string) (*Entry, func(), error) {
	s.mu.Lock()
	e := s.byID[id]
	if e == nil || e.SessionID != sessionID || !s.now().Before(e.ExpiresAt) {
		s.mu.Unlock()
		return nil, nil, ErrPreviewNotFound
	}
	if e.RootID == "" {
		s.mu.Unlock()
		return e, func() {}, nil
	}
	e.activeReaders++
	s.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			e.activeReaders--
			if e.activeReaders == 0 && !s.now().Before(e.ExpiresAt) {
				s.deleteLocked(id)
			}
		})
	}
	return e, release, nil
}
