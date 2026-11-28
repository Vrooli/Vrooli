package optimization

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	programv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization"
	programconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization/optimization_v1connect"
	_ "modernc.org/sqlite"
	internaloptimization "program-runtime/internal/optimization"
)

type connectFixtureAdapter struct{}

func (connectFixtureAdapter) Target() string { return "integration-fixture" }
func (connectFixtureAdapter) Validate(request internaloptimization.Request) error {
	if request.Target != "integration-fixture" || len(request.Candidates) != 2 || len(request.CaseIDs) != 1 {
		return internaloptimization.ErrInvalidRequest
	}
	return nil
}

func (connectFixtureAdapter) Acquire(_ context.Context, runID string, request internaloptimization.Request) (internaloptimization.Evidence, error) {
	digests := make(map[string]string, len(request.Candidates))
	for _, candidate := range request.Candidates {
		digests[candidate.ID] = candidate.ArtifactDigest
	}
	return internaloptimization.Evidence{Stage: internaloptimization.StageAcquire, ReceiptID: runID + "/acquire", Details: map[string]any{"candidate_artifact_digests": digests}}, nil
}

func (connectFixtureAdapter) Evaluate(_ context.Context, runID string, request internaloptimization.Request, _ []internaloptimization.Evidence) (internaloptimization.Evidence, error) {
	results := make(map[string]any, len(request.Candidates))
	for _, candidate := range request.Candidates {
		results[candidate.ID] = map[string]any{"case_count": 1, "case_digest": fixtureCaseDigest(request.CaseIDs), "score": 1.0}
	}
	return internaloptimization.Evidence{Stage: internaloptimization.StageEvaluate, ReceiptID: runID + "/evaluate", Details: map[string]any{
		"suite_id": request.SuiteID, "case_count": 1, "case_digest": fixtureCaseDigest(request.CaseIDs), "candidate_results": results,
	}}, nil
}

func (connectFixtureAdapter) CheckRights(_ context.Context, runID string, stage internaloptimization.Stage, request internaloptimization.Request, evidence []internaloptimization.Evidence) (internaloptimization.Evidence, error) {
	digest := internaloptimization.SubjectDigest(request)
	if stage == internaloptimization.StageRights {
		digest = internaloptimization.AcquiredSubjectDigest(request, evidence)
	}
	return internaloptimization.Evidence{Stage: stage, ReceiptID: runID + "/" + string(stage), Proof: &internaloptimization.Proof{
		Decision: "allow", ReceiptID: runID + "/" + string(stage) + "-proof", SubjectDigest: digest, ExpiresAt: time.Now().Add(time.Hour),
	}}, nil
}

func (connectFixtureAdapter) CheckDisk(_ context.Context, runID string, stage internaloptimization.Stage, request internaloptimization.Request, evidence []internaloptimization.Evidence) (internaloptimization.Evidence, error) {
	digest := internaloptimization.SubjectDigest(request)
	if stage == internaloptimization.StageDisk {
		digest = internaloptimization.AcquiredSubjectDigest(request, evidence)
	}
	return internaloptimization.Evidence{Stage: stage, ReceiptID: runID + "/" + string(stage), Proof: &internaloptimization.Proof{
		Decision: "allow", ReceiptID: runID + "/" + string(stage) + "-proof", SubjectDigest: digest,
		RequiredBytes: 4, AvailableBytes: 128, ExpiresAt: time.Now().Add(time.Hour),
	}}, nil
}

func (connectFixtureAdapter) Promote(_ context.Context, runID string, request internaloptimization.Request, _ []internaloptimization.Evidence) (internaloptimization.Evidence, error) {
	winner := "challenger"
	return internaloptimization.Evidence{Stage: internaloptimization.StagePromote, ReceiptID: runID + "/promote", Disposition: "promoted", Details: map[string]any{
		"winner_id": winner, "artifact_digest": "sha256:challenger",
	}}, nil
}

func (connectFixtureAdapter) Cleanup(_ context.Context, runID string, _ internaloptimization.Request, _ []internaloptimization.Evidence) (internaloptimization.Evidence, error) {
	return internaloptimization.Evidence{Stage: internaloptimization.StageCleanup, ReceiptID: runID + "/cleanup", Disposition: "cleanup_complete", Details: map[string]any{
		"removal_receipts": map[string]string{"incumbent": runID + "/remove/incumbent", "previous-role": runID + "/remove/previous-role"},
	}}, nil
}

