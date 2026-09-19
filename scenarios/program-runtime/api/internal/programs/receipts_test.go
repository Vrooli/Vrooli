package programs

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
)

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
