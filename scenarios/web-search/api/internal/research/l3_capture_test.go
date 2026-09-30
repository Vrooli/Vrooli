package research_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"web-search/internal/capture"
	"web-search/internal/research"
	"web-search/internal/research/agentmanager"
)

func TestL3AdmissionLeavesDurableCapture(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(capture.Schema)))
	outbox := capture.NewRepository(db, time.Now)
	am := &fakeAgentManager{spawnResult: agentmanager.RunResult{RunID: "run-1", TaskID: "task-1", Status: "pending"}}
	svc := research.NewService(research.Deps{AgentManager: am, AttemptOutbox: outbox})

	_, err := svc.RunL3(context.Background(), "what changed")
	require.NoError(t, err)
	claimed, err := outbox.Claim(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(claimed[0].Payload, &payload))
	require.Equal(t, "research.l3", payload["operation"])
	require.Equal(t, "unknown", payload["disposition"])
	require.Equal(t, "run-1", payload["run_id"])
}
