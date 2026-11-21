package templateengine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func runTemplateList[C any](deps HandlerDeps[C], ctx C, _ templatecontracts.TemplateListRequest) ([]templatecontracts.TemplateInfo, error) {
	return loadTemplates(deps.Root(ctx))
}

func runTemplateShow[C any](deps HandlerDeps[C], ctx C, req templatecontracts.TemplateShowRequest) (templatecontracts.TemplateInfo, error) {
	return loadTemplate(deps.Root(ctx), req.Name)
}

func runGenerate[C any](deps HandlerDeps[C], ctx C, req templatecontracts.GenerateRequest) (templatecontracts.GenerateResult, error) {
	info := req.TemplateInfo
	opts := req.Options
	prepared, err := prepareGenerate(deps.Root(ctx), info, opts)
	if err != nil {
		return templatecontracts.GenerateResult{}, err
	}
	if opts.DryRun {
		return prepared.result(opts.RunHooks, true), nil
	}
	if err := preflightGenerate(prepared, opts.Force); err != nil {
		return templatecontracts.GenerateResult{}, err
	}
	if err := writeGeneratedScenario(deps, ctx, prepared); err != nil {
		return templatecontracts.GenerateResult{}, err
	}
	if err := validateGeneratedScenarioResult(deps, ctx, prepared); err != nil {
		return templatecontracts.GenerateResult{}, err
	}
	result := prepared.result(opts.RunHooks, false)
	if opts.RunHooks {
		if err := runTemplateHooks(deps, ctx, prepared.destination, info.Manifest, prepared.values, deps.Stdout(ctx)); err != nil {
			return templatecontracts.GenerateResult{}, err
		}
	}
	return result, nil
}

type preparedGenerate struct {
	info        templatecontracts.TemplateInfo
	destination string
	values      map[string]string
	relocations []templatecontracts.ResolvedRelocation
	design      templatecontracts.ResolvedDesign
	provenance  templatecontracts.GenerationProvenance
}

func prepareGenerate(root string, info templatecontracts.TemplateInfo, opts templatecontracts.GenerateOptions) (preparedGenerate, error) {
	destination := cleanGenerateDestination(root, opts)
	values, err := buildTemplateValues(root, destination, info.Name, info.Manifest, opts.Values)
	if err != nil {
		return preparedGenerate{}, err
	}
	resolved, err := resolveRelocations(root, info, values)
	if err != nil {
		return preparedGenerate{}, err
	}
	design, err := resolveDesign(root, info, opts.Design, destination, values)
	if err != nil {
		return preparedGenerate{}, err
	}
	return preparedGenerate{
		info:        info,
		destination: destination,
		values:      values,
		relocations: resolved,
		design:      design,
		provenance:  buildGenerationProvenance(info, design, time.Now().UTC()),
	}, nil
}

func cleanGenerateDestination(root string, opts templatecontracts.GenerateOptions) string {
	destination := opts.Destination
	if destination == "" {
		destination = filepath.Join(root, "scenarios", opts.Values["SCENARIO_ID"])
	}
	if !filepath.IsAbs(destination) {
		destination = filepath.Join(root, filepath.FromSlash(destination))
	}
	return filepath.Clean(destination)
}

func (prepared preparedGenerate) result(runHooks, dryRun bool) templatecontracts.GenerateResult {
	return templatecontracts.GenerateResult{
		TemplateName: prepared.info.Name,
		DisplayName:  coalesce(prepared.values["SCENARIO_DISPLAY_NAME"], prepared.values["SCENARIO_ID"]),
		Destination:  prepared.destination,
		Values:       prepared.values,
		Manifest:     prepared.info.Manifest,
		Design:       prepared.design,
		Relocations:  prepared.relocations,
		Provenance:   prepared.provenance,
		RunHooks:     runHooks,
		DryRun:       dryRun,
	}
}

func preflightGenerate(prepared preparedGenerate, force bool) error {
	if err := removeExistingGenerateTarget(prepared.destination, "destination", force); err != nil {
		return err
	}
	for _, reloc := range prepared.relocations {
		if err := removeExistingGenerateTarget(reloc.To, "relocation target", force); err != nil {
			return err
		}
	}
	if err := preflightDesignCopies(prepared.design, force); err != nil {
		return err
	}
	return preflightDesignTemplateCollisions(prepared.info.Path, prepared.destination, prepared.design)
}

func removeExistingGenerateTarget(path, label string, force bool) error {
	if stat, err := os.Stat(path); err == nil && stat != nil {
		if !force {
			return fmt.Errorf("%s already exists: %s (use --force to overwrite)", label, path)
		}
		return os.RemoveAll(path)
	}
	return nil
}

func writeGeneratedScenario[C any](deps HandlerDeps[C], ctx C, prepared preparedGenerate) error {
	if err := copyTemplate(prepared.info.Path, prepared.destination, prepared.values, prepared.info.Manifest); err != nil {
		return err
	}
	if err := copyDesignAssets(prepared.design, prepared.values); err != nil {
		return err
	}
	if err := injectScenarioProvenance(prepared.destination, prepared.provenance); err != nil {
		return err
	}
	if err := renderOrientationManifest(prepared.destination, prepared.info.Manifest, prepared.values); err != nil {
		return err
	}
	if err := verifyTemplate(prepared.destination); err != nil {
		return err
	}
	return runRelocations(deps, ctx, deps.Root(ctx), deps.CommandEnv(ctx), prepared.relocations, prepared.values, deps.Stdout(ctx))
}

