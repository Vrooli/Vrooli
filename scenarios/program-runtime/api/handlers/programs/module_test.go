package programs

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/provenance"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
	internalprograms "program-runtime/internal/programs"
	"program-runtime/internal/tasks"
)

func TestEndpointsAreDeclared(t *testing.T) {
	if len(Endpoints) != 12 {
		t.Fatalf("endpoints=%d", len(Endpoints))
	}
}

func TestLearningFindingsProjectionBoundsAndRedactsAuthority(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(tasks.Schema())
	require.NoError(t, err)
	store := tasks.NewStore(db)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		ref := "prt_feedback_v1_" + strings.Repeat(fmt.Sprint(i), 43)
		require.NoError(t, store.PutLearningResult(ctx, tasks.LearningResult{FeedbackRef: ref, AttemptID: fmt.Sprint(i), Scope: "fixture-usage", Provenance: "operator"}))
		_, err = store.ApplyFeedback(ctx, tasks.Feedback{FeedbackRef: ref, ObservationID: fmt.Sprint(i), Disposition: "contradicted", Dimension: "usefulness", Evidence: []string{"downstream:rejected"}}, "operator")
		require.NoError(t, err)
	}
	h := &handler{learning: store}
	res, err := h.ListLearningFindings(ctx, connect.NewRequest(&programsv1.ListLearningFindingsRequest{Owner: "fixture", Limit: 2}))
	require.NoError(t, err)
	require.Len(t, res.Msg.Findings, 2)
	require.True(t, res.Msg.Truncated)
	body, err := protojson.Marshal(res.Msg)
	require.NoError(t, err)
	require.NotContains(t, string(body), "prt_feedback_v1_")
	require.NotContains(t, string(body), "claim_ref")
	require.Equal(t, "fixture", res.Msg.Findings[0].Owner)
	_, err = h.ListLearningFindings(ctx, connect.NewRequest(&programsv1.ListLearningFindingsRequest{Limit: 101}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	_, err = (&handler{}).ListLearningFindings(ctx, connect.NewRequest(&programsv1.ListLearningFindingsRequest{}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestDiscoveryEvalEndpointIsDeclared(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "programs_discovery_eval" {
			return
		}
	}
	t.Fatal("programs_discovery_eval is not declared; the CLI binding would bypass endpoint parity")
}

func TestAuthoringEvalReturnsTimingReceiptFields(t *testing.T) {
	suitePath := filepath.Join(t.TempDir(), "suite.json")
	require.NoError(t, os.WriteFile(suitePath, []byte(`{"name":"timing","floor":1,"cases":[{"id":"read","task":"read rows","oracle":{"kind":"stdout_contains","text":"rows"}}]}`), 0o600))
	h := &handler{authoring: internalprograms.AuthoringDeps{
		SuitePath: suitePath,
		AuthorWithProvenance: func(context.Context, string, string) (string, string, string, error) {
			return "print('rows')", "ollama", "test-model", nil
		},
		RunCase: func(context.Context, string, string) (string, string, int64, string, error) {
			return "rows", "", 3, "", nil
		},
	}}
	response, err := h.RunAuthoringEval(context.Background(), connect.NewRequest(&programsv1.RunAuthoringEvalRequest{}))
	require.NoError(t, err)
	require.Equal(t, internalprograms.AuthoringReceiptVersion, response.Msg.GetReceiptVersion())
	require.NotEmpty(t, response.Msg.GetCorpusSha256())
	require.NotEmpty(t, response.Msg.GetInstructionSha256())
	require.NotEmpty(t, response.Msg.GetStartedAt())
	require.NotEmpty(t, response.Msg.GetAuthorRole())
	require.NotEmpty(t, response.Msg.GetAuthorSchemaSha256())
	require.Equal(t, "unknown", response.Msg.GetRoutePolicyRevision())
	require.Len(t, response.Msg.GetResults(), 1)
	result := response.Msg.GetResults()[0]
	require.Equal(t, "measured", result.GetCaseStatus())
	require.EqualValues(t, 1, result.GetAttemptCount())
	require.Zero(t, result.GetRetryCount())
	require.Empty(t, result.GetFixtureCalls())
	require.NotEmpty(t, result.GetTaskSha256())
	require.NotEmpty(t, result.GetSourceSha256())
	require.Equal(t, "test-model", result.GetModel())
	require.Equal(t, "ollama", result.GetAuthorProvider())
	require.NotNil(t, result.ElapsedToFirstWorkingMs)
	require.GreaterOrEqual(t, result.GetGenerationElapsedMs(), int64(0))
	require.GreaterOrEqual(t, response.Msg.GetElapsedMs(), int64(0))
}

func TestSubmitProgramUsesVerifiedCallerAttribution(t *testing.T) {
	h := &handler{service: internalprograms.NewService(internalprograms.Options{AuthorizeSession: func(context.Context, string) (bool, error) { return false, nil }})}
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "verified-run", ProfileKey: "codex"})
	response, err := h.SubmitProgram(ctx, connect.NewRequest(&programsv1.SubmitProgramRequest{
		SessionId: "session", Source: "x = 1", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Async: true,
		Caller: &programsv1.Caller{RunId: "spoofed-run", AgentProfile: "spoofed-profile", SkillId: "skill", Harness: "cli"},
	}))
	require.NoError(t, err)
	require.Equal(t, "verified-run", response.Msg.Program.CallerRunId)
	require.Equal(t, "codex", response.Msg.Program.CallerAgentProfile)
	require.Equal(t, "skill", response.Msg.Program.CallerSkillId)
	require.Equal(t, "cli", response.Msg.Program.CallerHarness)
	require.True(t, response.Msg.Program.CallerIdentityVerified)
}

func TestSubmitProgramRequiresVerifiedIdentityForAgentProvenance(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"absent":             context.Background(),
		"unverified":         provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationAbsent, RunID: "claimed-run"}),
		"caller-fields-only": context.Background(),
	} {
		t.Run(name, func(t *testing.T) {
			admissionCalls := 0
			service := internalprograms.NewService(internalprograms.Options{AuthorizeSession: func(context.Context, string) (bool, error) {
				admissionCalls++
				return false, nil
			}})
			h := &handler{service: service}
			_, err := h.SubmitProgram(ctx, connect.NewRequest(&programsv1.SubmitProgramRequest{
				SessionId: "session", Source: "x = 1", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Async: true,
				Caller: &programsv1.Caller{RunId: "claimed-run", AgentProfile: "claimed-profile"},
			}))
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Zero(t, admissionCalls, "refusal must precede session admission and runner/binding effects")
			require.Empty(t, service.List(ctx, "session", false))
		})
	}
}

