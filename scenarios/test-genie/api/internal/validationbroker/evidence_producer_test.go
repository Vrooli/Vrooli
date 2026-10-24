package validationbroker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
)

type fakeEvidenceDeclarations struct{ err error }

func (f fakeEvidenceDeclarations) ResolveEvidenceProducer(_ context.Context, provider, producer, _ string) (*validationv1.PinnedEvidenceProducer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &validationv1.PinnedEvidenceProducer{Provider: provider, Producer: producer, Argv: []string{"/bin/true", "{run_id}", "{output_dir}"}, WorkingDirectory: "/tmp", OutputRoot: "/tmp/test-genie-evidence", TimeoutMilliseconds: 10000, MaximumOutputBytes: 64, DescriptorDigest: "sha256:descriptor"}, nil
}

type retainedSetTestAdmission struct {
	verified        []*scenariovalidationv1.RetainedEvidenceSet
	pinned          []string
	released        []string
	releaseFailures int
}

type recoveryNoopProducer struct{}

func (recoveryNoopProducer) ExecuteValidation(context.Context, *validationv1.ValidationReceipt, *validationv1.ValidationIntent, TransitionFunc) error {
	return nil
}

type recoveryReadErrorRepository struct {
	ReceiptRepository
	failedID string
	err      error
}

type retainedProducerRepository struct {
	ReceiptRepository
	producer       *validationv1.ValidationReceipt
	producerIntent *validationv1.ValidationIntent
}

func (r retainedProducerRepository) Get(ctx context.Context, receiptID string) (*validationv1.ValidationReceipt, error) {
	if receiptID == r.producer.GetReceiptId() {
		return proto.Clone(r.producer).(*validationv1.ValidationReceipt), nil
	}
	return r.ReceiptRepository.Get(ctx, receiptID)
}

func (r retainedProducerRepository) GetIntent(ctx context.Context, receiptID string) (*validationv1.ValidationIntent, error) {
	if receiptID == r.producer.GetReceiptId() {
		return proto.Clone(r.producerIntent).(*validationv1.ValidationIntent), nil
	}
	return r.ReceiptRepository.GetIntent(ctx, receiptID)
}

func (r recoveryReadErrorRepository) Get(ctx context.Context, id string) (*validationv1.ValidationReceipt, error) {
	if id == r.failedID {
		return nil, r.err
	}
	return r.ReceiptRepository.Get(ctx, id)
}

func activeRecoveryIntent(t *testing.T, repo *Repository, key, producerReceiptID string) *validationv1.ValidationReceipt {
	t.Helper()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:recovery"}
	intent := evidenceIntentForTest(key, identity)
	intent.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_INVESTIGATION
	intent.PinnedEvidenceProducer = nil
	if producerReceiptID != "" {
		intent.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{{
			ProducerReceiptId: producerReceiptID, Producer: "evidence-completeness", Target: "demo", RunId: "owner-run",
			CandidateIdentity: identity.GetIdentity(), CatalogDigest: "catalog-digest",
			Artifacts: []*commonv1.EvidenceRef{{Producer: "demo", ArtifactId: "artifact-1", Kind: "generic.file", Checksum: strings.Repeat("a", 64), SizeBytes: 8}},
		}}
	}
	admitted, err := repo.Admit(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := repo.Transition(context.Background(), admitted.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil {
		t.Fatal(err)
	}
	running, err := repo.Transition(context.Background(), queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil)
	if err != nil {
		t.Fatal(err)
	}
	return running
}

func TestRecoveryFailsOnlyInvalidRetainedReceiptAndPropagatesStorageOutage(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testsqllite(t))
	invalid := activeRecoveryIntent(t, repo, "bad-retained-bundle", "missing-producer-receipt")
	healthy := activeRecoveryIntent(t, repo, "unrelated-active-validation", "")
	leases := &retainedSetTestAdmission{}
	service := NewService(repo, nil)
	service.SetProducer(recoveryNoopProducer{})
	service.SetRetainedEvidenceAdmission(leases)
	started, err := service.Recover(ctx)
	if err != nil || started != 1 {
		t.Fatalf("recovery started=%d err=%v; invalid evidence should be isolated", started, err)
	}
	failed, err := repo.Get(ctx, invalid.GetReceiptId())
	if err != nil || failed.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED {
		t.Fatalf("invalid retained receipt was not failed: %+v err=%v", failed, err)
	}
	stillActive, err := repo.Get(ctx, healthy.GetReceiptId())
	if err != nil || stillActive.GetState() != validationv1.ReceiptState_RECEIPT_STATE_RUNNING {
		t.Fatalf("unrelated active receipt was hidden by bundle failure: %+v err=%v", stillActive, err)
	}
	if len(leases.released) != 1 || leases.released[0] != invalid.GetReceiptId() {
		t.Fatalf("invalid receipt cleanup owners=%v", leases.released)
	}

	outageRepo := recoveryReadErrorRepository{ReceiptRepository: repo, failedID: "producer-storage-key", err: errors.New("receipt store unavailable")}
	outageIntent := activeRecoveryIntent(t, repo, "storage-outage", "producer-storage-key")
	outage := NewService(outageRepo, nil)
	outage.SetProducer(recoveryNoopProducer{})
	outage.SetRetainedEvidenceAdmission(leases)
	if _, err := outage.Recover(ctx); err == nil || !strings.Contains(err.Error(), "receipt store unavailable") {
		t.Fatalf("storage-wide outage was hidden during recovery: %v", err)
	}
	if stored, err := repo.Get(ctx, outageIntent.GetReceiptId()); err != nil || stored.GetState() != validationv1.ReceiptState_RECEIPT_STATE_RUNNING {
		t.Fatalf("storage outage changed the affected receipt: %+v err=%v", stored, err)
	}
}

func TestTerminalEvidenceUnpinRetriesOnceAndExpiredLeaseBoundsMissedRelease(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testsqllite(t))
	active := activeRecoveryIntent(t, repo, "release-retry", "producer-receipt")
	leases := &retainedSetTestAdmission{releaseFailures: 1}
	service := NewService(repo, nil)
	service.SetRetainedEvidenceAdmission(leases)
	if _, err := service.Transition(ctx, active.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_FAILED, nil); err != nil {
		t.Fatal(err)
	}
	if len(leases.released) != 2 || leases.released[0] != active.GetReceiptId() || leases.released[1] != active.GetReceiptId() {
		t.Fatalf("terminal unpin retry owners=%v", leases.released)
	}

	missed := activeRecoveryIntent(t, repo, "release-expiry", "producer-receipt")
	leases.releaseFailures = 2
	if _, err := service.Transition(ctx, missed.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_FAILED, nil); err == nil || !strings.Contains(err.Error(), "existing 720h0m0s run-pin lease") {
		t.Fatalf("persistent unpin failure did not disclose bounded expiry: %v", err)
	}
	terminalReceipt, err := repo.Get(ctx, missed.GetReceiptId())
	if err != nil || terminalReceipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED {
		t.Fatalf("terminal receipt not persisted after unpin outage: %+v err=%v", terminalReceipt, err)
	}
	service.SetProducer(recoveryNoopProducer{})
	beforeRecovery := len(leases.released)
	if _, err := service.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if len(leases.released) != beforeRecovery {
		t.Fatalf("terminal receipt was retried by active-only recovery: %v", leases.released[beforeRecovery:])
	}
}

