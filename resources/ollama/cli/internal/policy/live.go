package policy

import (
	"os"
	"sync"
	"time"
)

// liveCheckInterval bounds how often a long-running reader stats the policy
// file. Edits take effect within this interval without a restart.
const liveCheckInterval = 5 * time.Second

// Live serves the policy file to a long-running process such as the activity
// edge. It reloads the file when its modification time or size changes and
// keeps the last valid policy when a reload fails, so a half-written edit
// never removes residency from live traffic.
type Live struct {
	Path string
	Logf func(format string, args ...any)
	Now  func() time.Time

	mu      sync.Mutex
	checked time.Time
	modTime time.Time
	size    int64
	policy  Policy
	loaded  bool
}

// NewLive returns a Live reader for path.
func NewLive(path string, logf func(format string, args ...any)) *Live {
	return &Live{Path: path, Logf: logf, Now: time.Now}
}

// Policy returns the current policy and whether one has loaded.
func (l *Live) Policy() (Policy, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.Now != nil {
		now = l.Now()
	}
	if l.loaded && now.Sub(l.checked) < liveCheckInterval {
		return l.policy, true
	}
	l.checked = now
	info, err := os.Stat(l.Path)
	if err != nil {
		l.logf("ollama model policy: stat %s: %v (keeping last loaded policy)\n", l.Path, err)
		return l.policy, l.loaded
	}
	if l.loaded && info.ModTime().Equal(l.modTime) && info.Size() == l.size {
		return l.policy, true
	}
	p, err := LoadFile(l.Path)
	if err != nil {
		l.logf("ollama model policy: reload %s: %v (keeping last loaded policy)\n", l.Path, err)
		return l.policy, l.loaded
	}
	l.policy, l.loaded, l.modTime, l.size = p, true, info.ModTime(), info.Size()
	return p, true
}

// KeepAliveSeconds applies the current policy's residency to model.
func (l *Live) KeepAliveSeconds(model string) (int, bool) {
	p, ok := l.Policy()
	if !ok {
		return 0, false
	}
	return p.KeepAliveSeconds(model)
}

func (l *Live) logf(format string, args ...any) {
	if l.Logf != nil {
		l.Logf(format, args...)
	}
}
