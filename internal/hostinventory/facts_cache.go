package hostinventory

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	platform "github.com/vrooli/platform-go"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/tuning"
)

const hostFactsSchemaVersion = 1

type hostFactsProbe func(context.Context, string) (json.RawMessage, error)

type hostFactsReader struct {
	Path   string
	TTL    map[string]time.Duration
	Probe  hostFactsProbe
	BootID func() string
	Now    func() time.Time
	// AcquireLock overrides the cross-process advisory lock (for testing).
	// Nil uses platform.AcquireFileLockContext.
	AcquireLock func(context.Context, string) (func(), error)
	mu          sync.Mutex
}

type hostFactsEntry struct {
	Schema    int             `json:"schema"`
	BootID    string          `json:"boot_id"`
	FetchedAt time.Time       `json:"fetched_at"`
	Class     string          `json:"class"`
	Value     json.RawMessage `json:"value"`
}

type hostFactsFile struct {
	Schema  int                           `json:"schema"`
	BootID  string                        `json:"boot_id"`
	Entries map[string]hostFactsFileEntry `json:"entries"`
}

type hostFactsFileEntry struct {
	FetchedAt time.Time       `json:"fetched_at"`
	Value     json.RawMessage `json:"value"`
}

// Read returns a fresh or cached fact class. An advisory file lock keeps
// independent CLI processes from probing the same expensive class
// simultaneously. The lock is a kernel flock, not a lockfile: a holder that
// dies releases it automatically, so one crashed process can never tax every
// later probe on the host (an orphaned .lock from 2026-08-27 did exactly
// that — every cache miss spun 5s and the autoheal GPU checks timed out).
func (r *hostFactsReader) Read(ctx context.Context, class string) (json.RawMessage, error) {
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
	if entry, ok := r.load(class); ok && entry.Schema == hostFactsSchemaVersion && entry.BootID == boot && now.Sub(entry.FetchedAt) < ttl {
		return append([]byte(nil), entry.Value...), nil
	}
	if err := os.MkdirAll(filepath.Dir(r.Path), tuning.PermPrivateDir); err != nil {
		return nil, err
	}
	acquire := r.AcquireLock
	if acquire == nil {
		acquire = platform.AcquireFileLockContext
	}
	release, err := acquire(ctx, r.Path+".lock")
	if err != nil {
		return nil, err
	}
	defer release()
	// Another process may have refreshed the class while this one waited.
	if entry, ok := r.load(class); ok && entry.Schema == hostFactsSchemaVersion && entry.BootID == boot && now.Sub(entry.FetchedAt) < ttl {
		return append([]byte(nil), entry.Value...), nil
	}
	value, err := r.Probe(ctx, class)
	if err != nil {
		return nil, err
	}
	if err := r.store(class, hostFactsEntry{Schema: hostFactsSchemaVersion, BootID: boot, FetchedAt: now, Class: class, Value: value}); err != nil {
		return nil, err
	}
	return append([]byte(nil), value...), nil
}

func (r *hostFactsReader) load(class string) (hostFactsEntry, bool) {
	b, err := os.ReadFile(r.Path)
	if err != nil {
		return hostFactsEntry{}, false
	}
	var cached hostFactsFile
	if json.Unmarshal(b, &cached) == nil && cached.Schema == hostFactsSchemaVersion && cached.Entries != nil {
		entry, ok := cached.Entries[class]
		if !ok {
			return hostFactsEntry{}, false
		}
		return hostFactsEntry{Schema: cached.Schema, BootID: cached.BootID, Class: class, FetchedAt: entry.FetchedAt, Value: entry.Value}, true
	}
	var legacy hostFactsEntry
	if json.Unmarshal(b, &legacy) != nil || legacy.Class != class {
		return hostFactsEntry{}, false
	}
	return legacy, true
}

func (r *hostFactsReader) store(class string, entry hostFactsEntry) error {
	if err := os.MkdirAll(filepath.Dir(r.Path), tuning.PermPrivateDir); err != nil {
		return err
	}
	cached := hostFactsFile{Schema: hostFactsSchemaVersion, BootID: entry.BootID, Entries: map[string]hostFactsFileEntry{}}
	if current, ok := r.readFile(); ok && current.Schema == hostFactsSchemaVersion && current.BootID == entry.BootID {
		cached = current
	}
	if cached.Entries == nil {
		cached.Entries = map[string]hostFactsFileEntry{}
	}
	cached.Entries[class] = hostFactsFileEntry{FetchedAt: entry.FetchedAt, Value: append([]byte(nil), entry.Value...)}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(r.Path, data, tuning.PermSecret)
}

func (r *hostFactsReader) readFile() (hostFactsFile, bool) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return hostFactsFile{}, false
	}
	var cached hostFactsFile
	if err := json.Unmarshal(data, &cached); err != nil {
		return hostFactsFile{}, false
	}
	return cached, true
}
