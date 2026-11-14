package depsapproved

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// goModRuleIDs mirror the API fix classes for Go module convergence: missing
// in-repo replaces and go.mod/go.sum drift from the import closure. Both resolve
// to the same per-surface convergence.
var goModRuleIDs = []string{"dependency.gomod.replace.missing", "dependency.go.tidy"}

// Message prefixes the API uses for surfaces it did not write.
const (
	fixMessageRefused = "refused "
	fixMessageFailed  = "failed "
)

// runReconcile is the operator entry point for Go module convergence: every
// selected surface's go.mod and go.sum are brought in line with its import
// closure (missing in-repo replaces, plus third-party requirements and
// checksums that shared in-repo modules pull in). It is dry-run by default
// (PreviewFix); --apply writes through ApplyFix; --check previews and exits
// non-zero when anything would change. Requirements a surface would newly
// record pass the dependency governance gate; refused surfaces are left
// untouched and reported with the `deps approved` command that unblocks them.
// This is the same SDA-owned path as the test-genie dependencies phase.
func runReconcile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps reconcile")
	var scenario, surface, module string
	var all, allModules, apply, check, jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Target scenario")
	fs.StringVar(&module, "module", "", "Target exactly one Go module directory (absolute, or relative to the working directory); may be outside the repository, e.g. a freshly generated scenario")
	fs.StringVar(&surface, "surface", "", "Only report/fix this surface (api|cli|ui|…)")
	fs.BoolVar(&all, "all", false, "Reconcile every discovered scenario")
	fs.BoolVar(&allModules, "all-modules", false, "Reconcile every buildable in-repo Go module (scenarios, resources, packages, control plane)")
	fs.BoolVar(&apply, "apply", false, "Write the converged go.mod/go.sum (default is a dry run)")
	fs.BoolVar(&check, "check", false, "Dry run that exits non-zero when any module would change")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return reconcileUsage()
	}
	selectors := 0
	for _, selected := range []bool{allModules, all, scenario != "", module != ""} {
		if selected {
			selectors++
		}
	}
	if selectors != 1 {
		return reconcileUsage()
	}
	if check && apply {
		return fmt.Errorf("--check is a dry run and may not be combined with --apply")
	}

	var targets []reconcileTarget
	switch {
	case module != "":
		dir, err := filepath.Abs(module)
		if err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			return fmt.Errorf("--module %s: no go.mod: %w", module, err)
		}
		// A bare module path converges exactly that module.
		targets = []reconcileTarget{{label: dir, req: &scenariovalidationv1.FixRequest{Path: dir, RuleIds: goModRuleIDs}}}
	case allModules:
		modulePaths, err := discoverBuildableModules()
		if err != nil {
			return err
		}
		for rel, goModPath := range modulePaths {
			// A bare module path converges exactly that module.
			targets = append(targets, reconcileTarget{label: filepath.Dir(rel), req: &scenariovalidationv1.FixRequest{Path: filepath.Dir(goModPath), RuleIds: goModRuleIDs}})
		}
		sort.Slice(targets, func(i, j int) bool { return targets[i].label < targets[j].label })
	case all:
		names, err := listScenarioNames(core)
		if err != nil {
			return err
		}
		for _, name := range names {
			targets = append(targets, reconcileTarget{label: name, req: &scenariovalidationv1.FixRequest{Scenario: name, RuleIds: goModRuleIDs}})
		}
	default:
		targets = []reconcileTarget{{label: scenario, req: &scenariovalidationv1.FixRequest{Scenario: scenario, RuleIds: goModRuleIDs}}}
	}

	client := validationClient(core)
	responses := make([]*scenariovalidationv1.FixResponse, len(targets))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for idx, target := range targets {
		idx, target := idx, target
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var resp *connect.Response[scenariovalidationv1.FixResponse]
			var err error
			if apply {
				resp, err = client.ApplyFix(context.Background(), connect.NewRequest(target.req))
			} else {
				resp, err = client.PreviewFix(context.Background(), connect.NewRequest(target.req))
			}
			if err != nil {
				responses[idx] = &scenariovalidationv1.FixResponse{Messages: []string{fmt.Sprintf("%s%s: %v", fixMessageFailed, target.label, err)}}
				return
			}
			responses[idx] = filterSurface(resp.Msg, surface)
		}()
	}
	wg.Wait()

	report := buildReconcileReport(targets, responses, apply, check)
	if jsonOutput {
		if err := support.PrintReportJSON(report); err != nil {
			return err
		}
	} else if err := printReconcileReport(report); err != nil {
		return err
	}
	return report.err()
}

type reconcileTarget struct {
	label string
	req   *scenariovalidationv1.FixRequest
}

// reconcileReport is the CLI's view of a reconcile run. go.mod edits carry
// their before/after content; go.sum edits are summarized, since checksum
// files are large and fully determined by the go.mod they accompany.
type reconcileReport struct {
	Mode    string            `json:"mode"`
	Targets int               `json:"targets"`
	Changes []reconcileChange `json:"changes,omitempty"`
	Refused []string          `json:"refused,omitempty"`
	Failed  []string          `json:"failed,omitempty"`
}

type reconcileChange struct {
	Target      string `json:"target"`
	File        string `json:"file"`
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
	Before      string `json:"before,omitempty"`
	After       string `json:"after,omitempty"`
}

