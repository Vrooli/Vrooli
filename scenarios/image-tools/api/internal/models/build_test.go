package models

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/capacity"
)

type buildCapacityStub struct {
	lease    capacity.Lease
	verdict  capacity.Verdict
	owner    string
	ram, cpu int64
	closed   bool
}

func (b *buildCapacityStub) AcquireNoReclaim(_ context.Context, owner string, ram, cpu int64) (capacity.Lease, capacity.Verdict, error) {
	b.owner, b.ram, b.cpu = owner, ram, cpu
	return b.lease, b.verdict, nil
}
func (b *buildCapacityStub) Close() error { b.closed = true; return nil }

type buildLeaseStub struct{ released bool }

func (l *buildLeaseStub) Release(context.Context) error   { l.released = true; return nil }
func (l *buildLeaseStub) Heartbeat(context.Context) error { return nil }

type buildOperationStub struct {
	intent            platform.OperationIntent
	ref               platform.OperationRef
	spec              platform.OperationSpec
	started, released bool
	startErr          error
	lookupCalls       int
	lookupsAfterStart error
	inspectCalls      int
	inspectFn         func(context.Context, platform.OperationRef, int) (platform.OperationObservation, error)
	stopCalled        bool
}

func (b *buildOperationStub) NewIntent(owner, attempt string, spec platform.OperationSpec) (platform.OperationIntent, error) {
	b.spec = spec
	return platform.NewOperationIntent(owner, attempt, spec)
}
func (b *buildOperationStub) Start(_ context.Context, intent platform.OperationIntent, spec platform.OperationSpec) (*platform.OwnedOperation, error) {
	if b.startErr != nil {
		return nil, b.startErr
	}
	b.intent, b.spec, b.started = intent, spec, true
	b.ref = platform.OperationRef{Intent: intent, Unit: "unit", NativeInvocationID: intent.InvocationID}
	return &platform.OwnedOperation{Ref: b.ref}, nil
}
func (b *buildOperationStub) Lookup(_ context.Context, intent platform.OperationIntent) (platform.OperationRef, error) {
	b.lookupCalls++
	if !b.started {
		if b.lookupCalls > 1 && b.lookupsAfterStart != nil {
			return platform.OperationRef{}, b.lookupsAfterStart
		}
		return platform.OperationRef{}, platform.ErrOperationOutcomeLost
	}
	return b.ref, nil
}
func (b *buildOperationStub) Inspect(_ context.Context, ref platform.OperationRef) (platform.OperationObservation, error) {
	b.inspectCalls++
	if b.inspectFn != nil {
		return b.inspectFn(context.Background(), ref, b.inspectCalls)
	}
	return platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, Result: "success", ExitKnown: true, ExecMainStatus: 0}, nil
}
func (b *buildOperationStub) Stop(context.Context, platform.OperationRef) (platform.OperationStopReceipt, error) {
	b.stopCalled = true
	return platform.OperationStopReceipt{NativeQuiescent: true}, nil
}
func (b *buildOperationStub) Release(context.Context, platform.OperationRef) error {
	b.released = true
	return nil
}

func TestOwnedBuildPersistsIntentAndUsesMatchingMemoryCeiling(t *testing.T) {
	db := newStateDB(t)
	store := NewInstallStore(db)
	owner := &buildOperationStub{}
	installer := &Installer{State: store, Build: &BuildConfig{Operations: owner}}
	threadEnv := []string{"OMP_NUM_THREADS=2", "MKL_NUM_THREADS=2", "OPENBLAS_NUM_THREADS=2", "NUMEXPR_NUM_THREADS=2", "VECLIB_MAXIMUM_THREADS=2"}
	if _, err, retain := installer.runOwnedBuild(context.Background(), "job-123", "big-lama", t.TempDir(), "/usr/bin/python3", []string{"-m", lamaBuildModule}, threadEnv); err != nil || retain {
		t.Fatal(err)
	}
	if !owner.started || !owner.released || owner.spec.Limits.MemoryMaxBytes != lamaBuildRAMBytes || owner.spec.Limits.CPUWeight != 100 || owner.spec.Limits.WallTime != lamaBuildTimeout {
		t.Fatalf("native operation was not bounded and released: started=%v released=%v limits=%+v", owner.started, owner.released, owner.spec.Limits)
	}
	for _, want := range []string{"OMP_NUM_THREADS=2", "MKL_NUM_THREADS=2", "OPENBLAS_NUM_THREADS=2", "NUMEXPR_NUM_THREADS=2", "VECLIB_MAXIMUM_THREADS=2"} {
		found := false
		for _, setting := range owner.spec.Env {
			if setting == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("native exporter environment is missing bounded thread setting %q", want)
		}
	}
	row, ok, err := store.getBuildOperation(context.Background(), "job-123")
	if err != nil || !ok || row.State != "terminal" || row.Intent != owner.intent {
		t.Fatalf("terminal operation intent was not durably retained: row=%+v ok=%v err=%v", row, ok, err)
	}
}

