package operator

import (
	"net/http"
	"path/filepath"
	"testing"

	clitest "vrooli-onboarding/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterAndPatchValidation(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "operator" || len(group.Subcommands) != 5 {
		t.Fatalf("unexpected operator group: %+v", group)
	}
	if err := group.Subcommands[1].Run([]string{"--body-file", "missing.json"}); err == nil {
		t.Fatal("missing patch file should fail")
	}
	if err := group.Subcommands[2].Run(nil); err == nil {
		t.Fatal("missing safeguard flags should fail")
	}
}

func TestCommandsUseV2ReadAndPatchRoutes(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready","scenarios":[],"host_safeguards":{}}`))
	}))
	group := Register(core)
	if err := group.Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	patchPath := filepath.Join(t.TempDir(), "patch.json")
	if err := clitest.WriteJSON(patchPath, map[string]any{"scenarios": map[string]any{"demo": map[string]any{"enabled": true}}}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[1].Run([]string{"--body-file", patchPath, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[2].Run([]string{"--name", "safe", "--key", "mode", "--value-json", `"strict"`, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[3].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[4].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
}
