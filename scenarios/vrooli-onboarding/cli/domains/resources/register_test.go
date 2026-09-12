package resources

import (
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
	resourcesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources/resourcesv1connect"

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
		case resourcesconnect.ResourcesServiceListResourcesProcedure:
			_, _ = w.Write([]byte(`{"count":1,"resources":[{"name":"postgres","status":"running","category":"database","installed":true}]}`))
		case resourcesconnect.ResourcesServiceGetResourceProcedure:
			_, _ = w.Write([]byte(`{"resource":{"name":"postgres","status":"running","category":"database","installed":true}}`))
		case resourcesconnect.ResourcesServiceGetResourceHealthProcedure:
			_, _ = w.Write([]byte(`{"healthyCount":1,"total":1,"checkedAt":"2026-01-01T00:00:00Z","resources":[{"name":"postgres","status":"healthy","category":"database","available":true}]}`))
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
