package resources

import (
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterAndGetValidation(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if group.Name != "resources" || len(group.Subcommands) != 3 {
		t.Fatalf("unexpected resources group: %+v", group)
	}
	if err := group.Subcommands[1].Run(nil); err == nil {
		t.Fatal("resources get without a name should fail")
	}
	if len(resourceRows(nil)) != 1 || len(healthRows(nil)) != 1 {
		t.Fatal("empty resource rows should be actionable")
	}
}

func TestCommandsRenderAPIResponses(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/resources":
			_, _ = w.Write([]byte(`{"count":1,"resources":[{"name":"postgres","status":"running","category":"database","installed":true}]}`))
		case "/api/v1/resources/postgres":
			_, _ = w.Write([]byte(`{"name":"postgres","status":"running","category":"database","installed":true}`))
		case "/api/v1/resources/health":
			_, _ = w.Write([]byte(`{"healthy_count":1,"total":1,"checked_at":"2026-01-01T00:00:00Z","resources":[{"name":"postgres","status":"healthy","category":"database","available":true}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	group := Register(core)
	if err := group.Subcommands[0].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[1].Run([]string{"postgres", "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := group.Subcommands[2].Run([]string{"--json"}); err != nil {
		t.Fatal(err)
	}
}
