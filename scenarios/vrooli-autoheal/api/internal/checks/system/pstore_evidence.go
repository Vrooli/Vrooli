// Package system: pstore evidence check.
//
// /sys/fs/pstore holds whatever the kernel managed to persist before a panic
// or hard reset. Filenames distinguish severity:
//   - dmesg-*  / console-*  : real kernel-side crash dumps   → CRITICAL
//   - pmsg-*                 : userspace messages only        → WARNING
//   - empty or missing       : nothing to investigate         → OK
//
// We never delete entries — preserving forensic evidence is the entire point.
// The operator clears them manually after triage.
package system

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"runtime"
	"strings"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// PstoreEvidenceReader is the narrow seam this check needs from the OS.
// Defined here (not in checks/filesystem.go) because no other check needs
// directory listings yet — adding a method to the global FileSystemReader
// would force every existing test stub to implement it.
type PstoreEvidenceReader interface {
	ReadDir(path string) ([]fs.DirEntry, error)
}

type osPstoreReader struct{}

func (osPstoreReader) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// DefaultPstoreReader is overridden in tests.
var DefaultPstoreReader PstoreEvidenceReader = osPstoreReader{}

// PstoreEvidenceCheck reports kernel crash artifacts surviving in /sys/fs/pstore.
type PstoreEvidenceCheck struct {
	pstorePath string
	reader     PstoreEvidenceReader
}

// PstoreEvidenceCheckOption configures a PstoreEvidenceCheck.
type PstoreEvidenceCheckOption func(*PstoreEvidenceCheck)

// WithPstoreReader injects a custom directory reader.
func WithPstoreReader(r PstoreEvidenceReader) PstoreEvidenceCheckOption {
	return func(c *PstoreEvidenceCheck) { c.reader = r }
}

// WithPstorePath overrides the default /sys/fs/pstore path (for testing).
func WithPstorePath(path string) PstoreEvidenceCheckOption {
	return func(c *PstoreEvidenceCheck) { c.pstorePath = path }
}

// NewPstoreEvidenceCheck builds a PstoreEvidenceCheck with sensible defaults.
func NewPstoreEvidenceCheck(opts ...PstoreEvidenceCheckOption) *PstoreEvidenceCheck {
	c := &PstoreEvidenceCheck{
		pstorePath: "/sys/fs/pstore",
		reader:     DefaultPstoreReader,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PstoreEvidenceCheck) ID() string    { return "system-pstore-evidence" }
func (c *PstoreEvidenceCheck) Title() string { return "Crash Dump Evidence (pstore)" }
func (c *PstoreEvidenceCheck) Description() string {
	return "Surfaces kernel panic / oops artifacts persisted in /sys/fs/pstore"
}

func (c *PstoreEvidenceCheck) Importance() string {
	return "Crash dumps are the only diagnostic record after an unclean reset; missing them means future hard resets will be opaque"
}
func (c *PstoreEvidenceCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *PstoreEvidenceCheck) IntervalSeconds() int       { return 300 }
func (c *PstoreEvidenceCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *PstoreEvidenceCheck) Run(ctx context.Context) checks.Result {
	r := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	if runtime.GOOS != "linux" {
		r.Status = checks.StatusOK
		r.Message = "pstore is Linux-only"
		r.Details["platform"] = runtime.GOOS
		return r
	}

	entries, err := c.reader.ReadDir(c.pstorePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		r.Status = checks.StatusOK
		r.Message = "pstore not configured (no /sys/fs/pstore)"
		r.Details["pstoreConfigured"] = false
		return r
	case errors.Is(err, fs.ErrPermission):
		r.Status = checks.StatusWarning
		r.Message = "pstore unreadable (EACCES) — autoheal needs read access to surface crash artifacts"
		r.Details["error"] = err.Error()
		return r
	case err != nil:
		r.Status = checks.StatusWarning
		r.Message = "Failed to read pstore: " + err.Error()
		r.Details["error"] = err.Error()
		return r
	}

	var crashDumps, consoleDumps, userspace []string
	for _, e := range entries {
		name := e.Name()
		switch {
		case strings.HasPrefix(name, "dmesg-"):
			crashDumps = append(crashDumps, name)
		case strings.HasPrefix(name, "console-"):
			consoleDumps = append(consoleDumps, name)
		case strings.HasPrefix(name, "pmsg-"):
			userspace = append(userspace, name)
		}
	}

	r.Details["pstoreConfigured"] = true
	r.Details["dmesgCount"] = len(crashDumps)
	r.Details["consoleCount"] = len(consoleDumps)
	r.Details["pmsgCount"] = len(userspace)
	if len(crashDumps) > 0 {
		r.Details["dmesgEntries"] = crashDumps
	}
	if len(consoleDumps) > 0 {
		r.Details["consoleEntries"] = consoleDumps
	}

	switch {
	case len(crashDumps) > 0 || len(consoleDumps) > 0:
		r.Status = checks.StatusCritical
		r.Message = "Kernel crash artifacts present in pstore — investigate before clearing"
	case len(userspace) > 0:
		r.Status = checks.StatusWarning
		r.Message = "Userspace pstore artifacts present (pmsg-*); no kernel crash recorded"
	default:
		r.Status = checks.StatusOK
		r.Message = "No crash artifacts in pstore"
	}
	r.Timestamp = time.Now()
	return r
}
