package components

import (
	"regexp"
	"sort"
)

// behaviorInventory deliberately stays static and transparent. It is not a
// claim of runtime equivalence; it catches the high-value signals that are
// commonly lost while de-scenario-ifying a component.
type behaviorInventory struct {
	Hooks     []string
	Keyboard  []string
	ARIA      []string
	Roles     []string
	Listeners []string
}

var (
	hookUseRE = regexp.MustCompile(`\buse[A-Z][A-Za-z0-9_]*\b`)
	ariaRE    = regexp.MustCompile(`\baria-[A-Za-z0-9_-]+`)
	roleRE    = regexp.MustCompile(`\brole\s*=`)
	keyRE     = regexp.MustCompile(`\bonKey(?:Down|Up|Press)\b`)
	listenRE  = regexp.MustCompile(`\baddEventListener\b`)
)

func inventoryBehavior(source string) behaviorInventory {
	return behaviorInventory{
		Hooks: hookUseRE.FindAllString(source, -1), Keyboard: keyRE.FindAllString(source, -1),
		ARIA: ariaRE.FindAllString(source, -1), Roles: roleRE.FindAllString(source, -1),
		Listeners: listenRE.FindAllString(source, -1),
	}
}

// BehaviorLossFindings reports statically-observable behavior signals present
// in origin but absent from harvested source. It is exported for calibration
// fixtures and intentionally returns the same structured findings as ingest.
func BehaviorLossFindings(origin, harvested, sourceFile string) []IngestFinding {
	from, to := inventoryBehavior(origin), inventoryBehavior(harvested)
	var findings []IngestFinding
	for _, check := range []struct {
		kind          string
		before, after []string
	}{
		{"hook", from.Hooks, to.Hooks}, {"keyboard-handler", from.Keyboard, to.Keyboard},
		{"aria", from.ARIA, to.ARIA}, {"role", from.Roles, to.Roles}, {"event-listener", from.Listeners, to.Listeners},
	} {
		missing := missingSignals(check.before, check.after)
		for _, signal := range missing {
			findings = append(findings, IngestFinding{Code: "behavior-lost", SourceFile: sourceFile, Message: "harvest removed " + check.kind + " signal " + signal})
		}
	}
	return findings
}

func missingSignals(before, after []string) []string {
	have := map[string]bool{}
	for _, signal := range after {
		have[signal] = true
	}
	missing := map[string]bool{}
	for _, signal := range before {
		if !have[signal] {
			missing[signal] = true
		}
	}
	out := make([]string, 0, len(missing))
	for signal := range missing {
		out = append(out, signal)
	}
	sort.Strings(out)
	return out
}
