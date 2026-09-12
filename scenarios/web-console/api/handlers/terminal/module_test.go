package terminal

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

func TestModuleMounts(t *testing.T) {
	m := Module(nil, LegacyDeps{Upload: http.NotFound, WS: http.NotFound}, nil)
	if m.Name != "terminal" || m.Mount == nil || len(m.Endpoints) == 0 {
		t.Fatalf("invalid module: %#v", m)
	}
	m.Mount(mux.NewRouter())
}
