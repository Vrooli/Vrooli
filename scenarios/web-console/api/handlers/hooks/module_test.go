package hooks

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
)

func TestModuleMounts(t *testing.T) {
	m := Module(Deps{Stop: http.NotFound, PromptSubmit: http.NotFound})
	if m.Name != "hooks" || m.Mount == nil || len(m.Endpoints) == 0 {
		t.Fatalf("invalid module: %#v", m)
	}
	m.Mount(mux.NewRouter())
}
