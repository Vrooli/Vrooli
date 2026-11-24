package main

import (
	"net/http"

	"scenario-to-cloud/instance"
	"scenario-to-cloud/internal/httputil"
)

// handleInstanceReadiness reports whether the disposable QEMU lane can run on
// this host. It is a read: an unready lane is a 200 report with state
// "unavailable" and one next action, never an opaque error.
func (s *Server) handleInstanceReadiness(w http.ResponseWriter, r *http.Request) {
	verify := r.URL.Query().Get("verify") == "1" || r.URL.Query().Get("verify") == "true"
	options := s.qemuReadiness
	options.VerifyImages = verify
	report := instance.Readiness(r.Context(), options)
	httputil.WriteJSON(w, http.StatusOK, report)
}
