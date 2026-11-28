package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

type countingModelSQL struct {
	SQLExecutor
	installReads int
	customReads  int
}

func (db *countingModelSQL) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if strings.Contains(query, "FROM model_install") {
		db.installReads++
	}
	return db.SQLExecutor.QueryRowContext(ctx, query, args...)
}

func (db *countingModelSQL) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, "FROM custom_model") {
		db.customReads++
	}
	return db.SQLExecutor.QueryContext(ctx, query, args...)
}

func TestLifecycleRejectsUnprovenReadyTransitions(t *testing.T) {
	for _, state := range []LifecycleState{ModelAbsent, ModelInstalling, ModelStale, ModelFailed, "unknown"} {
		if _, err := (ModelLifecycle{ModelID: "m", State: state}).Transition(ModelReady); err == nil {
			t.Errorf("%s -> ready must require preparation", state)
		}
	}
	state := ModelLifecycle{ModelID: "m", State: ModelAbsent}
	for _, next := range []LifecycleState{ModelInstalling, ModelInstalled, ModelBuilding, ModelSmoking, ModelReady, ModelStale, ModelInstalling, ModelFailed} {
		var err error
		state, err = state.Transition(next)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestLifecycleResolvesInstallRecordOnce(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	counting := &countingModelSQL{SQLExecutor: f.in.State.db}
	f.in.State = NewInstallStore(counting)
	state, err := f.in.Lifecycle(ctx, installTestModelID)
	if err != nil || state.State != ModelInstalled || state.Reason != "smoke_unsupported" {
		t.Fatalf("lifecycle state = %+v, err = %v", state, err)
	}
	if counting.installReads != 1 {
		t.Fatalf("lifecycle read install record %d times, want 1", counting.installReads)
	}
}

func TestInstallReusesCompletedInstallRecordDuringFinalLifecycle(t *testing.T) {
	f := newInstallFixture(t)
	counting := &countingModelSQL{SQLExecutor: f.in.State.db}
	f.in.State = NewInstallStore(counting)
	f.in.Custom = NewCustomStore(counting)
	if _, err := f.in.Install(context.Background(), installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	if counting.installReads != 2 {
		t.Fatalf("install read model_install %d times, want initial lifecycle + one combined artifact/pin lookup", counting.installReads)
	}
	if counting.customReads != 1 {
		t.Fatalf("install read custom model catalog %d times, want one resolution", counting.customReads)
	}
}

func TestInstallLifecyclePersistsSmokeAndInvalidatesChangedRuntime(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	m, err := f.in.Resolve(ctx, installTestModelID)
	if err != nil {
		t.Fatal(err)
	}
	m.Backend, m.Runtime.Family = BackendDiffusers, "instruct-pix2pix"
	if err := f.in.Custom.Add(ctx, m); err != nil {
		t.Fatal(err)
	}
	probes := 0
	f.in.Smoke = &SmokeConfig{Python: fakeInterp(t), LockHash: "runtime-1", Run: func(ctx context.Context, _ string, _, _ []string) ([]byte, error) {
		probes++
		state, err := f.in.Lifecycle(ctx, m.ID)
		if err != nil || state.State != ModelSmoking {
			t.Fatalf("active state: %+v, %v", state, err)
		}
		durable, err := f.in.State.lifecycle(ctx, m.ID)
		if err != nil || durable.State != ModelSmoking {
			t.Fatalf("durable state: %+v, %v", durable, err)
		}
		return []byte("loaded"), nil
	}}
	if _, err := f.in.Install(ctx, m.ID, nil); err != nil {
		t.Fatal(err)
	}
	state, err := f.in.Lifecycle(ctx, m.ID)
	if err != nil || !state.Ready() || state.ArtifactHash == "" || state.RuntimeHash != "runtime-1" {
		t.Fatalf("ready evidence: %+v, %v", state, err)
	}
	durable, err := f.in.State.lifecycle(ctx, m.ID)
	if err != nil || !durable.Ready() {
		t.Fatalf("durable ready: %+v, %v", durable, err)
	}
	f.in.Smoke.LockHash = "runtime-2"
	state, err = f.in.Lifecycle(ctx, m.ID)
	if err != nil || state.State != ModelStale || state.Reason != "smoke_pending" {
		t.Fatalf("stale: %+v, %v", state, err)
	}
	durable, err = f.in.State.lifecycle(ctx, m.ID)
	if err != nil || durable.State != ModelStale || durable.Reason != "smoke_pending" {
		t.Fatalf("durable stale lifecycle: %+v, %v", durable, err)
	}
	if ready, _ := f.in.RuntimeReadiness(ctx, m.ID); ready {
		t.Fatal("stale runtime admitted")
	}
	if _, err := f.in.Install(ctx, m.ID, nil); err != nil {
		t.Fatal(err)
	}
	if probes != 2 || f.downloads != 1 {
		t.Fatalf("probes=%d downloads=%d; changed runtime must resmoke existing weights", probes, f.downloads)
	}
}

func TestCanceledInstallPersistsFailureAfterContextCancellation(t *testing.T) {
	f := newInstallFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	f.in.Download = downloaderFunc(func(ctx context.Context, _, _ string, _ func(int64, int64)) error {
		state, err := f.in.Lifecycle(ctx, installTestModelID)
		if err != nil || state.State != ModelInstalling {
			t.Fatalf("install stage: %+v, %v", state, err)
		}
		cancel()
		return ctx.Err()
	})
	if _, err := f.in.Install(ctx, installTestModelID, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error: %v", err)
	}
	state, err := f.in.Lifecycle(context.Background(), installTestModelID)
	if err != nil || state.State != ModelFailed || state.Reason != "install_failed" {
		t.Fatalf("failure: %+v, %v", state, err)
	}
	if _, active := f.in.activeLifecycle.Load(installTestModelID); active {
		t.Fatal("cancellation retained active lifecycle")
	}
}

func TestLifecycleRefusesPythonSidecarWithoutApplicableSmoke(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	m, err := f.in.Resolve(ctx, installTestModelID)
	if err != nil {
		t.Fatal(err)
	}
	m.Backend = "python-sidecar"
	if err := f.in.Custom.Add(ctx, m); err != nil {
		t.Fatal(err)
	}
	if _, err := f.in.Install(ctx, m.ID, nil); err != nil {
		t.Fatal(err)
	}
	state, err := f.in.Lifecycle(ctx, m.ID)
	if err != nil || state.Ready() || state.Reason != "smoke_unsupported" {
		t.Fatalf("unproven sidecar: %+v, %v", state, err)
	}
}

func TestLifecycleReadsBackendEvidenceWithoutLaunchingProbes(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	m, err := f.in.Resolve(ctx, installTestModelID)
	if err != nil {
		t.Fatal(err)
	}
	m.Backend = "stable-diffusion.cpp" // this test exercises native backend evidence, not Python smoke
	if err := f.in.Custom.Add(ctx, m); err != nil {
		t.Fatal(err)
	}
	probes := 0
	prepared := false
	f.in.BackendReadiness = func(context.Context, Model) error {
		probes++
		if !prepared {
			return errors.New("backend missing")
		}
		return nil
	}
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		doctor, err := f.in.DoctorRuntimeReport(ctx, append(f.in.Reg.Models(), m), nil)
		if err != nil {
			t.Fatalf("doctor runtime: %v", err)
		}
		var projected *ModelLifecycle
		for i := range doctor.Models {
			if doctor.Models[i].ModelID == installTestModelID {
				projected = &doctor.Models[i]
				break
			}
		}
		if projected == nil || projected.State != ModelInstalled || projected.Reason != "needs_backend" {
			t.Fatalf("doctor lifecycle projection = %+v, want installed/needs_backend", projected)
		}
	}
	if probes != 1 {
		t.Fatalf("read paths launched %d probes, want only the install probe", probes)
	}
	prepared = true
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	state, err := f.in.Lifecycle(ctx, installTestModelID)
	if err != nil || !state.Ready() || probes != 2 {
		t.Fatalf("repaired state = %+v, probes=%d, %v", state, probes, err)
	}
}
