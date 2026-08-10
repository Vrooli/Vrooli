// Package applicability decides whether a known Test Genie phase should judge
// a target scenario. It is intentionally pure: callers provide the already
// discovered target facts, and evaluation performs no provider, network, or
// filesystem work.
package applicability

import (
	"fmt"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/providerdescriptor"
)

type Status string

const (
	StatusApplies       Status = "applies"
	StatusNotApplicable Status = "not_applicable"
	StatusInvalid       Status = "invalid"
	StatusUnknown       Status = "unknown"
)

const (
	CodeAnyPredicateMatched  = "applicability.any_matched"
	CodeAllPredicatesMatch   = "applicability.all_matched"
	CodeDefaultApplies       = "applicability.default_applies"
	CodeDefaultNotApplicable = "applicability.default_not_applicable"
	CodeDefaultUnknown       = "applicability.default_unknown"
	CodeInvalidDefault       = "applicability.invalid_default"
	CodeInvalidPredicate     = "applicability.invalid_predicate"
)

// Context is the target-scenario fact set used for declarative applicability.
// The maps should be precomputed by the orchestrator workspace layer.
type Context struct {
	HostOS                string
	TargetKind            string
	TargetID              string
	TargetRoot            string
	ScenarioName          string
	ScenarioDir           string
	HasUI                 bool
	HasAPI                bool
	Files                 map[string]bool
	PathGlobs             map[string][]string
	ScenarioDependencies  map[string]DependencyStatus
	ServiceCapabilities   map[string]bool
	ServiceTags           map[string]bool
	TestingConfigSections map[string]bool
}

type DependencyStatus string

const (
	DependencyAbsent   DependencyStatus = "absent"
	DependencyDisabled DependencyStatus = "disabled"
	DependencyPresent  DependencyStatus = "present"
)

type Reason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Phase   string   `json:"phase,omitempty"`
	Status  Status   `json:"status"`
	Reasons []Reason `json:"reasons,omitempty"`
}

func Evaluate(phase string, declaration providerdescriptor.Applicability, ctx Context) Result {
	anyReasons, anyMatched, invalid := evaluateAny(declaration.Any, ctx)
	if invalid != nil {
		return invalidResult(phase, *invalid)
	}
	if anyMatched {
		return Result{
			Phase:   phase,
			Status:  StatusApplies,
			Reasons: append([]Reason{{Code: CodeAnyPredicateMatched, Message: "at least one applicability predicate matched"}}, anyReasons...),
		}
	}

	allReasons, allMatched, invalid := evaluateAll(declaration.All, ctx)
	if invalid != nil {
		return invalidResult(phase, *invalid)
	}
	if allMatched {
		return Result{
			Phase:   phase,
			Status:  StatusApplies,
			Reasons: append([]Reason{{Code: CodeAllPredicatesMatch, Message: "all applicability predicates matched"}}, allReasons...),
		}
	}

	result := defaultResult(phase, declaration.Default)
	result.Reasons = append(result.Reasons, anyReasons...)
	result.Reasons = append(result.Reasons, allReasons...)
	return result
}

func evaluateAny(predicates []providerdescriptor.Predicate, ctx Context) ([]Reason, bool, *Reason) {
	var reasons []Reason
	for i, predicate := range predicates {
		reason, matched, invalid := evaluatePredicate(predicate, ctx)
		if invalid != nil {
			invalid.Message = fmt.Sprintf("any predicate %d is invalid: %s", i, invalid.Message)
			return reasons, false, invalid
		}
		reasons = append(reasons, reason)
		if matched {
			return reasons, true, nil
		}
	}
	return reasons, false, nil
}

func evaluateAll(predicates []providerdescriptor.Predicate, ctx Context) ([]Reason, bool, *Reason) {
	if len(predicates) == 0 {
		return nil, false, nil
	}
	var reasons []Reason
	for i, predicate := range predicates {
		reason, matched, invalid := evaluatePredicate(predicate, ctx)
		if invalid != nil {
			invalid.Message = fmt.Sprintf("all predicate %d is invalid: %s", i, invalid.Message)
			return reasons, false, invalid
		}
		reasons = append(reasons, reason)
		if !matched {
			return reasons, false, nil
		}
	}
	return reasons, true, nil
}

