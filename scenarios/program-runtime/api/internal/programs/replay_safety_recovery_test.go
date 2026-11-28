package programs

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"

	"program-runtime/internal/contracts"
)

type replaySafetyCountingRunner struct{ calls int }

func (r *replaySafetyCountingRunner) Execute(context.Context, string, string, bool) (Result, error) {
	r.calls++
	return Result{Stdout: `{"status":"ok"}`}, nil
}

func TestCurrentPotentiallyEffectfulContractsStayInterruptedWithoutProof(t *testing.T) {
	repoRoot, err := filepath.Abs("../../../../..")
	require.NoError(t, err)
	index := contracts.NewIndex()
	require.NoError(t, index.Load(repoRoot))

	repo := newMemoryRepository()
	var affected []*programsv1.Program
	for _, contract := range index.List() {
		if contract.ValidationError != "" || contract.SourceMissing || !contracts.ContractMayHaveEffects(contract) {
			continue
		}
		assessment := contracts.AssessReplaySafety(contract, nil, nil)
		require.Equal(t, contracts.ReplayUnsafe, assessment.Disposition, "%s lacks complete restart proof", contract.ID)
		program := &programsv1.Program{
			Id:          "pr5-restart-fixture-" + fmt.Sprintf("%03d", len(affected)),
			ProgramName: contract.Name, ProgramDigest: contract.Digest, Source: contract.Source,
			Status:     programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING,
			Provenance: programsv1.Provenance_PROVENANCE_TEST,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		}
		created, err := repo.Create(context.Background(), program)
		require.NoError(t, err)
		require.True(t, created)
		affected = append(affected, program)
	}
	require.NotEmpty(t, affected, "current contract fixture must include potential effects")
	t.Logf("fail-closed current effect/unknown contract count: %d", len(affected))

	runner := &replaySafetyCountingRunner{}
	service := NewService(Options{Runner: runner})
	service.repo = repo
	recovered, err := service.RecoverInterrupted(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(len(affected)), recovered)
	require.Zero(t, runner.calls, "recovery must never replay without a verified proof")
	for _, original := range affected {
		got, err := repo.Get(context.Background(), original.GetId())
		require.NoError(t, err)
		require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, got.GetStatus(), got.GetProgramName())
		require.Equal(t, "runtime_interrupted", got.GetFailureShape(), got.GetProgramName())
		require.Equal(t, programsv1.FailureCause_FAILURE_CAUSE_RUNTIME_INTERRUPTED, got.GetFailureCause(), got.GetProgramName())
		require.Contains(t, got.GetFailureDetail(), "downstream effects may have occurred", got.GetProgramName())
	}
}

