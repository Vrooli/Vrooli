package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
)

// installFixture wires an Installer over a fresh SQLite DB and a temp models
// root, with a controllable fake downloader and disk-space function.
type installFixture struct {
	in        *Installer
	root      string
	payload   []byte // bytes the fake downloader writes
	avail     int64  // bytes the fake disk reports
	downloads int
}

const installTestModelID = "install-test-model"

func newInstallFixture(t *testing.T) *installFixture {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	root := t.TempDir()
	// A realistic-sized generic artifact: above the 1 KiB validation floor and
	// not HTML, so it passes artifact validation (the fake downloader writes it).
	f := &installFixture{root: root, payload: bytes.Repeat([]byte("fake-model-weights\n"), 128), avail: int64(50) << 30}
	f.in = &Installer{
		Root:   root,
		Reg:    reg,
		Custom: NewCustomStore(d),
		State:  NewInstallStore(d),
		Download: downloaderFunc(func(_ context.Context, _, dest string, _ func(int64, int64)) error {
			f.downloads++
			return os.WriteFile(dest, f.payload, 0o644)
		}),
		DiskAvail: func(string) (int64, error) { return f.avail, nil },
	}
	if err := f.in.AddCustom(context.Background(), Model{
		ID:           installTestModelID,
		Name:         "Install Test Model",
		Operations:   []string{"upscale"},
		Tier:         TierNiceToHave,
		Backend:      "onnxruntime",
		SizeMBApprox: 16,
		Hardware:     Hardware{CPUCapable: true},
		CapabilityLabels: CapabilityLabels{
			License:       "MIT",
			CommercialUse: CommercialUseYes,
		},
		Source: Source{DownloadURL: "https://example.test/install-test-model.bin"},
	}); err != nil {
		t.Fatalf("add install test model: %v", err)
	}
	return f
}

