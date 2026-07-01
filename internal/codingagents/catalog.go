package codingagents

// Agent describes a coding-agent CLI resource known to the platform.
type Agent struct {
	Name        string
	ResourceCLI string
}

// Catalog is the single source of truth for coding-agent resources that expose
// the common resource CLI control surfaces.
var Catalog = []Agent{
	{Name: "claude-code", ResourceCLI: "resource-claude-code"},
	{Name: "codex", ResourceCLI: "resource-codex"},
	{Name: "opencode", ResourceCLI: "resource-opencode"},
	{Name: "grok", ResourceCLI: "resource-grok"},
}

// ResourceCLIs returns the resource CLI binary names in catalog order.
func ResourceCLIs() []string {
	out := make([]string, 0, len(Catalog))
	for _, agent := range Catalog {
		out = append(out, agent.ResourceCLI)
	}
	return out
}

// Names returns the resource names in catalog order.
func Names() []string {
	out := make([]string, 0, len(Catalog))
	for _, agent := range Catalog {
		out = append(out, agent.Name)
	}
	return out
}
