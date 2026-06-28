package validation

import "strings"

// This file holds the CLI and API branding rules — the surfaces the PRD's
// "uniform across UI/CLI/API" scope covers but the original UI-only rule set
// never touched. Both are conservative detect-only nudges (info): they fire only
// on a clear signal that the surface still carries no brand identity, so they do
// not generate fleet-wide noise. CLI is surface-conditional (skips without cli/).

const cliManifestRel = "cli/manifest.json"

// ruleCLIBranding nudges (info) when a CLI manifest exists but never surfaces
// the brand display name (its name is the slug and no description/title carries
// the brand). Rewriting human-facing CLI copy is a judgment call, so there is no
// deterministic fixer.
func ruleCLIBranding(c *scanContext) (Finding, bool) {
	content, ok := c.read(cliManifestRel)
	if !ok {
		return Finding{}, false
	}
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{}, false // has-display-name owns the missing-identity case
	}
	if strings.Contains(content, id.DisplayName) {
		return Finding{}, false // the brand name appears somewhere in the CLI surface
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "CLI surface does not carry the brand display name",
		Description:            "The CLI manifest never mentions service.displayName, so help/banner output reads as the raw slug rather than the product.",
		FilePath:               cliManifestRel,
		WhyItMatters:           "The CLI is a first-class surface; its help should present the product name, not just the command slug.",
		RecommendedRemediation: "Reference the display name in the CLI manifest description (or a title) so help output is branded.",
		Evidence:               map[string]any{"display_name": id.DisplayName},
	}, true
}

// apiTemplateMarkers betray an un-rebranded API service description.
var apiTemplateMarkers = []string{"template scenario", "notes domain", "todo", "[", "lorem ipsum"}

// ruleAPIBranding nudges (info) when the service description that an API/OpenAPI
// title would carry is empty or obvious template residue. Detect-only: the
// description is human-authored, so no deterministic fixer is offered.
func ruleAPIBranding(c *scanContext) (Finding, bool) {
	if _, ok := c.read(".vrooli/service.json"); !ok {
		return Finding{}, false // has-display-name owns the no-service.json case
	}
	id := c.identity()
	desc := strings.ToLower(strings.TrimSpace(id.Description))
	residue := desc == ""
	for _, m := range apiTemplateMarkers {
		if strings.Contains(desc, m) {
			residue = true
		}
	}
	if !residue {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "API service description is missing or template residue",
		Description:            "service.description (the title an API/OpenAPI surface advertises) is empty or still scaffold text.",
		FilePath:               ".vrooli/service.json",
		WhyItMatters:           "The served service/OpenAPI title is how API consumers identify the product; template text undercuts the brand.",
		RecommendedRemediation: "Set service.description to a real one-line product description.",
		Evidence:               map[string]any{"description": id.Description},
	}, true
}
