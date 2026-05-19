package lint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/lint/golang"
	"test-genie/internal/lint/nodejs"
	"test-genie/internal/lint/python"
)

type matchResult struct {
	Matched  bool
	Evidence []string
}

type handler interface {
	ID() string
	Match(component Component) matchResult
	Run(ctx context.Context, component Component) ComponentResult
}

type handlerRegistry struct {
	handlers map[string]handler
	order    []string
}

func newHandlerRegistry(config Config) *handlerRegistry {
	registry := &handlerRegistry{
		handlers: map[string]handler{},
		order:    []string{HandlerGoModule, HandlerNodePackage, HandlerPythonProject},
	}
	registry.register(goModuleHandler{config: config})
	registry.register(nodePackageHandler{config: config})
	registry.register(pythonProjectHandler{config: config})
	return registry
}

func (r *handlerRegistry) register(h handler) {
	r.handlers[h.ID()] = h
}

func (r *handlerRegistry) enabledHandlers(settings *Settings) []handler {
	out := make([]handler, 0, len(r.order))
	for _, id := range r.order {
		cfg, ok := settings.Handlers[id]
		if ok && !cfg.EnabledOrDefault() {
			continue
		}
		if h, exists := r.handlers[id]; exists {
			out = append(out, h)
		}
	}
	return out
}

func (r *handlerRegistry) get(id string) (handler, bool) {
	h, ok := r.handlers[id]
	return h, ok
}

type goModuleHandler struct{ config Config }

func (h goModuleHandler) ID() string { return HandlerGoModule }

func (h goModuleHandler) Match(component Component) matchResult {
	if fileExists(filepath.Join(component.AbsolutePath, "go.mod")) {
		return matchResult{Matched: true, Evidence: []string{"go.mod"}}
	}
	return matchResult{}
}

func (h goModuleHandler) Run(ctx context.Context, component Component) ComponentResult {
	linter := golang.New(golang.Config{
		Dir:           component.AbsolutePath,
		CommandLookup: h.config.CommandLookup,
		Runner:        h.config.CommandRunner,
	})
	result := linter.Lint(ctx)
	return componentResultFromGo(component, result)
}

type nodePackageHandler struct{ config Config }

func (h nodePackageHandler) ID() string { return HandlerNodePackage }

func (h nodePackageHandler) Match(component Component) matchResult {
	if fileExists(filepath.Join(component.AbsolutePath, "package.json")) {
		return matchResult{Matched: true, Evidence: []string{"package.json"}}
	}
	return matchResult{}
}

func (h nodePackageHandler) Run(ctx context.Context, component Component) ComponentResult {
	linter := nodejs.New(nodejs.Config{
		Dir:           component.AbsolutePath,
		CommandLookup: h.config.CommandLookup,
		Runner:        h.config.CommandRunner,
	})
	result := linter.Lint(ctx)
	return componentResultFromNode(component, result)
}

type pythonProjectHandler struct{ config Config }

func (h pythonProjectHandler) ID() string { return HandlerPythonProject }

func (h pythonProjectHandler) Match(component Component) matchResult {
	for _, indicator := range []string{"pyproject.toml", "setup.py", "requirements.txt", "pytest.ini", "mypy.ini"} {
		if fileExists(filepath.Join(component.AbsolutePath, indicator)) {
			return matchResult{Matched: true, Evidence: []string{indicator}}
		}
	}
	if component.CodeBearing {
		for _, evidence := range component.CodeEvidence {
			if evidence == "*.py" || filepath.Ext(evidence) == ".py" || evidence == "pyproject.toml" {
				return matchResult{Matched: true, Evidence: []string{evidence}}
			}
		}
	}
	return matchResult{}
}