func TestPersistentRecoveryMatrixOnlyInterruptsInFlightRegardlessOfContractEligibility(t *testing.T) {
	ctx := context.Background()
	db := newProgramsTestDB(t)
	repo := NewRepository(db)
	inputs := map[string]any{"request_key": "durable-intent-1"}

	idempotent := contracts.Contract{
		ID: "fixture-files.create", Scenario: "fixture-files", Name: "create", Source: "files.create()",
		Declaration: []byte(`{"replay_safety":{"version":1,"effects":[{"binding_id":"fixture-files/write/create","owner":"fixture-files","mode":"idempotent","key_input":"request_key","idempotency_guarantee":"owner_deduplicates_same_key"}]}}`),
		Bindings:    []contracts.BindingRef{{ID: "fixture-files/write/create", Effect: "write", Via: "binding"}},
		Inputs:      map[string]contracts.InputSpec{"request_key": {Type: "string", Required: true}},
	}
	ownerReceipt := contracts.Contract{
		ID: "fixture-files.publish", Scenario: "fixture-files", Name: "publish", Source: "files.publish()",
		Declaration: []byte(`{"replay_safety":{"version":1,"effects":[{"binding_id":"fixture-files/write/publish","owner":"fixture-files","mode":"owner_receipt","key_input":"request_key","receipt_binding":"fixture-files/receipt/get","receipt_id_field":"receipt_id","outcome_field":"effect_state"}]}}`),
		Bindings: []contracts.BindingRef{
			{ID: "fixture-files/write/publish", Effect: "write", Via: "binding"},
			{ID: "fixture-files/receipt/get", Effect: "read", Via: "binding"},
		},
		Inputs: map[string]contracts.InputSpec{"request_key": {Type: "string", Required: true}},
	}
	unsafe := contracts.Contract{
		ID: "fixture-files.delete", Scenario: "fixture-files", Name: "delete", Source: "files.delete()",
		Declaration: []byte(`{"name":"fixture-files.delete"}`),
		Bindings:    []contracts.BindingRef{{ID: "fixture-files/write/delete", Effect: "destructive", Via: "binding"}},
	}

	verifiedReceipt := contracts.EffectReceipt{
		BindingID: "fixture-files/write/publish", ReceiptBinding: "fixture-files/receipt/get", Owner: "fixture-files",
		EffectKey: "durable-intent-1", ReceiptID: "owner-receipt-1", Outcome: "applied", Verified: true,
	}
	contractCases := []struct {
		name        string
		contract    contracts.Contract
		receipts    []contracts.EffectReceipt
		disposition contracts.ReplayDisposition
	}{
		{name: "owner idempotency policy", contract: idempotent, disposition: contracts.ReplayIdempotent},
		{name: "complete owner receipt policy", contract: ownerReceipt, receipts: []contracts.EffectReceipt{verifiedReceipt}, disposition: contracts.ReplayWithReceipts},
		{name: "unsafe destructive policy", contract: unsafe, disposition: contracts.ReplayUnsafe},
	}
	for i, tc := range contractCases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := contracts.AssessReplaySafety(tc.contract, inputs, tc.receipts)
			require.Equal(t, tc.disposition, assessment.Disposition)
			status := programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED
			if i == 1 {
				status = programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING
			}
			p := &programsv1.Program{
				Id: fmt.Sprintf("persistent-inflight-%d", i), SessionId: "old-session", Source: tc.contract.Source,
				Provenance: programsv1.Provenance_PROVENANCE_TEST, Status: status,
				CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), OutputLimitBytes: 4096,
				ProgramName: tc.contract.ID, ProgramDigest: "fixture-digest-" + fmt.Sprint(i),
			}
			created, err := repo.Create(ctx, p)
			require.NoError(t, err)
			require.True(t, created)
		})
	}

	terminal := []*programsv1.Program{
		{Id: "persistent-success", Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, FailureShape: "", Stdout: "kept-success"},
		{Id: "persistent-failure", Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, FailureShape: "binding_refused", FailureDetail: "kept-failure"},
		{Id: "persistent-interrupted", Status: programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, FailureShape: "runtime_interrupted", FailureDetail: "kept-interrupted"},
	}
	for _, p := range terminal {
		p.SessionId = "old-session"
		p.Source = "retained source"
		p.Provenance = programsv1.Provenance_PROVENANCE_TEST
		p.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		require.NoError(t, repo.Save(ctx, p))
	}

	// A new service and repository wrapper model process reconstruction over the
	// same durable SQLite boundary. The proof above is classification-only:
	// program rows do not persist the validated input map or owner receipt.
	runner := &replaySafetyCountingRunner{}
	restarted := NewService(Options{Store: db, Runner: runner})
	recovered, err := restarted.RecoverInterrupted(ctx)
	require.NoError(t, err)
	require.EqualValues(t, len(contractCases), recovered)
	require.Zero(t, runner.calls, "recovery must not replay any persisted work")

	for i := range contractCases {
		got, err := restarted.Get(ctx, fmt.Sprintf("persistent-inflight-%d", i))
		require.NoError(t, err)
		require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, got.Status)
		require.Equal(t, "runtime_interrupted", got.FailureShape)
		require.Equal(t, programsv1.FailureCause_FAILURE_CAUSE_RUNTIME_INTERRUPTED, got.FailureCause)
		require.Contains(t, got.FailureDetail, "downstream effects may have occurred")
	}
	for _, want := range terminal {
		got, err := restarted.Get(ctx, want.Id)
		require.NoError(t, err)
		require.Equal(t, want.Status, got.Status, want.Id)
		require.Equal(t, want.FailureShape, got.FailureShape, want.Id)
		require.Equal(t, want.FailureDetail, got.FailureDetail, want.Id)
		require.Equal(t, want.Stdout, got.Stdout, want.Id)
	}
}
