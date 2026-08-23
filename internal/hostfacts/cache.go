// Package hostfacts provides a small cross-process TTL cache for expensive,
// non-emergency host probes. The emergency watchdog must keep using direct
// pressure reads; a stale cache must never hide saturation.
package hostfacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const schemaVersion = 1

type (
	Probe  func(context.Context, string) (json.RawMessage, error)
	Reader struct {
		Path   string
		TTL    map[string]time.Duration
		Probe  Probe
		BootID func() string
		Now    func() time.Time
		mu     sync.Mutex
	}
)

type entry struct {
	Schema    int             `json:"schema"`
	BootID    string          `json:"boot_id"`
	FetchedAt time.Time       `json:"fetched_at"`
	Class     string          `json:"class"`
	Value     json.RawMessage `json:"value"`
}

// file is the shared on-disk envelope. Keeping all classes together avoids
// making a GPU refresh evict an otherwise-warm inventory refresh.
type file struct {
	Schema  int                  `json:"schema"`
	BootID  string               `json:"boot_id"`
	Entries map[string]fileEntry `json:"entries"`
}

type fileEntry struct {
	FetchedAt time.Time       `json:"fetched_at"`
	Value     json.RawMessage `json:"value"`
}

func (r *Reader) Read(ctx context.Context, class string) (json.RawMessage, error) {
	if r == nil || r.Probe == nil {
		return nil, errors.New("host facts reader has no probe")
	}
	if class == "" {
		return nil, errors.New("host facts class is required")
	}
	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	boot := ""
	if r.BootID != nil {
		boot = r.BootID()
	}
	ttl := r.TTL[class]
	if ttl <= 0 {
		ttl = time.Minute
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.load(class); ok && e.Schema == schemaVersion && e.BootID == boot && now.Sub(e.FetchedAt) < ttl {
		return append([]byte(nil), e.Value...), nil
	}
	// O_EXCL makes concurrent short-lived CLI processes converge on one
	// refresh. A reader that observes another process refreshing waits for the
	// completed cache, then falls through only if the owner is wedged. Never
	// remove a lock created by another process.
	lock := r.Path + ".lock"
	for i := 0; i < 500; i++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			defer os.Remove(lock)
			break
		}
		if !errors.Is(err, os.ErrExist) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
		if e, ok := r.load(class); ok && e.Schema == schemaVersion && e.BootID == boot && now.Sub(e.FetchedAt) < ttl {
			return append([]byte(nil), e.Value...), nil
		}
	}
	value, err := r.Probe(ctx, class)
	if err != nil {
		return nil, err
	}
	if err := r.store(class, entry{Schema: schemaVersion, BootID: boot, FetchedAt: now, Class: class, Value: value}); err != nil {
		return nil, err
	}
	return append([]byte(nil), value...), nil
}

func (r *Reader) load(class string) (entry, bool) {
	b, err := os.ReadFile(r.Path)
	if err != nil {
		return entry{}, false
	}
	var cached file
	if json.Unmarshal(b, &cached) == nil && cached.Schema == schemaVersion && cached.Entries != nil {
		e, ok := cached.Entries[class]
		if !ok {
			return entry{}, false
		}
		return entry{Schema: cached.Schema, BootID: cached.BootID, Class: class, FetchedAt: e.FetchedAt, Value: e.Value}, true
	}
	// Read the pre-multi-class format so an upgrade never forces every caller
	// to probe at once. It is rewritten in the new format on the next refresh.
	var legacy entry
	if json.Unmarshal(b, &legacy) != nil || legacy.Class != class {
		return entry{}, false
	}
	return legacy, true
}

func (r *Reader) store(class string, e entry) error {
	if err := os.MkdirAll(filepath.Dir(r.Path), 0o700); err != nil {
		return err
	}
	cached := file{Schema: schemaVersion, BootID: e.BootID, Entries: map[string]fileEntry{}}
	if current, ok := r.readFile(); ok && current.Schema == schemaVersion && current.BootID == e.BootID {
		cached = current
	}
	if cached.Entries == nil {
		cached.Entries = map[string]fileEntry{}
	}
	cached.Entries[class] = fileEntry{FetchedAt: e.FetchedAt, Value: append([]byte(nil), e.Value...)}
	b, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", r.Path, os.Getpid())
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.Path)
}

func (r *Reader) readFile() (file, bool) {
	b, err := os.ReadFile(r.Path)
	if err != nil {
		return file{}, false
	}
	var cached file
	if err := json.Unmarshal(b, &cached); err != nil {
		return file{}, false
	}
	return cached, true
}
