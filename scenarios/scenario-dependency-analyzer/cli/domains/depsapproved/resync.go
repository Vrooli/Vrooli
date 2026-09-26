package depsapproved

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"scenario-dependency-analyzer/cli/internal/support"
)

// runResync detects named lockfile drift and sends every divergent package
// through the existing governed install RPC. The RPC remains dry-run unless
// the operator explicitly supplies --apply.
func runResync(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps resync")
	var scenario, surface string
	var apply, jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Target scenario")
	fs.StringVar(&surface, "surface", "", "Target JavaScript surface")
	fs.BoolVar(&apply, "apply", false, "Apply the governed resync")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || scenario == "" || surface == "" {
		return fmt.Errorf("usage: %s deps resync --scenario <name> --surface <ui|cli|api> [--apply] [--json]", support.AppName)
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 90*time.Second)
	validation := scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL)
	health, err := validation.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError("detect lockfile drift", err, nil)
	}
	nativeDetail := health.Msg.GetNativeDetail()
	if nativeDetail == nil {
		return fmt.Errorf("dependency health did not return native detail")
	}
	native := &healthv1.DependencyHealthResponse{}
	if err := nativeDetail.UnmarshalTo(native); err != nil {
		return fmt.Errorf("unpack dependency health: %w", err)
	}
	packages := []string{}
	for _, finding := range native.GetFindings() {
		if finding.GetRuleId() != "dependency.node.lockfile_drift" || finding.GetSurfaceId() != surface {
			continue
		}
		for _, name := range strings.Split(finding.GetObserved(), ",") {
			if name = strings.TrimSpace(name); name != "" {
				packages = append(packages, name)
			}
		}
	}
	sort.Strings(packages)
	packages = uniqueResyncStrings(packages)
	specs, err := resyncManifestSpecs(scenario, surface)
	if err != nil {
		return err
	}
	results := make([]*governancev1.InstallDependencyResponse, 0, len(packages))
	client := governanceClient(core)
	for _, name := range packages {
		resp, callErr := client.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: scenario, Surface: surface, Ecosystem: "npm", PackageName: name, Version: specs[name], Apply: apply}))
		if callErr != nil {
			return cliapp.WrapAPIError("resync dependency", callErr, nil)
		}
		results = append(results, resp.Msg)
	}
	if jsonOutput {
		return support.PrintReportJSON(struct {
			Scenario string                                    `json:"scenario"`
			Surface  string                                    `json:"surface"`
			Apply    bool                                      `json:"apply"`
			Packages []string                                  `json:"packages"`
			Results  []*governancev1.InstallDependencyResponse `json:"results"`
		}{scenario, surface, apply, packages, results})
	}
	lines := []string{fmt.Sprintf("Lockfile resync: %s/%s", scenario, surface), fmt.Sprintf("Packages: %d", len(packages)), fmt.Sprintf("Mode: %s", map[bool]string{true: "apply", false: "dry-run"}[apply])}
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("%s: %s", result.GetPackageManager(), result.GetCommand()))
	}
	return support.PrintList(false, cliapp.ListReport{Summary: lines, ResultsHeading: "Governed Resync"}, nil)
}

func resyncManifestSpecs(scenario, surface string) (map[string]string, error) {
	path := filepath.Join(cliutil.ResolveRepoRoot(), "scenarios", scenario, surface, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read resync manifest %s: %w", path, err)
	}
	var raw struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode resync manifest: %w", err)
	}
	out := map[string]string{}
	for _, group := range []map[string]string{raw.Dependencies, raw.DevDependencies, raw.OptionalDependencies} {
		for name, spec := range group {
			out[name] = spec
		}
	}
	return out, nil
}

func uniqueResyncStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}