func TestOwnedBuildCancellationStopsAndReleasesBeforeReturning(t *testing.T) {
	db := newStateDB(t)
	store := NewInstallStore(db)
	ctx, cancel := context.WithCancel(context.Background())
	owner := &buildOperationStub{}
	owner.inspectFn = func(_ context.Context, ref platform.OperationRef, call int) (platform.OperationObservation, error) {
		if call == 1 {
			cancel()
			return platform.OperationObservation{}, ctx.Err()
		}
		return platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, Result: "signal", ExitKnown: true}, nil
	}
	installer := &Installer{State: store, Build: &BuildConfig{Operations: owner}}
	_, err, retain := installer.runOwnedBuild(ctx, "job-cancel", "big-lama", t.TempDir(), "/usr/bin/python3", []string{"-m", lamaBuildModule}, nil)
	if err == nil || retain || !owner.stopCalled || !owner.released {
		t.Fatalf("cancellation must stop/reap/release before returning: err=%v retain=%v stop=%v release=%v", err, retain, owner.stopCalled, owner.released)
	}
}

func TestOwnedBuildAmbiguousStartRetainsRAMClaimUntilExpiry(t *testing.T) {
	db := newStateDB(t)
	store := NewInstallStore(db)
	owner := &buildOperationStub{startErr: errors.New("transport lost after dispatch"), lookupsAfterStart: platform.ErrOperationOutcomeLost}
	installer := &Installer{State: store, Build: &BuildConfig{Operations: owner}}
	_, err, retain := installer.runOwnedBuild(context.Background(), "job-ambiguous", "big-lama", t.TempDir(), "/usr/bin/python3", []string{"-m", lamaBuildModule}, nil)
	row, ok, readErr := store.getBuildOperation(context.Background(), "job-ambiguous")
	if err == nil || !retain || !ok || readErr != nil || row.State != "outcome_unknown" || owner.released {
		t.Fatalf("ambiguous dispatch must retain claim and intent: err=%v retain=%v row=%+v ok=%v readErr=%v released=%v", err, retain, row, ok, readErr, owner.released)
	}
}

func TestOwnedBuildStateWriteFailureCleansUpExactOperation(t *testing.T) {
	db := newStateDB(t)
	store := NewInstallStore(db)
	if _, err := db.Exec(`CREATE TRIGGER fail_build_state BEFORE UPDATE ON model_build_operation BEGIN SELECT RAISE(FAIL, 'injected state update failure'); END`); err != nil {
		t.Fatal(err)
	}
	owner := &buildOperationStub{}
	installer := &Installer{State: store, Build: &BuildConfig{Operations: owner}}
	_, err, retain := installer.runOwnedBuild(context.Background(), "job-state-fail", "big-lama", t.TempDir(), "/usr/bin/python3", []string{"-m", lamaBuildModule}, nil)
	if err == nil || retain || !owner.stopCalled || !owner.released {
		t.Fatalf("state write failure must clean up before claim release: err=%v retain=%v stop=%v release=%v", err, retain, owner.stopCalled, owner.released)
	}
}