func evaluatePredicate(predicate providerdescriptor.Predicate, ctx Context) (Reason, bool, *Reason) {
	if countPredicateFields(predicate) != 1 {
		return Reason{}, false, &Reason{Code: CodeInvalidPredicate, Message: "predicate must set exactly one supported field"}
	}
	switch {
	case strings.TrimSpace(predicate.HostOS) != "":
		want := strings.ToLower(strings.TrimSpace(predicate.HostOS))
		got := strings.ToLower(strings.TrimSpace(ctx.HostOS))
		if got == want {
			return Reason{Code: "applicability.host_os_matched", Message: fmt.Sprintf("host OS matched %s", want)}, true, nil
		}
		return Reason{Code: "applicability.host_os_mismatched", Message: fmt.Sprintf("host OS %s did not match %s", got, want)}, false, nil
	case strings.TrimSpace(predicate.TargetKind) != "":
		kind := strings.TrimSpace(predicate.TargetKind)
		if ctx.TargetKind == kind {
			return Reason{Code: "applicability.target_kind_matched", Message: fmt.Sprintf("target kind matched %s", kind)}, true, nil
		}
		return Reason{Code: "applicability.target_kind_mismatched", Message: fmt.Sprintf("target kind %s did not match %s", ctx.TargetKind, kind)}, false, nil
	case strings.TrimSpace(predicate.FileExists) != "":
		path := normalizePath(predicate.FileExists)
		if ctx.hasFile(path) {
			return Reason{Code: "applicability.file_exists", Message: fmt.Sprintf("target file %s exists", path)}, true, nil
		}
		return Reason{Code: "applicability.file_missing", Message: fmt.Sprintf("target file %s is absent", path)}, false, nil
	case strings.TrimSpace(predicate.PathGlob) != "":
		glob := strings.TrimSpace(predicate.PathGlob)
		if matches := ctx.PathGlobs[glob]; len(matches) > 0 {
			return Reason{Code: "applicability.path_glob_matched", Message: fmt.Sprintf("target path glob %s matched %s", glob, strings.Join(matches, ", "))}, true, nil
		}
		return Reason{Code: "applicability.path_glob_unmatched", Message: fmt.Sprintf("target path glob %s matched no files", glob)}, false, nil
	case strings.TrimSpace(predicate.ScenarioDependency) != "":
		dependency := normalizeKey(predicate.ScenarioDependency)
		switch ctx.dependencyStatus(dependency) {
		case DependencyPresent:
			return Reason{Code: "applicability.scenario_dependency_present", Message: fmt.Sprintf("target has enabled scenario dependency %s", dependency)}, true, nil
		case DependencyDisabled:
			return Reason{Code: "applicability.scenario_dependency_disabled", Message: fmt.Sprintf("target has disabled scenario dependency %s", dependency)}, false, nil
		default:
			return Reason{Code: "applicability.scenario_dependency_absent", Message: fmt.Sprintf("target lacks scenario dependency %s", dependency)}, false, nil
		}
	case strings.TrimSpace(predicate.ServiceCapability) != "":
		capability := normalizeKey(predicate.ServiceCapability)
		if ctx.hasCapability(capability) {
			return Reason{Code: "applicability.service_capability_present", Message: fmt.Sprintf("target declares service capability %s", capability)}, true, nil
		}
		return Reason{Code: "applicability.service_capability_absent", Message: fmt.Sprintf("target does not declare service capability %s", capability)}, false, nil
	case strings.TrimSpace(predicate.ServiceTag) != "":
		tag := normalizeKey(predicate.ServiceTag)
		if hasNormalizedKey(ctx.ServiceTags, tag) {
			return Reason{Code: "applicability.service_tag_present", Message: fmt.Sprintf("target declares service tag %s", tag)}, true, nil
		}
		return Reason{Code: "applicability.service_tag_absent", Message: fmt.Sprintf("target does not declare service tag %s", tag)}, false, nil
	case predicate.HasUI != nil:
		if ctx.HasUI == *predicate.HasUI {
			return Reason{Code: "applicability.has_ui_matched", Message: fmt.Sprintf("target UI availability matched %t", *predicate.HasUI)}, true, nil
		}
		return Reason{Code: "applicability.has_ui_mismatched", Message: fmt.Sprintf("target UI availability did not match %t", *predicate.HasUI)}, false, nil
	case predicate.HasAPI != nil:
		if ctx.HasAPI == *predicate.HasAPI {
			return Reason{Code: "applicability.has_api_matched", Message: fmt.Sprintf("target API availability matched %t", *predicate.HasAPI)}, true, nil
		}
		return Reason{Code: "applicability.has_api_mismatched", Message: fmt.Sprintf("target API availability did not match %t", *predicate.HasAPI)}, false, nil
	case strings.TrimSpace(predicate.TestingConfigSection) != "":
		section := normalizeKey(predicate.TestingConfigSection)
		if ctx.hasTestingConfigSection(section) {
			return Reason{Code: "applicability.testing_config_section_present", Message: fmt.Sprintf("target testing config has section %s", section)}, true, nil
		}
		return Reason{Code: "applicability.testing_config_section_absent", Message: fmt.Sprintf("target testing config lacks section %s", section)}, false, nil
	}
	return Reason{}, false, &Reason{Code: CodeInvalidPredicate, Message: "predicate must set exactly one supported field"}
}

