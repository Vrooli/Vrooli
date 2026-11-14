package notes

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

type migrationSourceFake struct {
	watermarks             []string
	watermarkReads         int
	snapshot               FleetSnapshot
	pointCount             uint64
	fenceWatermark         string
	fenceHeld              bool
	fenceStale             bool
	acquired               []string
	fenceReleaseContextErr error
}

func (s *migrationSourceFake) PinFleetSnapshot(_ context.Context, id string) (FleetSnapshot, error) {
	if s.snapshot.ID == "" {
		s.snapshot = FleetSnapshot{ID: id, Watermark: "10"}
	}
	return s.snapshot, nil
}

func (*migrationSourceFake) ReadFleetSnapshotPage(context.Context, string, FleetSnapshot, string, int) ([]FleetPoint, string, bool, error) {
	return nil, "", true, nil
}

func (*migrationSourceFake) ReadFleetChanges(context.Context, string, string, string, int) ([]FleetChange, string, error) {
	return nil, "", nil
}

func (*migrationSourceFake) CompareFleetSequence(_ context.Context, before, after string) (int, error) {
	left, err := strconv.ParseInt(before, 10, 64)
	if err != nil {
		return 0, err
	}
	right, err := strconv.ParseInt(after, 10, 64)
	if err != nil {
		return 0, err
	}
	switch {
	case left < right:
		return -1, nil
	case left > right:
		return 1, nil
	default:
		return 0, nil
	}
}

func (s *migrationSourceFake) FleetWatermark(context.Context) (string, error) {
	index := s.watermarkReads
	s.watermarkReads++
	if index >= len(s.watermarks) {
		index = len(s.watermarks) - 1
	}
	s.fenceWatermark = s.watermarks[index]
	return s.watermarks[index], nil
}

func (s *migrationSourceFake) ExpectedFleetPointCount(context.Context, string, string, string) (uint64, error) {
	return s.pointCount, nil
}

func (s *migrationSourceFake) AcquireFleetCutover(_ context.Context, expected string) (FleetCutoverFence, error) {
	s.acquired = append(s.acquired, expected)
	if s.fenceStale || expected != s.fenceWatermark {
		return nil, errors.New("stale fence watermark")
	}
	s.fenceHeld = true
	return &migrationFenceFake{source: s}, nil
}

type migrationFenceFake struct{ source *migrationSourceFake }

func (f *migrationFenceFake) CurrentWatermark(context.Context) (string, error) {
	if !f.source.fenceHeld {
		return "", errors.New("fence is not held")
	}
	return f.source.fenceWatermark, nil
}

func (f *migrationFenceFake) Release(ctx context.Context) error {
	f.source.fenceHeld = false
	f.source.fenceReleaseContextErr = ctx.Err()
	return nil
}

type migrationAdapterFake struct {
	source           *migrationSourceFake
	catchupTargets   []string
	operations       []string
	validation       FleetMigrationValidation
	promoteError     error
	promoted         bool
	rolledBack       bool
	recovered        bool
	cancelStage      context.CancelFunc
	cancelPromote    context.CancelFunc
	validationMutate func(*FleetMigrationValidation)
}

func (a *migrationAdapterFake) Stage(ctx context.Context, request FleetMigrationRequest, snapshot FleetSnapshot) error {
	a.operations = append(a.operations, "stage:"+snapshot.ID)
	scope, ok := aisearch.FleetAdmissionScopeFromContext(ctx)
	if !ok {
		return errors.New("Notes coordinator did not pass its parent admission scope")
	}
	admission, err := scope.AdmitQdrantFleetOperation(ctx, aisearch.FleetStage, request.ArmID, request.Config.Collection, request.PeakBytes)
	if err != nil {
		return err
	}
	if a.cancelStage != nil {
		a.cancelStage()
		if err := scope.CompleteQdrantFleetOperation(context.WithoutCancel(ctx), admission, false); err != nil {
			return err
		}
		return ctx.Err()
	}
	return scope.CompleteQdrantFleetOperation(ctx, admission, true)
}

func (a *migrationAdapterFake) CatchUp(_ context.Context, _ FleetMigrationRequest, _ FleetSnapshot, target string) (string, error) {
	a.operations = append(a.operations, "catch-up:"+target)
	a.catchupTargets = append(a.catchupTargets, target)
	return target, nil
}

