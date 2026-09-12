package domains

import (
	"seo-optimizer/cli/domains/seo"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups. The SEO Optimizer API is a
// small surface of four unrelated mutating endpoints (audit, keywords, content,
// competitors); exposing them as flat verbs keeps invocation direct
// (`seo-optimizer audit <url>`) and avoids a redundant intermediate noun.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		seo.Register(core),
	}
}

// SubcommandGroups is intentionally empty; no scenario-specific hierarchical
// groups exist yet. cli-core's built-in `status` command already covers the
// root `/health` probe.
func SubcommandGroups(core *cliapp.ScenarioApp) []cliapp.SubcommandGroup {
	_ = core
	return nil
}
