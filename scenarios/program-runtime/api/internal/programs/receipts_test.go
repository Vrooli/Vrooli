package programs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

func TestDeclaredKeySurvivesReceiptRedactionWithoutRestoringAuthority(t *testing.T) {
	db := newProgramsTestDB(t)
	runner := &admissionRunner{}
	s := NewService(Options{Store: db, Runner: runner})
	token := "prt_resume_v1_" + strings.Repeat("a", 43)
	source := `tasks.get(resume_token="` + token + `")`
	identity := Identity{ProgramName: "fixture.once", ProgramDigest: "digest", IdempotencyKey: "receipt", AdmissionDeadline: time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)}
	first, _, err := s.SubmitDeclared(t.Context(), "s", source, programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	s = NewService(Options{Store: db, Runner: runner})
	second, _, err := s.SubmitDeclared(t.Context(), "", source, programsv1.Provenance_PROVENANCE_TEST, false, false, identity, Caller{})
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id)
	require.NotContains(t, second.Source, token)
	require.Contains(t, second.Source, "unavailable after restart")
	require.EqualValues(t, 1, runner.calls.Load())
}

func TestTaskResumeReceiptIsLiveButNeverPlaintextInCorpus(t *testing.T) {
	ctx := context.Background()
	db := newProgramsTestDB(t)
	repo := NewRepository(db)
	token := "prt_resume_v1_" + strings.Repeat("a", 43)
	p := &programsv1.Program{Id: "receipt", SessionId: "session", Provenance: programsv1.Provenance_PROVENANCE_TEST, Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, CreatedAt: "2026-09-06T00:00:00Z", Source: `tasks.get(attempt_id="attempt",resume_token="` + token + `")`, Stdout: `{"learning":{"resume_token":"` + token + `"}}`, FailureDetail: "example " + token}
	require.NoError(t, repo.Save(ctx, p))
	var source, stdout, detail string
	require.NoError(t, db.QueryRow("SELECT source,stdout,failure_detail FROM programs WHERE id='receipt'").Scan(&source, &stdout, &detail))
	for _, field := range []string{source, stdout, detail} {
		require.NotContains(t, field, token)
		require.Contains(t, field, "prt_sealed_v1_")
	}
	live, err := repo.Get(ctx, p.Id)
	require.NoError(t, err)
	require.Equal(t, p.Source, live.Source)
	require.Equal(t, p.Stdout, live.Stdout)
	require.Equal(t, p.FailureDetail, live.FailureDetail)
	// A fresh API process has no decryption key. It preserves evidence but
	// cannot mint resume authority by reading a historical program row.
	recovered, err := NewRepository(db).Get(ctx, p.Id)
	require.NoError(t, err)
	require.NotContains(t, recovered.Stdout, token)
	require.Contains(t, recovered.Stdout, "unavailable after restart")
	// Stream updates must also protect incomplete tokens before the final frame.
	sealer := newReceiptSealer()
	for _, n := range []int{0, 1, 42, 43} {
		partial := token[:len("prt_resume_v1_")+n]
		sealed, err := sealer.seal(partial)
		require.NoError(t, err)
		require.NotContains(t, sealed, "prt_resume_v1_")
		require.Equal(t, partial, sealer.open(sealed))
	}
}

func TestCorpusLiteralSealedMarkersAndProtectedReceiptsRoundTrip(t *testing.T) {
	ctx := t.Context()
	db := newProgramsTestDB(t)
	repo := NewRepository(db)
	token := "prt_resume_v1_" + strings.Repeat("b", 43)
	marker := "prt_sealed_v1_literal_marker"
	p := &programsv1.Program{Id: "markers", SessionId: "s", CreatedAt: "2026-01-01T00:00:00Z", Status: programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING, Source: marker + " " + token, Stdout: marker + " " + token, FailureDetail: marker + " " + token}
	require.NoError(t, repo.Save(ctx, p))
	got, err := repo.Get(ctx, p.Id)
	require.NoError(t, err)
	require.Equal(t, p.Source, got.Source)
	require.Equal(t, p.Stdout, got.Stdout)
	require.Equal(t, p.FailureDetail, got.FailureDetail)
	var source, stdout, detail string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT source,stdout,failure_detail FROM programs WHERE id=?", p.Id).Scan(&source, &stdout, &detail))
	for _, value := range []string{source, stdout, detail} {
		require.NotContains(t, value, token)
	}
	restarted, err := NewRepository(db).Get(ctx, p.Id)
	require.NoError(t, err)
	for _, value := range []string{restarted.Source, restarted.Stdout, restarted.FailureDetail} {
		require.Contains(t, value, marker)
		require.Contains(t, value, "unavailable after restart")
		require.NotContains(t, value, token)
	}
	_, err = repo.InterruptUnfinished(ctx, "2026-01-01T01:00:00Z")
	require.NoError(t, err)
	interrupted, err := repo.Get(ctx, p.Id)
	require.NoError(t, err)
	require.Equal(t, interruptedDetail, interrupted.FailureDetail)
	require.Equal(t, p.Stdout, interrupted.Stdout)
}

func TestLegacySourceFormatSurvivesMutableUpsert(t *testing.T) {
	ctx := t.Context()
	db := newProgramsTestDB(t)
	_, err := db.ExecContext(ctx, `INSERT INTO programs (id,session_id,source,provenance,status,created_at) VALUES ('legacy','s','legacy source','1','running','2026-01-01')`)
	require.NoError(t, err)
	repo := NewRepository(db)
	p, err := repo.Get(ctx, "legacy")
	require.NoError(t, err)
	p.Source = "ignored replacement"
	p.Stdout = "literal prt_sealed_v1_abc"
	require.NoError(t, repo.Save(ctx, p))
	got, err := repo.Get(ctx, "legacy")
	require.NoError(t, err)
	require.Equal(t, "legacy source", got.Source)
	require.Equal(t, p.Stdout, got.Stdout)
}