func (a *migrationAdapterFake) Validate(_ context.Context, request FleetMigrationRequest, snapshot FleetSnapshot, watermark string, points uint64) (FleetMigrationValidation, error) {
	a.operations = append(a.operations, "validate")
	result := FleetMigrationValidation{GenerationID: request.Config.GenerationID, OwnerID: request.Config.OwnerID, Alias: request.Config.Alias, Collection: request.Config.Collection, RecipeID: request.Config.RecipeID, EmbeddingRole: request.Config.EmbeddingRole, Dimensions: request.Config.EmbeddingDimensions, SnapshotID: snapshot.ID, Watermark: watermark, PointCount: points, ReceiptDigest: "digest-bound-to-source-and-generation"}
	if a.validationMutate != nil {
		a.validationMutate(&result)
	}
	a.validation = result
	return result, nil
}

func (a *migrationAdapterFake) Promote(_ context.Context, _ FleetMigrationRequest, validation FleetMigrationValidation) error {
	a.operations = append(a.operations, "promote")
	if !a.source.fenceHeld {
		return errors.New("promotion did not hold the source fence")
	}
	if validation != a.validation {
		return errors.New("promotion validation receipt changed")
	}
	if a.promoteError != nil {
		return a.promoteError
	}
	if a.cancelPromote != nil {
		a.cancelPromote()
	}
	a.promoted = true
	return nil
}

func (a *migrationAdapterFake) Rollback(context.Context, string) error {
	a.operations = append(a.operations, "rollback")
	if !a.source.fenceHeld {
		return errors.New("rollback did not hold the source fence")
	}
	a.rolledBack = true
	return nil
}

func (a *migrationAdapterFake) Recover(context.Context, string) error {
	a.operations = append(a.operations, "recover")
	if !a.source.fenceHeld {
		return errors.New("recovery did not hold the source fence")
	}
	a.recovered = true
	return nil
}

type migrationLeaseFake struct {
	id                string
	operation         string
	armID             string
	targetCollections []string
	peakBytes         int64
	issued            []aisearch.FleetAdmission
	completed         []bool
	released          bool
	releaseContextErr error
}

func (l *migrationLeaseFake) ReceiptID() string { return l.id }
func (l *migrationLeaseFake) Release(ctx context.Context) error {
	l.released = true
	l.releaseContextErr = ctx.Err()
	return nil
}

func (l *migrationLeaseFake) AdmitQdrantFleetOperation(_ context.Context, operation aisearch.FleetOperation, armID, target string, peakBytes int64) (aisearch.FleetAdmission, error) {
	if armID != l.armID || peakBytes <= 0 || peakBytes > l.peakBytes {
		return aisearch.FleetAdmission{}, errors.New("scoped Qdrant operation exceeded its parent lease")
	}
	allowed := false
	for _, candidate := range l.targetCollections {
		if candidate == target {
			allowed = true
			break
		}
	}
	if !allowed {
		return aisearch.FleetAdmission{}, errors.New("scoped Qdrant operation target is outside its parent lease")
	}
	childID := fmt.Sprintf("%s/%d", l.id, len(l.issued)+1)
	proof := aisearch.FleetAdmission{OperationID: childID, ReceiptID: l.id, TargetID: target, PeakBytes: peakBytes, ObservedAt: time.Now().Add(-time.Second), ExpiresAt: time.Now().Add(time.Minute)}
	l.issued = append(l.issued, proof)
	_ = operation // The parent provider lease separately binds the authorized operation.
	return proof, nil
}

func (l *migrationLeaseFake) CompleteQdrantFleetOperation(_ context.Context, admission aisearch.FleetAdmission, successful bool) error {
	if admission.ReceiptID != l.id {
		return errors.New("Qdrant completion did not carry its parent lease receipt")
	}
	l.completed = append(l.completed, successful)
	return nil
}

type migrationAdmissionFake struct {
	operations []string
	leases     []*migrationLeaseFake
	reconciled []string
	deny       string
}

func (a *migrationAdmissionFake) Acquire(ctx context.Context, operation, armID string, targets []string, peakBytes int64) (FleetAdmissionLease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if operation == a.deny {
		return nil, errors.New("operation not admitted")
	}
	if armID == "" || len(targets) == 0 || peakBytes <= 0 {
		return nil, errors.New("invalid admission request")
	}
	a.operations = append(a.operations, operation)
	lease := &migrationLeaseFake{id: "receipt-" + operation + "-" + strconv.Itoa(len(a.leases)+1), operation: operation, armID: armID, targetCollections: append([]string(nil), targets...), peakBytes: peakBytes}
	a.leases = append(a.leases, lease)
	return lease, nil
}

