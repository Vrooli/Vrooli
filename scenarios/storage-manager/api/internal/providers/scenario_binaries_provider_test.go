package providers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/artifactlease"

	"github.com/vrooli/vrooli/packages/artifactledger"

	"storage-manager/internal/cleanup"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

type fakeLiveness struct {
	running map[string]bool
}

func (f *fakeLiveness) IsRunning(_ context.Context, path string) (bool, error) {
	return f.running[path], nil
}

func scenarioBinaryFixture(t *testing.T, running bool) (*ScenarioBinariesProvider, *cleanupfakes.FileSystem, string) {
	t.Helper()
	root, binary, fsys := scenarioBinaryFilesystem(t, running)
	provider := NewScenarioBinariesProvider(
		fsys,
		&fakeLiveness{running: map[string]bool{binary: running}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root, Ledger: artifactledger.NewAt(filepath.Join(t.TempDir(), "receipts"))},
	)
	return provider, fsys, binary
}

// scenarioBinaryFilesystem builds the fake install root without binding it to a
// provider, so a test can supply its own ledger or omit one entirely.
func scenarioBinaryFilesystem(t *testing.T, running bool) (string, string, *cleanupfakes.FileSystem) {
	t.Helper()
	_ = running
	// The artifacts are real files even though the directory listing is faked.
	// Removal goes through the attribution seam, which takes a real advisory
	// lock on the artifact family -- a fake filesystem cannot be locked, and
	// pretending otherwise would test a path production never takes.
	root := t.TempDir()
	binary := filepath.Join(root, "rcl-fixture-positive")
	metadata := binary + ".build.meta"
	manifest := binary + ".manifest.json"
	module := "/fake/scenarios/rcl-fixture-positive/cli"
	old := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	files := map[string]cleanup.FileInfo{
		root:     {Path: root, IsDir: true},
		binary:   {Path: binary, Size: 100, ModTime: old},
		metadata: {Path: metadata, Size: 50, ModTime: old},
		manifest: {Path: manifest, Size: 25, ModTime: old},
	}
	contents, _ := json.Marshal(map[string]string{"kind": "scenario", "module_path": module})
	for _, path := range []string{binary, metadata, manifest} {
		if err := os.WriteFile(path, make([]byte, files[path].Size), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fsys := &cleanupfakes.FileSystem{Root: root, Files: files, Contents: map[string][]byte{metadata: contents}, AllowRemove: true}
	return root, binary, fsys
}

// fixtureStart is the clock the scenario-binary fixtures run on.
var fixtureStart = time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

// previewAt runs a preview with an explicit clock, so a test can advance time
// without touching lease files by hand.
func previewAt(t *testing.T, provider *ScenarioBinariesProvider, now time.Time) cleanup.Preview {
	t.Helper()
	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{
		Scope:  cleanup.ObservationScope{Now: now},
		Policy: scenarioBinaryPolicy(),
	})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	return preview
}

// reclaimablePreview drives an artifact through the full grace path using the
// provider's own Preview.
//
// An absence must be observed at least twice and must have persisted for the
// grace window before anything may be reclaimed. Driving that through the real
// code rather than writing lease files by hand means these tests exercise the
// gate they are meant to be protected by.
func reclaimablePreview(t *testing.T, provider *ScenarioBinariesProvider) cleanup.Preview {
	t.Helper()
	previewAt(t, provider, fixtureStart)
	// The second observation uses the provider clock, which Apply also reads --
	// so the plan and its application agree about what time it is.
	preview := previewAt(t, provider, time.Time{})
	if len(preview.Items) == 0 {
		t.Fatalf("fixture did not become reclaimable after the full grace path; warnings=%v", preview.Warnings)
	}
	return preview
}

func scenarioBinaryPolicy() cleanup.ProviderPolicy {
	return cleanup.ProviderPolicy{Enabled: true, MinAge: 24 * time.Hour, ApprovalMode: cleanup.ApprovalModeOwner}
}

func TestScenarioBinariesProviderReportsOrphanedTriple(t *testing.T) {
	provider, _, binary := scenarioBinaryFixture(t, false)
	preview := reclaimablePreview(t, provider)
	if len(preview.Items) != 1 || preview.Items[0].Path != binary {
		t.Fatalf("preview items = %#v, want one triple rooted at %q", preview.Items, binary)
	}
	if preview.Items[0].Bytes != 175 {
		t.Fatalf("preview bytes = %d, want 175", preview.Items[0].Bytes)
	}
}

func TestScenarioBinariesProviderSkipsRunningBinaryWithWarning(t *testing.T) {
	provider, _, _ := scenarioBinaryFixture(t, true)
	previewAt(t, provider, fixtureStart)
	preview := previewAt(t, provider, time.Time{})
	if len(preview.Items) != 0 {
		t.Fatalf("running binary was reclaimable: %#v", preview.Items)
	}
	if len(preview.Warnings) != 1 || preview.Warnings[0] == "" {
		t.Fatalf("running binary warning = %#v", preview.Warnings)
	}
}

func TestScenarioBinariesProviderApplyRemovesTriple(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	preview := reclaimablePreview(t, provider)
	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "apply-1",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.ReclaimedBytes != 175 || len(fsys.Removed) != 3 {
		t.Fatalf("apply result = %#v removed=%#v, want 175 bytes and 3 artifacts", result, fsys.Removed)
	}
	if fsys.Removed[0] != binary {
		t.Fatalf("first removed path = %q, want binary %q", fsys.Removed[0], binary)
	}
}

func TestScenarioBinariesProviderIgnoresNonScenarioMetadata(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	metadata := binary + ".build.meta"
	contents, _ := json.Marshal(map[string]string{"kind": "resource", "module_path": "/missing/resource/cli"})
	fsys.Contents[metadata] = contents

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("non-scenario metadata was reported: %#v", preview.Items)
	}
}

