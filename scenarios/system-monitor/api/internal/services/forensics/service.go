// Package forensics surfaces crash-forensics signals (pstore artifacts,
// boot history, MCE summaries) as on-demand reads with short in-memory
// memoization. All endpoints degrade to {available: false} envelopes
// rather than erroring when their substrate is missing.
package forensics

import (
	"context"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/journal"
)

// memoTTL is the shared cache duration for forensics endpoints.
// Short enough to feel live, long enough to absorb dashboard polls.
const memoTTL = 30 * time.Second

// CommandExecutor abstracts shelling out for tests.
type CommandExecutor interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// FileSystem abstracts the few filesystem reads the forensics service
// needs. ReadDirFn returns directory entries; StatFn stats a single path.
type FileSystem struct {
	ReadDirFn func(name string) ([]fs.DirEntry, error)
	StatFn    func(name string) (fs.FileInfo, error)
}

// DefaultFileSystem returns a FileSystem backed by os.
func DefaultFileSystem() FileSystem {
	return FileSystem{
		ReadDirFn: os.ReadDir,
		StatFn:    os.Stat,
	}
}

// Envelope is the outer shape every forensics endpoint returns.
type Envelope struct {
	Available   bool        `json:"available"`
	Reason      string      `json:"reason,omitempty"`
	Data        interface{} `json:"data,omitempty"`
	GeneratedAt time.Time   `json:"generatedAt"`
}

// memoCache holds one cached envelope per endpoint key.
type memoCache struct {
	mu      sync.Mutex
	entries map[string]memoEntry
}

type memoEntry struct {
	value     Envelope
	fetchedAt time.Time
}

func (m *memoCache) get(key string, now time.Time) (Envelope, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[key]
	if !ok {
		return Envelope{}, false
	}
	if now.Sub(e.fetchedAt) >= memoTTL {
		return Envelope{}, false
	}
	return e.value, true
}

func (m *memoCache) set(key string, env Envelope, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.entries == nil {
		m.entries = map[string]memoEntry{}
	}
	m.entries[key] = memoEntry{value: env, fetchedAt: now}
}

// Service ties together the forensics subsystems. Construct via NewService
// and call the per-endpoint methods on it.
type Service struct {
	journal *journal.Reader
	exec    CommandExecutor
	fs      FileSystem
	now     func() time.Time
	cache   memoCache

	// PstoreDir is the path to inspect for pstore artifacts. Defaults to
	// /sys/fs/pstore. Overridable for tests.
	PstoreDir string
}

// NewService builds a forensics Service.
func NewService(j *journal.Reader, exec CommandExecutor, fsys FileSystem, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		journal:   j,
		exec:      exec,
		fs:        fsys,
		now:       now,
		PstoreDir: "/sys/fs/pstore",
	}
}
