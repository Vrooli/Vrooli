package prompts

import (
	"swarm-manager/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "prompts",
		Description: "Prompt catalog and skill operations",
		Subcommands: []cliapp.Command{
			support.APICommand("catalog", "List the swarm-manager prompt catalog", deps.PromptsCatalog),
			support.APICommand("skills", "List prompt skills used by swarm-manager", deps.PromptsSkills),
			support.APICommand("skill-get", "Get prompt skill details (--id ID)", deps.PromptsSkillGet),
			support.APICommand("skill-update", "Update prompt skill fields (--id ID --data JSON)", deps.PromptsSkillUpdate),
			support.APICommand("skill-versions", "Get prompt skill version history (--id ID)", deps.PromptsSkillVersions),
			support.APICommand("skill-revert", "Revert prompt skill to version (--id ID --version VERSION)", deps.PromptsSkillRevert),
			support.APICommand("preview", "Render a skill prompt with variables (--id ID)", deps.PromptsPreview),
			support.APICommand("experiment-results", "Show experiment results (--id EID or positional) [--json]", deps.PromptsExperiment),
		},
	}
}
