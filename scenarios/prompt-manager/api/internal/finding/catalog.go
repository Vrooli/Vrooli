package finding

import (
	"fmt"
	"sort"
	"strings"
)

// RuleGroup names the family a rule belongs to. It is a plain string so that
// each family can declare its own groups without this leaf package having to
// know them.
type RuleGroup string

// RuleCatalogEntry is the operator-facing identity for a validation rule. A
// rule without this metadata is not actionable enough to be registered.
type RuleCatalogEntry struct {
	ID          string
	Group       RuleGroup
	Severity    Severity
	Kind        Kind
	Description string
	Actuator    string
}

type RuleCatalog map[string]RuleCatalogEntry

// NewRuleCatalog rejects an entry missing any operator-facing field, and any
// duplicate identifier. Kind defaults to declaration: the overwhelming majority
// of rules read checked-in files, and a rule that silently defaulted to runtime
// would quietly leave the gate.
func NewRuleCatalog(entries ...RuleCatalogEntry) (RuleCatalog, error) {
	catalog := make(RuleCatalog, len(entries))
	for _, entry := range entries {
		if entry.ID == "" || entry.Group == "" || entry.Severity == "" || entry.Description == "" || entry.Actuator == "" {
			return nil, fmt.Errorf("rule catalog entry must include id, group, severity, description, and actuator")
		}
		if entry.Kind == "" {
			entry.Kind = KindDeclaration
		}
		if entry.Kind != KindDeclaration && entry.Kind != KindRuntime {
			return nil, fmt.Errorf("rule catalog entry %q has unknown kind %q", entry.ID, entry.Kind)
		}
		if _, exists := catalog[entry.ID]; exists {
			return nil, fmt.Errorf("rule catalog contains duplicate identifier %q", entry.ID)
		}
		catalog[entry.ID] = entry
	}
	return catalog, nil
}

// IDs returns every catalogued rule id in a stable order.
func (c RuleCatalog) IDs() []string {
	ids := make([]string, 0, len(c))
	for id := range c {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Markdown renders the operator-facing rule reference. It is deterministic so
// documentation generation fails on semantic catalog drift rather than
// producing noisy reorder-only diffs.
func (c RuleCatalog) Markdown() string {
	var b strings.Builder
	b.WriteString("| Rule | Group | Default severity | Kind | Description | Actuator |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	for _, id := range c.IDs() {
		entry := c[id]
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` | %s | %s |\n",
			entry.ID, entry.Group, entry.Severity, entry.Kind, entry.Description, entry.Actuator)
	}
	return b.String()
}
