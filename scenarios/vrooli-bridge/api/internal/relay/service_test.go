package relay_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/relay"
)

type fakeNodes struct{ node dispatch.TargetNode }

func (f fakeNodes) GetTarget(context.Context, string) (dispatch.TargetNode, error) {
	return f.node, nil
}

type fakePresence struct{ online, dispatchable bool }

func (f fakePresence) IsOnline(string) bool     { return f.online }
func (f fakePresence) Dispatchable(string) bool { return f.dispatchable }

type fakeAudit struct {
	mu      sync.Mutex
	records []audit.Record
}

func (f *fakeAudit) Append(_ context.Context, record audit.Record) (audit.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = append(f.records, record)
	return record, nil
}

type fakePusher struct {
	push   func(relay.Request)
	cancel func(string)
}

func (f fakePusher) Push(_ context.Context, _ string, request relay.Request) (int, error) {
	if f.push != nil {
		f.push(request)
	}
	return 1, nil
}

func (f fakePusher) Cancel(_ context.Context, _ string, correlationID, _ string) (int, error) {
	if f.cancel != nil {
		f.cancel(correlationID)
	}
	return 1, nil
}

func TestAdmissionParityWithTypedDispatch(t *testing.T) {
	manifest, _, err := dispatch.BuildManifest()
	require.NoError(t, err)
	node := dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}
	cases := []struct {
		name    string
		request relay.Request
		node    dispatch.TargetNode
	}{
		{name: "accepted", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: node},
		{name: "unsafe token", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test", Args: []string{"x;bad"}}, node: node},
		{name: "out of scope", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: dispatch.TargetNode{ID: "n1", Kind: "agent"}},
		{name: "unknown command", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "secrets list"}, node: node},
		{name: "revoked", request: relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test"}, node: dispatch.TargetNode{ID: "n1", Kind: "agent", Revoked: true, Scopes: node.Scopes}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			typedErr := dispatch.Admit(dispatch.Job{NodeID: tc.request.NodeID, Scenario: tc.request.Scenario, Verb: tc.request.Command, Args: tc.request.Args}, tc.node, manifest)
			relayErr := relay.Admit(tc.request, tc.node, manifest)
			if typedErr == nil || relayErr == nil {
				require.Equal(t, typedErr == nil, relayErr == nil)
				return
			}
			require.Equal(t, reflect.TypeOf(typedErr), reflect.TypeOf(relayErr))
			require.Equal(t, typedErr.Error(), relayErr.Error())
		})
	}
}

func TestCall_CatalogUnavailableFailsClosedBeforeNodeSideEffects(t *testing.T) {
	auditSink := &fakeAudit{}
	svc := relay.NewService(
		fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}},
		fakePresence{online: true, dispatchable: true},
		auditSink,
		fakePusher{},
		relay.NewBroker(),
		relay.WithCatalogError(errors.New("validate scenarios/demo/cli/manifest.json: malformed groups")),
	)

	_, err := svc.Call(context.Background(), relay.Request{Actor: "owner", NodeID: "n1", Scenario: "demo", Command: "scenario test"})
	var unavailable dispatch.ErrCatalogUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.Contains(t, err.Error(), "scenarios/demo/cli/manifest.json")
	require.Len(t, auditSink.records, 1)
	require.Equal(t, audit.OutcomeRejected, auditSink.records[0].Outcome)
}

func TestCallStreamsAndAuditsNodeCommandOutcome(t *testing.T) {
	broker := relay.NewBroker()
	auditSink := &fakeAudit{}
	pusher := fakePusher{push: func(request relay.Request) {
		go func() {
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindAccepted})
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindData, Data: []byte("ok\n")})
			_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindCompleted, ExitCode: 0})
		}()
	}}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}, fakePresence{online: true, dispatchable: true}, auditSink, pusher, broker)

	response, err := svc.Call(context.Background(), relay.Request{Actor: "owner", NodeID: "n1", Scenario: "demo", Command: "scenario test"})
	require.NoError(t, err)
	require.Equal(t, relay.KindCompleted, response.Kind)
	require.Equal(t, []byte("ok\n"), response.Data)
	require.Len(t, auditSink.records, 2)
	require.Equal(t, "n1", auditSink.records[0].NodeID)
	require.Equal(t, "scenario test", auditSink.records[0].Verb)
	require.Equal(t, audit.OutcomeCompleted, auditSink.records[len(auditSink.records)-1].Outcome)
}

