package agentsessioncontainment

import (
	"os"
	"testing"
)

// TestPrintRenderedUnits writes the units this safeguard converges so an
// operator can install them on a host whose control-plane binary is older
// than this change. It is skipped unless asked for.
func TestPrintRenderedUnits(t *testing.T) {
	dir := os.Getenv("VROOLI_RENDER_UNITS_TO")
	if dir == "" {
		t.Skip("set VROOLI_RENDER_UNITS_TO=<dir> to write the rendered slice units")
	}
	s := DefaultSettings()
	agents, err := Render(s)
	if err != nil {
		t.Fatal(err)
	}
	services, err := RenderServices(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []struct{ name, content string }{
		{agents.Primary().Name, agents.Primary().Content},
		{services.Primary().Name, services.Primary().Content},
	} {
		if err := os.WriteFile(dir+"/"+artifact.name, []byte(artifact.content), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s/%s:\n%s", dir, artifact.name, artifact.content)
	}
}
