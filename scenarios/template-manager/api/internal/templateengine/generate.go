package templateengine

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioexec"
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
		if err := runTemplateHooks(deps, ctx, prepared.destination, info.Manifest, deps.Stdout(ctx)); err != nil {
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
	return runRelocations(deps, ctx, prepared.info.Path, prepared.relocations, prepared.values, deps.Stdout(ctx))
}

func validateGeneratedScenarioResult[C any](deps HandlerDeps[C], ctx C, prepared preparedGenerate) error {
	issues := validateGeneratedScenario(prepared.destination, deps.RunSubprocess != nil, func(spec scenarioexec.SubprocessSpec) error {
		spec.Env = deps.CommandEnv(ctx)
		spec.Stdout = io.Discard
		spec.Stderr = deps.Stderr(ctx)
		return deps.RunSubprocess(ctx, spec)
	}, prepared.info.Name, prepared.info.Manifest)
	if len(issues) == 0 {
		return nil
	}
	return fmt.Errorf("%s", formatTemplateValidationIssues(issues))
}

func runTemplateHooks[C any](deps HandlerDeps[C], ctx C, destination string, manifest templatecontracts.TemplateManifest, output io.Writer) error {
	if output == nil {
		output = io.Discard
	}
	if len(manifest.PostHooks) == 0 {
		_, _ = fmt.Fprintln(output, "No post hooks defined for this template")
		return nil
	}
	for index, hook := range manifest.PostHooks {
		description := strings.TrimSpace(hook.Description)
		if description == "" {
			description = hook.Cmd
		}
		_, _ = fmt.Fprintf(output, "[Hook %d] %s\n", index+1, description)
		cwd := destination
		if strings.TrimSpace(hook.Cwd) != "" && hook.Cwd != "." {
			cwd = filepath.Join(destination, filepath.FromSlash(hook.Cwd))
		}
		if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
			Name:   "bash",
			Args:   []string{"-lc", hook.Cmd},
			Dir:    cwd,
			Env:    deps.CommandEnv(ctx),
			Stdout: output,
			Stderr: deps.Stderr(ctx),
		}); err != nil {
			return err
		}
	}
	return nil
}
