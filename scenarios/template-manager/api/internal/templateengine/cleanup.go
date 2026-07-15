package templateengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioexec"
	"github.com/vrooli/vrooli/internal/templatevalidation"
	templatecontracts "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func runTemplateCleanup[C any](deps HandlerDeps[C], ctx C, req templatecontracts.TemplateCleanupRequest) (templatecontracts.TemplateCleanupResult, error) {
	opts, err := templateCleanupOptions(deps.Root(ctx), req)
	if err != nil {
		return templatecontracts.TemplateCleanupResult{}, err
	}
	plan := templatevalidation.PlanCleanup(opts)
	result := templatevalidation.ExecuteCleanup(plan)
	if err := runProtoGenerateForCleanupResult(deps, ctx, &result); err != nil {
		return result, err
	}
	return result, templatevalidation.ResultError(result)
}

func cleanupStaleTemplateValidationRuns[C any](deps HandlerDeps[C], ctx C) {
	opts := templatevalidation.CleanupOptions{
		RepoRoot:  deps.Root(ctx),
		OlderThan: templatevalidation.DefaultCleanupOlderThan,
	}
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	_ = runProtoGenerateForCleanupResult(deps, ctx, &result)
}

func templateCleanupOptions(repoRoot string, req templatecontracts.TemplateCleanupRequest) (templatevalidation.CleanupOptions, error) {
	olderThan := templatevalidation.DefaultCleanupOlderThan
	if strings.TrimSpace(req.OlderThan) != "" {
		parsed, err := time.ParseDuration(req.OlderThan)
		if err != nil {
			return templatevalidation.CleanupOptions{}, fmt.Errorf("invalid --older-than duration %q: %w", req.OlderThan, err)
		}
		olderThan = parsed
	}
	return templatevalidation.CleanupOptions{
		RepoRoot:        repoRoot,
		OlderThan:       olderThan,
		IncludeRetained: req.IncludeRetained,
		RunID:           req.RunID,
		DryRun:          req.DryRun,
	}, nil
}

func runProtoGenerateForCleanupResult[C any](deps HandlerDeps[C], ctx C, result *templatevalidation.CleanupResult) error {
	if result == nil || result.DryRun || !result.NeedsProtoGenerate || len(result.Failures) > 0 {
		return nil
	}
	if deps.RunSubprocess == nil {
		return nil
	}
	protoDir := filepath.Join(deps.Root(ctx), "packages", "proto")
	if _, err := os.Stat(filepath.Join(protoDir, "Makefile")); err != nil {
		return nil
	}
	if err := deps.RunSubprocess(ctx, scenarioexec.SubprocessSpec{
		Name:   "make",
		Args:   []string{"generate"},
		Dir:    protoDir,
		Env:    deps.CommandEnv(ctx),
		Stdout: deps.Stderr(ctx),
		Stderr: deps.Stderr(ctx),
	}); err != nil {
		return fmt.Errorf("regenerate proto artifacts after template validation cleanup: %w", err)
	}
	result.ProtoGenerateRan = true
	return nil
}

func cleanupDeepValidationRelocations[C any](deps HandlerDeps[C], ctx C, relocations []templatecontracts.ResolvedRelocation) error {
	cleanupRelocationTargets(relocations)
	if !hasProtoRelocation(deps.Root(ctx), relocations) {
		return nil
	}
	return runProtoGenerateForCleanupResult(deps, ctx, &templatevalidation.CleanupResult{NeedsProtoGenerate: true})
}

func hasProtoRelocation(repoRoot string, relocations []templatecontracts.ResolvedRelocation) bool {
	protoSchemasRoot := filepath.Clean(filepath.Join(repoRoot, "packages", "proto", "schemas"))
	for _, relocation := range relocations {
		target := filepath.Clean(relocation.To)
		if target == protoSchemasRoot || strings.HasPrefix(target, protoSchemasRoot+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
