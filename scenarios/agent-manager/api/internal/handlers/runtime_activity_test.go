package handlers

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/maintenance"
	"agent-manager/internal/runtimeactivity"
	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func TestRuntimeActivityPublicTransportsPreservePhysicalProofAndUnknown(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "activity.db"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE runs(id TEXT,status TEXT,runner_pid INTEGER,runner_pgid INTEGER,finalization_status TEXT,execution_mode TEXT);CREATE TABLE workflow_executions(id TEXT,status TEXT);
 INSERT INTO runs VALUES('abandoned','running',0,0,'none','codec_pipe'),('physical','failed',33,33,'none','codec_pipe'),('queued','pending',0,0,'none','codec_pipe'),('imported','running',44,44,'none','imported');`)
	require.NoError(t, err)
	physical := true
	fail := false
	inventory := maintenance.NewBatchedInventory(db, func(_ context.Context, refs []maintenance.ExecutorRef) ([]maintenance.ExecutorEvidence, error) {
		require.Len(t, refs, 1)
		require.Equal(t, "physical", refs[0].ID)
		evidence := []maintenance.ExecutorEvidence{{ExecutorRef: refs[0], Alive: physical}}
		if fail {
			return evidence, errors.New("scope coverage unavailable")
		}
		return evidence, nil
	})
	h := &Handler{runtimeActivity: runtimeactivity.New(inventory)}
	mux := http.NewServeMux()
	path, handler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(h))
	mux.Handle(path, AgentOperationMiddleware(handler))
	mux.HandleFunc("/api/v1/runtime/activity", h.GetRuntimeActivity)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)
	for _, positive := range []bool{true, false} {
		physical = positive
		fail = !positive
		response, err := client.GetRuntimeActivity(t.Context(), connect.NewRequest(&apipb.GetRuntimeActivityRequest{}))
		require.NoError(t, err)
		got := response.Msg
		require.False(t, got.Complete)
		require.Nil(t, got.ExecutingRuns)
		require.Equal(t, uint32(1), got.LogicalQueuedRuns)
		require.NotEmpty(t, got.UnknownReasons)
		require.WithinDuration(t, time.Now(), got.ObservedAt.AsTime(), 5*time.Second)
		if positive {
			require.Equal(t, uint32(1), got.ObservedExecutingRuns)
		} else {
			require.Zero(t, got.ObservedExecutingRuns)
		}
		rest, err := server.Client().Get(server.URL + "/api/v1/runtime/activity")
		require.NoError(t, err)
		require.Equal(t, 200, rest.StatusCode)
		var decoded apipb.GetRuntimeActivityResponse
		err = func() error {
			defer rest.Body.Close()
			return protojson.UnmarshalOptions{}.Unmarshal(readActivityBody(t, rest), &decoded)
		}()
		require.NoError(t, err)
		decoded.ObservedAt = got.ObservedAt
		require.True(t, proto.Equal(got, &decoded))
	}
	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM runs WHERE id='abandoned'`).Scan(&status))
	require.Equal(t, "running", status)
}

func readActivityBody(t *testing.T, r *http.Response) []byte {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return data
}