func (a *migrationAdmissionFake) Reconcile(_ context.Context, armID string) error {
	if armID == "" {
		return errors.New("arm ID required")
	}
	a.reconciled = append(a.reconciled, armID)
	for _, lease := range a.leases {
		if !lease.released {
			lease.released = true
		}
	}
	return nil
}

func migrationFixture() (FleetMigrationCoordinatorFixture, FleetMigrationRequest) {
	source := &migrationSourceFake{watermarks: []string{"10", "11", "11", "11"}, pointCount: 3, fenceWatermark: "11"}
	config := FleetGenerationConfig{GenerationID: "arm-8", OwnerID: "notes.api", Alias: "notes-live", Collection: "notes-candidate", RecipeID: "recipe-8", EmbeddingRole: "embedding.notes.v8", EmbeddingDimensions: 2}
	request := FleetMigrationRequest{ArmID: "arm-8", PeakBytes: 128_000_000, Config: config, Source: source}
	adapter := &migrationAdapterFake{source: source}
	admission := &migrationAdmissionFake{}
	coordinator := NotesFleetMigrationCoordinator{Adapter: adapter, Admission: admission}
	return FleetMigrationCoordinatorFixture{coordinator: coordinator, source: source, adapter: adapter, admission: admission}, request
}

type FleetMigrationCoordinatorFixture struct {
	coordinator NotesFleetMigrationCoordinator
	source      *migrationSourceFake
	adapter     *migrationAdapterFake
	admission   *migrationAdmissionFake
}

func TestNotesFleetMigrationPrepareCatchupCommitHoldsFenceAndBindsReceipt(t *testing.T) {
	fixture, request := migrationFixture()
	prepared, err := fixture.coordinator.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if prepared.Snapshot.ID != "notes-arm-8" || prepared.Watermark != "11" || prepared.PointCount != 3 || prepared.ReceiptDigest == "" {
		t.Fatalf("prepared migration omitted pinned source validation: %+v", prepared)
	}
	if len(fixture.adapter.catchupTargets) != 2 || fixture.adapter.catchupTargets[0] != "10" || fixture.adapter.catchupTargets[1] != "11" {
		t.Fatalf("catch-up did not close concurrent source changes: %v", fixture.adapter.catchupTargets)
	}
	if err := fixture.coordinator.Commit(context.Background(), request, prepared); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if !fixture.adapter.promoted || fixture.source.fenceHeld || len(fixture.source.acquired) != 1 || fixture.source.acquired[0] != "11" {
		t.Fatalf("promotion did not run under and release exact cutover fence: adapter=%+v source=%+v", fixture.adapter, fixture.source)
	}
	if len(fixture.admission.operations) != 5 || fixture.admission.operations[0] != "fleet-stage" || fixture.admission.operations[1] != "fleet-catch-up" || fixture.admission.operations[2] != "fleet-catch-up" || fixture.admission.operations[3] != "fleet-validate" || fixture.admission.operations[4] != "fleet-promote" {
		t.Fatalf("prepare/commit operation admission order = %v", fixture.admission.operations)
	}
}

func TestNotesFleetMigrationFailsClosedWithoutPeakIdentityOrAdmission(t *testing.T) {
	fixture, request := migrationFixture()
	tests := []struct {
		name   string
		mutate func(*FleetMigrationRequest)
	}{
		{name: "no peak estimate", mutate: func(r *FleetMigrationRequest) { r.PeakBytes = 0 }},
		{name: "unknown collection", mutate: func(r *FleetMigrationRequest) { r.Config.Collection = "" }},
		{name: "missing recipe", mutate: func(r *FleetMigrationRequest) { r.Config.RecipeID = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := request
			test.mutate(&candidate)
			if _, err := fixture.coordinator.Prepare(context.Background(), candidate); err == nil {
				t.Fatal("Prepare() succeeded without required identity or byte estimate")
			}
		})
	}
	if len(fixture.admission.operations) != 0 || len(fixture.adapter.operations) != 0 {
		t.Fatalf("invalid migration reached owner or admission: admission=%v owner=%v", fixture.admission.operations, fixture.adapter.operations)
	}
}

