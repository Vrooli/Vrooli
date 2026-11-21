package usagemodes

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"agent-manager/internal/decisionpage"
	"agent-manager/internal/usagelabel"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/usagemode"
)

// A usage mode decision's page. Every kind shows the same picture: which
// models new work uses today and which it would use with the mode on, the
// kinds of work the mode leaves alone, and what each answer does.

const usageModeLead = "A usage mode makes new agent work use cheaper AI models, so your plan's allowance lasts longer. Work that is already running is never changed."

// modePage is the page for a decision about mode m, or about the one-tier-down
// mode that would start on runner when m is nil.
type modePage struct {
	facts   operatordecision.Facts
	rows    []operatordecision.Row
	extra   string
	effects map[string]string
	noReply string
}

func (n *Notifier) render(ctx context.Context, p modePage, m *usagemode.Mode, runner string) string {
	page := operatordecision.NewPage(p.facts.Subject).Raw(decisionpage.Style).Lead(usageModeLead).Facts(p.rows...)
	if p.extra != "" {
		page.Raw(p.extra)
	}
	if rows := n.modelRows(ctx, m, runner); len(rows) > 0 {
		page.Section("What changes for new work", "").Table([]string{"Runs on", "Usually", "With the mode on"}, rows)
	}
	if m != nil && len(m.ExemptRoles) > 0 {
		names := make([]string, 0, len(m.ExemptRoles))
		for _, role := range m.ExemptRoles {
			names = append(names, roleName(role))
		}
		page.Section("Work the mode leaves alone", "These kinds of work keep their usual, stronger models: "+strings.Join(names, ", ")+".")
	}
	page.Options(p.facts, p.effects)
	return page.IfNoAnswer(p.noReply).HTML()
}

// modelRows lists the model swaps a mode makes: the mode's own map when it
// froze one, otherwise the runner's current one-tier-down map from the
// resource policy, which is what a save mode applies.
func (n *Notifier) modelRows(ctx context.Context, m *usagemode.Mode, runner string) [][]string {
	runners := map[string]usagemode.RunnerSettings{}
	if m != nil {
		for name, settings := range m.Runners {
			runners[name] = settings
		}
	} else if runner != "" {
		runners[runner] = usagemode.RunnerSettings{}
	}
	names := make([]string, 0, len(runners))
	for name := range runners {
		names = append(names, name)
	}
	sort.Strings(names)
	var rows [][]string
	for _, name := range names {
		settings := runners[name]
		swaps := settings.ModelMap
		if len(swaps) == 0 && settings.OverrideModel != "" {
			rows = append(rows, []string{runnerTitle(name), "Every model it lowers", settings.OverrideModel})
			continue
		}
		if len(swaps) == 0 {
			swaps = n.tierMap(ctx, name)
		}
		from := make([]string, 0, len(swaps))
		for model := range swaps {
			if !strings.Contains(model, "-20") { // skip dated full ids that repeat an alias
				from = append(from, model)
			}
		}
		sort.Strings(from)
		for _, model := range from {
			rows = append(rows, []string{runnerTitle(name), model, swaps[model]})
		}
	}
	return rows
}

// tierMap is the runner's one-tier-down map from the resource policy, or nil
// when the policy cannot be read; the page then just leaves the table out.
func (n *Notifier) tierMap(ctx context.Context, runner string) map[string]string {
	if n.Modes == nil || n.Modes.Tiers == nil {
		return nil
	}
	tiers, err := n.Modes.Tiers.Tiers(ctx, resourceRunner(runner))
	if err != nil {
		return nil
	}
	settings, err := tiers.Settings()
	if err != nil {
		return nil
	}
	return settings.ModelMap
}

// roleName turns a role key such as "code.review" into "code review".
func roleName(role string) string {
	return strings.ToLower(decisionpage.Name(role))
}

// allowanceName is "your Claude 5-hour allowance".
func allowanceName(provider, window string, minutes int64) string {
	return fmt.Sprintf("your %s %s allowance", usagelabel.Provider(provider), usagelabel.Window(minutes, window))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