func TestDoubleUnpinFailureStillPropagatesTerminalReceipt(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testsqllite(t))
	leases := &retainedSetTestAdmission{releaseFailures: 2}
	set := &scenariovalidationv1.RetainedEvidenceSet{
		ProducerReceiptId: "producer-receipt", Producer: "evidence-completeness", Target: "demo", RunId: "owner-run",
		CandidateIdentity: "ci:v1:abc", CatalogDigest: "sha256:catalog",
		Artifacts: []*commonv1.EvidenceRef{{Producer: "demo", ArtifactId: "receipt", Kind: "generic.file", Checksum: strings.Repeat("a", 64), SizeBytes: 8}},
	}
	producerIntent := evidenceIntentForTest("successful-producer", &validationv1.SourceIdentity{SchemaVersion: 1, Identity: set.GetCandidateIdentity()})
	producerIntent.PinnedEvidenceProducer.Provider = "provider"
	producerIntent.PinnedEvidenceProducer.Producer = set.GetProducer()
	producerIntent.Targets[0].Id = set.GetTarget()
	producerReceipt := &validationv1.ValidationReceipt{
		ReceiptId: set.GetProducerReceiptId(), State: validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED,
		AdmittedIdentity:    proto.Clone(producerIntent.GetExpectedIdentity()).(*validationv1.SourceIdentity),
		ProducedEvidenceSet: proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet),
		Children:            []*validationv1.ChildOperation{{ChildId: evidenceProducerChildID(producerIntent.GetPinnedEvidenceProducer()), OperationId: set.GetRunId(), State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED}},
	}
	serviceRepo := retainedProducerRepository{ReceiptRepository: repo, producer: producerReceipt, producerIntent: producerIntent}
	service := NewService(serviceRepo, nil)
	service.SetRetainedEvidenceAdmission(leases)
	parentIntent := validIntent("terminal-release-test", "parent")
	parentIntent.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet)}
	parentResponse, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: parentIntent}))
	if err != nil {
		t.Fatal(err)
	}
	childIntent := proto.Clone(parentIntent).(*validationv1.ValidationIntent)
	childIntent.IdempotencyKey = "attached-observer"
	childResponse, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: childIntent}))
	if err != nil {
		t.Fatal(err)
	}
	parent, child := parentResponse.Msg.GetReceipt(), childResponse.Msg.GetReceipt()
	if child.GetState() != validationv1.ReceiptState_RECEIPT_STATE_ATTACHED {
		t.Fatalf("observer receipt state=%s, want attached", child.GetState())
	}
	if _, err := service.Transition(ctx, parent.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil); err != nil {
		t.Fatal(err)
	}
	terminalReceipt, err := service.Transition(ctx, parent.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, nil)
	if err == nil || !strings.Contains(err.Error(), "injected unpin failure") {
		t.Fatalf("double unpin failure was not visible: %v", err)
	}
	if terminalReceipt == nil || terminalReceipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("terminal receipt was not returned after cleanup failure: %+v", terminalReceipt)
	}
	storedChild, err := repo.Get(ctx, child.GetReceiptId())
	if err != nil || storedChild.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("terminal propagation skipped after unpin failure: child=%+v err=%v", storedChild, err)
	}
	if len(leases.released) != 2 || leases.released[0] != parent.GetReceiptId() || leases.released[1] != parent.GetReceiptId() {
		t.Fatalf("release retry owners=%v, want only parent twice", leases.released)
	}
}

func (a *retainedSetTestAdmission) Verify(_ context.Context, set *scenariovalidationv1.RetainedEvidenceSet) error {
	a.verified = append(a.verified, proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet))
	return nil
}
func (a *retainedSetTestAdmission) Pin(_ context.Context, receiptID string, _ *scenariovalidationv1.RetainedEvidenceSet) error {
	a.pinned = append(a.pinned, receiptID)
	return nil
}
func (a *retainedSetTestAdmission) Release(_ context.Context, receiptID string, _ *scenariovalidationv1.RetainedEvidenceSet) error {
	a.released = append(a.released, receiptID)
	if a.releaseFailures > 0 {
		a.releaseFailures--
		return errors.New("injected unpin failure")
	}
	return nil
}

type retainedSetTestRuns struct {
	*fakeSuiteRuns
	owner *runmanager.Manager
}

func (r *retainedSetTestRuns) RetainedEvidenceSet(scenario, runID, receiptID, producer, identity string) (*scenariovalidationv1.RetainedEvidenceSet, error) {
	return r.owner.RetainedEvidenceSet(scenario, runID, receiptID, producer, identity)
}

func newRetainedSetTestRuns(t *testing.T, runID string) *retainedSetTestRuns {
	t.Helper()
	root := t.TempDir()
	outputDir := filepath.Join(sharedartifacts.RunDir(root, runID), "evidence-producer", "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "receipt.json"), []byte("owner output"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.RefreshArtifactCatalog(root, runID, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedartifacts.WriteArtifactCatalog(root, catalog); err != nil {
		t.Fatal(err)
	}
	owner := runmanager.New(&cancellationFenceExecutor{}, root).WithArtifactRootResolver(func(string) (string, error) { return root, nil })
	t.Cleanup(owner.Shutdown)
	return &retainedSetTestRuns{fakeSuiteRuns: &fakeSuiteRuns{}, owner: owner}
}

