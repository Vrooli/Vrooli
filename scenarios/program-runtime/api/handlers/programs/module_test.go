package programs

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"fmt"
	"github.com/stretchr/testify/require"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
	"program-runtime/internal/tasks"
	"strings"
	"testing"
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