func TestRecoverPendingBuildStopsExactPriorOperationBeforeReplacement(t *testing.T) {
	db := newStateDB(t)
	store := NewInstallStore(db)
	owner := &buildOperationStub{started: true}
	spec := platform.OperationSpec{Path: "/usr/bin/python3", Args: []string{"-m", lamaBuildModule}, Env: []string{}, Dir: t.TempDir(), Limits: platform.OperationLimits{MemoryMaxBytes: lamaBuildRAMBytes, TasksMax: 4096, CPUWeight: 100, WallTime: lamaBuildTimeout}}
	intent, err := owner.NewIntent("image-tools-model-build", "job-prior", spec)
	if err != nil {
		t.Fatal(err)
	}
	owner.intent, owner.spec = intent, spec
	owner.ref = platform.OperationRef{Intent: intent, Unit: "unit", NativeInvocationID: intent.InvocationID}
	if err := store.saveBuildOperation(context.Background(), buildOperationRecord{JobID: "job-prior", ModelID: "big-lama", Intent: intent, State: "starting"}); err != nil {
		t.Fatal(err)
	}
	if err := store.updateBuildOperation(context.Background(), buildOperationRecord{JobID: "job-prior", ModelID: "big-lama", Intent: intent, State: "running"}); err != nil {
		t.Fatal(err)
	}
	owner.inspectFn = func(_ context.Context, ref platform.OperationRef, call int) (platform.OperationObservation, error) {
		if call == 1 {
			return platform.OperationObservation{Ref: ref, State: platform.OperationRunning, Populated: true}, nil
		}
		return platform.OperationObservation{Ref: ref, State: platform.OperationStoppedOutcomeKnown, Result: "signal", ExitKnown: true}, nil
	}
	installer := &Installer{State: store, Build: &BuildConfig{Operations: owner}}
	if err := installer.recoverPendingBuild(context.Background(), "big-lama"); err != nil {
		t.Fatal(err)
	}
	row, ok, err := store.getBuildOperation(context.Background(), "job-prior")
	if err != nil || !ok || row.State != "recovered" || !owner.stopCalled || !owner.released {
		t.Fatalf("prior operation was not stopped and reconciled: row=%+v ok=%v err=%v stop=%v released=%v", row, ok, err, owner.stopCalled, owner.released)
	}
}

