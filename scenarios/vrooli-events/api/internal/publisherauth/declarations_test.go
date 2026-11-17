package publisherauth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/eventpublisher"
	"github.com/vrooli/api-core/serviceprincipal"
)

// Every scenario that publishes events as itself declares a
// "<source>-event-publisher" principal in its own .vrooli/service-principals.json,
// and the operator grants them all with one command. A declaration is plain data
// vrooli-events cannot import, so this test holds each one to the contract the bus
// verifies: the resource, audience and single scope, and a name that is this
// scenario's own, so a scenario cannot ask for another's identity.
func TestEveryEventPublisherDeclarationIsExactlyWhatTheBusVerifies(t *testing.T) {
	scenarios := filepath.Join("..", "..", "..", "..")
	entries, err := os.ReadDir(scenarios)
	if err != nil {
		t.Fatal(err)
	}
	declared := map[string]bool{}
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(scenarios, entry.Name(), filepath.FromSlash(serviceprincipal.DeclarationFile)))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		declarations, err := serviceprincipal.ParseDeclarations(raw)
		if err != nil {
			t.Fatalf("%s: %v", entry.Name(), err)
		}
		for _, d := range declarations {
			source, isPublisher := eventpublisher.SourceOf(d.Name)
			if !isPublisher && d.Resource != eventpublisher.Resource {
				continue
			}
			switch {
			case !isPublisher:
				t.Errorf("%s declares %s for the event-publisher resource, but its name does not end in %q, so the bus would read no source off it", entry.Name(), d.Name, eventpublisher.Suffix)
			case d.Resource != eventpublisher.Resource, d.Audience != eventpublisher.Audience, len(d.Scopes) != 1 || d.Scopes[0] != eventpublisher.Scope:
				t.Errorf("%s declares %+v; the bus verifies resource %s, audience %s and exactly scope %s", entry.Name(), d, eventpublisher.Resource, eventpublisher.Audience, eventpublisher.Scope)
			case source != entry.Name():
				t.Errorf("%s declares the event publisher of %s, which is another scenario's", entry.Name(), source)
			default:
				declared[source] = true
			}
		}
	}
	// The scenarios that publish operator-facing events (decision requests,
	// withdrawals, answers, notices) through the shared eventbus client or the
	// raw publishers of prompt-manager.
	for _, source := range []string{
		"agent-manager", "data-backup-manager", "deployment-manager", "notification-hub", "prompt-manager", "research-desk",
		"scenario-to-cloud", "storage-manager", "swarm-manager", "system-monitor", "vrooli-autoheal", "web-console",
	} {
		if !declared[source] {
			t.Errorf("%s publishes events but declares no %s", source, eventpublisher.Principal(source))
		}
	}
}
