package validationrun

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	core "github.com/vrooli/api-core/validationrun"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"workflow-health/internal/testutil/db"
)

func TestRepositoryPersistsReplaySafeRunAndTerminalEvidence(t *testing.T) {
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(Schema)))
	repo := Repository{DB: database}
	created := Record{Run: core.Run{ID: "run-1", Target: core.Target{Scenario: "demo", Path: "/tmp/demo"}, IdempotencyKey: "request-1", ParentRunID: "parent-1", State: core.StateQueued, CreatedAt: time.Now().UTC(), Version: 1}, ETA: time.Minute, ExecutionBinding: &scenariovalidationv1.DesktopValidationBinding{TargetId: "target-1", CdpEndpoint: "http://127.0.0.1:9222", RendererId: "renderer-1", ContextId: "ctx-1", ProfileId: "normal"}}
	require.NoError(t, repo.Create(context.Background(), created))
	found, err := repo.FindByIdempotency(context.Background(), "request-1")
	require.NoError(t, err)
	require.Equal(t, created.Run.ID, found.Run.ID)
	require.Equal(t, created.Run.ParentRunID, found.Run.ParentRunID)
	require.Equal(t, "target-1", found.ExecutionBinding.GetTargetId())
	next, err := core.Transition(found.Run, core.EventClaim, time.Now().UTC())
	require.NoError(t, err)
	next.Version = found.Run.Version + 1
	found.Run = next
	require.NoError(t, repo.Update(context.Background(), found, 1))
	require.Error(t, repo.Update(context.Background(), found, 1), "stale commits must not overwrite recovery state")
}
