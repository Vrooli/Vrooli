package control

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

func TestControlCommandsReadApplyAndExport(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch || r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"scenarios":[],"tools":[],"safeguards":[]}`))
	}))
	groups := CommandGroups(core)
	if err := groups[0].Commands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := groups[1].Commands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	subs := SubcommandGroups(core)
	if err := subs[0].Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := subs[2].Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := subs[2].Subcommands[1].Run([]string{"--name", "safe", "--key", "mode", "--value-json", `"strict"`}); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "union.json")
	if err := subs[1].Subcommands[0].Run([]string{"--output", output}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatal(err)
	}
}

func TestControlRejectsIncompleteSafeguardConfig(t *testing.T) {
	if err := setConfig(nil, nil); err == nil {
		t.Fatal("incomplete safeguard config should fail before using the API")
	}
}