func TestProducerReceiptSetPassesUnchangedIntoValidationIntent(t *testing.T) {
	ctx := context.Background()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:owner-handoff"}
	providerIdentity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:bas-provider"}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../"))
	pinned, err := (DescriptorEvidenceResolver{RepoRoot: repoRoot}).ResolveEvidenceProducer(ctx, "browser-automation-studio", "evidence-completeness", "browser-automation-studio")
	if err != nil {
		t.Fatal(err)
	}
	pinned.SourceIdentity = providerIdentity.GetIdentity()
	intent := evidenceIntentForTest("owner-handoff", identity)
	intent.Targets[0].Id = "browser-automation-studio"
	intent.PinnedEvidenceProducer = pinned
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	runID := validationSuiteRunID(admission.Receipt.GetReceiptId(), "browser-automation-studio", 1)
	root := t.TempDir()
	outputDir := filepath.Join(sharedartifacts.RunDir(root, runID), "evidence-producer", "output")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "receipt.json"), []byte("owner output"), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.RefreshArtifactCatalog(root, runID, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedartifacts.WriteArtifactCatalog(root, catalog); err != nil {
		t.Fatal(err)
	}
	owner := runmanager.New(&cancellationFenceExecutor{}, root).WithArtifactRootResolver(func(string) (string, error) { return root, nil })
	defer owner.Shutdown()
	runs := &retainedSetTestRuns{fakeSuiteRuns: &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}, owner: owner}
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, providerIdentity, identity, providerIdentity}})
	if err := producer.ExecuteValidation(ctx, admission.Receipt, intent, repo.Transition); err != nil {
		t.Fatal(err)
	}
	producerReceipt, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	set := producerReceipt.GetProducedEvidenceSet()
	if producerReceipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || set == nil || set.GetCatalogDigest() == "" || len(set.GetArtifacts()) != 1 {
		t.Fatalf("producer did not publish owner set: receipt=%+v", producerReceipt)
	}
	consumer := proto.Clone(intent).(*validationv1.ValidationIntent)
	consumer.IdempotencyKey = "owner-handoff-consumer"
	consumer.Purpose = validationv1.ValidationPurpose_VALIDATION_PURPOSE_INVESTIGATION
	consumer.PinnedEvidenceProducer = nil
	consumer.RetainedEvidenceSets = []*scenariovalidationv1.RetainedEvidenceSet{proto.Clone(set).(*scenariovalidationv1.RetainedEvidenceSet)}
	consumerAdmission := &retainedSetTestAdmission{}
	service := NewService(repo, nil)
	service.SetRetainedEvidenceAdmission(consumerAdmission)
	changed := proto.Clone(consumer).(*validationv1.ValidationIntent)
	changed.RetainedEvidenceSets[0].Artifacts[0].Checksum = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: changed})); err == nil {
		t.Fatal("caller-modified owner checksum was admitted")
	}
	if len(consumerAdmission.verified) != 0 {
		t.Fatal("changed input reached catalog consumption")
	}
	created, err := service.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: consumer}))
	if err != nil {
		t.Fatal(err)
	}
	if len(consumerAdmission.verified) != 1 || !proto.Equal(consumerAdmission.verified[0], set) {
		t.Fatalf("consumer did not receive unchanged producer set: %+v", consumerAdmission.verified)
	}
	consumerID := created.Msg.GetReceipt().GetReceiptId()
	if len(consumerAdmission.pinned) != 1 || consumerAdmission.pinned[0] != consumerID {
		t.Fatalf("consumer evidence lease owner=%v want %s", consumerAdmission.pinned, consumerID)
	}
	if _, err := service.Transition(ctx, consumerID, validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, nil); err != nil {
		t.Fatal(err)
	}
	if len(consumerAdmission.released) != 1 || consumerAdmission.released[0] != consumerID {
		t.Fatalf("terminal cleanup released lease owners=%v want only %s", consumerAdmission.released, consumerID)
	}
}