type downloaderFunc func(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error

func (fn downloaderFunc) Download(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error {
	return fn(ctx, rawURL, dest, emit)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestEstimateInstallSecondsAt(t *testing.T) {
	tests := []struct {
		name        string
		sizeMB      int
		mbPerSecond int
		want        int
	}{
		{name: "unknown size", sizeMB: 0, mbPerSecond: 20, want: 0},
		{name: "default throughput", sizeMB: 30, mbPerSecond: 0, want: 2},
		{name: "minimum nonzero eta", sizeMB: 4, mbPerSecond: 20, want: 1},
		{name: "custom measured throughput", sizeMB: 120, mbPerSecond: 30, want: 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EstimateInstallSecondsAt(tt.sizeMB, tt.mbPerSecond); got != tt.want {
				t.Fatalf("EstimateInstallSecondsAt(%d, %d) = %d, want %d", tt.sizeMB, tt.mbPerSecond, got, tt.want)
			}
		})
	}
}

func TestEstimateInstallSecondsDefault(t *testing.T) {
	if got := EstimateInstallSeconds(30); got != 2 {
		t.Fatalf("EstimateInstallSeconds(30) = %d, want 2", got)
	}
}

func TestInstallRejectsConditionalCommercialRightsBeforeAcquisition(t *testing.T) {
	f := newInstallFixture(t)
	var conditionalID string
	for _, model := range f.in.Reg.Models() {
		if model.CapabilityLabels.CommercialUse == CommercialUseConditional {
			conditionalID = model.ID
			break
		}
	}
	if conditionalID == "" {
		t.Fatal("seed should include a conditional-rights model for this regression")
	}
	if _, err := f.in.Install(context.Background(), conditionalID, nil); err == nil || !strings.Contains(err.Error(), "commercial rights") {
		t.Fatalf("install error = %v, want commercial-rights rejection", err)
	}
	if f.downloads != 0 {
		t.Fatalf("conditional model caused %d artifact downloads", f.downloads)
	}
}

func TestInstallStoreListAndDelete(t *testing.T) {
	ctx := context.Background()
	st := NewInstallStore(newStateDB(t))
	at := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	records := []InstallRecord{
		{ID: "b", Installed: true, Path: "/models/b", Checksum: "sum-b", SizeBytes: 20, InstalledAt: at},
		{ID: "a", Installed: true, Path: "/models/a", Checksum: "sum-a", SizeBytes: 10, InstalledAt: at},
	}
	for _, rec := range records {
		if err := st.Set(ctx, rec); err != nil {
			t.Fatalf("set %s: %v", rec.ID, err)
		}
	}
	list, err := st.List(ctx)
	if err != nil {
		t.Fatalf("list installs: %v", err)
	}
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("installs should be ordered by id, got %+v", list)
	}
	if !list[0].InstalledAt.Equal(at) {
		t.Fatalf("installed_at round trip mismatch: %v", list[0].InstalledAt)
	}
	if err := st.Delete(ctx, "a"); err != nil {
		t.Fatalf("delete install: %v", err)
	}
	if _, ok, err := st.Get(ctx, "a"); err != nil || ok {
		t.Fatalf("deleted install still present: ok=%v err=%v", ok, err)
	}
}

func TestInstalledBuildCandidateFetchesMissingPinnedInputBeforeBuild(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	model, ok := f.in.Reg.ByID("big-lama-1024-export")
	if !ok {
		t.Fatal("missing LaMa export candidate")
	}
	// Exercise the native build pipeline with a synthetic explicitly attested
	// custom entry. The shipped seed row is conditional and must remain blocked.
	model.CapabilityLabels.CommercialUse = CommercialUseYes
	model.CapabilityLabels.CommercialUseNotes = "test fixture attestation"
	sourcePayload := bytes.Repeat([]byte("pinned source\n"), 128)
	checkpointPayload := append([]byte{0x10, 0, 0, 0, 0, 0, 0, 0, '{'}, bytes.Repeat([]byte{' '}, 2048)...)
	sourceHash := sha256Hex(sourcePayload)
	checkpointHash := sha256Hex(checkpointPayload)
	model.Source.Assets[0].SHA256 = sourceHash
	model.Source.Assets[0].MinBytes = 1
	model.Source.Assets[1].SHA256 = checkpointHash
	model.Source.Assets[1].MinBytes = 1
	model.BuildProfile.SourceSHA256 = sourceHash
	model.BuildProfile.CheckpointSHA256 = checkpointHash
	f.in.Reg.byID[model.ID] = model
	for i := range f.in.Reg.models {
		if f.in.Reg.models[i].ID == model.ID {
			f.in.Reg.models[i] = model
			break
		}
	}

	dir := f.in.modelDir(model.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, model.BuildProfile.SourceAsset), sourcePayload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := f.in.State.Set(ctx, InstallRecord{ID: model.ID, Installed: true, Path: dir, Checksum: sourceHash}); err != nil {
		t.Fatal(err)
	}
	f.in.Download = downloaderFunc(func(_ context.Context, rawURL, dest string, _ func(int64, int64)) error {
		f.downloads++
		payload := sourcePayload
		if strings.Contains(rawURL, "big-lama.safetensors") {
			payload = checkpointPayload
		}
		return os.WriteFile(dest, payload, 0o644)
	})

	_, err := f.in.Install(ctx, model.ID, nil)
	if err == nil || !strings.Contains(err.Error(), "operation owner") {
		t.Fatalf("install should fetch the missing pinned asset before reaching the unavailable build seam, got %v", err)
	}
	if f.downloads != 2 {
		t.Fatalf("incomplete installed candidate should reconcile both pinned assets, downloads=%d", f.downloads)
	}
	if _, err := os.Stat(filepath.Join(dir, model.BuildProfile.CheckpointAsset)); err != nil {
		t.Fatalf("missing safetensors input was not fetched: %v", err)
	}
}

func TestCustomStoreListUpsertAndDelete(t *testing.T) {
	ctx := context.Background()
	st := NewCustomStore(newStateDB(t))
	first := Model{ID: "custom-a", Name: "A", Operations: []string{"upscale"}, Tier: TierNiceToHave, Backend: "onnxruntime", Hardware: Hardware{CPUCapable: true}}
	second := first
	second.Name = "A2"
	if err := st.Add(ctx, first); err != nil {
		t.Fatalf("add custom: %v", err)
	}
	if err := st.Add(ctx, second); err != nil {
		t.Fatalf("upsert custom: %v", err)
	}
	list, err := st.List(ctx)
	if err != nil {
		t.Fatalf("list custom: %v", err)
	}
	if len(list) != 1 || list[0].Name != "A2" {
		t.Fatalf("custom list after upsert = %+v", list)
	}
	if err := st.Delete(ctx, "custom-a"); err != nil {
		t.Fatalf("delete custom: %v", err)
	}
	list, err = st.List(ctx)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("custom list after delete = %+v", list)
	}
}

