//go:build linux

package procsampler

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

// procFS abstracts the /proc reads the Linux sampler performs so unit tests can
// inject a fixture tree (a real directory under t.TempDir, or an in-memory
// fake) instead of the host's live /proc. All reads tolerate ENOENT/ESRCH:
// a process that exits between listing and reading is simply skipped.
type procFS interface {
	// PIDs lists the numeric process directories under /proc.
	PIDs() ([]int, error)
	// ReadFile reads /proc/<pid>/<name> (e.g. "stat", "comm", "cmdline").
	ReadFile(pid int, name string) ([]byte, error)
	// Readlink resolves a /proc/<pid>/<name> symlink (e.g. "cwd").
	Readlink(pid int, name string) (string, error)
}

// osProcFS reads the host's real /proc.
type osProcFS struct{ root string }

func (f osProcFS) base() string {
	if f.root != "" {
		return f.root
	}
	return "/proc"
}

func (f osProcFS) PIDs() ([]int, error) {
	entries, err := os.ReadDir(f.base())
	if err != nil {
		return nil, err
	}
	pids := make([]int, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // non-pid entries like "self", "cpuinfo"
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func (f osProcFS) ReadFile(pid int, name string) ([]byte, error) {
	return os.ReadFile(f.base() + "/" + strconv.Itoa(pid) + "/" + name)
}

func (f osProcFS) Readlink(pid int, name string) (string, error) {
	return os.Readlink(f.base() + "/" + strconv.Itoa(pid) + "/" + name)
}

// linuxSampler is the production /proc walk.
type linuxSampler struct {
	fs    procFS
	delta *cpuDeltaTracker
	now   func() time.Time
}

// NewSampler returns the Linux /proc sampler reading the host's real /proc.
func NewSampler() Sampler {
	return &linuxSampler{
		fs:    osProcFS{},
		delta: newCPUDeltaTracker(),
		now:   time.Now,
	}
}

// newSamplerWithFS builds a sampler over an injected procFS (tests).
func newSamplerWithFS(fs procFS, now func() time.Time) *linuxSampler {
	if now == nil {
		now = time.Now
	}
	return &linuxSampler{fs: fs, delta: newCPUDeltaTracker(), now: now}
}

// Sample walks /proc once. Errors reading a single process are swallowed
// (the process likely exited mid-walk — ESRCH); only a failure to list /proc
// itself is returned.
func (s *linuxSampler) Sample(ctx context.Context) ([]ProcessSample, error) {
	pids, err := s.fs.PIDs()
	if err != nil {
		return nil, err
	}

	samples := make([]ProcessSample, 0, len(pids))
	for _, pid := range pids {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		sample, ok := s.readProcess(pid)
		if !ok {
			continue
		}
		samples = append(samples, sample)
	}

	s.delta.apply(samples, s.now())
	sortByCPUDesc(samples)
	return samples, nil
}

// readProcess assembles one ProcessSample from /proc/<pid>/{stat,cmdline,cwd}.
// It returns ok=false when the process disappears or the stat line is
// unparseable.
func (s *linuxSampler) readProcess(pid int) (ProcessSample, bool) {
	statRaw, err := s.fs.ReadFile(pid, "stat")
	if err != nil {
		return ProcessSample{}, false
	}
	sample, ok := parseStat(string(statRaw))
	if !ok {
		return ProcessSample{}, false
	}
	sample.PID = pid

	// cmdline: nul-separated args; the comm in stat is already the short name.
	if cmdRaw, err := s.fs.ReadFile(pid, "cmdline"); err == nil {
		sample.Cmdline = renderCmdline(cmdRaw)
	}
	// cwd: best-effort symlink resolution (requires permission; root sees all).
	if cwd, err := s.fs.Readlink(pid, "cwd"); err == nil {
		sample.Cwd = cwd
	}
	return sample, true
}

// parseStat parses /proc/<pid>/stat. The comm field (2nd) is wrapped in
// parentheses and may itself contain spaces or parentheses, so we split on the
// last ')' to find the start of the space-delimited fields that follow.
//
// Field indices after comm (0-based from the field *after* the closing paren):
//
//	[0]=state [1]=ppid ... [11]=utime [12]=stime ... [17]=num_threads ... [21]=rss(pages)
//
// In /proc terms these are the documented fields 3,4,14,15,20,24.
func parseStat(line string) (ProcessSample, bool) {
	line = strings.TrimSpace(line)
	openParen := strings.IndexByte(line, '(')
	closeParen := strings.LastIndexByte(line, ')')
	if openParen < 0 || closeParen < 0 || closeParen < openParen {
		return ProcessSample{}, false
	}

	comm := line[openParen+1 : closeParen]
	rest := strings.Fields(line[closeParen+1:])
	// We need up to field index 21 (rss). Require enough fields.
	if len(rest) < 22 {
		return ProcessSample{}, false
	}

	sample := ProcessSample{Comm: comm}
	sample.PPID = atoiSafe(rest[1])
	sample.utime = atouSafe(rest[11])
	sample.stime = atouSafe(rest[12])
	sample.Threads = atoiSafe(rest[17])

	// rss is in pages; convert to KiB using the page size.
	rssPages := atoiSafe(rest[21])
	sample.RSSKB = int64(rssPages) * int64(pageSizeKB())
	return sample, true
}

// renderCmdline turns the nul-separated /proc cmdline into a space-joined
// string. A kernel thread has an empty cmdline; callers fall back to comm.
func renderCmdline(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	parts := strings.Split(strings.TrimRight(string(raw), "\x00"), "\x00")
	return strings.Join(parts, " ")
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atouSafe(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n
}

// pageSizeKB returns the system page size in KiB (4 on every common Linux
// configuration). os.Getpagesize is cgo-free.
func pageSizeKB() int {
	ps := os.Getpagesize()
	if ps <= 0 {
		return 4
	}
	return ps / 1024
}