func TestEvidenceProducerDrainAndRecoveryKeepAdmissionDeadlineAndChild(t *testing.T) {
	ctx := context.Background()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:deadline-owner"}
	repo := NewRepository(testsqllite(t))
	intent := evidenceIntentForTest("deadline-owner", identity)
	intent.Targets[0].Id = "demo"
	intent.PinnedEvidenceProducer.TimeoutMilliseconds = 80
	intent.PinnedEvidenceProducer.SourceIdentity = identity.GetIdentity()
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	admittedAt := admission.Receipt.GetCreatedAt().AsTime()
	runID := validationSuiteRunID(admission.Receipt.GetReceiptId(), "demo", 1)
	runs := newRetainedSetTestRuns(t, runID)
	var waitRuns []string
	var deadlines []time.Time
	var waitCount atomic.Int32
	runs.waitFn = func(waitCtx context.Context, _, childRunID string) (runmanager.LiveStatus, error) {
		deadline, ok := waitCtx.Deadline()
		if !ok {
			return runmanager.LiveStatus{}, errors.New("producer observer has no deadline")
		}
		waitRuns = append(waitRuns, childRunID)
		deadlines = append(deadlines, deadline)
		if waitCount.Add(1) == 1 {
			return runmanager.LiveStatus{}, errors.New("simulated observer loss")
		}
		// Model a slow terminal/drain path that outlasts the command's declared
		// 80 ms timeout but completes inside the original cleanup reserve.
		timer := time.NewTimer(120 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-waitCtx.Done():
			return runmanager.LiveStatus{}, waitCtx.Err()
		case <-timer.C:
			return runmanager.LiveStatus{Status: sharedruns.StatusPassed}, nil
		}
	}
	defer runs.owner.Shutdown()
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, identity, identity}})
	if err := producer.ExecuteValidation(ctx, admission.Receipt, intent, repo.Transition); !errors.Is(err, errEvidencePending) {
		t.Fatalf("first observation err=%v, want retained pending child", err)
	}
	pending, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if len(pending.GetChildren()) != 1 || pending.GetChildren()[0].GetOperationId() != runID || pending.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING {
		t.Fatalf("durable producer child after observer loss=%+v", pending.GetChildren())
	}
	// Restart consumes some of the original observation window; it must not
	// renew the producer's timeout-plus-drain deadline.
	time.Sleep(40 * time.Millisecond)
	if err := producer.ExecuteValidation(ctx, pending, intent, repo.Transition); err != nil {
		t.Fatalf("recovered producer observation: %v", err)
	}
	terminal, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil || terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED || terminal.GetProducedEvidenceSet() == nil {
		t.Fatalf("slow producer did not publish after drain: receipt=%+v err=%v", terminal, err)
	}
	wantDeadline := admittedAt.Add(80*time.Millisecond + evidenceProducerCleanupReserve)
	if len(deadlines) != 2 || !deadlines[0].Equal(wantDeadline) || !deadlines[1].Equal(wantDeadline) {
		t.Fatalf("observation deadlines=%v, want original admission deadline %s", deadlines, wantDeadline)
	}
	if len(waitRuns) != 2 || waitRuns[0] != runID || waitRuns[1] != runID {
		t.Fatalf("recovery observed child runs=%v, want same durable child %s", waitRuns, runID)
	}
	if len(runs.starts) != 2 || runs.starts[0].Input.Request.RunID != runID || runs.starts[1].Input.Request.RunID != runID || len(runs.accepted) != 1 {
		t.Fatalf("recovery replayed producer admission: starts=%d accepted=%v", len(runs.starts), runs.accepted)
	}
	if time.Since(admittedAt) <= 80*time.Millisecond {
		t.Fatal("slow drain test did not outlast the declared producer command timeout")
	}

	longDeadline, err := evidenceProductionDeadline(admittedAt, 45*time.Minute, evidenceProducerCleanupReserve)
	if err != nil {
		t.Fatal(err)
	}
	if !longDeadline.Equal(admittedAt.Add(46 * time.Minute)) {
		t.Fatalf("generic producer observation deadline=%s, want its 45m pinned timeout plus reserve", longDeadline)
	}
}

func TestExpiredEvidenceProducerRecoveryFencesWithoutStarting(t *testing.T) {
	ctx := context.Background()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:expired-recovery"}
	intent := evidenceIntentForTest("expired-recovery", identity)
	intent.PinnedEvidenceProducer.TimeoutMilliseconds = 50
	repo := NewRepository(testsqllite(t))
	repo.now = func() time.Time { return time.Now().Add(-evidenceProducerCleanupReserve - time.Second) }
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	runID := validationSuiteRunID(admission.Receipt.GetReceiptId(), "demo", 1)
	queued, err := repo.Transition(ctx, admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil {
		t.Fatal(err)
	}
	running, err := repo.Transition(ctx, queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(receipt *validationv1.ValidationReceipt) error {
		receipt.AdmittedIdentity = proto.Clone(identity).(*validationv1.SourceIdentity)
		receipt.Children = append(receipt.Children, &validationv1.ChildOperation{ChildId: evidenceProducerChildID(intent.GetPinnedEvidenceProducer()), OperationId: runID, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runs := &fakeSuiteRuns{}
	producer := NewRunProducer(runs)
	if err := producer.ExecuteValidation(ctx, running, intent, repo.Transition); err != nil {
		t.Fatal(err)
	}
	terminal, err := repo.Get(ctx, running.GetReceiptId())
	if err != nil || terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || terminal.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_DEADLINE_EXCEEDED {
		t.Fatalf("expired producer terminal=%+v err=%v", terminal, err)
	}
	if len(runs.starts) != 0 || len(runs.accepted) != 0 {
		t.Fatalf("expired recovery launched work: starts=%d accepted=%v", len(runs.starts), runs.accepted)
	}
	if len(runs.evidenceAborts) != 1 || runs.evidenceAborts[0].Input.Request.RunID != runID {
		t.Fatalf("expired recovery did not fence the same child: %+v", runs.evidenceAborts)
	}
	if len(terminal.GetChildren()) != 1 || terminal.GetChildren()[0].GetOperationId() != runID || terminal.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
		t.Fatalf("expired child was not durably drained: %+v", terminal.GetChildren())
	}
}

func TestEvidenceProducerDeadlineExpiryDuringWaitDrainsLateSuccess(t *testing.T) {
	ctx := context.Background()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:wait-expiry"}
	intent := evidenceIntentForTest("wait-expiry", identity)
	intent.PinnedEvidenceProducer.TimeoutMilliseconds = 50
	repo := NewRepository(testsqllite(t))
	admittedAt := time.Now().Add(-50*time.Millisecond - evidenceProducerCleanupReserve + 80*time.Millisecond)
	repo.now = func() time.Time { return admittedAt }
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	runID := validationSuiteRunID(admission.Receipt.GetReceiptId(), "demo", 1)
	runs := &fakeSuiteRuns{evidenceAbortStatus: sharedruns.StatusPassed}
	runs.waitFn = func(_ context.Context, _, observedRunID string) (runmanager.LiveStatus, error) {
		if observedRunID != runID {
			return runmanager.LiveStatus{}, fmt.Errorf("wait run %q, want %q", observedRunID, runID)
		}
		time.Sleep(120 * time.Millisecond)
		// Some owners can deliver a late terminal success even after the
		// observation context expires. The broker must still abort/drain and
		// refuse to publish that output.
		return runmanager.LiveStatus{Status: sharedruns.StatusPassed}, nil
	}
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, identity}})
	if err := producer.ExecuteValidation(ctx, admission.Receipt, intent, repo.Transition); err != nil {
		t.Fatal(err)
	}
	terminal, err := repo.Get(ctx, admission.Receipt.GetReceiptId())
	if err != nil || terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || terminal.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_DEADLINE_EXCEEDED || terminal.GetProducedEvidenceSet() != nil {
		t.Fatalf("late Wait success escaped producer deadline: receipt=%+v err=%v", terminal, err)
	}
	if len(runs.starts) != 1 || len(runs.accepted) != 1 || runs.starts[0].Input.Request.RunID != runID {
		t.Fatalf("expired observation replayed child: starts=%d accepted=%v", len(runs.starts), runs.accepted)
	}
	if len(runs.evidenceAborts) != 1 || runs.evidenceAborts[0].Input.Request.RunID != runID {
		t.Fatalf("deadline expiry did not drain the same child: %+v", runs.evidenceAborts)
	}
	if len(terminal.GetChildren()) != 1 || terminal.GetChildren()[0].GetOperationId() != runID || terminal.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_SUCCEEDED {
		t.Fatalf("confirmed owner terminal state not retained honestly: %+v", terminal.GetChildren())
	}
}

func TestCheckedInBASEvidenceProducerDeclarationResolvesBoundedOutput(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(source), "../../../../../"))
	pinned, err := (DescriptorEvidenceResolver{RepoRoot: repoRoot}).ResolveEvidenceProducer(context.Background(), "browser-automation-studio", "evidence-completeness", "browser-automation-studio")
	if err != nil {
		t.Fatal(err)
	}
	wantWorkdir := filepath.Join(repoRoot, "scenarios", "browser-automation-studio")
	if pinned.GetWorkingDirectory() != wantWorkdir || pinned.GetOutputRoot() != "/dev/shm/tg-output" || pinned.GetTimeoutMilliseconds() != 10*60*1000 || pinned.GetMaximumOutputBytes() != 16<<20 {
		t.Fatalf("resolved producer bounds/workdir = %+v", pinned)
	}
	if got := pinned.GetArgv(); len(got) != 3 || got[0] != "node" || got[1] != "api/cmd/evidence-completeness-cohort/qualification.mjs" || got[2] != "{output_dir}" {
		t.Fatalf("resolved producer argv=%v", got)
	}
}

