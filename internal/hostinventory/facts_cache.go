package hostinventory

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	mu     sync.Mutex
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

// Read returns a fresh or cached fact class. A file lock keeps independent CLI
// processes from probing the same expensive class simultaneously.
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
	lock := r.Path + ".lock"
	for i := 0; i < 500; i++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, tuning.PermSecret)
		if err == nil {
			_ = f.Close()
			defer os.Remove(lock)
			break
		}
		if !errors.Is(err, os.ErrExist) {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(tuning.HostFactsRetryInterval()):
		}
		if entry, ok := r.load(class); ok && entry.Schema == hostFactsSchemaVersion && entry.BootID == boot && now.Sub(entry.FetchedAt) < ttl {
			return append([]byte(nil), entry.Value...), nil
		}
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
