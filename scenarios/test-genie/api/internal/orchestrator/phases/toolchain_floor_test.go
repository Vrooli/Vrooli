package phases

import (
	"strings"
	"testing"

	"github.com/vrooli/envkit-go"
)

func TestPhaseCommandEnvCarriesFloor(t *testing.T) {
	env := phaseCommandEnv(envkit.Env{"GOFLAGS=-mod=mod", "HOME=/h"})
	var goflags, gomaxprocs string
	for _, entry := range env {
		if strings.HasPrefix(entry, "GOFLAGS=") {
			goflags = strings.TrimPrefix(entry, "GOFLAGS=")
		}
		if strings.HasPrefix(entry, "GOMAXPROCS=") {
			gomaxprocs = strings.TrimPrefix(entry, "GOMAXPROCS=")
		}
	}
	if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") || gomaxprocs == "" {
		t.Fatalf("phase command env lacks the floor: %#v", env)
	}
}