type mutableEvidenceDeclarations struct {
	err error
}

type cancellationFenceExecutor struct{ calls atomic.Int32 }

func (e *cancellationFenceExecutor) ExecuteWithEvents(_ context.Context, input execution.SuiteExecutionInput, _ orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	e.calls.Add(1)
	return &orchestrator.SuiteExecutionResult{RunID: input.Request.RunID, ScenarioName: input.Request.ScenarioName, Success: true, Verdict: "PRODUCER_COMPLETED"}, nil
}

type barrierEvidenceAborter struct {
	entered chan struct{}
	release chan struct{}
}

func (a *barrierEvidenceAborter) AbortValidation(context.Context, *validationv1.ValidationReceipt, string, string) error {
	return nil
}
func (a *barrierEvidenceAborter) AbortEvidenceValidation(_ context.Context, receipt *validationv1.ValidationReceipt, _ *validationv1.ValidationIntent, _, _ string) error {
	for _, child := range receipt.GetChildren() {
		if child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING || child.GetState() == validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
			child.State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
			child.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
		}
	}
	close(a.entered)
	<-a.release
	return nil
}

func (f *mutableEvidenceDeclarations) ResolveEvidenceProducer(_ context.Context, provider, producer, _ string) (*validationv1.PinnedEvidenceProducer, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &validationv1.PinnedEvidenceProducer{Provider: provider, Producer: producer, Argv: []string{"/bin/true", "{run_id}", "{output_dir}"}, WorkingDirectory: "/tmp", OutputRoot: "/tmp/test-genie-evidence", TimeoutMilliseconds: 10000, MaximumOutputBytes: 64, DescriptorDigest: "sha256:descriptor"}, nil
}

func TestEvidenceProductionTypedAdmissionAndUnchangedKeyReplay(t *testing.T) {
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	repo := NewRepository(testsqllite(t))
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, identity, identity}})
	service := NewService(repo, nil)
	service.SetProducer(producer)
	service.SetIdentityResolver(producer.identity)
	declarations := &mutableEvidenceDeclarations{}
	service.SetEvidenceDeclarationResolver(declarations)
	request := &validationv1.CreateEvidenceProductionRequest{IdempotencyKey: "producer-key", CallerScenario: "qualification", Provider: "owner", Producer: "refresh", CandidateScenario: "demo", ExpectedCandidateIdentity: proto.Clone(identity).(*validationv1.SourceIdentity)}
	created, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	terminal := waitForTerminalReceipt(t, service, created.Msg.GetReceipt())
	if terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || terminal.GetProducedEvidenceSet() != nil || terminal.GetAchievedStrength() != validationv1.ValidationStrength_VALIDATION_STRENGTH_UNSPECIFIED {
		t.Fatalf("producer without an owner catalog set was treated as published: %+v", terminal)
	}
	if len(runs.starts) != 1 || runs.starts[0].Input.EvidenceProducer == nil || runs.starts[0].Input.Request.RunID == "" {
		t.Fatalf("producer not routed through pinned run admission: %+v", runs.starts)
	}
	if runs.starts[0].Input.EvidenceProducer.Argv[1] == "{run_id}" {
		t.Fatal("server run id placeholder was not resolved")
	}
	declarations.err = errors.New("descriptor changed after admission")
	replayed, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if replayed.Msg.GetReceipt().GetReceiptId() != created.Msg.GetReceipt().GetReceiptId() {
		t.Fatalf("unchanged key replay receipt=%q want %q", replayed.Msg.GetReceipt().GetReceiptId(), created.Msg.GetReceipt().GetReceiptId())
	}
	changed := proto.Clone(request).(*validationv1.CreateEvidenceProductionRequest)
	changed.Producer = "different"
	if _, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(changed)); err == nil {
		t.Fatal("changed producer intent reused an existing idempotency key")
	}
}

