package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/lifecycle"
	common "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

const bootA = "72cc2f05-18b9-44bb-aa8a-dc824f53c8f2"
const bootB = "7af44c84-90bd-42e0-b82a-57a03cdf2171"

func testHostStore(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(hostMaintenanceSchema); err != nil {
		t.Fatal(err)
	}
	return db
}

func testHostOwner(t *testing.T, db *sql.DB, boot, instance string, hooks lifecycle.Hooks) *hostMaintenance {
	t.Helper()
	host, err := newHostMaintenance(t.Context(), db, boot, instance, hooks)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

func prepareHostTest(t *testing.T, host *hostMaintenance, id string) *common.LifecycleStanding {
	t.Helper()
	res, err := host.Prepare(t.Context(), connect.NewRequest(&common.LifecyclePrepareRequest{OperationId: id, Reason: hostReasonPrefix + host.boot}))
	if err != nil {
		t.Fatal(err)
	}
	out := res.Msg.GetStanding()
	if !out.AdmissionClosed || out.FencePersistence != common.LifecycleFencePersistence_LIFECYCLE_FENCE_PERSISTENCE_OPERATION_DURABLE {
		t.Fatalf("unexpected prepare: %v", out)
	}
	return out
}

func assertHostGate(t *testing.T, host *hostMaintenance, status int) {
	t.Helper()
	handler := host.Mount(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil))
	if response.Code != status {
		t.Fatalf("admission returned %d, want %d", response.Code, status)
	}
}

func TestHostMaintenanceReceiptSurvivesProcessAndBoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "host.db")
	db := testHostStore(t, path)
	hooks := lifecycle.Hooks{Snapshot: func(context.Context) (lifecycle.Inventory, error) {
		return lifecycle.Inventory{Complete: true, Work: []*common.LifecycleWorkItem{{Id: "goal", Status: "native_goal_continuation"}}}, nil
	}}
	host := testHostOwner(t, db, bootA, "process-a", hooks)
	prepared := prepareHostTest(t, host, "reboot")
	assertHostGate(t, host, http.StatusServiceUnavailable)
	drain := &common.LifecycleDrainRequest{OperationId: "reboot", FenceToken: prepared.FenceToken, Revision: prepared.Revision}
	check := &common.LifecycleStatusRequest{OperationId: "reboot", FenceToken: prepared.FenceToken, Revision: prepared.Revision}
	release := &common.LifecycleResumeRequest{OperationId: "reboot", FenceToken: prepared.FenceToken, Revision: prepared.Revision}
	first, err := host.Drain(t.Context(), connect.NewRequest(drain))
	if err != nil || !first.Msg.GetStanding().GetDrained() || len(first.Msg.GetStanding().GetWork()) != 0 {
		t.Fatalf("drain: %v, %v", first, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db = testHostStore(t, path)
	host = testHostOwner(t, db, bootA, "process-b", hooks)
	assertHostGate(t, host, http.StatusServiceUnavailable)
	verified, err := host.Status(t.Context(), connect.NewRequest(check))
	if err != nil {
		t.Fatal(err)
	}
	if out := verified.Msg.GetStanding(); !out.Drained || out.ProviderInstanceId != "process-b" || out.FenceToken != prepared.FenceToken || out.Revision != prepared.Revision {
		t.Fatalf("restart changed the checkpoint: %v", out)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	host = testHostOwner(t, testHostStore(t, path), bootB, "process-c", hooks)
	assertHostGate(t, host, http.StatusServiceUnavailable)
	next, err := host.Status(t.Context(), connect.NewRequest(check))
	if err != nil || next.Msg.GetStanding().GetDrained() {
		t.Fatalf("prior boot checkpoint became dispatchable: %v, %v", next, err)
	}
	if _, err := host.Drain(t.Context(), connect.NewRequest(drain)); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("old boot accepted a new drain: %v", err)
	}
	for i := 0; i < 2; i++ {
		open, err := host.Resume(t.Context(), connect.NewRequest(release))
		if err != nil {
			t.Fatal(err)
		}
		if out := open.Msg.GetStanding(); out.AdmissionClosed || out.Phase != common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN || out.FenceToken != prepared.FenceToken || out.Revision <= prepared.Revision {
			t.Fatalf("bad release: %v", out)
		}
	}
	assertHostGate(t, host, http.StatusNoContent)
	host = testHostOwner(t, testHostStore(t, path), bootB, "process-d", hooks)
	assertHostGate(t, host, http.StatusNoContent)
	out, err := host.Status(t.Context(), connect.NewRequest(&common.LifecycleStatusRequest{OperationId: "reboot"}))
	if err != nil || out.Msg.GetStanding().GetPhase() != common.LifecycleMaintenancePhase_LIFECYCLE_MAINTENANCE_PHASE_OPEN {
		t.Fatalf("lost release receipt on restart: %v, %v", out, err)
	}
}

func TestHostMaintenanceWarmAbortAndWork(t *testing.T) {
	var running, blocked, quiesced atomic.Bool
	running.Store(true)
	hooks := lifecycle.Hooks{
		Quiesce: func(context.Context) error { quiesced.Store(true); return nil },
		Resume:  func(context.Context) error { quiesced.Store(false); return nil },
		Snapshot: func(context.Context) (lifecycle.Inventory, error) {
			report := lifecycle.Inventory{Complete: true}
			if blocked.Load() {
				report.Work = []*common.LifecycleWorkItem{{Id: "pane", Kind: "tmux", Status: "explicit_host_recovery_required"}}
			} else if running.Load() {
				report.Remaining = 1
				report.Work = []*common.LifecycleWorkItem{{Id: "turn", Status: "ordinary_turn_running"}}
			}
			return report, nil
		},
	}
	host := testHostOwner(t, testHostStore(t, filepath.Join(t.TempDir(), "abort.db")), bootA, "process-a", hooks)
	if _, err := host.Status(t.Context(), connect.NewRequest(&common.LifecycleStatusRequest{OperationId: "absent"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing operation is not wire NotFound: %v", err)
	}
	prepared := prepareHostTest(t, host, "abort")
	msg := &common.LifecycleDrainRequest{OperationId: "abort", FenceToken: prepared.FenceToken, Revision: prepared.Revision, Timeout: durationpb.New(time.Second)}
	seen, err := host.checkpoint(t.Context(), msg)
	if err != nil || seen.GetDrained() || seen.GetRemaining() != 1 || !quiesced.Load() {
		t.Fatalf("ordinary work was not left running: %v, %v", seen, err)
	}
	blocked.Store(true)
	if _, err := host.Drain(t.Context(), connect.NewRequest(msg)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unsupported pane was not blocking: %v", err)
	}
	if _, err := host.Resume(t.Context(), connect.NewRequest(&common.LifecycleResumeRequest{OperationId: "abort", FenceToken: "wrong", Revision: 1})); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("false receipt released admission: %v", err)
	}
	_, err = host.Resume(t.Context(), connect.NewRequest(&common.LifecycleResumeRequest{OperationId: "abort", FenceToken: prepared.FenceToken, Revision: 1}))
	if err != nil || quiesced.Load() {
		t.Fatalf("warm abort did not release: %v", err)
	}
	assertHostGate(t, host, http.StatusNoContent)
	blocked.Store(false)
	prepared = prepareHostTest(t, host, "retry")
	msg.OperationId, msg.FenceToken, msg.Revision = "retry", prepared.FenceToken, prepared.Revision
	running.Store(false)
	drained, err := host.Drain(t.Context(), connect.NewRequest(msg))
	if err != nil || !drained.Msg.GetStanding().GetDrained() {
		t.Fatalf("completed ordinary work did not drain: %v, %v", drained, err)
	}
}

func TestHostMaintenanceSharesAdmissionAndOwnsLifecycle(t *testing.T) {
	host := testHostOwner(t, testHostStore(t, filepath.Join(t.TempDir(), "shared.db")), bootA, "process", lifecycle.Hooks{})
	entered, finish, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
	handler := host.Mount(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-finish
		w.WriteHeader(http.StatusNoContent)
	}))
	go func() {
		defer close(exited)
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil))
	}()
	defer func() { <-exited }()
	defer close(finish)
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not enter shared admission")
	}

	first := prepareHostTest(t, host, "owner")
	repeat := prepareHostTest(t, host, "owner")
	if repeat.FenceToken != first.FenceToken || repeat.Revision != first.Revision {
		t.Fatal("prepare retry replaced the durable tuple")
	}
	req := httptest.NewRequest(http.MethodPost, "/common.v1.LifecycleMaintenanceService/Resume", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	httpReceipt := httptest.NewRecorder()
	handler.ServeHTTP(httpReceipt, req)
	if httpReceipt.Code != http.StatusServiceUnavailable {
		t.Fatalf("ordinary lifecycle bypassed host ownership: %d", httpReceipt.Code)
	}
	drain := &common.LifecycleDrainRequest{OperationId: "owner", FenceToken: first.FenceToken, Revision: first.Revision}
	standing, err := host.checkpoint(t.Context(), drain)
	if err != nil || standing.GetDrained() || standing.GetAdmitting() != 1 {
		t.Fatalf("host did not count admitted work: %v, %v", standing, err)
	}
}

