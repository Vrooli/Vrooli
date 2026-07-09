package templateengine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
	. "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts" //nolint:revive // see template_runtime.go rationale
)

// runTemplateDrift answers `vrooli scenario template drift`. For each target
// scenario it loads the recorded provenance from .vrooli/service.json, re-hashes
// the current template, and reports any mismatch.
func runTemplateDrift[C any](deps HandlerDeps[C], ctx C, req TemplateDriftRequest) (cliout.Format, TemplateDriftReport, error) {
	root := deps.Root(ctx)
	format, err := deps.OutputFormat(ctx)
	if err != nil {
		return "", TemplateDriftReport{}, err
	}
	if req.JSON {
		format = cliout.FormatJSON
	}

	var targets []scenariomodel.Scenario
	env := scenariomodel.SandboxEnvFromEnv()
	if req.All {
		all, err := scenariomodel.Discover(root, env)
		if err != nil {
			return "", TemplateDriftReport{}, fmt.Errorf("discover scenarios: %w", err)
		}
		targets = all
	} else {
		scenario, err := scenariomodel.Load(root, req.Scenario, env)
		if err != nil {
			if errors.Is(err, scenariomodel.ErrNotFound) {
				return "", TemplateDriftReport{}, fmt.Errorf("scenario not found: %s", req.Scenario)
			}
			return "", TemplateDriftReport{}, fmt.Errorf("load scenario %s: %w", req.Scenario, err)
		}
		targets = []scenariomodel.Scenario{scenario}
	}

	report := TemplateDriftReport{Scenarios: make([]TemplateDriftScenarioReport, 0, len(targets))}
	for _, sc := range targets {
		entry := analyzeDriftForScenario(root, sc, req.Verbose)
		if req.All && entry.Status == TemplateDriftStatusNoProvenance {
			// In --all mode, scenarios without template provenance (handwritten
			// or pre-provenance) are noise — skip them silently.
			continue
		}
		report.Scenarios = append(report.Scenarios, entry)
	}
	return format, report, nil
}

func analyzeDriftForScenario(root string, sc scenariomodel.Scenario, verbose bool) TemplateDriftScenarioReport {
	out := TemplateDriftScenarioReport{Scenario: sc.Slug}
	gen := sc.Manifest.Generation
	if gen == nil || strings.TrimSpace(gen.Template.ID) == "" {
		out.Status = TemplateDriftStatusNoProvenance
		return out
	}
	out.TemplateID = gen.Template.ID
	out.RecordedVersion = gen.Template.Version
	out.RecordedManifest = gen.ManifestSha
	out.RecordedContent = gen.ContentSha

	info, err := loadTemplate(root, gen.Template.ID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "template not found") {
			out.Status = TemplateDriftStatusTemplateGone
			return out
		}
		out.Status = TemplateDriftStatusHashError
		out.Message = fmt.Sprintf("load template: %v", err)
		return out
	}
	out.CurrentVersion = strings.TrimSpace(info.Manifest.Version)

	manifestSha, contentSha, hashErr := computeTemplateHashes(info)
	out.CurrentManifest = manifestSha
	out.CurrentContent = contentSha
	if hashErr != nil {
		out.Status = TemplateDriftStatusHashError
		out.Message = fmt.Sprintf("hash template: %v", hashErr)
		return out
	}

	// A scenario generated before drift tracking shipped will have no recorded
	// hashes. There's nothing to compare against, so we can't claim drift — but
	// we also shouldn't silently report "in sync" because that would be a lie.
	if strings.TrimSpace(gen.ManifestSha) == "" && strings.TrimSpace(gen.ContentSha) == "" {
		out.Status = TemplateDriftStatusMissingHashes
		return out
	}

	out.ManifestDrifted = !hashesEqualOrSkippable(gen.ManifestSha, manifestSha)
	out.ContentDrifted = !hashesEqualOrSkippable(gen.ContentSha, contentSha)
	if out.Drifted() {
		out.Status = TemplateDriftStatusDrifted
	} else {
		out.Status = TemplateDriftStatusOK
	}

	if verbose && out.ContentDrifted {
		out.FileDiffs = collectVerboseDiffs(info, sc)
	}
	return out
}

// hashesEqualOrSkippable returns true when the hashes match. If the recorded
// hash is empty we treat that as "can't compare for this hash" — the missing-
// hashes status above already short-circuits the all-empty case, so this only
// fires when one of the two hashes was recorded but the other wasn't (which
// can happen if hashing partially failed at generation time).
func hashesEqualOrSkippable(recorded, current string) bool {
	r := strings.TrimSpace(recorded)
	c := strings.TrimSpace(current)
	if r == "" {
		return true
	}
	return r == c
}

// collectVerboseDiffs reports which files the current template ships that are
// not present at their template-relative path in the scenario destination.
// Path-substituted entries (those containing `{{...}}`) are skipped because we
// can't resolve them without the original generation values.
//
// This is a presence check, not a content diff: scenarios are expected to edit
// inherited files, so byte-level "this file changed" reports would be noise.
// What's actionable is "the template now ships file X and your scenario
// doesn't have it" — that's what this surfaces.
func collectVerboseDiffs(info TemplateInfo, sc scenariomodel.Scenario) []TemplateDriftFileDiff {
	files, err := templateFilesByPath(info.Path, info.Manifest)
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var diffs []TemplateDriftFileDiff
	for _, rel := range paths {
		if strings.Contains(rel, "{{") {
			continue
		}
		target := filepath.Join(sc.Path, filepath.FromSlash(rel))
		if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
			diffs = append(diffs, TemplateDriftFileDiff{Path: rel, Reason: "added_in_template"})
		}
	}
	return diffs
}

// EnrichInfoDriftFlag is called from the scenario info handler to set the
// TemplateDrifted bit on an InfoOutput. The check is best-effort: any failure
// to load the template or compute its current hash leaves TemplateDrifted
// false, because `info` must keep working even when drift detection can't run.
func EnrichInfoDriftFlag(root string, out *InfoOutput) {
	if out == nil || out.Scenario.Generation == nil {
		return
	}
	gen := out.Scenario.Generation
	if strings.TrimSpace(gen.Template.ID) == "" {
		return
	}
	if strings.TrimSpace(gen.ManifestSha) == "" && strings.TrimSpace(gen.ContentSha) == "" {
		return
	}
	info, err := loadTemplate(root, gen.Template.ID)
	if err != nil {
		return
	}
	manifestSha, contentSha, err := computeTemplateHashes(info)
	if err != nil {
		return
	}
	manifestDrift := !hashesEqualOrSkippable(gen.ManifestSha, manifestSha)
	contentDrift := !hashesEqualOrSkippable(gen.ContentSha, contentSha)
	out.Scenario.TemplateDrifted = manifestDrift || contentDrift
}