func TestEvidenceProductionRejectsMidRunCandidateAndProviderDrift(t *testing.T) {
	candidate := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	provider := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:provider"}
	candidateDrift := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate-drift"}
	providerDrift := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:provider-drift"}
	for _, tc := range []struct {
		name       string
		identities []*validationv1.SourceIdentity
		observed   string
	}{
		{name: "candidate changed while running", identities: []*validationv1.SourceIdentity{provider, candidate, candidate, provider, candidateDrift}, observed: candidateDrift.GetIdentity()},
		{name: "provider changed while running", identities: []*validationv1.SourceIdentity{provider, candidate, candidate, provider, candidate, providerDrift}, observed: candidate.GetIdentity()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
			producer := NewRunProducer(runs, &sequenceIdentityResolver{values: tc.identities})
			service := NewService(NewRepository(testsqllite(t)), producer)
			service.SetProducer(producer)
			service.SetIdentityResolver(producer.identity)
			service.SetEvidenceDeclarationResolver(fakeEvidenceDeclarations{})
			request := &validationv1.CreateEvidenceProductionRequest{IdempotencyKey: "drift-key", CallerScenario: "qualification", Provider: "owner", Producer: "refresh", CandidateScenario: "demo", ExpectedCandidateIdentity: proto.Clone(candidate).(*validationv1.SourceIdentity)}
			created, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(request))
			if err != nil {
				t.Fatal(err)
			}
			terminal := waitForTerminalReceipt(t, service, created.Msg.GetReceipt())
			if terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_FAILED || terminal.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_IDENTITY_CHANGED {
				t.Fatalf("drift receipt state/reason=%s/%s detail=%s", terminal.GetState(), terminal.GetReasonCode(), terminal.GetDetail())
			}
			if terminal.GetObservedIdentity().GetIdentity() != tc.observed {
				t.Fatalf("observed identity=%q want %q", terminal.GetObservedIdentity().GetIdentity(), tc.observed)
			}
			if len(runs.starts) != 1 {
				t.Fatalf("drift caused %d producer starts, want exactly one", len(runs.starts))
			}
		})
	}
}

func TestEvidenceProducerAbortRoutesPendingAndActiveScenarioChildren(t *testing.T) {
	for _, state := range []validationv1.ChildOperationState{validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING, validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING} {
		t.Run(state.String(), func(t *testing.T) {
			intent := evidenceIntentForTest("abort-"+state.String(), &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"})
			repo := NewRepository(testsqllite(t))
			admission, err := repo.Admit(context.Background(), intent)
			if err != nil {
				t.Fatal(err)
			}
			queued, err := repo.Transition(context.Background(), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
			if err != nil {
				t.Fatal(err)
			}
			runID := validationSuiteRunID(queued.GetReceiptId(), "demo", 1)
			childID := "scenario:owner:evidence:refresh"
			receipt, err := repo.Transition(context.Background(), queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
				r.AdmittedIdentity = proto.Clone(intent.GetExpectedIdentity()).(*validationv1.SourceIdentity)
				r.Children = append(r.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: state, Owner: "test-genie", OperationId: runID})
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			runs := &fakeSuiteRuns{}
			service := NewService(repo, NewRunProducer(runs))
			response, err := service.AbortValidationWork(context.Background(), connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: receipt.GetReceiptId(), RequestedBy: "operator", Reason: "stop producer"}))
			if err != nil {
				t.Fatal(err)
			}
			if response.Msg.GetReceipt().GetState() != validationv1.ReceiptState_RECEIPT_STATE_CANCELLED {
				t.Fatalf("abort state=%s", response.Msg.GetReceipt().GetState())
			}
			if len(runs.aborts) != 1 || runs.aborts[0] != "owner:"+runID {
				t.Fatalf("producer abort route=%v", runs.aborts)
			}
		})
	}
}

