package bindings

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	"program-runtime/internal/testutil/db"
)

func newInvocationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	return d
}

func TestInvocationRepositoryPersistsRefusal(t *testing.T) { // [REQ:PRT-P1-008]
	d := newInvocationTestDB(t)
	repo := NewInvocationRepository(d)
	now := time.Date(2026, 8, 11, 18, 0, 0, 0, time.UTC)
	require.NoError(t, repo.RecordInvocation(context.Background(), Invocation{InvocationID: "inv_refused", BindingID: "demo/ops/delete", TargetScenario: "demo", SessionID: "sess_1", ProgramID: "prog_1", Provenance: "PROVENANCE_AGENT", Outcome: "refused", Reason: "missing explicit grant", LatencyMS: 12, OccurredAt: now}))
	rows, err := repo.ListInvocations(context.Background(), now.Add(-time.Minute), "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "refused", rows[0].Outcome)
	require.Equal(t, "prog_1", rows[0].ProgramID)
}

type fakeInvocationRecorder struct{ rows []Invocation }

func (f fakeInvocationRecorder) RecordInvocation(context.Context, Invocation) error { return nil }
func (f fakeInvocationRecorder) ListInvocations(context.Context, time.Time, string, string) ([]Invocation, error) {
	return f.rows, nil
}

func TestBindingConditionUsesNearestRankAndReportsUninstrumentedDrift(t *testing.T) { // [REQ:PRT-P1-008]
	now := time.Now().UTC()
	r := &Registry{
		bindings: []*bindingsv1.Binding{{Id: "demo/read/list", Scenario: "demo"}},
		recorder: fakeInvocationRecorder{rows: []Invocation{
			{BindingID: "demo/read/list", SessionID: "s1", Outcome: "success", LatencyMS: 10, OccurredAt: now},
			{BindingID: "demo/read/list", SessionID: "s1", Outcome: "success", LatencyMS: 20, OccurredAt: now},
			{BindingID: "demo/read/list", SessionID: "s2", Outcome: "success", LatencyMS: 30, OccurredAt: now},
			{BindingID: "demo/read/list", SessionID: "s3", Outcome: "success", LatencyMS: 40, OccurredAt: now},
			{BindingID: "demo/read/list", SessionID: "s4", Outcome: "success", LatencyMS: 50, OccurredAt: now},
		}},
		artifactMtime: now.Add(-time.Hour),
	}
	response, err := r.Conditions(context.Background(), "", "", time.Hour)
	require.NoError(t, err)
	require.Len(t, response.Conditions, 1)
	condition := response.Conditions[0]
	require.Equal(t, int64(50), condition.Serving.LatencyP95Ms)
	require.Equal(t, int64(30), condition.Serving.LatencyP50Ms)
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_HEALTHY, condition.Status)
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, condition.Freshness.DriftStatus)
}

func TestBindingConditionReportsDormantAndNotHealthy(t *testing.T) { // [REQ:PRT-P1-008]
	r := &Registry{bindings: []*bindingsv1.Binding{{Id: "demo/read/never", Scenario: "demo"}}, recorder: fakeInvocationRecorder{}}
	response, err := r.Conditions(context.Background(), "", "", time.Hour)
	require.NoError(t, err)
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_DORMANT, response.Conditions[0].Status)
	require.Contains(t, response.Conditions[0].Verdict, "exercise.invocations=0")
	require.Equal(t, bindingsv1.ConditionStatus_CONDITION_STATUS_UNINSTRUMENTED, response.Conditions[0].Freshness.Family.Status)
}