func TestHostMaintenanceReleasesPartialPrepareAfterRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.db")
	store := testHostStore(t, path)
	first := testHostOwner(t, store, bootA, "process-a", lifecycle.Hooks{})
	prepared := prepareHostTest(t, first, "partial")
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	next := testHostOwner(t, testHostStore(t, path), bootA, "process-b", lifecycle.Hooks{})
	assertHostGate(t, next, http.StatusServiceUnavailable)
	current, err := next.Status(t.Context(), connect.NewRequest(&common.LifecycleStatusRequest{OperationId: "partial"}))
	if err != nil || current.Msg.GetStanding().GetDrained() || current.Msg.GetStanding().GetFenceToken() != prepared.FenceToken {
		t.Fatalf("partial prepare changed after restart: %v, %v", current, err)
	}
	_, err = next.Resume(t.Context(), connect.NewRequest(&common.LifecycleResumeRequest{OperationId: "partial", FenceToken: prepared.FenceToken, Revision: prepared.Revision}))
	if err != nil {
		t.Fatal(err)
	}
	assertHostGate(t, next, http.StatusNoContent)
}

func TestHostMaintenanceSocketRequiresManagedLiveRuntime(t *testing.T) {
	for _, test := range []struct {
		name, goos, managed, instance, variant, want string
	}{
		{"managed default", "linux", "true", " id ", "", "id"},
		{"managed live", "linux", " TRUE ", "id", " live ", "id"},
		{"shadow", "linux", "true", "id", "shadow", ""},
		{"unmanaged", "linux", "", "id", "live", ""},
		{"missing instance", "linux", "true", " ", "live", ""},
		{"unsupported OS", "darwin", "true", "id", "live", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := map[string]string{
				"VROOLI_LIFECYCLE_MANAGED":   test.managed,
				"VROOLI_RUNTIME_INSTANCE_ID": test.instance,
				"VROOLI_VARIANT":             test.variant,
			}
			if got := hostMaintenanceInstance(test.goos, func(key string) string { return env[key] }); got != test.want {
				t.Fatalf("host socket instance = %q, want %q", got, test.want)
			}
		})
	}
}
