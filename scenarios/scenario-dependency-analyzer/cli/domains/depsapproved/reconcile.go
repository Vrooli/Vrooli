package depsapproved

import (
	"context"
	"fmt"
	"io/fs"
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

// goModReplaceRuleID mirrors the API fix-class id; the CLI scopes the Fix RPC to
// this class so `deps reconcile` only ever touches in-repo go.mod replaces.
const goModReplaceRuleID = "dependency.gomod.replace.missing"

// runReconcile is the operator entry point for repairing missing in-repo go.mod
// replaces. It is dry-run by default (PreviewFix); --apply writes + tidies via
// ApplyFix. It flows through the same SDA-owned reconcile primitive as the
// test-genie dependencies phase, so detection and remediation never drift.
func runReconcile(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps reconcile")
	var scenario, surface string
	var all, allModules, apply, jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Target scenario")
	fs.StringVar(&surface, "surface", "", "Only report/fix this surface (api|cli|ui|…)")
	fs.BoolVar(&all, "all", false, "Reconcile every discovered scenario")
	fs.BoolVar(&allModules, "all-modules", false, "Reconcile every buildable in-repo Go module")
	fs.BoolVar(&apply, "apply", false, "Write the replaces (default is a dry run)")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return reconcileUsage()
	}
	if allModules && (all || scenario != "") || !allModules && all == (scenario != "") {
		return reconcileUsage()
	}

	targets := []string{scenario}
	var modulePaths map[string]string
	if allModules {
		var err error
		modulePaths, err = discoverBuildableModules()
		if err != nil {
			return err
		}
		targets = make([]string, 0, len(modulePaths))
		for path := range modulePaths {
			targets = append(targets, path)
		}
		sort.Strings(targets)
	}
	if all {
		var err error
		targets, err = listScenarioNames(core)
		if err != nil {
			return err
		}
	}

	client := validationClient(core)
	responses := make([]*scenariovalidationv1.FixResponse, len(targets))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for idx, name := range targets {
		idx, name := idx, name
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			req := &scenariovalidationv1.FixRequest{Scenario: name, RuleIds: []string{goModReplaceRuleID}}
			if allModules {
				req.Scenario = filepath.Base(filepath.Dir(modulePaths[name]))
				req.Path = filepath.Dir(modulePaths[name])
			}
			var resp *connect.Response[scenariovalidationv1.FixResponse]
			var err error
			if apply {
				resp, err = client.ApplyFix(context.Background(), connect.NewRequest(req))
			} else {
				resp, err = client.PreviewFix(context.Background(), connect.NewRequest(req))
			}
			if err != nil {
				responses[idx] = &scenariovalidationv1.FixResponse{Scenario: name, Messages: []string{fmt.Sprintf("%s: %v", name, err)}}
				return
			}
			responses[idx] = filterSurface(resp.Msg, surface)
		}()
	}
	wg.Wait()

	if jsonOutput {
		if len(responses) == 1 {
			return printProto(responses[0])
		}
		return printReconcileJSON(responses)
	}
	return printReconcileReport(responses, apply)
}

func reconcileUsage() error {
	return fmt.Errorf("usage: %s deps reconcile (--scenario <name> | --all | --all-modules) [--surface api|cli|ui] [--apply] [--json]", support.AppName)
}

func discoverBuildableModules() (map[string]string, error) {
	root := cliutil.ResolveRepoRoot()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache", "phase-cache":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "templates/") || strings.Contains(rel, "/bas/") {
			return nil
		}
		out[rel] = path
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

func printReconcileJSON(responses []*scenariovalidationv1.FixResponse) error {
	for _, resp := range responses {
		if len(responses) == 1 {
			return printProto(resp)
		}
		break
	}
	return support.PrintReportJSON(struct {
		Responses []*scenariovalidationv1.FixResponse `json:"responses"`
	}{Responses: responses})
}

func printReconcileReport(responses []*scenariovalidationv1.FixResponse, applied bool) error {
	results := make([]string, 0)
	total := 0
	for _, resp := range responses {
		for _, c := range resp.GetCandidates() {
			total++
			results = append(results, fmt.Sprintf("%s: %s — %s", resp.GetScenario(), c.GetFilePath(), strings.TrimPrefix(c.GetDescription(), "Add ")))
		}
	}
	headline := "No missing in-repo go.mod replaces."
	if total > 0 {
		if applied {
			headline = fmt.Sprintf("Reconciled %d surface go.mod file(s).", total)
		} else {
			headline = fmt.Sprintf("%d surface go.mod file(s) need a local replace (dry run).", total)
		}
	}
	report := cliapp.ListReport{
		Summary:        []string{headline},
		ResultsHeading: "Surfaces",
		Results:        results,
		RetrievalHints: []string{
			fmt.Sprintf("%s deps reconcile --all --json", support.AppName),
			fmt.Sprintf("%s deps reconcile --scenario <name> --apply", support.AppName),
		},
	}
	return support.PrintList(false, report, nil)
}