func (h pythonProjectHandler) Run(ctx context.Context, component Component) ComponentResult {
	linter := python.New(python.Config{
		Dir:           component.AbsolutePath,
		CommandLookup: h.config.CommandLookup,
		Runner:        h.config.CommandRunner,
	})
	result := linter.Lint(ctx)
	return componentResultFromPython(component, result)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func componentResultFromGo(component Component, result *golang.Result) ComponentResult {
	return ComponentResult{
		Component:    component,
		Matched:      true,
		Success:      result.Success,
		Issues:       convertGoIssues(result.Issues),
		TypeErrors:   result.TypeErrors,
		LintWarnings: result.LintWarnings,
		ToolsUsed:    append([]string(nil), result.ToolsUsed...),
		Skipped:      result.Skipped,
		SkipReason:   result.SkipReason,
		Observations: append([]Observation(nil), result.Observations...),
	}
}

func componentResultFromNode(component Component, result *nodejs.Result) ComponentResult {
	return ComponentResult{
		Component:    component,
		Matched:      true,
		Success:      result.Success,
		Issues:       convertNodeIssues(result.Issues),
		TypeErrors:   result.TypeErrors,
		LintWarnings: result.LintWarnings,
		ToolsUsed:    append([]string(nil), result.ToolsUsed...),
		Skipped:      result.Skipped,
		SkipReason:   result.SkipReason,
		Observations: append([]Observation(nil), result.Observations...),
	}
}

func componentResultFromPython(component Component, result *python.Result) ComponentResult {
	return ComponentResult{
		Component:    component,
		Matched:      true,
		Success:      result.Success,
		Issues:       convertPythonIssues(result.Issues),
		TypeErrors:   result.TypeErrors,
		LintWarnings: result.LintWarnings,
		ToolsUsed:    append([]string(nil), result.ToolsUsed...),
		Skipped:      result.Skipped,
		SkipReason:   result.SkipReason,
		Observations: append([]Observation(nil), result.Observations...),
	}
}

func convertGoIssues(issues []golang.Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, Issue{
			File:     issue.File,
			Line:     issue.Line,
			Column:   issue.Column,
			Message:  issue.Message,
			Severity: Severity(issue.Severity),
			Rule:     issue.Rule,
			Source:   issue.Source,
		})
	}
	return out
}

func convertNodeIssues(issues []nodejs.Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, Issue{
			File:     issue.File,
			Line:     issue.Line,
			Column:   issue.Column,
			Message:  issue.Message,
			Severity: Severity(issue.Severity),
			Rule:     issue.Rule,
			Source:   issue.Source,
		})
	}
	return out
}

func convertPythonIssues(issues []python.Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		out = append(out, Issue{
			File:     issue.File,
			Line:     issue.Line,
			Column:   issue.Column,
			Message:  issue.Message,
			Severity: Severity(issue.Severity),
			Rule:     issue.Rule,
			Source:   issue.Source,
		})
	}
	return out
}

func resolveHandler(component Component, settings *Settings, registry *handlerRegistry) (handler, []string, error) {
	override, hasOverride := settings.Components[component.Name]
	if hasOverride && override.Handler != "" {
		h, ok := registry.get(override.Handler)
		if !ok {
			return nil, nil, fmt.Errorf("component %s references unknown lint handler %q", component.Name, override.Handler)
		}
		match := h.Match(component)
		if !match.Matched {
			return nil, nil, fmt.Errorf("component %s forces handler %q but does not match its lint contract", component.Name, override.Handler)
		}
		return h, match.Evidence, nil
	}

	var matched []handler
	var evidence []string
	for _, h := range registry.enabledHandlers(settings) {
		match := h.Match(component)
		if !match.Matched {
			continue
		}
		matched = append(matched, h)
		evidence = append(evidence, match.Evidence...)
	}

	switch len(matched) {
	case 0:
		return nil, nil, nil
	case 1:
		return matched[0], evidence, nil
	default:
		ids := make([]string, 0, len(matched))
		for _, h := range matched {
			ids = append(ids, h.ID())
		}
		return nil, nil, fmt.Errorf("component %s matches multiple lint handlers: %s", component.Name, strings.Join(ids, ", "))
	}
}
