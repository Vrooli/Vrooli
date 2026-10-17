package lifecycle

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/tuning"
)

// Lifecycle run logs are retained outside the logs root. The logs root is a
// regenerable runtime-home entry, and storage-manager's disk-pressure
// recovery reclaims it without an age floor: on 2026-10-07 (91% disk) it
// emptied ~/.vrooli/logs every recovery cycle, so a failed live UI build left
// no log (O18). The last runs of each instance are therefore copied into the
// contract-protected state root, bounded by count and by size per run.
const (
	lifecycleRunLogDir = "lifecycle-runs"
	// lifecycleRunLogKeep is how many runs are kept per instance.
	lifecycleRunLogKeep = 10
	// lifecycleRunLogHeadBytes and lifecycleRunLogTailBytes bound one run:
	// its first and last bytes are kept, the middle is dropped.
	lifecycleRunLogHeadBytes = 2 << 20
	lifecycleRunLogTailBytes = 1 << 20
)

// LifecycleRunLogsDir resolves the retained run logs of one instance:
// <home>/.vrooli/state/lifecycle-runs/<instance>.
func LifecycleRunLogsDir(home, instance string) (string, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, lifecycleRunLogDir, instance), nil
}

// retainedRunLog keeps the head and the tail of one run's output.
type retainedRunLog struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	written int64
	tail    []byte
	dropped int64
}

// openRetainedRunLog creates the retained log of one run. Retention is best
// effort: a nil result means the run proceeds without it.
func openRetainedRunLog(home string, ctx lifecycleLogContext, runID string) *retainedRunLog {
	if strings.TrimSpace(home) == "" {
		return nil
	}
	dir, err := LifecycleRunLogsDir(home, ctx.Scenario)
	if err != nil {
		return nil
	}
	if _, err := config.EnsureOwnedDir(dir); err != nil {
		return nil
	}
	name := sanitizeRunLogPart(runID) + "." + sanitizeRunLogPart(ctx.Operation) + "." + sanitizeRunLogPart(ctx.Phase) + ".log"
	path := filepath.Join(dir, name)
	file, err := config.OpenOwnedFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, tuning.PermFile)
	if err != nil {
		return nil
	}
	return &retainedRunLog{file: file, path: path}
}

func (l *retainedRunLog) Write(p []byte) (int, error) {
	if l == nil {
		return len(p), nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	rest := p
	if room := lifecycleRunLogHeadBytes - l.written; room > 0 {
		head := rest
		if int64(len(head)) > room {
			head = head[:room]
		}
		n, _ := l.file.Write(head)
		l.written += int64(n)
		rest = rest[len(head):]
	}
	if len(rest) > 0 {
		l.tail = append(l.tail, rest...)
		if over := len(l.tail) - lifecycleRunLogTailBytes; over > 0 {
			l.dropped += int64(over)
			l.tail = append(l.tail[:0], l.tail[over:]...)
		}
	}
	return len(p), nil
}

// close writes the kept tail and prunes the instance's older runs.
func (l *retainedRunLog) close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	if l.dropped > 0 {
		_, _ = fmt.Fprintf(l.file, "\n[vrooli] %d bytes of this run were omitted; the last %d bytes follow\n", l.dropped, len(l.tail))
	}
	_, _ = l.file.Write(l.tail)
	_ = l.file.Close()
	l.mu.Unlock()
	_ = config.ChownToInvokingUser(l.path)
	pruneRetainedRunLogs(filepath.Dir(l.path), lifecycleRunLogKeep)
}

// pruneRetainedRunLogs keeps the newest keep runs, by modification time
// (callers may supply their own run IDs, so names need not sort by time).
func pruneRetainedRunLogs(dir string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type runLog struct {
		name    string
		modTime int64
	}
	logs := make([]runLog, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		logs = append(logs, runLog{name: entry.Name(), modTime: info.ModTime().UnixNano()})
	}
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].modTime != logs[j].modTime {
			return logs[i].modTime < logs[j].modTime
		}
		return logs[i].name < logs[j].name
	})
	for len(logs) > keep {
		_ = os.Remove(filepath.Join(dir, logs[0].name))
		logs = logs[1:]
	}
}

func sanitizeRunLogPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.', r == '@':
			return r
		default:
			return '_'
		}
	}, value)
}

// teeRetained adds the retained run log to a lifecycle log writer.
func teeRetained(log io.Writer, retained *retainedRunLog) io.Writer {
	if retained == nil {
		return log
	}
	return io.MultiWriter(log, retained)
}
