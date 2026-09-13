package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const AdmissionPath = "/api/v1/maintenance/admission"

type Handler struct {
	gate      *Gate
	observe   func(context.Context) (Inventory, error)
	owners    authn.TokenVerifier
	interlock *ScenarioInterlock
}

// NewHandler must be mounted only after every orchestration admission path uses
// gate. observe is the owner's durable and physical drain projection, never a
// client-provided count. This constructor does not grant authority or mint tokens.
func NewHandler(gate *Gate, observe func(context.Context) (Inventory, error), owners authn.TokenVerifier, interlock *ScenarioInterlock) (*Handler, error) {
	if gate == nil || observe == nil || owners == nil || interlock == nil {
		return nil, fmt.Errorf("maintenance requires a shared gate, owner drain observer and owner verifier")
	}
	return &Handler{gate: gate, observe: observe, owners: owners, interlock: interlock}, nil
}

// standing brackets the read projection with the durable fence revision and
// active admissions. A concurrent resume/admit/enter cannot create an empty proof.
func (h *Handler) standing(ctx context.Context) (Standing, error) {
	before, err := h.gate.Status(ctx)
	if err != nil {
		return before, err
	}
	inv, observeErr := h.observe(ctx)
	after, err := h.gate.Status(ctx)
	after.Inventory = &inv
	after.LifecycleInterlock = ScenarioLockV1
	if err != nil {
		return after, err
	}
	if observeErr != nil {
		return after, observeErr
	}
	if inv.Remaining == nil || *inv.Remaining < 0 || len(inv.Unknown) > 0 {
		return after, fmt.Errorf("executor inventory is incomplete")
	}
	after.Remaining = inv.Remaining
	if before.Revision != after.Revision {
		return after, ErrConflict
	}
	after.Drained = after.Closed && before.Admitting == 0 && after.Admitting == 0 && *inv.Remaining == 0 && len(inv.Work) == 0 && len(inv.Executors) == 0
	return after, nil
}

type maintenanceRequest struct {
	Reason         string `json:"reason"`
	Revision       int64  `json:"revision"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if req.Method == http.MethodGet && req.URL.Path == AdmissionPath {
		state, err := h.standing(req.Context())
		h.respond(w, state, err)
		return
	}
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	operation := strings.TrimPrefix(req.URL.Path, AdmissionPath+"/")
	if operation != "enter" && operation != "resume" && operation != "wait" {
		http.NotFound(w, req)
		return
	}
	token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		http.Error(w, "owner credential required", http.StatusUnauthorized)
		return
	}
	owner, err := h.owners.Verify(req.Context(), strings.TrimSpace(token))
	if err != nil {
		http.Error(w, "owner credential could not be verified", http.StatusUnauthorized)
		return
	}
	if !owner.Verified || owner.Kind != identity.ActorHuman || strings.TrimSpace(owner.Subject) == "" || !maintenanceScope(owner.Scopes) {
		http.Error(w, "owner maintenance authority required", http.StatusForbidden)
		return
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 4096))
	decoder.DisallowUnknownFields()
	var input maintenanceRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid maintenance request", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "one maintenance request is required", http.StatusBadRequest)
		return
	}
	switch operation {
	case "enter":
		if strings.TrimSpace(input.Reason) == "" || len(input.Reason) > 512 {
			http.Error(w, "reason must contain 1-512 bytes", http.StatusBadRequest)
			return
		}
		state, err := h.gate.Enter(req.Context(), owner.Subject, input.Reason)
		h.respond(w, state, err)
	case "resume":
		release, err := h.interlock.acquire(req.Context())
		if err != nil {
			// In particular do not acquire gate.mu on lock contention: the
			// lifecycle owner may be reading status under that lock.
			h.respond(w, nil, err)
			return
		}
		defer release()
		state, err := h.gate.Resume(req.Context(), owner.Subject, input.Revision)
		h.respond(w, state, err)
	case "wait":
		if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 120 {
			http.Error(w, "timeoutSeconds must be 1-120", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), time.Duration(input.TimeoutSeconds)*time.Second)
		defer cancel()
		var inventory Inventory
		state, err := h.gate.Wait(ctx, func(ctx context.Context) (int, error) {
			var err error
			inventory, err = h.observe(ctx)
			if err != nil {
				return -1, err
			}
			if inventory.Remaining == nil || len(inventory.Unknown) > 0 {
				return -1, fmt.Errorf("executor inventory is incomplete")
			}
			if *inventory.Remaining == 0 && (len(inventory.Work) > 0 || len(inventory.Executors) > 0) {
				return -1, fmt.Errorf("executor inventory is inconsistent")
			}
			return *inventory.Remaining, nil
		}, time.Second)
		state.Inventory = &inventory
		state.LifecycleInterlock = ScenarioLockV1
		h.respond(w, state, err)
	}
}

func maintenanceScope(scopes []string) bool {
	for _, scope := range scopes {
		switch scope {
		case "*", "agent-manager:write":
			return true
		}
	}
	return false
}

func (h *Handler) respond(w http.ResponseWriter, state any, err error) {
	if err == nil {
		_ = json.NewEncoder(w).Encode(state)
		return
	}
	code := http.StatusServiceUnavailable
	if errors.Is(err, ErrConflict) {
		code = http.StatusConflict
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		code = http.StatusRequestTimeout
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(struct {
		State any    `json:"state"`
		Error string `json:"error"`
	}{state, err.Error()})
}
