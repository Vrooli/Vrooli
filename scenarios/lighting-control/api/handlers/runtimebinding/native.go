package runtimebinding

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vrooli/api-core/lightingbinding"
	"github.com/vrooli/api-core/runtimediagnostics"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

// StartNativeFromEnv uses the existing lifecycle-injected private directory.
// Ordinary standalone scenario startup has no native owner socket or hardware
// work. The public HTTP handler never receives this handler or its authority.
func StartNativeFromEnv(ctx context.Context, service Service, getenv func(string) string) (func(), error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	directory := getenv(runtimediagnostics.DirectoryEnv)
	if directory == "" {
		return func() {}, nil
	}
	identity := lightingbinding.Identity{Scenario: getenv("VROOLI_SCENARIO"), Variant: getenv("VROOLI_VARIANT"), RuntimeInstanceID: getenv("VROOLI_RUNTIME_INSTANCE_ID"), Generation: getenv("VROOLI_COMPONENT_GENERATION"), PID: os.Getpid()}
	// Shadow/test instances never become the machine's backend owner.
	if identity.Scenario != "lighting-control" || identity.Variant != "live" {
		return func() {}, nil
	}
	listener, err := lightingbinding.Listen(lightingbinding.Directory(directory), identity)
	if err != nil {
		return nil, err
	}
	runtimeCtx, cancel := context.WithCancel(ctx)
	handler := newNativeHandler(identity, service)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 2 * time.Second, MaxHeaderBytes: 4096, BaseContext: func(net.Listener) context.Context { return runtimeCtx }}
	done := make(chan struct{})
	go func() { defer close(done); _ = server.Serve(listener) }()
	return func() {
		cancel()
		shutdownCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		_ = server.Shutdown(shutdownCtx)
		_ = server.Close()
		<-done
		// Endpoint removal precedes drain even if shutdown has exhausted its budget.
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer drainCancel()
		_ = service.Quiesce(drainCtx)
	}, nil
}

type nativeHandler struct {
	mu          sync.Mutex
	state       lightingbinding.State
	lastRequest lightingbinding.Request
	lastReceipt lightingbinding.Receipt
	service     Service
}

func newNativeHandler(identity lightingbinding.Identity, service Service) *nativeHandler {
	return &nativeHandler{state: lightingbinding.State{Version: lightingbinding.Version, Identity: identity}, service: service}
}

func (h *nativeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet && r.URL.Path == "/v1/state" {
		_ = json.NewEncoder(w).Encode(h.state)
		return
	}
	if r.Method != http.MethodPost || r.URL.Path != "/v1/apply" {
		http.NotFound(w, r)
		return
	}
	var request lightingbinding.Request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, 400, errors.New("invalid native lighting request"))
		return
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, 400, errors.New("expected one native lighting request"))
		return
	}
	if request.Identity != h.state.Identity || request.OperationID == "" || len(request.OperationID) > 128 {
		writeError(w, 409, errors.New("native lighting identity mismatch"))
		return
	}
	if request.Binding != nil {
		if err := request.Binding.Validate(); err != nil {
			writeError(w, 400, err)
			return
		}
	}
	if request.OperationID == h.lastRequest.OperationID {
		if !sameRequest(request, h.lastRequest) {
			writeError(w, 409, errors.New("native lighting operation ID reused"))
			return
		}
		_ = json.NewEncoder(w).Encode(h.lastReceipt)
		return
	}
	if request.ExpectedRevision != h.state.Revision || h.state.Revision == ^uint64(0) {
		writeError(w, 409, lightingbinding.ErrConflict)
		return
	}
	// The request owns this revision before any SDK effect. Holding the mutex
	// serializes state reads and retries until its outcome is retained.
	h.state.Revision++
	h.state.Binding = nil
	h.state.ReconcilePending = false
	var zones []lightingbinding.ZoneResult
	var err error
	if request.Binding == nil {
		err = h.service.Quiesce(r.Context())
	} else {
		err = h.service.BindBackend(r.Context(), request.Binding.Address, request.Binding.Incarnation)
		if err == nil {
			var results []*controlv1.ReconcileZoneResult
			results, err = h.service.Reconcile(r.Context())
			if err == nil {
				zones, h.state.ReconcilePending, err = nativeZoneResults(results)
			}
		}
		if err == nil {
			binding := *request.Binding
			h.state.Binding = &binding
		}
	}
	if err != nil {
		// A failed bind/reconcile cannot leave an endpoint usable behind an
		// unbound receipt. Drain independently of a caller's canceled connection.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cleanupErr := h.service.Quiesce(cleanupCtx)
		cancel()
		err = errors.Join(err, cleanupErr)
	}
	receipt := lightingbinding.Receipt{State: h.state, OperationID: request.OperationID, Zones: zones}
	if err != nil {
		receipt.Error = err.Error()
	}
	h.lastRequest = request
	h.lastReceipt = receipt
	_ = json.NewEncoder(w).Encode(receipt)
}

func sameRequest(a, b lightingbinding.Request) bool {
	if a.Identity != b.Identity || a.ExpectedRevision != b.ExpectedRevision || a.OperationID != b.OperationID {
		return false
	}
	if a.Binding == nil || b.Binding == nil {
		return a.Binding == nil && b.Binding == nil
	}
	return *a.Binding == *b.Binding
}

// Native receipts preserve command evidence independently of attachment. A
// disconnected/not-enrolled controller may remain pending without a command.
// Failed, unacknowledged or mismatched commands cannot establish
// a successful reconciled binding.
func nativeZoneResults(results []*controlv1.ReconcileZoneResult) ([]lightingbinding.ZoneResult, bool, error) {
	zones := make([]lightingbinding.ZoneResult, 0, len(results))
	pending := false
	var failure error
	for _, zone := range results {
		if zone == nil || zone.Result == nil {
			failure = errors.Join(failure, errors.New("missing zone reconciliation evidence"))
			continue
		}
		result := zone.Result
		zones = append(zones, lightingbinding.ZoneResult{ControllerID: zone.ControllerId, ZoneID: zone.ZoneId, Operation: int32(result.Operation), Reason: result.Reason, LocalWriteCompleted: result.LocalWriteCompleted, BackendAcknowledged: result.BackendAcknowledged, BackendAckStatus: result.BackendAckStatus, CachedModeReadback: int32(result.CachedModeReadback)})
		switch result.Operation {
		case controlv1.OperationState_OPERATION_STATE_DEFERRED, controlv1.OperationState_OPERATION_STATE_UNSUPPORTED:
			if result.LocalWriteCompleted || result.BackendAcknowledged {
				failure = errors.Join(failure, errors.New("deferred zone has contradictory command evidence"))
			} else {
				pending = true
			}
		case controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED:
			if !result.LocalWriteCompleted || !result.BackendAcknowledged || result.BackendAckStatus != 0 || result.CachedModeReadback == controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH {
				failure = errors.Join(failure, errors.New("zone command acknowledgment or cached readback failed"))
			}
		default:
			failure = errors.Join(failure, errors.New("zone reconciliation failed or remains unverified"))
		}
	}
	return zones, pending, failure
}
