// Package system: kernel panic evidence check.
//
// kdump writes a complete record of a panic to /var/crash, but writes it where
// no scenario can read it — the dmesg is mode 0600 root and the vmcore beside
// it is roughly the size of system RAM. On 2026-08-19 this host panicked in the
// ntfs3 write path, kdump captured 5.8 GB of evidence, and nothing in Vrooli
// noticed; the cause had to be recovered by hand.
//
// The kdump_observability safeguard exports a bounded summary into the shared
// host-observability directory. This check reads that summary and reports the
// panic, so a crash becomes a visible incident rather than a silent reboot.
package system

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostobservability"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// CrashDumpManifestReader is the narrow seam this check needs. It is an
// interface so tests can supply a manifest without a filesystem.
type CrashDumpManifestReader interface {
	ReadCrashDumpManifest(exportDir string) (hostobservability.CrashDumpManifest, error)
}

type osCrashDumpReader struct{}

func (osCrashDumpReader) ReadCrashDumpManifest(exportDir string) (hostobservability.CrashDumpManifest, error) {
	return hostobservability.ReadCrashDumpManifest(exportDir)
}

// PanicEvidenceCheck reports kernel panics captured by kdump.
type PanicEvidenceCheck struct {
	exportPath string
	reader     CrashDumpManifestReader
	// retainedBytesWarn is the retained-vmcore size above which the check warns
	// on its own. A crash directory that grows unbounded is a second outage
	// waiting to happen, and this host has filled its root filesystem before.
	retainedBytesWarn int64
}

// PanicEvidenceCheckOption configures a PanicEvidenceCheck.
type PanicEvidenceCheckOption func(*PanicEvidenceCheck)

// WithCrashDumpReader injects a manifest reader.
func WithCrashDumpReader(r CrashDumpManifestReader) PanicEvidenceCheckOption {
	return func(c *PanicEvidenceCheck) { c.reader = r }
}

// WithCrashDumpExportPath overrides the export directory.
func WithCrashDumpExportPath(path string) PanicEvidenceCheckOption {
	return func(c *PanicEvidenceCheck) { c.exportPath = path }
}

// WithRetainedBytesWarning overrides the retained-vmcore warning threshold.
func WithRetainedBytesWarning(bytes int64) PanicEvidenceCheckOption {
	return func(c *PanicEvidenceCheck) { c.retainedBytesWarn = bytes }
}

// NewPanicEvidenceCheck builds a PanicEvidenceCheck with sensible defaults.
func NewPanicEvidenceCheck(opts ...PanicEvidenceCheckOption) *PanicEvidenceCheck {
	c := &PanicEvidenceCheck{
		exportPath:        hostobservability.CrashDumpExportPath(),
		reader:            osCrashDumpReader{},
		retainedBytesWarn: 32 << 30, // 32 GiB of retained dumps is worth surfacing
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PanicEvidenceCheck) ID() string    { return "system-panic-evidence" }
func (c *PanicEvidenceCheck) Title() string { return "Kernel Panic Evidence (kdump)" }
func (c *PanicEvidenceCheck) Description() string {
	return "Surfaces kernel panics captured by kdump via the host-observability crash dump export"
}

func (c *PanicEvidenceCheck) Importance() string {
	return "A panic is the most severe thing a host can do; without this the only record sits in a root-only directory nothing reads"
}
func (c *PanicEvidenceCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *PanicEvidenceCheck) IntervalSeconds() int       { return 300 }
func (c *PanicEvidenceCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *PanicEvidenceCheck) Run(ctx context.Context) (r checks.Result) {
	r = checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	defer func() {
		if r.Timestamp.IsZero() {
			r.Timestamp = time.Now()
		}
	}()
	if checkOS != "linux" {
		r.Status = checks.StatusNotApplicable
		r.Message = "kdump is Linux-only"
		r.Details["platform"] = checkOS
		return r
	}

	r.Details["exportPath"] = c.exportPath
	manifest, err := c.reader.ReadCrashDumpManifest(c.exportPath)
	if err != nil {
		// A missing export is a coverage gap, not a clean bill of health. Saying
		// "no panics" when nothing has been collected is exactly the failure
		// this check exists to end.
		r.Status = checks.StatusWarning
		r.Details["exportReadable"] = false
		r.Details["error"] = err.Error()
		r.Details["coverageGap"] = true
		switch {
		case errors.Is(err, fs.ErrNotExist):
			r.Message = "Panic evidence coverage gap: no kdump summary export found"
			r.Details["coverageGapReason"] = "crashdump_export_missing"
		case errors.Is(err, fs.ErrPermission):
			r.Message = "Panic evidence coverage gap: kdump summary export is not readable by this user"
			r.Details["coverageGapReason"] = "crashdump_export_permission_denied"
		default:
			r.Message = "Panic evidence coverage gap: kdump summary export could not be read"
			r.Details["coverageGapReason"] = "crashdump_export_unreadable"
		}
		r.Details["recommendations"] = []string{
			"run project setup with sudo to apply the kdump_observability safeguard",
			"confirm the runtime user is in the vrooli-observability group after a new login session",
		}
		return r
	}

	r.Details["exportReadable"] = true
	r.Details["collectedAt"] = manifest.CollectedAt
	r.Details["dumpCount"] = len(manifest.Dumps)
	r.Details["retainedBytes"] = manifest.TotalBytes()
	r.Details["retainVmcores"] = manifest.RetainVmcores

	newest, ok := manifest.Newest()
	if !ok {
		if manifest.TotalBytes() > c.retainedBytesWarn {
			// Defensive: no dumps but disk held means the manifest and the
			// directory disagree, which is worth a look.
			r.Status = checks.StatusWarning
			r.Message = "kdump export reports no panics but retains crash data"
			return r
		}
		r.Status = checks.StatusOK
		r.Message = "No kernel panics captured"
		return r
	}

	r.Details["latestStamp"] = newest.Stamp
	r.Details["latestSummary"] = newest.Summary
	r.Details["latestReason"] = newest.Reason
	r.Details["latestComm"] = newest.Comm

	if manifest.TotalBytes() > c.retainedBytesWarn {
		r.Details["retentionPressure"] = true
		r.Details["recommendations"] = []string{
			"lower retain_vmcores on the kdump_observability safeguard, or archive the dumps you still need",
		}
	}

	r.Status = checks.StatusCritical
	r.Message = "Kernel panic captured by kdump — investigate before clearing"
	if newest.Reason != "" {
		r.Message = "Kernel panic captured by kdump: " + newest.Reason
	}
	return r
}

// ensure the check satisfies the registry contract at compile time.
var _ checks.Check = (*PanicEvidenceCheck)(nil)
