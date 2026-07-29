package renders

import (
	"net/http"
	"testing"

	"asset-studio/cli/internal/testutil"
)

func TestRegisterIncludesRenderInspection(t *testing.T) {
	app := testutil.NewTestApp(t, http.NewServeMux())
	group := Register(app)
	if group.Name != "renders" || len(group.Subcommands) != 4 || group.Subcommands[0].Name != "show" || group.Subcommands[1].Name != "set-campaign-budget" || group.Subcommands[2].Name != "regenerate" || group.Subcommands[3].Name != "analyze-conformance" {
		t.Fatalf("render command group = %#v", group)
	}
}