func defaultResult(phase string, value string) Result {
	switch Status(strings.TrimSpace(value)) {
	case StatusApplies:
		return Result{Phase: phase, Status: StatusApplies, Reasons: []Reason{{Code: CodeDefaultApplies, Message: "applicability default is applies"}}}
	case StatusNotApplicable:
		return Result{Phase: phase, Status: StatusNotApplicable, Reasons: []Reason{{Code: CodeDefaultNotApplicable, Message: "applicability default is not_applicable"}}}
	case StatusUnknown:
		return Result{Phase: phase, Status: StatusUnknown, Reasons: []Reason{{Code: CodeDefaultUnknown, Message: "applicability default is unknown"}}}
	default:
		return invalidResult(phase, Reason{Code: CodeInvalidDefault, Message: "applicability default must be applies, not_applicable, or unknown"})
	}
}

func invalidResult(phase string, reason Reason) Result {
	return Result{Phase: phase, Status: StatusInvalid, Reasons: []Reason{reason}}
}

func (ctx Context) hasFile(path string) bool {
	if ctx.Files == nil {
		return false
	}
	if ctx.Files[path] {
		return true
	}
	if ctx.Files[filepath.ToSlash(filepath.Join(ctx.ScenarioDir, path))] {
		return true
	}
	return ctx.Files[strings.TrimPrefix(path, "./")]
}

func (ctx Context) hasCapability(capability string) bool {
	return hasNormalizedKey(ctx.ServiceCapabilities, capability)
}

func (ctx Context) dependencyStatus(dependency string) DependencyStatus {
	if ctx.ScenarioDependencies == nil {
		return DependencyAbsent
	}
	if status, ok := ctx.ScenarioDependencies[dependency]; ok {
		return status
	}
	for name, status := range ctx.ScenarioDependencies {
		if normalizeKey(name) == dependency {
			return status
		}
	}
	return DependencyAbsent
}

func (ctx Context) hasTestingConfigSection(section string) bool {
	return hasNormalizedKey(ctx.TestingConfigSections, section)
}

func hasNormalizedKey(values map[string]bool, key string) bool {
	if values == nil {
		return false
	}
	if values[key] {
		return true
	}
	for candidate, enabled := range values {
		if enabled && normalizeKey(candidate) == key {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func countPredicateFields(predicate providerdescriptor.Predicate) int {
	count := 0
	if strings.TrimSpace(predicate.HostOS) != "" {
		count++
	}
	if strings.TrimSpace(predicate.TargetKind) != "" {
		count++
	}
	if strings.TrimSpace(predicate.FileExists) != "" {
		count++
	}
	if strings.TrimSpace(predicate.PathGlob) != "" {
		count++
	}
	if strings.TrimSpace(predicate.ScenarioDependency) != "" {
		count++
	}
	if strings.TrimSpace(predicate.ServiceCapability) != "" {
		count++
	}
	if strings.TrimSpace(predicate.ServiceTag) != "" {
		count++
	}
	if predicate.HasUI != nil {
		count++
	}
	if predicate.HasAPI != nil {
		count++
	}
	if strings.TrimSpace(predicate.TestingConfigSection) != "" {
		count++
	}
	return count
}