func TestEvidenceProducerCancellationFenceAndTerminalWriteRecovery(t *testing.T) {
	ctx := context.Background()
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	intent := evidenceIntentForTest("cancel-fence-write-recovery", identity)
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	queued, err := repo.Transition(ctx, admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil {
		t.Fatal(err)
	}
	runID := validationSuiteRunID(queued.GetReceiptId(), "demo", 1)
	childID := evidenceProducerChildID(intent.GetPinnedEvidenceProducer())
	receipt, err := repo.Transition(ctx, queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
		r.AdmittedIdentity = proto.Clone(identity).(*validationv1.SourceIdentity)
		r.Children = append(r.Children, &validationv1.ChildOperation{ChildId: childID, Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING, Owner: "test-genie", OperationId: runID})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	executor := &cancellationFenceExecutor{}
	owner := runmanager.New(executor, root)
	defer owner.Shutdown()
	producer := NewRunProducer(owner)
	service := NewService(repo, producer)
	service.SetProducer(producer)
	if _, err := repo.db.ExecContext(ctx, `CREATE TRIGGER reject_cancel BEFORE UPDATE OF state ON validation_receipts WHEN NEW.state = 9 BEGIN SELECT RAISE(ABORT, 'injected transient terminal write failure'); END`); err != nil {
		t.Fatal(err)
	}
	request := connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: receipt.GetReceiptId(), RequestedBy: "operator", Reason: "cancel before manager start"})
	if _, err := service.AbortValidationWork(ctx, request); err == nil {
		t.Fatal("injected cancelled terminal write unexpectedly succeeded")
	}
	interrupted, err := repo.Get(ctx, receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.GetState() != validationv1.ReceiptState_RECEIPT_STATE_RUNNING || interrupted.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
		t.Fatalf("failed terminal write partially persisted parent/child: %s/%s", interrupted.GetState(), interrupted.GetChildren()[0].GetState())
	}
	if _, err := repo.db.ExecContext(ctx, `DROP TRIGGER reject_cancel`); err != nil {
		t.Fatal(err)
	}
	response, err := service.AbortValidationWork(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	final := response.Msg.GetReceipt()
	if final.GetState() != validationv1.ReceiptState_RECEIPT_STATE_CANCELLED || final.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
		t.Fatalf("recovered cancellation parent/child=%s/%s", final.GetState(), final.GetChildren()[0].GetState())
	}
	options, err := evidenceProducerStartOptions(final, intent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Start(options); err != nil {
		t.Fatalf("late Start after recovered cancellation: %v", err)
	}
	if got := executor.calls.Load(); got != 0 {
		t.Fatalf("late producer Start executed %d times", got)
	}
}

func TestEvidenceProducerParentChildTerminalRaceOrders(t *testing.T) {
	for _, workerWins := range []bool{true, false} {
		name := "abort transition wins"
		if workerWins {
			name = "worker transition wins"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
			intent := evidenceIntentForTest("race-"+name, identity)
			repo := NewRepository(testsqllite(t))
			admission, err := repo.Admit(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			queued, err := repo.Transition(ctx, admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
			if err != nil {
				t.Fatal(err)
			}
			runID := validationSuiteRunID(queued.GetReceiptId(), "demo", 1)
			receipt, err := repo.Transition(ctx, queued.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
				r.AdmittedIdentity = proto.Clone(identity).(*validationv1.SourceIdentity)
				r.Children = append(r.Children, &validationv1.ChildOperation{ChildId: evidenceProducerChildID(intent.GetPinnedEvidenceProducer()), Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, OperationId: runID})
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			aborter := &barrierEvidenceAborter{entered: make(chan struct{}), release: make(chan struct{})}
			service := NewService(repo, aborter)
			response := make(chan *validationv1.ValidationReceipt, 1)
			failure := make(chan error, 1)
			go func() {
				result, callErr := service.AbortValidationWork(ctx, connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: receipt.GetReceiptId(), RequestedBy: "operator", Reason: "race test"}))
				if callErr != nil {
					failure <- callErr
					return
				}
				response <- result.Msg.GetReceipt()
			}()
			<-aborter.entered
			if workerWins {
				_, err = repo.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, func(r *validationv1.ValidationReceipt) error {
					r.Children[0].State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED
					r.Children[0].ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			close(aborter.release)
			var final *validationv1.ValidationReceipt
			select {
			case err := <-failure:
				t.Fatal(err)
			case final = <-response:
			case <-time.After(time.Second):
				t.Fatal("abort barrier did not settle")
			}
			if !workerWins {
				// The producer's delayed terminal write loses to the service's atomic
				// parent+child cancellation commit and must not rewrite the child.
				_, err = repo.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, func(r *validationv1.ValidationReceipt) error {
					r.Children[0].State = validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				final, err = repo.Get(ctx, receipt.GetReceiptId())
				if err != nil {
					t.Fatal(err)
				}
			}
			if final.GetState() != validationv1.ReceiptState_RECEIPT_STATE_CANCELLED || final.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED {
				t.Fatalf("race left incoherent terminal parent/child: %s/%s", final.GetState(), final.GetChildren()[0].GetState())
			}
		})
	}
}

func TestEvidenceProducerTerminalOutcomesAndPersistenceRecovery(t *testing.T) {
	for _, tc := range []struct {
		status string
		state  validationv1.ReceiptState
		child  validationv1.ChildOperationState
	}{
		{sharedruns.StatusFailed, validationv1.ReceiptState_RECEIPT_STATE_FAILED, validationv1.ChildOperationState_CHILD_OPERATION_STATE_FAILED},
		{sharedruns.StatusAborted, validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, validationv1.ChildOperationState_CHILD_OPERATION_STATE_CANCELLED},
	} {
		t.Run(tc.status, func(t *testing.T) {
			ctx := context.Background()
			identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
			intent := evidenceIntentForTest("terminal-"+tc.status, identity)
			repo := NewRepository(testsqllite(t))
			admitted, err := repo.Admit(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: tc.status}}
			producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity}})
			persistErr := errors.New("terminal receipt write unavailable")
			failTerminal := func(ctx context.Context, id string, next validationv1.ReceiptState, mutate func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error) {
				if terminal(next) {
					return nil, persistErr
				}
				return repo.Transition(ctx, id, next, mutate)
			}
			err = producer.ExecuteValidation(ctx, admitted.Receipt, intent, failTerminal)
			if !errors.Is(err, errEvidencePending) || !errors.Is(err, persistErr) {
				t.Fatalf("terminal write failure must remain recoverable, got %v", err)
			}
			retained, err := repo.Get(ctx, admitted.Receipt.GetReceiptId())
			if err != nil || terminal(retained.GetState()) {
				t.Fatalf("failed persistence reported completion: %v %v", retained, err)
			}
			if err := producer.ExecuteValidation(ctx, retained, intent, repo.Transition); err != nil {
				t.Fatal(err)
			}
			final, err := repo.Get(ctx, admitted.Receipt.GetReceiptId())
			if err != nil || final.GetState() != tc.state || len(final.GetChildren()) != 1 || final.GetChildren()[0].GetState() != tc.child {
				t.Fatalf("terminal outcome = %v, error = %v", final, err)
			}
			if tc.status == sharedruns.StatusAborted && final.GetReasonCode() != validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_ABORTED {
				t.Fatalf("abort lost its reason: %v", final)
			}
			if len(runs.starts) != 2 || runs.starts[0].Input.Request.RunID != runs.starts[1].Input.Request.RunID {
				t.Fatalf("recovery changed the original child: %+v", runs.starts)
			}
		})
	}
}

func TestEvidenceProductionRefusesProviderSourceDriftBeforeNewLaunch(t *testing.T) {
	candidate := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	changedProvider := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:provider-drift"}
	runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{candidate, changedProvider}})
	service := NewService(NewRepository(testsqllite(t)), nil)
	service.SetProducer(producer)
	service.SetIdentityResolver(producer.identity)
	service.SetEvidenceDeclarationResolver(fakeEvidenceDeclarations{})
	request := &validationv1.CreateEvidenceProductionRequest{IdempotencyKey: "source-drift", CallerScenario: "qualification", Provider: "owner", Producer: "refresh", CandidateScenario: "demo", ExpectedCandidateIdentity: proto.Clone(candidate).(*validationv1.SourceIdentity)}
	if _, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(request)); err == nil {
		t.Fatal("provider source drift launched producer")
	}
	if len(runs.starts) != 0 {
		t.Fatalf("source drift dispatched producer: %d starts", len(runs.starts))
	}
}

func TestEvidenceProductionUnknownDeclarationAndCandidateMismatchRefuseBeforeDispatch(t *testing.T) {
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	for _, tc := range []struct {
		name         string
		declarations fakeEvidenceDeclarations
		observed     *validationv1.SourceIdentity
	}{
		{name: "unknown declaration", declarations: fakeEvidenceDeclarations{err: errors.New("unknown producer")}, observed: identity},
		{name: "candidate mismatch", declarations: fakeEvidenceDeclarations{}, observed: &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:changed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := &fakeSuiteRuns{wait: runmanager.LiveStatus{Status: sharedruns.StatusPassed}}
			repo := NewRepository(testsqllite(t))
			producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{tc.observed}})
			service := NewService(repo, nil)
			service.SetProducer(producer)
			service.SetIdentityResolver(producer.identity)
			service.SetEvidenceDeclarationResolver(tc.declarations)
			request := &validationv1.CreateEvidenceProductionRequest{IdempotencyKey: "refuse-key", CallerScenario: "qualification", Provider: "owner", Producer: "refresh", CandidateScenario: "demo", ExpectedCandidateIdentity: proto.Clone(identity).(*validationv1.SourceIdentity)}
			if _, err := service.CreateEvidenceProduction(context.Background(), connect.NewRequest(request)); err == nil {
				t.Fatal("unsafe producer admission succeeded")
			}
			if len(runs.starts) != 0 {
				t.Fatalf("refusal dispatched producer: %+v", runs.starts)
			}
		})
	}
}