func buildReconcileReport(targets []reconcileTarget, responses []*scenariovalidationv1.FixResponse, apply, check bool) reconcileReport {
	report := reconcileReport{Mode: "preview", Targets: len(targets)}
	if apply {
		report.Mode = "apply"
	} else if check {
		report.Mode = "check"
	}
	for idx, resp := range responses {
		for _, candidate := range resp.GetCandidates() {
			change := reconcileChange{
				Target:      targets[idx].label,
				File:        candidate.GetFilePath(),
				RuleID:      candidate.GetRuleId(),
				Description: candidate.GetDescription(),
				Applied:     candidate.GetApplied(),
			}
			if filepath.Base(change.File) == "go.mod" {
				change.Before, change.After = candidate.GetBefore(), candidate.GetAfter()
			}
			report.Changes = append(report.Changes, change)
		}
		for _, message := range resp.GetMessages() {
			switch {
			case strings.HasPrefix(message, fixMessageRefused):
				report.Refused = append(report.Refused, strings.TrimPrefix(message, fixMessageRefused))
			case strings.HasPrefix(message, fixMessageFailed):
				report.Failed = append(report.Failed, strings.TrimPrefix(message, fixMessageFailed))
			}
		}
	}
	sort.Strings(report.Refused)
	sort.Strings(report.Failed)
	return report
}

// err makes refusals and failures — and, for --check, any pending change —
// a non-zero exit so the command can gate a workflow.
func (r reconcileReport) err() error {
	if len(r.Refused) > 0 || len(r.Failed) > 0 {
		// The error carries each outcome so callers that keep only stderr (a
		// generator, a lifecycle hook) still show the exact next step.
		lines := make([]string, 0, len(r.Refused)+len(r.Failed))
		for _, refused := range r.Refused {
			lines = append(lines, "refused "+refused)
		}
		for _, failed := range r.Failed {
			lines = append(lines, "failed "+failed)
		}
		return fmt.Errorf("deps reconcile: %d surface requirement(s) refused, %d surface(s) failed:\n  %s", len(r.Refused), len(r.Failed), strings.Join(lines, "\n  "))
	}
	if r.Mode == "check" && len(r.Changes) > 0 {
		return fmt.Errorf("deps reconcile --check: %d file(s) are not converged; run with --apply", len(r.Changes))
	}
	return nil
}

func reconcileUsage() error {
	return fmt.Errorf("usage: %s deps reconcile (--scenario <name> | --module <dir> | --all | --all-modules) [--surface api|cli|ui] [--apply | --check] [--json]", support.AppName)
}

func discoverBuildableModules() (map[string]string, error) {
	return discoverBuildableModulesAt(cliutil.ResolveRepoRoot())
}

func discoverBuildableModulesAt(root string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			switch d.Name() {
			// Documentation, evidence, fixture, template, and BAS directories
			// hold retained or synthetic modules that are never build surfaces.
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache", "phase-cache", "coverage",
				"docs", "testdata", "evidence", "fixtures", ".vrooli", "bas", "templates":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		out[filepath.ToSlash(rel)] = path
		return nil
	})
	return out, err
}

func validationClient(core *cliapp.ScenarioApp) scenariovalidationconnect.ScenarioValidationServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 90*time.Second)
	return scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
}

func listScenarioNames(core *cliapp.ScenarioApp) ([]string, error) {
	body, err := core.Get("/scenarios", nil)
	if err != nil {
		return nil, err
	}
	var resp []map[string]interface{}
	if err := support.Decode(body, &resp); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(resp))
	for _, item := range resp {
		if name := strings.TrimSpace(support.String(item["name"])); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// filterSurface drops candidates that do not belong to the requested surface.
func filterSurface(resp *scenariovalidationv1.FixResponse, surface string) *scenariovalidationv1.FixResponse {
	surface = strings.TrimSpace(surface)
	if resp == nil || surface == "" {
		return resp
	}
	needle := "/" + surface + "/"
	kept := resp.GetCandidates()[:0:0]
	for _, c := range resp.GetCandidates() {
		if strings.Contains(filepathSlash(c.GetFilePath()), needle) {
			kept = append(kept, c)
		}
	}
	resp.Candidates = kept
	return resp
}

func filepathSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

func printReconcileReport(report reconcileReport) error {
	results := make([]string, 0, len(report.Changes)+len(report.Refused)+len(report.Failed))
	for _, change := range report.Changes {
		results = append(results, fmt.Sprintf("%s: %s — %s", change.Target, change.File, change.Description))
	}
	for _, refused := range report.Refused {
		results = append(results, "REFUSED "+refused)
	}
	for _, failed := range report.Failed {
		results = append(results, "FAILED "+failed)
	}
	var headline string
	switch {
	case len(report.Changes) == 0 && len(report.Refused) == 0 && len(report.Failed) == 0:
		headline = fmt.Sprintf("All %d target(s) are converged on their import closure.", report.Targets)
	case report.Mode == "apply":
		headline = fmt.Sprintf("Converged %d file(s); %d requirement(s) refused; %d surface(s) failed.", len(report.Changes), len(report.Refused), len(report.Failed))
	default:
		headline = fmt.Sprintf("%d file(s) would converge (dry run); %d requirement(s) refused; %d surface(s) failed.", len(report.Changes), len(report.Refused), len(report.Failed))
	}
	return support.PrintList(false, cliapp.ListReport{
		Summary:        []string{headline},
		ResultsHeading: "Surfaces",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps reconcile --all-modules --json", support.AppName),
			fmt.Sprintf("%s deps reconcile --scenario <name> --apply", support.AppName),
		},
	}, nil)
}
