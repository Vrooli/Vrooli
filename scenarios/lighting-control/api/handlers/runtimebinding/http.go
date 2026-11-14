package runtimebinding

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/vrooli/api-core/identity"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

type Service interface {
	BindBackend(context.Context, string, string) error
	Quiesce(context.Context) error
	Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error)
}

type bindRequest struct {
	Address     string `json:"address"`
	Incarnation string `json:"incarnation"`
	Generation  string `json:"generation"`
}

// Register exposes the narrow scenario half of the root lighting owner
// protocol. Both operations require an authenticated, verified human with
// write capability; the endpoint accepts no command or arbitrary arguments.
func Register(mux *http.ServeMux, service Service, generation string) {
	mux.HandleFunc("POST /internal/lighting/backend/bind", func(w http.ResponseWriter, r *http.Request) {
		if err := requireVerifiedWriter(r.Context()); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		var request bindRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid backend binding request"))
			return
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, errors.New("backend binding request must contain one JSON object"))
			return
		}
		if strings.TrimSpace(generation) == "" || strings.TrimSpace(request.Generation) != generation {
			writeError(w, http.StatusConflict, errors.New("backend binding does not target this scenario generation"))
			return
		}
		if strings.TrimSpace(request.Address) == "" || strings.TrimSpace(request.Incarnation) == "" || len(request.Incarnation) > 512 || strings.ContainsRune(request.Incarnation, 0) {
			writeError(w, http.StatusBadRequest, errors.New("exact loopback address and verified backend incarnation are required"))
			return
		}
		if err := service.BindBackend(r.Context(), request.Address, request.Incarnation); err != nil {
			writeError(w, http.StatusFailedDependency, err)
			return
		}
		results, err := service.Reconcile(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		zones := make([]map[string]any, 0, len(results))
		for _, zone := range results {
			if zone == nil || zone.Result == nil {
				continue
			}
			zones = append(zones, map[string]any{
				"controller_id": zone.ControllerId, "zone_id": zone.ZoneId,
				"operation": zone.Result.Operation.String(), "reason": zone.Result.Reason,
				"local_write_completed": zone.Result.LocalWriteCompleted,
				"backend_acknowledged":  zone.Result.BackendAcknowledged,
				"backend_ack_status":    zone.Result.BackendAckStatus,
				"cached_mode_readback":  zone.Result.CachedModeReadback.String(),
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"bound": true, "zones": zones, "physical_output_verified": false})
	})
	mux.HandleFunc("POST /internal/lighting/backend/quiesce", func(w http.ResponseWriter, r *http.Request) {
		if err := requireVerifiedWriter(r.Context()); err != nil {
			writeError(w, http.StatusForbidden, err)
			return
		}
		if err := service.Quiesce(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"quiesced":true}`)
	})
}

func requireVerifiedWriter(ctx context.Context) error {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.Verified || !principal.IsHuman() ||
		!(slices.Contains(principal.Scopes, "lighting-control:write") || slices.Contains(principal.Scopes, "*")) ||
		!(slices.Contains(principal.Scopes, "lighting-control:backend-owner") || slices.Contains(principal.Scopes, "*")) {
		return errors.New("verified human identity with lighting-control:write and lighting-control:backend-owner is required")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
