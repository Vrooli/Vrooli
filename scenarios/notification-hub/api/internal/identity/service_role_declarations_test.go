package identity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/hubsender"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/api-core/serviceprincipal"
)

// hostPrograms are the programs that send to the Hub as themselves without being
// scenarios: the operator's own host-local programs. A host program has no
// .vrooli folder, so a scenario declares its principal on its behalf. The map is
// the one place that says which source belongs to which declaring scenario; any
// other source must be the declaring scenario's own name.
var hostPrograms = map[string]string{
	// The portfolio steward (~/.vrooli/portfolio/steward) raises its asks through
	// the Hub CLI from a timer. It reads them back as this sender.
	"portfolio-steward": "notification-hub",
}

// A scenario that talks to the Hub as itself declares its principal in its own
// .vrooli/service-principals.json, and the operator grants them with one command
// (serviceprincipal.IssueCommand). A declaration is plain data the Hub cannot
// import, so this test holds every declaration for a Hub role to the contract the
// Hub verifies: the resource, audience and single scope, and the name suffix the
// Hub reads the source off.
func TestEveryServiceRoleDeclarationIsExactlyWhatTheHubVerifies(t *testing.T) {
	roles := []struct {
		name     string
		role     principalrole.Role
		required []string // sources known to hold the role today
	}{
		{"ask-reader", askreader.Role, []string{"agent-manager", "data-backup-manager", "deployment-manager", "research-desk", "storage-manager", "swarm-manager", "vrooli-autoheal"}},
		// Senders: the scenarios that tell the operator when a resolution of their own
		// asks was refused (operatordecision.RefusalNotifier).
		{"sender", hubsender.Role, []string{"agent-manager", "data-backup-manager", "deployment-manager", "portfolio-steward", "storage-manager", "swarm-manager", "vrooli-autoheal"}},
	}
	scenarios := filepath.Join("..", "..", "..", "..")
	entries, err := os.ReadDir(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	declarations := map[string][]serviceprincipal.Declaration{}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(scenarios, entry.Name(), filepath.FromSlash(serviceprincipal.DeclarationFile)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		declared, err := serviceprincipal.ParseDeclarations(raw)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		declarations[entry.Name()] = declared
	}
	for _, r := range roles {
		found := map[string]bool{}
		for scenario, declared := range declarations {
			for _, d := range declared {
				source, isRole := r.role.SourceOf(d.Name)
				if !isRole && d.Resource != r.role.Resource {
					continue
				}
				found[source] = true
				switch {
				case !isRole:
					t.Errorf("%s declares %s for the %s resource, but its name does not end in %q, so the Hub would not read a source off it", scenario, d.Name, r.name, r.role.Suffix)
				case d.Resource != r.role.Resource, d.Audience != r.role.Audience, len(d.Scopes) != 1 || d.Scopes[0] != r.role.Scope:
					t.Errorf("%s declares %+v; the Hub verifies %s resource %s, audience %s and exactly scope %s", scenario, d, r.name, r.role.Resource, r.role.Audience, r.role.Scope)
				case hostPrograms[source] == scenario:
				case source != scenario && !strings.HasPrefix(source, scenario):
					t.Errorf("%s declares the %s of %s, which is another scenario's", scenario, r.name, source)
				}
			}
		}
		for _, source := range r.required {
			if !found[source] {
				t.Errorf("%s holds the %s role but declares no %s", source, r.name, r.role.Principal(source))
			}
		}
	}
}