func validateGeneratedScenarioResult[C any](deps HandlerDeps[C], ctx C, prepared preparedGenerate) error {
	issues := validateGeneratedScenario(prepared.destination, deps.RunSubprocess != nil, func(spec shell.Spec) error {
		var err error
		spec.Env, err = templateHookEnv(deps.CommandEnv(ctx), map[string]string{"GOWORK": "off"})
		if err != nil {
			return err
		}
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, prepared.info.Name, prepared.info.Manifest)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%s", formatTemplateValidationIssues(issues))
}

func runTemplateHooks[C any](deps HandlerDeps[C], ctx C, destination string, manifest templatecontracts.TemplateManifest, values map[string]string, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(output, "No post hooks defined for this template")
		return nil
	}
	resolved := make([]resolvedTemplateHook, 0, len(manifest.PostHooks))
	for index, hook := range manifest.PostHooks {
		name, args, description, err := resolveTemplateHook(hook, values)
		if err != nil {
			return fmt.Errorf("post hook %d: %w", index+1, err)
		}
		envOverrides, err := resolveHookValuesMap(hook.Env, values)
		if err != nil {
			return fmt.Errorf("post hook %d: %w", index+1, err)
		}
		env, err := templateHookEnv(deps.CommandEnv(ctx), envOverrides)
		if err != nil {
			return fmt.Errorf("post hook %d: %w", index+1, err)
		}
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		resolved = append(resolved, resolvedTemplateHook{name: name, args: args, description: description, cwd: cwd, env: env})
	}
	for index, item := range resolved {
		_, _ = fmt.Fprintf(output, "[Hook %d] %s\n", index+1, item.description)
		if err := deps.RunSubprocess(ctx, shell.Spec{
			Name:   item.name,
			Args:   item.args,
			Dir:    item.cwd,
			Env:    item.env,
			Stdout: output,
			Stderr: deps.Stderr(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}

type resolvedTemplateHook struct {
	name        string
	args        []string
	description string
	cwd         string
	env         []string
}

var scenarioIDValue = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func resolveTemplateHook(hook templatecontracts.TemplateHook, values map[string]string) (string, []string, string, error) {
	if len(hook.Argv) == 0 {
		return "", nil, "", fmt.Errorf("argv must declare an executable")
	}
	argv, err := resolveHookValues(hook.Argv, values)
	if err != nil {
		return "", nil, "", err
	}
	name := strings.TrimSpace(argv[0])
	if name == "" {
		return "", nil, "", fmt.Errorf("argv[0] must declare a non-empty executable")
	}
	args := append([]string(nil), argv[1:]...)
	description := strings.TrimSpace(hook.Description)
	if description == "" {
		description = strings.Join(hook.Argv, " ")
	}
	return name, args, description, nil
}

func resolveHookValues(values []string, supplied map[string]string) ([]string, error) {
	resolved := make([]string, len(values))
	for i, original := range values {
		value := original
		var b strings.Builder
		for len(value) > 0 {
			start := strings.Index(value, "{{")
			close := strings.Index(value, "}}")
			if start < 0 {
				if close >= 0 || strings.ContainsAny(value, "{}") {
					return nil, fmt.Errorf("malformed hook token in %q", original)
				}
				b.WriteString(value)
				break
			}
			if close >= 0 && close < start {
				return nil, fmt.Errorf("malformed hook token in %q", original)
			}
			b.WriteString(value[:start])
			end := strings.Index(value[start+2:], "}}")
			if end < 0 {
				return nil, fmt.Errorf("malformed hook token in %q", original)
			}
			end += start + 2
			key := value[start+2 : end]
			if key != "SCENARIO_ID" {
				return nil, fmt.Errorf("unsupported hook token %q", key)
			}
			replacement, ok := supplied[key]
			if !ok || replacement == "" {
				return nil, fmt.Errorf("hook token %s has no value", key)
			}
			if !scenarioIDValue.MatchString(replacement) {
				return nil, fmt.Errorf("hook token %s has invalid value %q", key, replacement)
			}
			b.WriteString(replacement)
			value = value[end+2:]
		}
		resolved[i] = b.String()
	}
	return resolved, nil
}

func resolveHookValuesMap(values map[string]string, supplied map[string]string) (map[string]string, error) {
	resolved := make(map[string]string, len(values))
	for key, value := range values {
		items, err := resolveHookValues([]string{value}, supplied)
		if err != nil {
			return nil, fmt.Errorf("environment %s: %w", key, err)
		}
		resolved[key] = items[0]
	}
	return resolved, nil
}

func templateHookEnv(base []string, overrides map[string]string) ([]string, error) {
	env := append([]string(nil), base...)
	keys := make([]string, 0, len(overrides))
	for key := range overrides {
		if strings.TrimSpace(key) == "" || strings.Contains(key, "=") {
			return nil, fmt.Errorf("environment key %q is invalid", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		assignment := key + "=" + overrides[key]
		replaced := false
		for index, entry := range env {
			existingKey, _, ok := strings.Cut(entry, "=")
			if ok && existingKey == key {
				env[index] = assignment
				replaced = true
			}
		}
		if !replaced {
			env = append(env, assignment)
		}
	}
	return env, nil
}
