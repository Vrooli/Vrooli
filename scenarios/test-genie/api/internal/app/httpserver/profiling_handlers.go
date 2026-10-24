package httpserver

import (
	"crypto/subtle"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
)

// handleAdmissionProfile provides deliberately opt-in, token-protected
// process profiles for an active overload investigation. Profiles contain Go
// runtime stacks and allocation metadata, not request bodies. The operator
// supplies the run id as a correlation header which is echoed only in the
// response header and never persisted.
func (s *Server) handleAdmissionProfile(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("TEST_GENIE_PROFILING_ENABLED") != "1" {
		s.writeError(w, http.StatusNotFound, "profiling is disabled")
		return
	}
	token := strings.TrimSpace(os.Getenv("TEST_GENIE_PROFILING_TOKEN"))
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(r.Header.Get("X-Test-Genie-Profile-Token"))) != 1 {
		s.writeError(w, http.StatusForbidden, "profiling token is required")
		return
	}
	if runID := strings.TrimSpace(r.Header.Get("X-Test-Genie-Run-ID")); runID != "" {
		w.Header().Set("X-Test-Genie-Run-ID", runID)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	switch kind {
	case "heap":
		w.Header().Set("Content-Disposition", `attachment; filename="test-genie-heap.pprof"`)
		if err := pprof.Lookup("heap").WriteTo(w, 0); err != nil {
			s.log("write heap profile", map[string]interface{}{"error": err.Error()})
		}
	case "cpu":
		seconds := 10
		if raw := strings.TrimSpace(r.URL.Query().Get("seconds")); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 1 && parsed <= 30 {
				seconds = parsed
			}
		}
		w.Header().Set("Content-Disposition", `attachment; filename="test-genie-cpu.pprof"`)
		if err := pprof.StartCPUProfile(w); err != nil {
			s.writeError(w, http.StatusConflict, "CPU profiling is already active")
			return
		}
		defer pprof.StopCPUProfile()
		time.Sleep(time.Duration(seconds) * time.Second)
	default:
		s.writeError(w, http.StatusBadRequest, "kind must be heap or cpu")
	}
}