func TestEvidenceProducerReattachesPendingChildWithoutNewRunID(t *testing.T) {
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	intent := evidenceIntentForTest("reattach", identity)
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	runs := newRetainedSetTestRuns(t, validationSuiteRunID(admission.Receipt.GetReceiptId(), "demo", 1))
	runs.waitErr = errors.New("observer lost")
	runs.responseErrors = []error{errors.New("start response lost")}
	runs.wait = runmanager.LiveStatus{Status: sharedruns.StatusPassed}
	producer := NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, identity, identity, identity}})
	err = producer.ExecuteValidation(context.Background(), admission.Receipt, intent, repo.Transition)
	if !errors.Is(err, errEvidencePending) {
		t.Fatalf("first execution error=%v", err)
	}
	stored, err := repo.Get(context.Background(), admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.GetChildren()) != 1 || stored.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_PENDING {
		t.Fatalf("uncertain launch child was not retained pending: %+v", stored.GetChildren())
	}
	runID := stored.GetChildren()[0].GetOperationId()
	err = producer.ExecuteValidation(context.Background(), stored, intent, repo.Transition)
	if !errors.Is(err, errEvidencePending) {
		t.Fatalf("second execution error=%v", err)
	}
	stored, err = repo.Get(context.Background(), admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetChildren()[0].GetState() != validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING {
		t.Fatalf("observed nonterminal child state=%s", stored.GetChildren()[0].GetState())
	}
	runs.waitErr = nil
	err = producer.ExecuteValidation(context.Background(), stored, intent, repo.Transition)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs.starts) != 3 || runs.starts[0].Input.Request.RunID != runID || runs.starts[1].Input.Request.RunID != runID || runs.starts[2].Input.Request.RunID != runID {
		t.Fatalf("reattachment created/replayed another run: %+v", runs.starts)
	}
	stored, err = repo.Get(context.Background(), admission.Receipt.GetReceiptId())
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("recovered receipt state=%s detail=%s", stored.GetState(), stored.GetDetail())
	}
}

func TestEvidenceProductionRecoverReattachesPinnedChildAfterServiceRestart(t *testing.T) {
	identity := &validationv1.SourceIdentity{SchemaVersion: 1, Identity: "ci:v1:candidate"}
	intent := evidenceIntentForTest("restart-key", identity)
	repo := NewRepository(testsqllite(t))
	admission, err := repo.Admit(context.Background(), intent)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := repo.Transition(context.Background(), admission.Receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil {
		t.Fatal(err)
	}
	runID := validationSuiteRunID(receipt.GetReceiptId(), "demo", 1)
	receipt, err = repo.Transition(context.Background(), receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, func(r *validationv1.ValidationReceipt) error {
		r.AdmittedIdentity = proto.Clone(identity).(*validationv1.SourceIdentity)
		r.Children = append(r.Children, &validationv1.ChildOperation{ChildId: "scenario:owner:evidence:refresh", Kind: validationv1.ChildOperationKind_CHILD_OPERATION_KIND_TEST_RUN, State: validationv1.ChildOperationState_CHILD_OPERATION_STATE_RUNNING, Owner: "test-genie", OperationId: runID})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runs := newRetainedSetTestRuns(t, runID)
	runs.wait = runmanager.LiveStatus{Status: sharedruns.StatusPassed}
	restarted := NewService(repo, nil)
	restarted.SetProducer(NewRunProducer(runs, &sequenceIdentityResolver{values: []*validationv1.SourceIdentity{identity, identity}}))
	if count, err := restarted.Recover(context.Background()); err != nil || count != 1 {
		t.Fatalf("recovery count=%d err=%v", count, err)
	}
	terminal := waitForTerminalReceipt(t, restarted, receipt)
	if terminal.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		t.Fatalf("recovered state=%s detail=%s", terminal.GetState(), terminal.GetDetail())
	}
	if len(runs.starts) != 1 || runs.starts[0].Input.Request.RunID != runID {
		t.Fatalf("recovery launched a different child: %+v", runs.starts)
	}
}

func evidenceIntentForTest(key string, identity *validationv1.SourceIdentity) *validationv1.ValidationIntent {
	return &validationv1.ValidationIntent{SchemaVersion: 1, IdempotencyKey: key, CallerScenario: "qualification", Targets: []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: "demo"}}, Purpose: validationv1.ValidationPurpose_VALIDATION_PURPOSE_EVIDENCE_PRODUCTION, RequiredStrength: validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED, ExpectedIdentity: proto.Clone(identity).(*validationv1.SourceIdentity), ContentInputs: []*validationv1.ContentInputRoot{{Name: "candidate", Root: "scenarios/demo", Selections: []*validationv1.InputSelection{{Glob: "**", Required: true}}}}, EvidencePolicy: &validationv1.EvidencePolicy{}, ReusePolicy: &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_NEVER}, ConcurrencyPolicy: &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_EXCLUSIVE}, DeadlinePolicy: &validationv1.DeadlinePolicy{QueueBudget: durationpb.New(time.Minute), ExecutionBudget: durationpb.New(time.Minute), MaximumAttempts: 1}, PinnedEvidenceProducer: &validationv1.PinnedEvidenceProducer{Provider: "owner", Producer: "refresh", Argv: []string{"/bin/true", "{run_id}", "{output_dir}"}, WorkingDirectory: "/tmp", OutputRoot: "/tmp/output", TimeoutMilliseconds: 10000, MaximumOutputBytes: 64, DescriptorDigest: "sha256:descriptor", SourceIdentity: identity.GetIdentity()}}
}