func TestInstallerResolveCustomAndMissing(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	m, err := f.in.Resolve(ctx, installTestModelID)
	if err != nil {
		t.Fatalf("resolve custom: %v", err)
	}
	if m.ID != installTestModelID {
		t.Fatalf("resolved model id = %q", m.ID)
	}
	if _, err := f.in.Resolve(ctx, "missing-model"); !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

// TestInstall_DownloadsPinsChecksumAndRecords proves the happy path: a seed model
// downloads, the checksum is pinned (it was empty in the seed), and the install
// state is persisted and readable.
func TestInstall_DownloadsPinsChecksumAndRecords(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	rec, err := f.in.Install(ctx, installTestModelID, nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !rec.Installed {
		t.Fatalf("expected installed record")
	}
	if rec.Checksum != sha256Hex(f.payload) {
		t.Fatalf("checksum not pinned to computed hash: %q", rec.Checksum)
	}
	if !f.in.Installed(ctx, installTestModelID) {
		t.Fatalf("Installed should report true after install")
	}
	got, ok, err := f.in.State.Get(ctx, installTestModelID)
	if err != nil || !ok {
		t.Fatalf("get install: ok=%v err=%v", ok, err)
	}
	if got.Checksum != rec.Checksum || !got.Installed {
		t.Fatalf("persisted record mismatch: %+v", got)
	}
}

// TestInstall_Idempotent proves a second install is a no-op (no re-download).
func TestInstall_Idempotent(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if f.downloads != 1 {
		t.Fatalf("expected exactly one download, got %d", f.downloads)
	}
}

func TestAcquireInstalledArtifactsReadsRecordOnce(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("install fixture model: %v", err)
	}
	m, err := f.in.Resolve(ctx, installTestModelID)
	if err != nil {
		t.Fatal(err)
	}
	counting := &countingModelSQL{SQLExecutor: f.in.State.db}
	f.in.State = NewInstallStore(counting)

	rec, err := f.in.acquireArtifacts(ctx, m, nil)
	if err != nil {
		t.Fatalf("reuse installed artifacts: %v", err)
	}
	if !rec.Installed || rec.Path == "" || rec.Checksum != sha256Hex(f.payload) {
		t.Fatalf("reused install record = %+v", rec)
	}
	if counting.installReads != 1 {
		t.Fatalf("idempotent artifact reuse read model_install %d times, want exactly once", counting.installReads)
	}
}

func TestInstall_RevalidatesExistingArtifactsBeforeReuse(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("first install: %v", err)
	}
	path := filepath.Join(f.in.modelDir(installTestModelID), "install-test-model.bin")
	if err := os.WriteFile(path, bytes.Repeat([]byte("corrupt"), 256), 0o644); err != nil {
		t.Fatalf("corrupt installed artifact: %v", err)
	}
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("repair install: %v", err)
	}
	if f.downloads != 2 {
		t.Fatalf("repair should reacquire the invalid artifact; downloads=%d", f.downloads)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, f.payload) {
		t.Fatalf("repaired artifact = %d bytes, err=%v", len(got), err)
	}
}

