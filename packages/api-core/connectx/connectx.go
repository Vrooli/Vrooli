// Package connectx provides small integration helpers for mounting Connect-RPC
// handlers into Vrooli's existing HTTP routers.
package connectx

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// ServiceMount is one Connect service handler returned by generated code.
//
// Generated New<Service>Handler functions return both a path and an
// http.Handler. RegisterServices mounts that pair on the scenario router.
// Existing logging, recovery, auth, and CORS middleware continue to compose
// normally because Connect handlers are standard http.Handler values.
type ServiceMount struct {
	Path    string
	Handler http.Handler
}

// RegisterServices mounts Connect handlers onto router.
func RegisterServices(router *mux.Router, mounts ...ServiceMount) {
	if router == nil {
		return
	}
	for _, mount := range mounts {
		path := normalizePath(mount.Path)
		if path == "" || mount.Handler == nil {
			continue
		}
		router.PathPrefix(path).Handler(mount.Handler)
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/") + "/"
}