func TestCallNamesAndCancelsResponseLimit(t *testing.T) {
	broker := relay.NewBroker()
	var cancelled string
	pusher := fakePusher{
		push: func(request relay.Request) {
			go func() {
				_ = broker.Deliver(context.Background(), "n1", relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindData, Data: []byte("too-large")})
			}()
		},
		cancel: func(correlationID string) { cancelled = correlationID },
	}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}, fakePresence{online: true, dispatchable: true}, nil, pusher, broker)

	response, err := svc.Call(context.Background(), relay.Request{NodeID: "n1", Scenario: "demo", Command: "scenario test", MaxResponseBytes: 3})
	var limit relay.ErrResponseLimit
	require.ErrorAs(t, err, &limit)
	require.Equal(t, relay.KindFailed, response.Kind)
	require.Equal(t, relay.ResponseLimitReason, response.Reason[:len(relay.ResponseLimitReason)])
	require.NotEmpty(t, cancelled)
}

type stagedPusher struct {
	mu       sync.Mutex
	broker   *relay.Broker
	calls    int
	first    int
	complete bool
}

type measuredRelayPusher struct{ broker *relay.Broker }

func (p measuredRelayPusher) Route(context.Context, string, relay.Request) (string, uint64) {
	return "sse-relay", 3
}

func (p measuredRelayPusher) Push(_ context.Context, nodeID string, request relay.Request) (int, error) {
	go func() {
		time.Sleep(2 * time.Millisecond)
		_ = p.broker.Deliver(context.Background(), nodeID, relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindCompleted, ExitCode: 0})
	}()
	return 1, nil
}

func (measuredRelayPusher) Cancel(context.Context, string, string, string) (int, error) {
	return 1, nil
}

// REM-07: when the direct peer route is unavailable, the supported signed
// channel relay remains usable and reports provider cost plus measured
// end-to-end latency on the receipt.
func TestRelayFallbackRecordsRouteCostAndLatency(t *testing.T) {
	b := relay.NewBroker()
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}, fakePresence{online: true, dispatchable: true}, nil, measuredRelayPusher{broker: b}, b)
	response, err := svc.Call(context.Background(), relay.Request{CommandID: "relay-fallback", NodeID: "n1", Scenario: "demo", Command: "scenario test"})
	require.NoError(t, err)
	require.Equal(t, relay.KindCompleted, response.Kind)
	require.Equal(t, "sse-relay", response.Route)
	require.Equal(t, uint64(3), response.RouteCostUnits)
	require.Greater(t, response.RouteLatencyMS, uint64(0))
	receipt, err := svc.Reconcile(context.Background(), relay.ReconcileRequest{NodeID: "n1", CommandID: "relay-fallback"})
	require.NoError(t, err)
	require.Equal(t, response.Route, receipt.Response.Route)
	require.Equal(t, response.RouteCostUnits, receipt.Response.RouteCostUnits)
}

func (p *stagedPusher) Push(_ context.Context, _ string, request relay.Request) (int, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.mu.Unlock()
	if call == 1 && p.first == 0 {
		return 0, nil
	}
	if p.complete {
		go func() {
			_ = p.broker.Deliver(context.Background(), request.NodeID, relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindCompleted, ExitCode: 0})
		}()
	}
	return 1, nil
}

func (p *stagedPusher) Cancel(context.Context, string, string, string) (int, error) { return 1, nil }

// REM-04: a route that never admits the command is durably marked
// not_admitted, so the same command identity can be resumed without changing
// its payload or accidentally creating a second side effect.
func TestCallLossBeforeEffectCanResumeSameCommand(t *testing.T) {
	broker := relay.NewBroker()
	pusher := &stagedPusher{broker: broker, complete: true}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}, fakePresence{online: true, dispatchable: true}, nil, pusher, broker)
	request := relay.Request{CommandID: "command-before-effect", NodeID: "n1", Scenario: "demo", Command: "scenario test"}
	_, err := svc.Call(context.Background(), request)
	require.Error(t, err, "a node that reports no delivery must not be treated as an admitted effect")
	record, err := svc.Reconcile(context.Background(), relay.ReconcileRequest{NodeID: "n1", CommandID: request.CommandID})
	require.NoError(t, err)
	require.Equal(t, relay.StateNotAdmitted, record.State)

	response, err := svc.Call(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, relay.KindCompleted, response.Kind)
	pusher.mu.Lock()
	require.Equal(t, 2, pusher.calls)
	pusher.mu.Unlock()
}

