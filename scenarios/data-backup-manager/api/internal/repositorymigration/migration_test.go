package repositorymigration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeStore struct{ rows map[string]Migration }

func (s *fakeStore) Get(_ context.Context, id string) (Migration, bool, error) {
	m, ok := s.rows[id]
	return m, ok, nil
}
func (s *fakeStore) Put(_ context.Context, m Migration) error { s.rows[m.ID] = m; return nil }

type fakeEffects struct {
	calls                    []string
	failDrill                string
	proof                    CapacityProof
	finalProof               *CapacityProof
	capacityCalls            int
	switched                 bool
	captureCount, drillCount int
	wrongCaptureDestination  bool
	wrongDrillDestination    bool
	wrongCapacityTargets     bool
}

func (f *fakeEffects) ProveCapacity(_ context.Context, planID, oldDestinationID, d string, targets []string, _ bool) (CapacityProof, error) {
	f.calls = append(f.calls, "capacity")
	f.capacityCalls++
	p := f.proof
	if f.capacityCalls > 1 && f.finalProof != nil {
		p = *f.finalProof
	}
	p.PlanID = planID
	p.OldDestinationID = oldDestinationID
	p.DestinationID = d
	p.TargetIDs = append([]string(nil), targets...)
	if f.wrongCapacityTargets {
		p.TargetIDs = []string{"target-from-another-plan"}
	}
	return p, nil
}

func (f *fakeEffects) Capture(_ context.Context, p, t, d, k string) (CaptureReceipt, error) {
	f.calls = append(f.calls, "capture:"+t)
	f.captureCount++
	if f.wrongCaptureDestination {
		d = "other-repository"
	}
	return CaptureReceipt{TargetID: t, DestinationID: d, SnapshotID: "snap-" + t, Bytes: 100, Verified: true}, nil
}

func (f *fakeEffects) Drill(_ context.Context, _, t, d, s, k string) (DrillReceipt, error) {
	f.calls = append(f.calls, "drill:"+t)
	if t == f.failDrill {
		return DrillReceipt{}, errors.New("synthetic restore failure")
	}
	f.drillCount++
	if f.wrongDrillDestination {
		d = "other-repository"
	}
	return DrillReceipt{TargetID: t, DestinationID: d, SnapshotID: s, RestoreID: "restore-" + t, ScratchReceiptID: "scratch-restore-" + t, Verified: true}, nil
}

func (f *fakeEffects) SwitchBindings(_ context.Context, p, o, n string, targets []string, k string) error {
	f.calls = append(f.calls, "switch")
	f.switched = true
	return nil
}

func newFixture() (*Service, *fakeStore, *fakeEffects, []string) {
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	st := &fakeStore{rows: map[string]Migration{}}
	fx := &fakeEffects{proof: CapacityProof{AvailableBytes: 2000, RequiredBytes: 1400, DeviceIdentity: "device-1", CredentialAuthority: "secret-ref", MeasuredAt: now, ExpiresAt: now.Add(time.Hour), Receipt: "capacity-receipt"}}
	targets := make([]string, RequiredTargetCount)
	for i := range targets {
		targets[i] = fmt.Sprintf("target-%02d", i)
	}
	return New(st, fx, fakeClock{now}), st, fx, targets
}

func TestMigrationCapturesAndDrillsExact14BeforeSwitch(t *testing.T) {
	svc, st, fx, targets := newFixture()
	m, err := svc.Begin(context.Background(), "migration-1", "plan-1", "old-repo", "new-repo", targets)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stage != StageCapacity {
		t.Fatalf("stage=%s", m.Stage)
	}
	m, err = svc.Resume(context.Background(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if m.Stage != StageComplete || !fx.switched || fx.captureCount != 14 || fx.drillCount != 14 {
		t.Fatalf("migration=%+v effects=%+v", m, fx)
	}
	for _, item := range m.Items {
		if item.Capture == nil || item.Drill == nil || item.Capture.SnapshotID != item.Drill.SnapshotID {
			t.Fatalf("missing linked evidence: %+v", item)
		}
	}
	again, err := svc.Resume(context.Background(), m.ID)
	if err != nil || again.Stage != StageComplete || len(fx.calls) != 31 {
		t.Fatalf("complete replay must be inert: calls=%d err=%v", len(fx.calls), err)
	}
	if st.rows[m.ID].OldDestinationID != "old-repo" {
		t.Fatal("old repository identity not retained")
	}
}

func TestMigrationNeverSwitchesUntilAllScratchDrillsVerify(t *testing.T) {
	svc, _, fx, targets := newFixture()
	fx.failDrill = targets[7]
	m, err := svc.Begin(context.Background(), "migration-2", "plan-1", "old", "new", targets)
	if err != nil {
		t.Fatal(err)
	}
	m, err = svc.Resume(context.Background(), m.ID)
	if err == nil || fx.switched || m.Stage != StageDrill {
		t.Fatalf("failed drill switched or lost resume stage: stage=%s switched=%v err=%v", m.Stage, fx.switched, err)
	}
	fx.failDrill = ""
	m, err = svc.Resume(context.Background(), m.ID)
	if err != nil || m.Stage != StageComplete || !fx.switched {
		t.Fatalf("resume failed: %+v %v", m, err)
	}
}

func TestMigrationBindsCaptureAndDrillReceiptsToSelectedDestination(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		wrongCapture, wrongDrill bool
	}{
		{name: "capture on another repository", wrongCapture: true},
		{name: "drill from another repository", wrongDrill: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, fx, targets := newFixture()
			fx.wrongCaptureDestination = tc.wrongCapture
			fx.wrongDrillDestination = tc.wrongDrill
			m, err := svc.Begin(context.Background(), "migration-destination-binding", "plan-1", "old-repo", "new-repo", targets)
			if err != nil {
				t.Fatal(err)
			}
			got, err := svc.Resume(context.Background(), m.ID)
			if err == nil || fx.switched || got.Stage == StageComplete {
				t.Fatalf("unbound destination receipt advanced migration: stage=%s switched=%v err=%v", got.Stage, fx.switched, err)
			}
		})
	}
}