// TestInstall_InsufficientDisk proves the disk-space gate refuses before any
// download happens.
func TestInstall_InsufficientDisk(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	f.avail = 1 << 20 // 1 MiB, far below sd-1.5's ~2.1 GB

	_, err := f.in.Install(ctx, installTestModelID, nil)
	if !errors.Is(err, ErrInsufficientDisk) {
		t.Fatalf("expected ErrInsufficientDisk, got %v", err)
	}
	if f.downloads != 0 {
		t.Fatalf("download must not run when disk is insufficient")
	}
}

// TestInstall_ChecksumMismatch proves a pinned checksum that disagrees with the
// downloaded bytes fails and removes the partial download.
func TestInstall_ChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	// Pre-pin a checksum that won't match the fake payload.
	if err := f.in.State.Set(ctx, InstallRecord{ID: installTestModelID, Checksum: "deadbeef"}); err != nil {
		t.Fatalf("pre-pin: %v", err)
	}
	_, err := f.in.Install(ctx, installTestModelID, nil)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
	if dirExists(filepath.Join(f.root, "models", installTestModelID)) {
		t.Fatalf("partial download dir should be removed on mismatch")
	}
}

// TestRemove_DeletesWeightsAndRecord proves removal clears disk + state.
func TestRemove_DeletesWeightsAndRecord(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := f.in.Remove(ctx, installTestModelID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if f.in.Installed(ctx, installTestModelID) {
		t.Fatalf("Installed should be false after remove")
	}
	if _, ok, _ := f.in.State.Get(ctx, installTestModelID); ok {
		t.Fatalf("install record should be gone after remove")
	}
}

func TestRemove_InterruptedCleanupDoesNotReportPartialModelInstalled(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	modelDir := f.in.modelDir(installTestModelID)
	quarantineDir := f.in.removalDir(installTestModelID)
	if err := os.WriteFile(filepath.Join(modelDir, "extra-weight.bin"), []byte("extra"), 0o644); err != nil {
		t.Fatalf("add synthetic second weight: %v", err)
	}

	// Model the process stopping after the complete model directory is moved
	// away from its serving path and cleanup has removed only part of it. The
	// install row deliberately remains, matching the crash boundary before the
	// final state deletion.
	interrupted := errors.New("simulated interruption during quarantine cleanup")
	f.in.removeAll = func(path string) error {
		if path != quarantineDir {
			t.Fatalf("cleanup path = %q, want quarantine %q", path, quarantineDir)
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			t.Fatal("quarantine has no file to partially remove")
		}
		if err := os.Remove(filepath.Join(path, entries[0].Name())); err != nil {
			return err
		}
		return interrupted
	}
	if err := f.in.Remove(ctx, installTestModelID); !errors.Is(err, interrupted) {
		t.Fatalf("Remove error = %v, want simulated interruption", err)
	}
	if dirExists(modelDir) {
		t.Fatal("interrupted removal left the serving model path visible")
	}
	entries, err := os.ReadDir(quarantineDir)
	if err != nil {
		t.Fatalf("read partial quarantine: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("partial quarantine entries = %d, want one remaining synthetic weight", len(entries))
	}
	if _, ok, err := f.in.State.Get(ctx, installTestModelID); err != nil || !ok {
		t.Fatalf("install row should remain at interrupted boundary: ok=%v err=%v", ok, err)
	}

	// A fresh gate and Installer represent API restart. The persisted record
	// must not make the quarantined, partially removed tree available again.
	restarted := &Installer{Root: f.root, Reg: f.in.Reg, Custom: f.in.Custom, State: f.in.State}
	if restarted.Installed(ctx, installTestModelID) {
		t.Fatal("partial quarantined model was reported installed after restart")
	}
	if err := restarted.Remove(ctx, installTestModelID); err != nil {
		t.Fatalf("retry interrupted removal: %v", err)
	}
	if pathExists(quarantineDir) {
		t.Fatal("retry left the interrupted quarantine behind")
	}
	if _, ok, err := f.in.State.Get(ctx, installTestModelID); err != nil || ok {
		t.Fatalf("retry should clear the install row: ok=%v err=%v", ok, err)
	}
}

func TestRemove_RefusesWhileModelIsInUse(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	f.in.UseGate = NewModelUseGate(t.TempDir())
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	modelDir := filepath.Join(f.root, "models", installTestModelID)
	release, err := f.in.UseGate.Acquire(installTestModelID)
	if err != nil {
		t.Fatalf("acquire use lease: %v", err)
	}
	defer release()

	if err := f.in.Remove(ctx, installTestModelID); !errors.Is(err, ErrModelInUse) {
		t.Fatalf("Remove while model is in use should return ErrModelInUse, got %v", err)
	}
	if !dirExists(modelDir) {
		t.Fatal("Remove deleted weights while model was in use")
	}
	if _, ok, err := f.in.State.Get(ctx, installTestModelID); err != nil || !ok {
		t.Fatalf("Remove changed install state while model was in use: ok=%v err=%v", ok, err)
	}

	release()
	if err := f.in.Remove(ctx, installTestModelID); err != nil {
		t.Fatalf("Remove after releasing model use: %v", err)
	}
}

func TestInstall_RefusesWhileModelIsInUse(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	f.in.UseGate = NewModelUseGate(t.TempDir())
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	modelDir := filepath.Join(f.root, "models", installTestModelID)
	release, err := f.in.UseGate.Acquire(installTestModelID)
	if err != nil {
		t.Fatalf("acquire use lease: %v", err)
	}
	defer release()

	if _, err := f.in.Install(ctx, installTestModelID, nil); !errors.Is(err, ErrModelInUse) {
		t.Fatalf("Install while model is in use should return ErrModelInUse, got %v", err)
	}
	if !dirExists(modelDir) {
		t.Fatal("Install removed or replaced weights while model was in use")
	}
}

func TestModelUseGateConcurrentReadersAndMutators(t *testing.T) {
	gate := NewModelUseGate(t.TempDir())
	const workers = 8
	const iterations = 100
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < iterations; i++ {
				if release, err := gate.Acquire(installTestModelID); err == nil {
					release()
				} else if !errors.Is(err, ErrModelInUse) {
					t.Errorf("Acquire returned unexpected error: %v", err)
				}
				if release, err := gate.AcquireExclusive(installTestModelID); err == nil {
					release()
				} else if !errors.Is(err, ErrModelInUse) {
					t.Errorf("AcquireExclusive returned unexpected error: %v", err)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

// TestAddCustom_RoundTripAndLocalInstall proves a custom local model is rejected
// when it shadows a seed id or points at a missing path, accepted otherwise, and
// "installed" by reference without a download.
func TestAddCustom_RoundTripAndLocalInstall(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	// Shadowing a seed id is rejected.
	if err := f.in.AddCustom(ctx, Model{ID: "sd-1.5"}); !errors.Is(err, ErrCustomShadowsSeed) {
		t.Fatalf("expected ErrCustomShadowsSeed, got %v", err)
	}

	// A real local file backs a custom upscale model.
	weights := filepath.Join(t.TempDir(), "my-upscaler.bin")
	if err := os.WriteFile(weights, []byte("custom-weights"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	custom := Model{
		ID:               "my-upscaler",
		Name:             "My Upscaler",
		Operations:       []string{"upscale"},
		Tier:             TierNiceToHave,
		Backend:          "onnxruntime",
		Hardware:         Hardware{CPUCapable: true},
		CapabilityLabels: CapabilityLabels{License: "MIT", CommercialUse: CommercialUseYes},
		Source:           Source{LocalPath: weights},
	}
	if err := f.in.AddCustom(ctx, custom); err != nil {
		t.Fatalf("add custom: %v", err)
	}

	rec, err := f.in.Install(ctx, "my-upscaler", nil)
	if err != nil {
		t.Fatalf("install custom: %v", err)
	}
	if rec.Path != weights {
		t.Fatalf("custom install should point at the local path, got %q", rec.Path)
	}
	if f.downloads != 0 {
		t.Fatalf("custom local model must not download")
	}
	if err := f.in.Remove(ctx, "my-upscaler"); err != nil {
		t.Fatalf("remove custom local model reference: %v", err)
	}
	if !pathExists(weights) {
		t.Fatal("removing a custom local model deleted its referenced weights")
	}
	if _, ok, err := f.in.State.Get(ctx, "my-upscaler"); err != nil || ok {
		t.Fatalf("custom local install reference was not removed: ok=%v err=%v", ok, err)
	}
}

func TestAddCustomValidationErrors(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	withoutStore := *f.in
	withoutStore.Custom = nil
	if err := withoutStore.AddCustom(ctx, Model{ID: "x"}); err == nil {
		t.Fatal("expected unavailable custom store error")
	}
	if err := f.in.AddCustom(ctx, Model{}); err == nil {
		t.Fatal("expected empty custom id error")
	}
	base := Model{ID: "unclear-license", Name: "Unclear License", Operations: []string{"upscale"}, Tier: TierNiceToHave, Backend: "onnxruntime", Enabled: true, Hardware: Hardware{CPUCapable: true}}
	missingLicense := base
	missingLicense.CapabilityLabels.CommercialUse = CommercialUseYes
	if err := f.in.AddCustom(ctx, missingLicense); err == nil || !strings.Contains(err.Error(), "license is required") {
		t.Fatalf("expected custom model without license to be rejected, got %v", err)
	}
	conditional := base
	conditional.CapabilityLabels = CapabilityLabels{License: "unknown", CommercialUse: CommercialUseConditional}
	if err := f.in.AddCustom(ctx, conditional); err == nil || !strings.Contains(err.Error(), "explicitly commercial-clean") {
		t.Fatalf("expected conditional custom license to be rejected, got %v", err)
	}
	if err := f.in.AddCustom(ctx, Model{ID: "missing-local", Source: Source{LocalPath: filepath.Join(t.TempDir(), "missing.bin")}}); !errors.Is(err, ErrLocalPathMissing) {
		t.Fatalf("expected ErrLocalPathMissing, got %v", err)
	}
}

// TestInstall_WeightlessModelsAlwaysInstalled proves weightless models
// (RequiresWeights == false) are reported installed without any download, and
// Install no-ops to an installed record. This is what lets deterministic and
// pure-Go ops run with zero model provisioning.
func TestInstall_WeightlessModelsAlwaysInstalled(t *testing.T) {
	for _, id := range []string{"naturalize-detail-v1", "normals-from-depth", "laplacian-blur", "goimagehash", "gozxing"} {
		t.Run(id, func(t *testing.T) {
			f := newInstallFixture(t)
			m, ok := f.in.Reg.ByID(id)
			if !ok {
				t.Fatalf("seed missing model %q", id)
			}
			if m.RequiresWeights() {
				t.Fatalf("model %q should be weightless (backend %q)", id, m.Backend)
			}
			if !f.in.Installed(context.Background(), id) {
				t.Fatal("weightless model should report installed before any download")
			}
			rec, err := f.in.Install(context.Background(), id, nil)
			if err != nil {
				t.Fatalf("install weightless model: %v", err)
			}
			if !rec.Installed {
				t.Fatal("install record for weightless model should be Installed")
			}
			if f.downloads != 0 {
				t.Fatalf("weightless model must not download anything; got %d downloads", f.downloads)
			}
		})
	}
}