// REM-05: once delivery is acknowledged but the reply route is lost, the
// durable unknown receipt is returned on retry and no alternate Call is sent.
func TestCallLossAfterEffectRequiresReconciliationBeforeFallback(t *testing.T) {
	broker := relay.NewBroker()
	pusher := &stagedPusher{broker: broker, first: 1, complete: false}
	svc := relay.NewService(fakeNodes{node: dispatch.TargetNode{ID: "n1", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}, fakePresence{online: true, dispatchable: true}, nil, pusher, broker)
	request := relay.Request{CommandID: "command-after-effect", NodeID: "n1", Scenario: "demo", Command: "scenario test", TimeoutSeconds: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	response, err := svc.Call(ctx, request)
	require.Error(t, err)
	require.Equal(t, relay.KindOutcomeUnknown, response.Kind)

	reconciled, err := svc.Reconcile(context.Background(), relay.ReconcileRequest{NodeID: "n1", CommandID: request.CommandID})
	require.NoError(t, err)
	require.Equal(t, relay.StateOutcomeUnknown, reconciled.State)
	require.Equal(t, relay.KindOutcomeUnknown, reconciled.Response.Kind)

	retry, err := svc.Call(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, relay.KindOutcomeUnknown, retry.Kind)
	pusher.mu.Lock()
	require.Equal(t, 1, pusher.calls, "retry returns the original receipt instead of invoking a fallback effect")
	pusher.mu.Unlock()
}

// REM-05: command state and the original response remain available after a
// store is reopened, which is the Bridge restart boundary.
func TestSQLiteCommandStorePersistsReceiptAcrossReopen(t *testing.T) {
	d := dbtest.NewSQLite(t)
	_, err := d.ExecContext(context.Background(), relay.Schema())
	require.NoError(t, err)
	request := relay.Request{CommandID: "persistent-command", CorrelationID: "corr-1", Actor: "owner", NodeID: "n1", Scenario: "demo", Command: "scenario test", Args: []string{"--json"}}
	storeA, err := relay.NewSQLiteCommandStore(d)
	require.NoError(t, err)
	_, err = storeA.Reserve(context.Background(), request)
	require.NoError(t, err)
	response := relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindCompleted, Data: []byte("ok"), ExitCode: 0}
	require.NoError(t, storeA.Update(context.Background(), request.CommandID, request.NodeID, relay.StateSubmitted, relay.Response{CorrelationID: request.CorrelationID, Kind: relay.KindAccepted}))
	require.NoError(t, storeA.Update(context.Background(), request.CommandID, request.NodeID, relay.StateCompleted, response))

	storeB, err := relay.NewSQLiteCommandStore(d)
	require.NoError(t, err)
	record, err := storeB.Get(context.Background(), request.CommandID, request.NodeID)
	require.NoError(t, err)
	require.Equal(t, relay.StateCompleted, record.State)
	require.Equal(t, []byte("ok"), record.Response.Data)
	require.Equal(t, []string{"--json"}, record.Request.Args)
}

// AC6: a Web Console retry after a lost response or process restart must use
// the same command ID. The production Relay service must return its SQLite
// receipt without dispatching the harness lifecycle action a second time.
func TestCallReplaysHarnessLifecycleReceiptAfterServiceReconstruction(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, relay.Schema())
	require.NoError(t, err)
	storeA, err := relay.NewSQLiteCommandStore(db)
	require.NoError(t, err)
	brokerA := relay.NewBroker()
	pushes := 0
	pusher := fakePusher{push: func(request relay.Request) {
		pushes++
		require.NoError(t, brokerA.Deliver(ctx, request.NodeID, relay.Response{
			CorrelationID: request.CorrelationID,
			Kind:          relay.KindCompleted,
			Data:          []byte(`{"success":true,"harness_update":{"action":"update","success":true}}`),
		}))
	}}
	nodes := fakeNodes{node: dispatch.TargetNode{ID: "ac6-node", Kind: "agent", Scopes: []string{"vrooli-bridge:write", "vrooli:write"}}}
	presence := fakePresence{online: true, dispatchable: true}
	request := relay.Request{
		CommandID: "0c7e6530-45ab-45e5-a63f-ef4d8ac6ac06", CorrelationID: "ac6-correlation-1",
		Actor: "owner", NodeID: "ac6-node", Scenario: "vrooli", Command: "resource update",
		Args: []string{"codex", "--action", "update", "--json"}, TimeoutSeconds: 1,
	}
	firstService := relay.NewService(nodes, presence, nil, pusher, brokerA, relay.WithCommandStore(storeA))
	first, err := firstService.Call(ctx, request)
	require.NoError(t, err)
	require.Equal(t, relay.KindCompleted, first.Kind)
	require.Equal(t, "ac6-correlation-1", first.CorrelationID)

	// Reopen the file-backed journal and create a new service and command-store
	// instance to model the Bridge API restart boundary.
	var seq int
	var dbName, dbPath string
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA database_list`).Scan(&seq, &dbName, &dbPath))
	require.Equal(t, "main", dbName)
	require.NotEmpty(t, dbPath)
	require.NoError(t, db.Close())
	dsn, err := storage.SQLiteDSNAt(filepath.Clean(dbPath), storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	db, err = sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.PingContext(ctx))
	storeB, err := relay.NewSQLiteCommandStore(db)
	require.NoError(t, err)
	secondService := relay.NewService(nodes, presence, nil, pusher, relay.NewBroker(), relay.WithCommandStore(storeB))
	second, err := secondService.Call(ctx, request)
	require.NoError(t, err)
	require.Equal(t, first.Kind, second.Kind)
	require.Equal(t, first.CorrelationID, second.CorrelationID)
	require.Equal(t, first.Data, second.Data)
	require.Equal(t, 1, pushes, "replaying a completed harness action must not push another effect")

	receipt, err := secondService.Reconcile(ctx, relay.ReconcileRequest{NodeID: request.NodeID, CommandID: request.CommandID})
	require.NoError(t, err)
	require.Equal(t, relay.StateCompleted, receipt.State)
	require.Equal(t, request.CommandID, receipt.CommandID)
}

func TestCommandStateTransitionMatrixRejectsInvalidEdges(t *testing.T) {
	states := []relay.CommandState{
		relay.StateReserved, relay.StateSubmitted, relay.StateNotAdmitted,
		relay.StateCompleted, relay.StateFailed, relay.StateOutcomeUnknown,
	}
	want := map[relay.CommandState]map[relay.CommandState]bool{
		relay.StateReserved:    {relay.StateSubmitted: true},
		relay.StateSubmitted:   {relay.StateNotAdmitted: true, relay.StateCompleted: true, relay.StateFailed: true, relay.StateOutcomeUnknown: true},
		relay.StateNotAdmitted: {relay.StateReserved: true},
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"->"+string(to), func(t *testing.T) {
				record := relay.CommandRecord{State: from}
				err := record.Transition(to)
				if want[from][to] {
					require.NoError(t, err)
					require.Equal(t, to, record.State)
					return
				}
				require.ErrorIs(t, err, relay.ErrInvalidCommandTransition)
				require.Equal(t, from, record.State, "rejected transition must preserve the old state")
			})
		}
	}

	for _, invalid := range []relay.CommandState{"", "unknown"} {
		record := relay.CommandRecord{State: relay.StateReserved}
		require.ErrorIs(t, record.Transition(invalid), relay.ErrInvalidCommandTransition)
		require.Equal(t, relay.StateReserved, record.State)
	}
	record := relay.CommandRecord{State: "unknown"}
	require.ErrorIs(t, record.Transition(relay.StateReserved), relay.ErrInvalidCommandTransition)
	require.Equal(t, relay.CommandState("unknown"), record.State)
}

func TestCommandStoresRejectInvalidTransitionWithoutMutation(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, relay.Schema())
	require.NoError(t, err)
	sqliteStore, err := relay.NewSQLiteCommandStore(db)
	require.NoError(t, err)
	stores := []struct {
		name  string
		store relay.CommandStore
	}{
		{name: "memory", store: relay.NewMemoryCommandStore()},
		{name: "sqlite", store: sqliteStore},
	}
	for _, item := range stores {
		t.Run(item.name, func(t *testing.T) {
			request := relay.Request{CommandID: "invalid-" + item.name, NodeID: "node", Scenario: "demo", Command: "test"}
			_, err := item.store.Reserve(ctx, request)
			require.NoError(t, err)
			response := relay.Response{Kind: relay.KindCompleted}
			err = item.store.Update(ctx, request.CommandID, request.NodeID, relay.StateCompleted, response)
			require.ErrorIs(t, err, relay.ErrInvalidCommandTransition)
			record, err := item.store.Get(ctx, request.CommandID, request.NodeID)
			require.NoError(t, err)
			require.Equal(t, relay.StateReserved, record.State)
		})
	}
}

func TestMigrateAddsRouteTelemetryToExistingJournal(t *testing.T) {
	db := dbtest.OpenSQLiteMemory(t)
	_, err := db.Exec(`CREATE TABLE relay_commands (command_id TEXT PRIMARY KEY, fingerprint TEXT NOT NULL, correlation_id TEXT NOT NULL, actor TEXT NOT NULL, node_id TEXT NOT NULL, scenario TEXT NOT NULL, command TEXT NOT NULL, args_json TEXT NOT NULL, timeout_seconds INTEGER NOT NULL, max_response_bytes INTEGER NOT NULL, state TEXT NOT NULL, response_kind TEXT NOT NULL, response_data BLOB, response_reason TEXT NOT NULL, response_exit_code INTEGER NOT NULL, response_total_bytes INTEGER NOT NULL, updated_at TEXT NOT NULL)`)
	require.NoError(t, err)
	require.NoError(t, relay.Migrate(context.Background(), db))
	require.NoError(t, relay.Migrate(context.Background(), db), "migration is idempotent")
	for _, name := range []string{"route_name", "route_cost_units", "route_latency_ms"} {
		var found int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM pragma_table_info('relay_commands') WHERE name = ?`, name).Scan(&found))
		require.Equal(t, 1, found, "migration must add %s", name)
	}
}
