package audio_runtime

import (
	"testing"

	"github.com/gorilla/mux"
)

func TestModuleMounts(t *testing.T) {
	m := Module(Deps{})
	if m.Name != "audio_runtime" || m.Mount == nil || len(m.Endpoints) == 0 {
		t.Fatalf("invalid module: %#v", m)
	}
	m.Mount(mux.NewRouter())
}