func TestScenarioBinariesProviderIgnoresMissingModulePath(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)
	metadata := binary + ".build.meta"
	contents, _ := json.Marshal(map[string]string{"kind": "scenario"})
	fsys.Contents[metadata] = contents

	preview, err := provider.Preview(context.Background(), cleanup.PreviewRequest{Policy: scenarioBinaryPolicy()})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview.Items) != 0 {
		t.Fatalf("missing module_path was reported: %#v", preview.Items)
	}
}

// [REQ:VROOLI-ARTIFACT-ATTRIBUTION]
// This reaper decides for itself what is an orphan, which makes it the removal
// hardest to reconstruct after the fact. Every artifact it deletes must leave a
// receipt naming the rule that fired.
func TestScenarioBinariesProviderWritesReceiptsForEveryRemoval(t *testing.T) {
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	root, binary, fsys := scenarioBinaryFilesystem(t, false)
	provider := NewScenarioBinariesProvider(
		fsys,
		&fakeLiveness{running: map[string]bool{}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root, Ledger: artifactledger.NewAt(receiptDir)},
	)

	preview := reclaimablePreview(t, provider)
	if _, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "apply-receipts",
		Preview:         preview,
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	receipts, err := artifactledger.NewAt(receiptDir).Read()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	removed := 0
	sawBinary := false
	for _, receipt := range receipts {
		if receipt.Outcome != artifactledger.OutcomeRemoved {
			continue
		}
		removed++
		if receipt.Predicate == "" {
			t.Fatalf("receipt records a reclamation without its rule: %+v", receipt)
		}
		if receipt.Component != "storage-manager.ScenarioBinariesProvider" {
			t.Fatalf("receipt does not name the reaper: %+v", receipt)
		}
		if receipt.Path == binary {
			sawBinary = true
		}
	}
	if removed != 3 {
		t.Fatalf("got %d removal receipts, want one per triple member", removed)
	}
	if !sawBinary {
		t.Fatalf("no receipt named the reclaimed binary %q", binary)
	}
}

// A provider with no ledger must refuse rather than reclaim unrecorded.
func TestScenarioBinariesProviderRefusesWithoutALedger(t *testing.T) {
	root, _, fsys := scenarioBinaryFilesystem(t, false)
	withLedger := NewScenarioBinariesProvider(fsys, &fakeLiveness{running: map[string]bool{}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root, Ledger: artifactledger.NewAt(filepath.Join(t.TempDir(), "r"))})
	preview := reclaimablePreview(t, withLedger)

	unledgered := NewScenarioBinariesProvider(fsys, &fakeLiveness{running: map[string]bool{}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root})
	result, err := unledgered.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: unledgered.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "apply-no-ledger",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.Applied || result.ReclaimedBytes != 0 || len(fsys.Removed) != 0 {
		t.Fatalf("reclaimed without a ledger: %#v removed=%#v", result, fsys.Removed)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("refusal produced no warning explaining why nothing was reclaimed")
	}
}

