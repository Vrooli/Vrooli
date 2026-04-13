package watchdog

import (
	"os"
	"path/filepath"
	"testing"

	"vrooli-autoheal/internal/platform"
)

func TestResolveVrooliRootCanonicalizesContractDescendant(t *testing.T) {
	plat := &platform.Capabilities{Platform: platform.Linux}
	probe := newFakeProbe()
	root := newWatchdogContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	probe.env["VROOLI_ROOT"] = nested

	d := detectorWithProbe(plat, probe)
	if got := d.resolveVrooliRoot(); got != root {
		t.Fatalf("resolveVrooliRoot() = %q, want %q", got, root)
	}
}