func TestNotesFleetMigrationCommitRejectsStaleFenceAndFailedPromotion(t *testing.T) {
	fixture, request := migrationFixture()
	prepared, err := fixture.coordinator.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	fixture.source.fenceStale = true
	if err := fixture.coordinator.Commit(context.Background(), request, prepared); err == nil {
		t.Fatal("Commit() accepted stale source fence")
	}
	if fixture.adapter.promoted {
		t.Fatal("stale fence reached promotion")
	}
	fixture.source.fenceStale = false
	fixture.adapter.promoteError = errors.New("owner promotion interrupted")
	if err := fixture.coordinator.Commit(context.Background(), request, prepared); err == nil {
		t.Fatal("Commit() hid owner promotion failure")
	}
	if fixture.source.fenceHeld {
		t.Fatal("failed promotion leaked source fence")
	}
	lastLease := fixture.admission.leases[len(fixture.admission.leases)-1]
	if lastLease.released {
		t.Fatal("failed promotion released its admission before provider recovery reconciled the effect")
	}
}

func TestNotesFleetMigrationValidationRejectsEncoderIdentityDrift(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*FleetMigrationValidation)
	}{
		{name: "generation", mutate: func(v *FleetMigrationValidation) { v.GenerationID = "arm-other" }},
		{name: "role", mutate: func(v *FleetMigrationValidation) { v.EmbeddingRole = "embedding.other" }},
		{name: "dimensions", mutate: func(v *FleetMigrationValidation) { v.Dimensions++ }},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := migrationFixture()
			fixture.adapter.validationMutate = test.mutate
			if _, err := fixture.coordinator.Prepare(context.Background(), request); err == nil {
				t.Fatal("Prepare() accepted a validation receipt with encoder identity drift")
			}
		})
	}
}

func TestNotesFleetMigrationRollbackAndRestartRecoveryAreAdmitted(t *testing.T) {
	fixture, request := migrationFixture()
	if err := fixture.coordinator.Rollback(context.Background(), request.ArmID, []string{request.Config.Collection}, request.PeakBytes, request.Source); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if err := fixture.coordinator.Recover(context.Background(), request.ArmID, []string{request.Config.Collection}, request.PeakBytes, request.Source); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if !fixture.adapter.rolledBack || !fixture.adapter.recovered || len(fixture.admission.operations) != 2 || fixture.admission.operations[0] != "fleet-rollback" || fixture.admission.operations[1] != "fleet-recover" {
		t.Fatalf("rollback/recovery did not pass through operation admission: adapter=%+v admission=%v", fixture.adapter, fixture.admission.operations)
	}
	if len(fixture.admission.reconciled) != 1 || fixture.admission.reconciled[0] != request.ArmID {
		t.Fatalf("successful provider recovery did not reconcile retained arm leases: %v", fixture.admission.reconciled)
	}
	if fixture.source.fenceHeld {
		t.Fatal("rollback or recovery leaked the Notes source fence")
	}
}

func TestNotesFleetMigrationCancellationRetainsAdmissionAndLeavesRecoverableArm(t *testing.T) {
	fixture, request := migrationFixture()
	ctx, cancel := context.WithCancel(context.Background())
	fixture.adapter.cancelStage = cancel
	if _, err := fixture.coordinator.Prepare(ctx, request); err == nil {
		t.Fatal("Prepare() succeeded after cancellation during staging")
	}
	if len(fixture.admission.leases) != 1 || fixture.admission.leases[0].released {
		t.Fatalf("cancellation released stage admission before provider recovery: %+v", fixture.admission.leases)
	}
	fixture.adapter.cancelStage = nil
	if err := fixture.coordinator.Recover(context.Background(), request.ArmID, []string{request.Config.Collection}, request.PeakBytes, request.Source); err != nil {
		t.Fatalf("Recover() after interrupted stage error = %v", err)
	}
	if !fixture.adapter.recovered {
		t.Fatal("interrupted stage was not offered to durable owner recovery")
	}
	if !fixture.admission.leases[0].released || len(fixture.admission.reconciled) != 1 || fixture.admission.reconciled[0] != request.ArmID {
		t.Fatal("successful provider recovery did not explicitly reconcile the prior uncertain stage lease")
	}
}

func TestNotesFleetMigrationCancellationDuringPromotionStillReleasesFence(t *testing.T) {
	fixture, request := migrationFixture()
	prepared, err := fixture.coordinator.Prepare(context.Background(), request)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	fixture.adapter.cancelPromote = cancel
	if err := fixture.coordinator.Commit(ctx, request, prepared); err != nil {
		t.Fatalf("Commit() error after owner completed promotion = %v", err)
	}
	if fixture.source.fenceHeld || fixture.source.fenceReleaseContextErr != nil {
		t.Fatalf("cancellation leaked fence or canceled its release context: source=%+v", fixture.source)
	}
}