func TestMigrationReprovesCapacityBeforeSwitch(t *testing.T) {
	svc, _, fx, targets := newFixture()
	final := fx.proof
	final.AvailableBytes = final.RequiredBytes - 1
	fx.finalProof = &final
	m, err := svc.Begin(context.Background(), "migration-final-capacity", "plan-1", "old", "new", targets)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Resume(context.Background(), m.ID)
	if err == nil || fx.switched || got.Stage != StageSwitch || fx.capacityCalls != 2 {
		t.Fatalf("final capacity failure must preserve the old plan binding: stage=%s switched=%v proofs=%d err=%v", got.Stage, fx.switched, fx.capacityCalls, err)
	}
}

func TestMigrationRejectsUnknownInsufficientAndExpiredCapacity(t *testing.T) {
	for _, mutate := range []func(*CapacityProof){func(p *CapacityProof) { p.AvailableBytes = p.RequiredBytes - 1 }, func(p *CapacityProof) { p.Receipt = "" }, func(p *CapacityProof) { p.ExpiresAt = p.MeasuredAt }} {
		svc, _, fx, targets := newFixture()
		mutate(&fx.proof)
		m, _ := svc.Begin(context.Background(), "migration-capacity", "p", "old", "new", targets)
		got, err := svc.Resume(context.Background(), m.ID)
		if err == nil || fx.captureCount != 0 || fx.switched || got.Capacity != nil {
			t.Fatalf("invalid capacity admitted effects: %+v err=%v", got, err)
		}
	}
}

func TestMigrationRejectsCapacityProofForDifferentTargetSet(t *testing.T) {
	svc, _, fx, targets := newFixture()
	fx.wrongCapacityTargets = true
	m, err := svc.Begin(context.Background(), "migration-capacity-target-binding", "plan-1", "old", "new", targets)
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Resume(context.Background(), m.ID)
	if err == nil || fx.captureCount != 0 || fx.switched || got.Capacity != nil {
		t.Fatalf("capacity proof for different target set admitted effects: %+v err=%v", got, err)
	}
}

func TestMigrationRequiresExactly14UniqueExplicitTargets(t *testing.T) {
	svc, _, _, targets := newFixture()
	for _, set := range [][]string{targets[:13], append(append([]string{}, targets[:13]...), targets[0]), append(append([]string{}, targets[:13]...), "")} {
		if _, err := svc.Begin(context.Background(), "id", "p", "old", "new", set); err == nil {
			t.Fatalf("accepted invalid target set: %v", set)
		}
	}
}

func TestMigrationFileStorePersistsAndRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	store := &FileStore{Root: root}
	svc, _, fx, targets := newFixture()
	svc.store = store
	m, err := svc.Begin(context.Background(), "durable-1", "p", "old", "new", targets)
	if err != nil {
		t.Fatal(err)
	}
	m, err = svc.Resume(context.Background(), m.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc2 := New(store, fx, fakeClock{time.Now().UTC()})
	again, err := svc2.Resume(context.Background(), m.ID)
	if err != nil || again.Stage != StageComplete {
		t.Fatalf("durable replay: %s %v", again.Stage, err)
	}
	if _, _, err := store.Get(context.Background(), "../bad"); err == nil {
		t.Fatal("path traversal id accepted")
	}
}
