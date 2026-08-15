package glossary

import (
	"net/http"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
	"vrooli-onboarding/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterAndRows(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	if len(group.Commands) != 1 || group.Commands[0].Name != "glossary" {
		t.Fatalf("unexpected glossary group: %+v", group)
	}
	if len(rows(nil)) != 1 || len(rows([]support.GlossaryEntry{{Term: "x", Category: "core", Description: "y"}})) != 1 {
		t.Fatal("glossary rows failed")
	}
}

func TestCommandQueriesAndPrintsGlossary(t *testing.T) {
	core := clitest.NewTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/glossary" || r.URL.Query().Get("q") != "postgres" {
			t.Fatalf("unexpected glossary request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"count":1,"query":"postgres","entries":[{"term":"Postgres","category":"resource","description":"database"}]}`))
	}))
	command := Register(core).Commands[0]
	if err := command.Run([]string{"--query", "postgres", "--json"}); err != nil {
		t.Fatal(err)
	}
}
