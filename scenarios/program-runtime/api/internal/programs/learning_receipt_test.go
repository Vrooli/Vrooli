package programs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

const liveLearningReceipt = `{"task_id":"task_1","attempt_id":"attempt_1","attempt_number":1,"outcome":"unknown","delivery":"delivered","resume_token":"prt_resume_v1_abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG","advice":[],"steps":[],"attempts":[{"attempt_id":"attempt_1","recall_status":"no_match"}]}`

func TestSQLiteRepositoryPersistsLearningReceiptWithoutResumeToken(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(newProgramsTestDB(t))
	want := &programsv1.Program{Id: "prog_learning", SessionId: "sess", Source: "learn.outcome('unknown', [])", Provenance: programsv1.Provenance_PROVENANCE_OPERATOR, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: "2026-09-09T00:00:00Z", LearningJson: liveLearningReceipt}
	require.NoError(t, repo.Save(ctx, want))
	// Save must not mutate the caller's live record: the submit response keeps the token.
	require.Equal(t, liveLearningReceipt, want.GetLearningJson())

	got, err := repo.Get(ctx, "prog_learning")
	require.NoError(t, err)
	require.NotEmpty(t, got.GetLearningJson())
	require.NotContains(t, got.GetLearningJson(), "resume_token")
	require.NotContains(t, got.GetLearningJson(), "prt_resume_v1_")
	var receipt map[string]any
	require.NoError(t, json.Unmarshal([]byte(got.GetLearningJson()), &receipt))
	require.Equal(t, "task_1", receipt["task_id"])
	require.Equal(t, "attempt_1", receipt["attempt_id"])
	require.Equal(t, "unknown", receipt["outcome"])
	require.Equal(t, "delivered", receipt["delivery"])
	attempts := receipt["attempts"].([]any)
	require.Equal(t, "no_match", attempts[0].(map[string]any)["recall_status"])

	// Listing reads the same column.
	listed, err := repo.List(ctx, "sess", true)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, got.GetLearningJson(), listed[0].GetLearningJson())

	// A second Save of the already-stripped receipt is byte-stable.
	require.NoError(t, repo.Save(ctx, got))
	again, err := repo.Get(ctx, "prog_learning")
	require.NoError(t, err)
	require.Equal(t, got.GetLearningJson(), again.GetLearningJson())
}

func TestSQLiteRepositoryLeavesLearningEmptyWithoutLearnVerbs(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(newProgramsTestDB(t))
	require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "prog_plain", SessionId: "sess", Source: "print(1)", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: "2026-09-09T00:00:00Z"}))
	got, err := repo.Get(ctx, "prog_plain")
	require.NoError(t, err)
	require.Empty(t, got.GetLearningJson())
}

func TestPersistedLearningJSONRejectsNonObjectReceipts(t *testing.T) {
	require.Equal(t, "", persistedLearningJSON(""))
	require.Equal(t, "", persistedLearningJSON("   "))
	require.Equal(t, "", persistedLearningJSON(`"prt_resume_v1_x"`))
	require.Equal(t, "", persistedLearningJSON(`{not json`))
	require.Equal(t, `{"outcome":"ok"}`, persistedLearningJSON(`{ "resume_token": "prt_resume_v1_x", "outcome": "ok" }`))
}

func TestCompactLearningJSONNormalisesKernelReceipts(t *testing.T) {
	require.Equal(t, "", compactLearningJSON(nil))
	require.Equal(t, "", compactLearningJSON(json.RawMessage("null")))
	require.Equal(t, "", compactLearningJSON(json.RawMessage("{broken")))
	require.Equal(t, `{"a":1,"b":[1,2]}`, compactLearningJSON(json.RawMessage(" {\"a\": 1,\n \"b\": [1, 2]} ")))
}

func TestEnsureCompatibilityAddsLearningJSONToExistingDatabase(t *testing.T) {
	ctx := context.Background()
	d := newProgramsTestDB(t)
	_, err := d.ExecContext(ctx, `ALTER TABLE programs DROP COLUMN learning_json`)
	require.NoError(t, err)
	require.NoError(t, EnsureCompatibility(ctx, d))
	repo := NewRepository(d)
	require.NoError(t, repo.Save(ctx, &programsv1.Program{Id: "prog_upgraded", SessionId: "sess", Source: "x", Provenance: programsv1.Provenance_PROVENANCE_AGENT, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: "2026-09-09T00:00:00Z", LearningJson: `{"outcome":"ok"}`}))
	got, err := repo.Get(ctx, "prog_upgraded")
	require.NoError(t, err)
	require.Equal(t, `{"outcome":"ok"}`, got.GetLearningJson())
}

func TestServiceCarriesLearningReceiptOntoProgramRecord(t *testing.T) {
	s := NewService(Options{Runner: fakeRunner{result: Result{Stdout: "ok", LearningJSON: liveLearningReceipt}}})
	submitted, err := s.Submit(context.Background(), "s1", "learn.outcome('unknown', [])", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, submitted.GetStatus())
	// The live submit response keeps the bearer token for the caller.
	require.Contains(t, submitted.GetLearningJson(), "resume_token")

	got, err := s.Get(context.Background(), submitted.GetId())
	require.NoError(t, err)
	require.NotEmpty(t, got.GetLearningJson())
	require.NotContains(t, got.GetLearningJson(), "resume_token")
	require.Contains(t, got.GetLearningJson(), `"delivery":"delivered"`)
}

func TestServiceCarriesLearningReceiptOnFailedProgram(t *testing.T) {
	s := NewService(Options{Runner: fakeRunner{result: Result{LearningJSON: `{"outcome":"failed","delivery":"delivered"}`}, err: errors.New("ValueError: boom")}})
	submitted, err := s.Submit(context.Background(), "s1", "learn.outcome('failed', [])", programsv1.Provenance_PROVENANCE_OPERATOR, false)
	require.NoError(t, err)
	require.Equal(t, programsv1.ProgramStatus_PROGRAM_STATUS_FAILED, submitted.GetStatus())
	got, err := s.Get(context.Background(), submitted.GetId())
	require.NoError(t, err)
	require.Contains(t, got.GetLearningJson(), `"outcome":"failed"`)
}