func TestBuildUsesExactNoReclaimRAMAdmissionAndReleases(t *testing.T) {
	dir := t.TempDir()
	for name, contents := range map[string][]byte{"source.zip": []byte("source"), "checkpoint.zip": []byte("checkpoint")} {
		if err := os.WriteFile(filepath.Join(dir, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sourceHash, _, _ := hashFile(filepath.Join(dir, "source.zip"))
	checkpointHash, _, _ := hashFile(filepath.Join(dir, "checkpoint.zip"))
	lease := &buildLeaseStub{}
	broker := &buildCapacityStub{lease: lease, verdict: capacity.Verdict{Kind: "grant"}}
	runCalled := false
	installer := &Installer{Build: &BuildConfig{
		Python: "python", ReferenceONNX: "reference.onnx", Capacity: func(context.Context) (BuildCapacityBroker, error) { return broker, nil },
		Run: func(context.Context, string, []string, []string) ([]byte, error) {
			runCalled = true
			return nil, errors.New("stop after admission")
		},
	}}
	model := Model{ID: "big-lama-test", BuildProfile: &BuildProfile{Kind: "big-lama-onnx", SourceAsset: "source.zip", CheckpointAsset: "checkpoint.zip", SourceSHA256: sourceHash, CheckpointSHA256: checkpointHash}}
	_, err := installer.ensureBuild(context.Background(), model, dir, "", func(int, string) {})
	if err == nil || !runCalled {
		t.Fatalf("expected exporter to run after grant and return its sentinel error; err=%v called=%v", err, runCalled)
	}
	if broker.owner != "image-tools:model-build:big-lama-test" || broker.ram != lamaBuildRAMBytes || broker.cpu != lamaBuildCPUMillicores {
		t.Fatalf("unexpected capacity request: owner=%q ram=%d cpu=%d", broker.owner, broker.ram, broker.cpu)
	}
	if !lease.released || !broker.closed {
		t.Fatalf("capacity lifecycle incomplete: released=%v closed=%v", lease.released, broker.closed)
	}
}

func TestBuildDoesNotStartWithoutSharedBrokerGrant(t *testing.T) {
	dir := t.TempDir()
	for name, contents := range map[string][]byte{"source.zip": []byte("source"), "checkpoint.zip": []byte("checkpoint")} {
		if err := os.WriteFile(filepath.Join(dir, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sourceHash, _, _ := hashFile(filepath.Join(dir, "source.zip"))
	checkpointHash, _, _ := hashFile(filepath.Join(dir, "checkpoint.zip"))
	broker := &buildCapacityStub{verdict: capacity.Verdict{Kind: "deny", Reason: "memory PSI above threshold"}}
	runCalled := false
	installer := &Installer{Build: &BuildConfig{
		Python: "python", ReferenceONNX: "reference.onnx", Capacity: func(context.Context) (BuildCapacityBroker, error) { return broker, nil },
		Run: func(context.Context, string, []string, []string) ([]byte, error) { runCalled = true; return nil, nil },
	}}
	model := Model{ID: "big-lama-test", BuildProfile: &BuildProfile{Kind: "big-lama-onnx", SourceAsset: "source.zip", CheckpointAsset: "checkpoint.zip", SourceSHA256: sourceHash, CheckpointSHA256: checkpointHash}}
	_, err := installer.ensureBuild(context.Background(), model, dir, "", func(int, string) {})
	if err == nil || !strings.Contains(err.Error(), "memory PSI above threshold") || runCalled {
		t.Fatalf("denial must prevent exporter start: err=%v called=%v", err, runCalled)
	}
	if !broker.closed {
		t.Fatal("capacity broker was not closed after denial")
	}
}

func TestBuildOutputReadyRequiresMatchingOutputAndReceipt(t *testing.T) {
	dir := t.TempDir()
	output := []byte("validated onnx artifact")
	outputPath := filepath.Join(dir, "lama_1024.onnx")
	if err := os.WriteFile(outputPath, output, 0o600); err != nil {
		t.Fatal(err)
	}
	sum, size, err := hashFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	profile := &BuildProfile{
		SourceAsset: "lama-source.zip", CheckpointAsset: "big-lama.safetensors",
		SourceSHA256: "source-hash", CheckpointSHA256: "checkpoint-hash",
		OutputFilename: "lama_1024.onnx", OutputSHA256: sum, Resolution: 1024, Opset: 17,
	}
	receipt := buildReceipt{
		Schema: "vrooli.image-tools.lama-onnx-build/v2", SourceArchiveSHA256: profile.SourceSHA256,
		CheckpointSHA256: profile.CheckpointSHA256, CheckpointBytes: 204348820, OutputSHA256: sum, OutputBytes: size,
		CheckpointKeyPrefix: lamaCheckpointKeyPrefix,
		CapacityClaimBytes:  lamaBuildRAMBytes, ProcessPeakRSSBytes: 14 << 30,
		CPUClaimMillicores: lamaBuildCPUMillicores, TorchThreadLimit: 2,
		OperationCPUWeight: 100, CPUQuotaMillicores: 0,
		ProcessPeakRSSScope:     "single exporter process kernel high-water mark; excludes other cgroup processes",
		ProcessPeakRSSAfterLoad: 3 << 30, ProcessPeakRSSAfter512: 7 << 30,
		ProcessPeakRSSAfterExport: 13 << 30, ProcessPeakRSSAfterParity: 14 << 30,
		Resolution: []int{1024, 1024}, Opset: 17, OutputFinite: true, ParityMaxAbs: 0.01,
		UpstreamUtilsSourceSHA256: lamaUtilsSourceSHA256, UpstreamUtilsPreparedSHA256: lamaUtilsPreparedSHA256,
		SourcePreparation: lamaSourcePreparation, ReferenceONNXSHA256: lamaReferenceONNXSHA256,
		ReferenceProbeResolution: []int{512, 512}, ReferenceProbeMaxAbs: 0.1, ReferenceProbeTolerance: lamaReferenceTolerance,
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build-provenance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ready := buildOutputReady(profile, dir); !ready || got != sum {
		t.Fatalf("valid receipt should make output ready: hash=%q ready=%v", got, ready)
	}
	receipt.CapacityClaimBytes = int64(7 << 30)
	data, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build-provenance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ready := buildOutputReady(profile, dir); ready {
		t.Fatal("prior under-budget build receipt must not make the model ready")
	}
	receipt.CapacityClaimBytes = lamaBuildRAMBytes
	receipt.ProcessPeakRSSBytes = lamaBuildRAMBytes + 1
	data, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build-provenance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ready := buildOutputReady(profile, dir); ready {
		t.Fatal("receipt with process peak above native limit must not make the model ready")
	}
	receipt.ProcessPeakRSSBytes = 14 << 30
	data, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "build-provenance.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(outputPath, []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ready := buildOutputReady(profile, dir); ready {
		t.Fatal("tampered ONNX bytes must invalidate the build receipt")
	}
	profile.OutputSHA256 = "another-output-digest"
	if _, ready := buildOutputReady(profile, dir); ready {
		t.Fatal("a profile-pinned digest mismatch must keep the model unbuilt")
	}
}
