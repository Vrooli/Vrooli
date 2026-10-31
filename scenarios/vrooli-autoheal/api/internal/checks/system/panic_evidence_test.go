package system

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostobservability"
)

type stubCrashDumpReader struct {
	manifest hostobservability.CrashDumpManifest
	err      error
}

func (s stubCrashDumpReader) ReadCrashDumpManifest(string) (hostobservability.CrashDumpManifest, error) {
	return s.manifest, s.err
}

func runPanicCheck(t *testing.T, opts ...PanicEvidenceCheckOption) checks.Result {
	t.Helper()
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "panic evidence check is Linux-only")
	}
	return NewPanicEvidenceCheck(opts...).Run(context.Background())
}

// The whole point of this check: a captured panic must surface as critical,
// carrying the banner and faulting command an incident report leads with.
func TestPanicEvidenceReportsCapturedPanic(t *testing.T) {
	manifest := hostobservability.CrashDumpManifest{
		RetainVmcores: 2,
		Dumps: []hostobservability.CrashDump{{
			Stamp:   "202608191459",
			Summary: "202608191459.dmesg",
			Reason:  "kernel BUG at fs/iomap/buffered-io.c:1061!",
			Comm:    "kopia",
			Bytes:   5_783_371_785,
		}},
	}
	r := runPanicCheck(t, WithCrashDumpReader(stubCrashDumpReader{manifest: manifest}))

	if r.Status != checks.StatusCritical {
		t.Fatalf("status = %v, want critical", r.Status)
	}
	if !strings.Contains(r.Message, "fs/iomap/buffered-io.c:1061") {
		t.Errorf("message should carry the panic banner, got %q", r.Message)
	}
	if r.Details["latestComm"] != "kopia" {
		t.Errorf("latestComm = %v, want kopia", r.Details["latestComm"])
	}
	if r.Details["latestStamp"] != "202608191459" {
		t.Errorf("latestStamp = %v", r.Details["latestStamp"])
	}
}

// An absent export means nothing has been collected. Reporting that as "no
// panics" is the exact failure this check exists to end.
func TestMissingExportIsACoverageGapNotHealth(t *testing.T) {
	r := runPanicCheck(t, WithCrashDumpReader(stubCrashDumpReader{err: fs.ErrNotExist}))

	if r.Status != checks.StatusWarning {
		t.Fatalf("status = %v, want warning", r.Status)
	}
	if r.Details["coverageGap"] != true {
		t.Error("a missing export must be marked as a coverage gap")
	}
	if r.Details["coverageGapReason"] != "crashdump_export_missing" {
		t.Errorf("coverageGapReason = %v", r.Details["coverageGapReason"])
	}
}

// A permission error is a distinct, actionable condition: the safeguard ran but
// this user is not in the observability group yet.
func TestPermissionDeniedIsDistinguished(t *testing.T) {
	r := runPanicCheck(t, WithCrashDumpReader(stubCrashDumpReader{err: fs.ErrPermission}))

	if r.Details["coverageGapReason"] != "crashdump_export_permission_denied" {
		t.Fatalf("coverageGapReason = %v", r.Details["coverageGapReason"])
	}
	recs, ok := r.Details["recommendations"].([]string)
	if !ok || len(recs) == 0 {
		t.Fatal("a coverage gap should carry recommendations")
	}
}

func TestNoPanicsIsHealthy(t *testing.T) {
	r := runPanicCheck(t, WithCrashDumpReader(stubCrashDumpReader{
		manifest: hostobservability.CrashDumpManifest{RetainVmcores: 2},
	}))

	if r.Status != checks.StatusOK {
		t.Fatalf("status = %v, want ok", r.Status)
	}
}

// Retained vmcores are each roughly the size of RAM. Growth is worth surfacing
// alongside the panic, because filling the root filesystem is its own outage.
func TestRetentionPressureIsSurfaced(t *testing.T) {
	manifest := hostobservability.CrashDumpManifest{
		Dumps: []hostobservability.CrashDump{
			{Stamp: "202608191459", Reason: "Oops: invalid opcode", Bytes: 40 << 30},
		},
	}
	r := runPanicCheck(t,
		WithCrashDumpReader(stubCrashDumpReader{manifest: manifest}),
		WithRetainedBytesWarning(32<<30),
	)

	if r.Details["retentionPressure"] != true {
		t.Fatalf("expected retention pressure to be flagged; details: %v", r.Details)
	}
	if r.Status != checks.StatusCritical {
		t.Errorf("a real panic still outranks retention pressure, got %v", r.Status)
	}
}

// End-to-end against a real manifest file, so the collector's JSON shape and the
// reader's expectations cannot drift apart silently.
func TestReadsARealManifestFile(t *testing.T) {
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "panic evidence check is Linux-only")
	}
	dir := t.TempDir()
	body := map[string]any{
		"collectedAt":   "2026-08-19T19:30:00Z",
		"sourcePath":    "/var/crash",
		"retainVmcores": 2,
		"dumpCount":     2,
		"dumps": []map[string]any{
			{"stamp": "202608180101", "summary": "202608180101.dmesg", "reason": "Oops: invalid opcode", "comm": "node", "bytes": 1024},
			{"stamp": "202608191459", "summary": "202608191459.dmesg", "reason": "kernel BUG at fs/iomap/buffered-io.c:1061!", "comm": "kopia", "bytes": 2048},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, hostobservability.ManifestFilename), raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	r := NewPanicEvidenceCheck(WithCrashDumpExportPath(dir)).Run(context.Background())

	if r.Status != checks.StatusCritical {
		t.Fatalf("status = %v, want critical; details %v", r.Status, r.Details)
	}
	// Newest first, regardless of the order the collector emitted.
	if r.Details["latestStamp"] != "202608191459" {
		t.Errorf("latestStamp = %v, want the newest crash", r.Details["latestStamp"])
	}
	if r.Details["latestComm"] != "kopia" {
		t.Errorf("latestComm = %v", r.Details["latestComm"])
	}
}
