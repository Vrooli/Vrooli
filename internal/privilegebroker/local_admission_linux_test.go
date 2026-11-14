//go:build linux

package privilegebroker

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// This opt-in probe reads the real registry, interface and socket ownership.
// It never constructs an executor or contacts UFW/the privilege broker.
func TestLiveLocalAdmissionListenerReadOnly(t *testing.T) {
	path := os.Getenv("VROOLI_ADMISSION_BINDING_FILE")
	if path == "" {
		t.Skip("requires an explicit local binding fixture")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var binding LocalAdmissionBinding
	if err := json.Unmarshal(data, &binding); err != nil {
		t.Fatal(err)
	}
	if err := binding.Validate(); err != nil {
		t.Fatal(err)
	}
	home, err := config.HomeDir()
	if err != nil {
		t.Fatal(err)
	}
	dbPath, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	port, err := resolveAdmissionListener(ctx, LocalAdmissionPolicy{UID: uint32(os.Getuid()), RuntimeDB: dbPath}, binding)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("verified live %s/%s listener on port %d for UID %d; no firewall operations", binding.Scenario, binding.Listener, port, os.Getuid())
}