func fixtureCaseDigest(caseIDs []string) string {
	encoded, _ := json.Marshal(caseIDs)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func TestConnectSubmitWaitCompletesDurableOptimizationRun(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "optimization.db"))
	require.NoError(t, err)
	// The fixture store and manager share one SQLite file while the worker is
	// committing stage receipts; serialize connections to avoid SQLITE_BUSY.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	store, err := internaloptimization.NewSQLStore(context.Background(), db)
	require.NoError(t, err)
	engine := &internaloptimization.Engine{Store: store, Adapters: map[string]internaloptimization.Adapter{"integration-fixture": connectFixtureAdapter{}}}
	manager, err := internaloptimization.NewManager(engine)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NoError(t, manager.Start(ctx))
	t.Cleanup(manager.Stop)
	service, err := internaloptimization.NewService(manager)
	require.NoError(t, err)

	path, handler := programconnect.NewOptimizationServiceHandler(&handler{service: service})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := programconnect.NewOptimizationServiceClient(server.Client(), server.URL)

	_, err = client.Submit(context.Background(), connect.NewRequest(&programv1.SubmitOptimizationRequest{
		RequestJson: `{"run_id":"first","run_id":"second"}`,
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "malformed request JSON is caller input, not an unavailable owner")

	invalidShape := internaloptimization.Request{
		RunID: "connect-invalid-target-shape", Target: "integration-fixture", Subject: map[string]any{"role": "fixture"},
		Candidates: []internaloptimization.Candidate{{ID: "incumbent", ArtifactDigest: "sha256:incumbent"}, {ID: "challenger", ArtifactDigest: "sha256:challenger"}},
		SuiteID:    "fixed-fixture-suite", CaseIDs: []string{"case-1", "case-2"}, MinimumCases: 1,
	}
	invalidJSON, err := json.Marshal(invalidShape)
	require.NoError(t, err)
	_, err = client.Submit(context.Background(), connect.NewRequest(&programv1.SubmitOptimizationRequest{RequestJson: string(invalidJSON)}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "target-owned validation failures are caller input")

	request := internaloptimization.Request{
		RunID: "connect-integration-run", Target: "integration-fixture", Subject: map[string]any{"role": "fixture"},
		Candidates: []internaloptimization.Candidate{
			{ID: "incumbent", ArtifactDigest: "sha256:incumbent"},
			{ID: "challenger", ArtifactDigest: "sha256:challenger"},
		},
		PredecessorID: "previous-role", SuiteID: "fixed-fixture-suite", CaseIDs: []string{"case-1"}, MinimumCases: 1,
	}
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	submitted, err := client.Submit(context.Background(), connect.NewRequest(&programv1.SubmitOptimizationRequest{RequestJson: string(encoded)}))
	require.NoError(t, err)
	require.Equal(t, request.RunID, submitted.Msg.GetRunId())
	resubmitted, err := client.Submit(context.Background(), connect.NewRequest(&programv1.SubmitOptimizationRequest{RequestJson: string(encoded)}))
	require.NoError(t, err, "same immutable request reuses its durable run")
	require.Equal(t, request.RunID, resubmitted.Msg.GetRunId())
	changed := request
	changed.Subject = map[string]any{"role": "different"}
	changedJSON, err := json.Marshal(changed)
	require.NoError(t, err)
	_, err = client.Submit(context.Background(), connect.NewRequest(&programv1.SubmitOptimizationRequest{RequestJson: string(changedJSON)}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a run ID cannot be reused for different immutable inputs")

	waited, err := client.Wait(context.Background(), connect.NewRequest(&programv1.WaitOptimizationRequest{RunId: request.RunID, WaitMs: 5000}))
	require.NoError(t, err)
	require.True(t, waited.Msg.GetTerminal())
	var record internaloptimization.Record
	require.NoError(t, json.Unmarshal([]byte(waited.Msg.GetRecordJson()), &record))
	require.Equal(t, internaloptimization.StageComplete, record.Stage)
	require.Equal(t, "challenger", record.WinnerID)
	require.Len(t, record.Evidence, 8)
	last, err := store.Load(context.Background(), request.RunID)
	require.NoError(t, err)
	require.Equal(t, record.Revision, last.Revision, "terminal Connect receipt must match the durable SQLite journal")
}
