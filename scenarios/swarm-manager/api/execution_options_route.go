package main

import (
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// executionOptions proxies Agent Manager's authoritative runner/model catalog
// so the operator surface never maintains a second list of providers.
func (s *Server) executionOptions(w http.ResponseWriter, r *http.Request) {
	role := strings.TrimSpace(r.URL.Query().Get("role"))
	if role == "" {
		role = "code.smart"
	}
	options, err := s.agentSvc.ListExecutionOptions(r.Context(), role)
	if err != nil {
		apierr.MapError(w, "[execution-options] list", err)
		return
	}
	if err := httputil.ProtoJSON(w, &apipb.ListExecutionOptionsResponse{Options: options}); err != nil {
		apierr.MapError(w, "[execution-options] encode", apierr.Internal("failed to encode execution options"))
	}
}
