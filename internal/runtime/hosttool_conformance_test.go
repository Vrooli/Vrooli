package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
)

// TestImageToolsHostToolsAreRegistered is the platform-side drift ratchet for
// the host-tool-fetch contract: every host tool the image-tools scenario
// declares in its service.json must resolve to a registered runtime tool
// handler (i.e. an internal/tools/<name>/tool.json exists). The root module
// cannot import the scenario module, so the binding is asserted across the file
// boundary by reading the scenario's service.json directly. A provider that
// references a host tool with no platform definition is a red build.
func TestImageToolsHostToolsAreRegistered(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller path")
	}
	servicePath := filepath.Clean(filepath.Join(
		filepath.Dir(currentFile), "..", "..",
		"scenarios", "image-tools", ".vrooli", "service.json",
	))
	data, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read image-tools service.json: %v", err)
	}
	var manifest struct {
		HostTools []struct {
			Name string `json:"name"`
		} `json:"hostTools"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse service.json: %v", err)
	}
	if len(manifest.HostTools) == 0 {
		t.Fatal("image-tools service.json declares no hostTools; expected backend host tools")
	}

	reg, err := ensureRegistry()
	if err != nil {
		t.Fatalf("ensureRegistry: %v", err)
	}
	for _, ht := range manifest.HostTools {
		if reg.lookup(hostreq.KindTool, ht.Name) == nil {
			t.Errorf("image-tools declares host tool %q, but no internal/tools/%s/tool.json is registered", ht.Name, ht.Name)
		}
	}
}
