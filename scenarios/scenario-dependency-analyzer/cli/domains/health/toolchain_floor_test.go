package health

import (
	"testing"

	"github.com/vrooli/envkit-go"
)

func TestFreshnessProbeComposesGoflags(t *testing.T) {
	env := envkit.Toolchain(freshnessGoEnv(envkit.Env{"GOFLAGS=-mod=mod", "GOWORK=/tmp/go.work"}, true), envkit.ToolchainOptions{Width: 4})
	if !hasEntry(env, "GOFLAGS=-mod=mod -p=4") || !hasEntry(env, "GOWORK=off") || !hasEntry(env, "GOPROXY=off") {
		t.Fatalf("freshness env = %#v", env)
	}
	if online := freshnessGoEnv(envkit.Env{}, false); hasEntry(online, "GOPROXY=off") {
		t.Fatalf("online env sets GOPROXY=off: %#v", online)
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
