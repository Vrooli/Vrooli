package relay_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"

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
	require.NoError(t, storeA.Update(context.Background(), request.CommandID, request.NodeID, relay.StateCompleted, response))

	storeB, err := relay.NewSQLiteCommandStore(d)
	require.NoError(t, err)
	record, err := storeB.Get(context.Background(), request.CommandID, request.NodeID)
	require.NoError(t, err)
	require.Equal(t, relay.StateCompleted, record.State)
	require.Equal(t, []byte("ok"), record.Response.Data)
	require.Equal(t, []string{"--json"}, record.Request.Args)
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
