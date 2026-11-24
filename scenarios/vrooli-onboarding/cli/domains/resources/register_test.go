package resources

import (
	"io"
	"net/http"
	"strings"
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

func TestResourceCommandFamilyNormalizesTargetBeforeDispatch(t *testing.T) {
	bodies := map[string]string{}
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies[r.URL.Path] = string(body)
		switch r.URL.Path {
		case resourcesconnect.ResourcesServiceListResourcesProcedure:
			_, _ = w.Write([]byte(`{"count":0,"resources":[]}`))
		case resourcesconnect.ResourcesServiceGetResourceProcedure:
			_, _ = w.Write([]byte(`{"resource":{"name":"postgres","status":"running"}}`))
		case resourcesconnect.ResourcesServiceGetResourceHealthProcedure:
			_, _ = w.Write([]byte(`{"healthyCount":0,"total":0,"resources":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	commands := Register(core).Subcommands
	for index, args := range [][]string{
		{"--target", " remote ", "--json"},
		{"postgres", "--target", " remote ", "--json"},
		{"--target", " remote ", "--json"},
	} {
		if err := commands[index].Run(args); err != nil {
			t.Fatalf("command %d error: %v", index, err)
		}
	}
	if len(bodies) != 3 {
		t.Fatalf("request count = %d, want all three resource commands", len(bodies))
	}
	for procedure, body := range bodies {
		if !strings.Contains(body, `"target":"remote"`) {
			t.Errorf("%s body = %q, want trimmed target remote", procedure, body)
		}
	}
	if err := commands[0].Run([]string{"--target= ", "--json"}); err == nil || !strings.Contains(err.Error(), "--target is required") {
		t.Fatalf("blank target error = %v", err)
	}
	if len(bodies) != 3 {
		t.Fatalf("blank target reached API; request count = %d", len(bodies))
	}
}
