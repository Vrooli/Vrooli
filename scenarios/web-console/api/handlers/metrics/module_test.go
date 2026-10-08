package metrics

import (
	"testing"

	"github.com/gorilla/mux"
)

func TestModuleMounts(t *testing.T) {
	m := Module(nil, nil)
	if m.Name != "metrics" || m.Mount == nil || len(m.Endpoints) == 0 {
		t.Fatalf("invalid module: %#v", m)
	}
	m.Mount(mux.NewRouter())
}
