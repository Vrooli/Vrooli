package observability

import (
	"net/http"
	"net/http/pprof"
)

// RegisterPprof mounts stdlib pprof handlers when enabled. It intentionally
// uses the provided mux rather than http.DefaultServeMux so profiling routes
// stay scoped to the scenario server.
func RegisterPprof(mux *http.ServeMux, enabled bool) {
	if !enabled || mux == nil {
		return
	}
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
