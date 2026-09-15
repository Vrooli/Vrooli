// Package follow exposes owner-only branch-follow policies: set a node to a
// branch, stop, list, and check now. It is a REST exception like readiness:
// an owner operational setting with a CLI wrapper, not a workflow RPC.
package follow

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"vrooli-bridge/internal/auth"
	internalfollow "vrooli-bridge/internal/follow"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/registry"
)

// Schema contributes the branch-follow policy table.
func Schema() string { return internalfollow.Schema() }

// NewProvisioner dispatches follow updates through the shared provisioning
// service, so each one is admitted, audited, and tracked like `provision sync`.
func NewProvisioner(svc provision.Service) internalfollow.Provisioner {
	return provisioner{svc: svc}
}

type provisioner struct{ svc provision.Service }

func (p provisioner) Provision(ctx context.Context, nodeID, revision, actor string) (string, error) {
	decision, err := p.svc.Sync(ctx, provision.SyncInput{Actor: actor, NodeID: nodeID, TargetRevision: revision})
	return decision.OpID, err
}

// NewNodeChecker confirms a node exists before a policy is written for it.
func NewNodeChecker(svc registry.Service) internalfollow.NodeChecker {
	return func(ctx context.Context, nodeID string) error {
		_, err := svc.Get(ctx, nodeID)
		return err
	}
}

// RunScheduler checks every followed node shortly after start and then every
// interval until ctx ends.
func RunScheduler(ctx context.Context, svc *internalfollow.Service, interval time.Duration, logger *log.Logger) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		for _, policy := range svc.CheckAll(ctx) {
			if strings.HasPrefix(policy.LastResult, "dispatched ") || strings.Contains(policy.LastResult, "not dispatched") || strings.HasPrefix(policy.LastResult, "could not") {
				logger.Printf("follow: node %s on %s: %s", policy.NodeID, policy.Branch, policy.LastResult)
			}
		}
		timer.Reset(interval)
	}
}

type followRequest struct {
	Branch    string `json:"branch"`
	RepoURL   string `json:"repo_url"`
	UpdateNow bool   `json:"update_now"`
}

// Module mounts the owner-only branch-follow endpoints.
func Module(svc *internalfollow.Service) module.Module {
	owner := func(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if _, err := auth.RequireOwner(r.Context()); err != nil {
				http.Error(w, "owner authentication required", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
	list := owner(func(w http.ResponseWriter, r *http.Request) {
		policies, err := svc.List(r.Context())
		if err != nil {
			writeError(w, err)
			return
		}
		if policies == nil {
			policies = []internalfollow.Policy{}
		}
		writeJSON(w, map[string]any{"policies": policies})
	})
	set := owner(func(w http.ResponseWriter, r *http.Request) {
		var body followRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
			http.Error(w, "invalid branch-follow request: "+err.Error(), http.StatusBadRequest)
			return
		}
		policy, err := svc.Follow(r.Context(), mux.Vars(r)["node"], body.Branch, body.RepoURL, body.UpdateNow)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, policy)
	})
	unset := owner(func(w http.ResponseWriter, r *http.Request) {
		if err := svc.Unfollow(r.Context(), mux.Vars(r)["node"]); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, map[string]string{"status": "unfollowed"})
	})
	check := owner(func(w http.ResponseWriter, r *http.Request) {
		policy, err := svc.Check(r.Context(), mux.Vars(r)["node"])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, policy)
	})
	return module.Module{Name: "follow", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/follow", list).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/follow/{node}", set).Methods(http.MethodPut)
		r.HandleFunc("/api/v1/follow/{node}", unset).Methods(http.MethodDelete)
		r.HandleFunc("/api/v1/follow/{node}/check", check).Methods(http.MethodPost)
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var invalid internalfollow.ErrInvalid
	var missing registry.ErrNodeNotFound
	switch {
	case errors.As(err, &invalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, internalfollow.ErrNotFollowing), errors.As(err, &missing):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		// The branch head could not be read or the store failed: the owner's
		// request was valid, an upstream dependency was not.
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}
