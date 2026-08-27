package hostreqkittest

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type goodHandler struct{}

func (goodHandler) Name() string           { return "fixture" }
func (goodHandler) Kind() hostreqspec.Kind { return hostreqspec.KindTool }

func (goodHandler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.ItemStatus{Name: "fixture", Kind: hostreqspec.KindTool, SupportClass: hostreqkit.SupportSupported}
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
	} else if host.OS != "linux" {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
	}
	return status
}

func (goodHandler) Apply(_ hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}
	switch status.SupportClass {
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		return status, nil
	}
	status.ExecutionState = hostreqkit.ExecutionWouldApply
	return status, nil
}

type mutant struct {
	rule string
}

func (m mutant) Name() string {
	if m.rule == "wrong_name" {
		return "mutant"
	}
	return goodHandler{}.Name()
}

func (m mutant) Kind() hostreqspec.Kind {
	if m.rule == "wrong_kind" {
		return hostreqspec.KindSafeguard
	}
	return goodHandler{}.Kind()
}

func (m mutant) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	if m.rule == "manual_inspect" && requirement.Manual {
		return hostreqkit.ItemStatus{Name: "fixture", Kind: hostreqspec.KindTool, SupportClass: hostreqkit.SupportSupported}
	}
	return goodHandler{}.Inspect(host, requirement)
}

func (m mutant) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch m.rule {
	case "dry_run_apply":
		if opts.DryRun && status.SupportClass == hostreqkit.SupportSupported {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			return status, nil
		}
	case "unsupported_apply":
		if status.SupportClass == hostreqkit.SupportUnsupported {
			status.Applied = true
			status.ExecutionState = hostreqkit.ExecutionApplied
			return status, nil
		}
	case "already_applied":
		if status.Applied {
			status.ExecutionState = hostreqkit.ExecutionWouldApply
			return status, nil
		}
	}
	return goodHandler{}.Apply(host, status, opts)
}

type recordingT struct{ failed bool }

func (t *recordingT) Helper() {}

func (t *recordingT) Run(_ string, f func(suiteT)) bool {
	child := &recordingT{}
	f(child)
	if child.failed {
		t.failed = true
	}
	return !child.failed
}

func (t *recordingT) Errorf(string, ...any) { t.failed = true }

func fixtureCase(rule string) Case {
	return Case{
		NewHandler: func() hostreqkit.Handler {
			if rule == "" {
				return goodHandler{}
			}
			return mutant{rule: rule}
		},
		Name:               "fixture",
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: []string{"linux"},
		InstallCommand:     "fixture install",
	}
}

func TestSuiteAcceptsGoodHandler(t *testing.T) {
	RunSuite(t, fixtureCase(""))
}

func TestSuiteRejectsEachMutant(t *testing.T) {
	for _, rule := range []string{"wrong_name", "wrong_kind", "dry_run_apply", "unsupported_apply", "manual_inspect", "already_applied"} {
		t.Run(rule, func(t *testing.T) {
			recorder := &recordingT{}
			runSuite(recorder, fixtureCase(rule))
			if !recorder.failed {
				t.Fatalf("RunSuite accepted mutant %q", rule)
			}
		})
	}
}