// [REQ:VROOLI-ARTIFACT-SEAM]
// Preview decided this binary was an orphan; Apply runs later. Between the two,
// another agent can recreate the scenario that owns it -- concurrent agents are
// a design property here. Apply previously re-checked liveness but never the
// orphan predicate, so a freshly rebuilt CLI could be deleted on a stale
// observation.
func TestScenarioBinariesProviderAbandonsWhenTheOwnerReappears(t *testing.T) {
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	root, binary, fsys := scenarioBinaryFilesystem(t, false)
	provider := NewScenarioBinariesProvider(
		fsys,
		&fakeLiveness{running: map[string]bool{}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root, Ledger: artifactledger.NewAt(receiptDir)},
	)

	preview := reclaimablePreview(t, provider)

	// The owning scenario is regenerated after the plan was made.
	fsys.Files["/fake/scenarios/rcl-fixture-positive/cli"] = cleanup.FileInfo{
		Path:  "/fake/scenarios/rcl-fixture-positive/cli",
		IsDir: true,
	}

	result, err := provider.Apply(context.Background(), cleanup.ApplyRequest{
		ProviderVersion: provider.Metadata().Version,
		ApprovalMode:    cleanup.ApprovalModeOwner,
		IdempotencyKey:  "apply-owner-returned",
		Preview:         preview,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(fsys.Removed) != 0 {
		t.Fatalf("the reaper deleted a CLI whose scenario exists again: %#v", fsys.Removed)
	}
	if result.ReclaimedBytes != 0 {
		t.Fatalf("reclaimed %d bytes from a non-orphan", result.ReclaimedBytes)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("the binary should still be on disk: %v", err)
	}

	receipts, err := artifactledger.NewAt(receiptDir).Read()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	abandoned := 0
	for _, receipt := range receipts {
		if receipt.Outcome == artifactledger.OutcomeAbandoned {
			abandoned++
			if receipt.Error == "" {
				t.Fatalf("abandoned receipt does not say why: %+v", receipt)
			}
		}
		if receipt.Outcome == artifactledger.OutcomeRemoved {
			t.Fatalf("a removal was recorded for a restored owner: %+v", receipt)
		}
	}
	if abandoned == 0 {
		t.Fatalf("the guard fired but left no evidence; receipts=%+v", receipts)
	}
}

// [REQ:VROOLI-ARTIFACT-LEASE]
// The exit criterion for the lease work: a scenario that disappears and comes
// back inside the grace window keeps its CLI.
//
// Everything here is a fixture. The "scenario module" is an entry in a fake
// filesystem map and the "CLI" is a file in t.TempDir(); no real scenario, CLI,
// or install root is touched.
func TestScenarioRecreatedInsideGraceKeepsItsCLI(t *testing.T) {
	const module = "/fake/scenarios/rcl-fixture-positive/cli"
	root, binary, fsys := scenarioBinaryFilesystem(t, false)
	provider := NewScenarioBinariesProvider(
		fsys,
		&fakeLiveness{running: map[string]bool{}},
		cleanupfakes.Clock{Time: fixtureStart.Add(artifactlease.DefaultGrace + time.Hour)},
		ScenarioBinariesProviderConfig{Root: root, Ledger: artifactledger.NewAt(filepath.Join(t.TempDir(), "receipts"))},
	)

	// The owner vanishes and is observed missing.
	previewAt(t, provider, fixtureStart)

	// It is regenerated well inside the grace window.
	fsys.Files[module] = cleanup.FileInfo{Path: module, IsDir: true}
	previewAt(t, provider, fixtureStart.Add(time.Hour))

	// It vanishes again, and is observed once more.
	delete(fsys.Files, module)
	preview := previewAt(t, provider, time.Time{})

	if len(preview.Items) != 0 {
		t.Fatalf("the CLI became reclaimable despite its scenario returning inside the grace window: %#v", preview.Items)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("the CLI should still be on disk: %v", err)
	}

	lease, found, err := artifactlease.Load(binary)
	if err != nil || !found {
		t.Fatalf("lease: %v found=%v", err, found)
	}
	if lease.Observations != 1 {
		t.Fatalf("observations = %d; the reappearance should have reset the count to zero before the new sighting", lease.Observations)
	}
}

// An observation is not authority to delete. The first time an owner is seen
// missing, nothing may be reclaimed however old the binary is.
func TestFirstObservationNeverReclaims(t *testing.T) {
	provider, fsys, binary := scenarioBinaryFixture(t, false)

	preview := previewAt(t, provider, fixtureStart)

	if len(preview.Items) != 0 {
		t.Fatalf("a single observation authorized a reclamation: %#v", preview.Items)
	}
	if len(fsys.Removed) != 0 {
		t.Fatalf("preview removed something: %#v", fsys.Removed)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("the binary should be untouched: %v", err)
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("the refusal produced no reason")
	}
}
