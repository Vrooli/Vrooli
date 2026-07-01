package agentpolicycli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/codingagents"
)

func TestSupportedAgentsMatchesCodingAgentCatalog(t *testing.T) {
	want := codingagents.ResourceCLIs()
	if !reflect.DeepEqual(SupportedAgents, want) {
		t.Fatalf("SupportedAgents = %v, want catalog resource CLIs %v", SupportedAgents, want)
	}
}

func TestHelpMentionsEverySupportedAgent(t *testing.T) {
	var out strings.Builder
	RenderCommandHelp(&out)
	help := out.String()
	for _, agent := range SupportedAgents {
		if !strings.Contains(help, agent) {
			t.Fatalf("help missing supported agent %q:\n%s", agent, help)
		}
	}
}
