package domains

import (
	"ai-chatbot-manager/cli/domains/abtests"
	"ai-chatbot-manager/cli/domains/analytics"
	"ai-chatbot-manager/cli/domains/chat"
	"ai-chatbot-manager/cli/domains/chatbots"
	"ai-chatbot-manager/cli/domains/crm"
	"ai-chatbot-manager/cli/domains/escalations"
	"ai-chatbot-manager/cli/domains/tenants"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. Single-verb read-only surfaces
// live here (chat message send, analytics summary).
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		chat.Register(core),
		analytics.Register(core),
	}
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		chatbots.Register(core),
		escalations.Register(core),
		tenants.Register(core),
		abtests.Register(core),
		crm.Register(core),
	}
}
