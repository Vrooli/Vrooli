package gomodreconcile

import (
	"testing"

	"github.com/vrooli/envkit-go"
)

func TestReconcileComposesGoflags(t *testing.T) {
	env := envkit.Toolchain(reconcileGoEnv(envkit.Env{"GOFLAGS=-p=4", "GOWORK=/tmp/go.work", "HOME=/h"}), reconcileToolchain)
	if !hasEntry(env, "GOFLAGS=-p=4 -mod=mod") || !hasEntry(env, "GOWORK=off") || !hasEntry(env, "HOME=/h") {
		t.Fatalf("reconcile env = %#v", env)
	}
}

func hasEntry(env envkit.Env, entry string) bool {
	for _, candidate := range env {
		if candidate == entry {
			return true
		}
	}
	return false
}
