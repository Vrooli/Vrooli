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

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostobservability"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
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
	pstorePath     string
	exportPath     string
	reader         PstoreEvidenceReader
	exportOverride bool
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

// WithPstoreExportPath overrides the host-observability export path.
func WithPstoreExportPath(path string) PstoreEvidenceCheckOption {
	return func(c *PstoreEvidenceCheck) {
		c.exportPath = path
		c.exportOverride = true
	}
}

// NewPstoreEvidenceCheck builds a PstoreEvidenceCheck with sensible defaults.
func NewPstoreEvidenceCheck(opts ...PstoreEvidenceCheckOption) *PstoreEvidenceCheck {
	exportPath := strings.TrimSpace(os.Getenv(hostobservability.EnvPstoreExportDir))
	if exportPath == "" {
		exportPath = hostobservability.PstoreExportDir
	}
	c := &PstoreEvidenceCheck{
		pstorePath: hostobservability.PstoreSourceDir,
		exportPath: exportPath,
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

func (c *PstoreEvidenceCheck) Run(ctx context.Context) (r checks.Result) {
	r = checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	defer func() {
		if r.Timestamp.IsZero() {
			r.Timestamp = time.Now()
		}
	}()
	if runtime.GOOS != "linux" {
		r.Status = checks.StatusOK
		r.Message = "pstore is Linux-only"
		r.Details["platform"] = runtime.GOOS
		return r
	}

	entries, sourcePath, sourceKind, directErr, exportErr := c.readBestSource()
	r.Details["pstoreConfigured"] = !errors.Is(directErr, fs.ErrNotExist)
	r.Details["pstoreMounted"] = !errors.Is(directErr, fs.ErrNotExist)
	r.Details["directReadable"] = directErr == nil
	r.Details["exportConfigured"] = c.exportOverride || exportErr == nil || errors.Is(exportErr, fs.ErrPermission)
	r.Details["exportReadable"] = exportErr == nil
	exportFresh := sourceKind != "export" || hasPstoreManifest(entries)
	r.Details["exportFresh"] = exportFresh
	r.Details["sourcePath"] = sourcePath
	r.Details["sourceKind"] = sourceKind
	if directErr != nil {
		r.Details["directError"] = directErr.Error()
	}
	if exportErr != nil {
		r.Details["exportError"] = exportErr.Error()
	}
	if directErr != nil && exportErr != nil && !errors.Is(directErr, fs.ErrNotExist) {
		r.Status = checks.StatusWarning
		r.Message = "Crash artifact coverage gap: pstore is active but autoheal cannot read a direct or exported source"
		r.Details["coverageGap"] = true
		r.Details["coverageGapReason"] = coverageGapReason(directErr, exportErr)
		r.Details["recommendations"] = []string{
			"run project setup with sudo to apply the pstore_observability safeguard",
			"confirm the runtime user is in the vrooli-observability group after a new login session",
		}
		return r
	}
	if errors.Is(directErr, fs.ErrNotExist) && exportErr != nil {
		r.Status = checks.StatusOK
		r.Message = "pstore not configured"
		r.Details["pstoreConfigured"] = false
		return r
	}
	if sourcePath == "" {
		r.Status = checks.StatusWarning
		r.Message = "Failed to read pstore: no readable direct or exported source"
		r.Details["coverageGap"] = true
		r.Details["coverageGapReason"] = "no_readable_source"
		return r
	}
	if sourceKind == "export" && directErr != nil && !exportFresh {
		r.Status = checks.StatusWarning
		r.Message = "Crash artifact coverage gap: pstore export is readable but no collector manifest is present"
		r.Details["coverageGap"] = true
		r.Details["coverageGapReason"] = "pstore_export_manifest_missing"
		r.Details["recommendations"] = []string{
			"run project setup with sudo to apply the pstore_observability safeguard",
			"check systemctl status vrooli-pstore-collector.service",
		}
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
	r.Details["artifactCount"] = len(crashDumps) + len(consoleDumps) + len(userspace)
	if len(crashDumps) > 0 {
		r.Details["dmesgEntries"] = crashDumps
	}
	if len(consoleDumps) > 0 {
		r.Details["consoleEntries"] = consoleDumps
	}

	switch {
	case len(crashDumps) > 0 || len(consoleDumps) > 0:
		r.Status = checks.StatusCritical
		r.Message = "Kernel crash artifacts present in pstore export/direct source — investigate before clearing"
	case len(userspace) > 0:
		r.Status = checks.StatusWarning
		r.Message = "Userspace pstore artifacts present (pmsg-*); no kernel crash recorded"
	default:
		r.Status = checks.StatusOK
		r.Message = "No crash artifacts in pstore"
	}
	return r
}

func (c *PstoreEvidenceCheck) readBestSource() ([]fs.DirEntry, string, string, error, error) {
	exportEntries, exportErr := c.reader.ReadDir(c.exportPath)
	directEntries, directErr := c.reader.ReadDir(c.pstorePath)
	if exportErr == nil {
		return exportEntries, c.exportPath, "export", directErr, nil
	}
	if directErr == nil {
		return directEntries, c.pstorePath, "direct", nil, exportErr
	}
	return nil, "", "", directErr, exportErr
}

func coverageGapReason(directErr, exportErr error) string {
	switch {
	case errors.Is(directErr, fs.ErrPermission) && errors.Is(exportErr, fs.ErrNotExist):
		return "direct_pstore_permission_denied_export_missing"
	case errors.Is(directErr, fs.ErrPermission) && errors.Is(exportErr, fs.ErrPermission):
		return "direct_and_export_permission_denied"
	case errors.Is(directErr, fs.ErrPermission):
		return "direct_pstore_permission_denied_export_unreadable"
	default:
		return "pstore_unreadable"
	}
}

func hasPstoreManifest(entries []fs.DirEntry) bool {
	for _, entry := range entries {
		if entry.Name() == hostobservability.ManifestFilename {
			return true
		}
	}
	return false
}