func TestSubmitProgramPreservesNonAgentProvenanceWithoutAgentIdentity(t *testing.T) {
	service := internalprograms.NewService(internalprograms.Options{AuthorizeSession: func(context.Context, string) (bool, error) { return false, nil }})
	h := &handler{service: service}
	response, err := h.SubmitProgram(context.Background(), connect.NewRequest(&programsv1.SubmitProgramRequest{
		SessionId: "session", Source: "x = 1", Provenance: programsv1.Provenance_PROVENANCE_OPERATOR, Async: true,
	}))
	require.NoError(t, err)
	require.Equal(t, programsv1.Provenance_PROVENANCE_OPERATOR, response.Msg.GetProgram().GetProvenance())
}

func TestSubmitProgramRefusesMissingSessionAuthorizer(t *testing.T) {
	h := &handler{service: internalprograms.NewService(internalprograms.Options{})}
	_, err := h.SubmitProgram(context.Background(), connect.NewRequest(&programsv1.SubmitProgramRequest{SessionId: "session", Source: "print(1)", Provenance: programsv1.Provenance_PROVENANCE_OPERATOR}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestProgramReadRoutesEnforceVerifiedOwner(t *testing.T) {
	service := internalprograms.NewService(internalprograms.Options{})
	h := &handler{service: service}
	owner := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "owner-run", ProfileKey: "codex"})
	other := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "other-run", ProfileKey: "codex"})
	p, _, err := service.SubmitDeclared(owner, "session", "x = 1", programsv1.Provenance_PROVENANCE_AGENT, false, false, internalprograms.Identity{}, internalprograms.Caller{RunID: "spoofed-run"}, true)
	require.NoError(t, err)
	require.True(t, p.CallerIdentityVerified)

	get := &programsv1.GetProgramRequest{Id: p.Id}
	_, err = h.GetProgram(context.Background(), connect.NewRequest(get))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = h.GetProgram(other, connect.NewRequest(get))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	got, err := h.GetProgram(owner, connect.NewRequest(get))
	require.NoError(t, err)
	require.Equal(t, p.Id, got.Msg.Program.Id)

	wait := &programsv1.WaitForProgramRequest{Id: p.Id, TimeoutMillis: 1}
	_, err = h.WaitForProgram(context.Background(), connect.NewRequest(wait))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = h.WaitForProgram(other, connect.NewRequest(wait))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	waited, err := h.WaitForProgram(owner, connect.NewRequest(wait))
	require.NoError(t, err)
	require.Equal(t, p.Id, waited.Msg.Program.Id)

	list := &programsv1.ListProgramsRequest{SessionId: "session"}
	for _, ctx := range []context.Context{context.Background(), other} {
		listed, err := h.ListPrograms(ctx, connect.NewRequest(list))
		require.NoError(t, err)
		require.Empty(t, listed.Msg.Programs)
	}
	listed, err := h.ListPrograms(owner, connect.NewRequest(list))
	require.NoError(t, err)
	require.Len(t, listed.Msg.Programs, 1)
	require.Equal(t, p.Id, listed.Msg.Programs[0].Id)
}

// TestWaitEndpointIsDeclared pins the block-once primitive to the declared
// surface. It is named separately from the count so a future change that
// removes it fails with a reason rather than an arithmetic mismatch.
func TestWaitEndpointIsDeclared(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "programs_wait" {
			return
		}
	}
	t.Fatal("programs_wait is not declared; callers would fall back to polling GetProgram")
}
