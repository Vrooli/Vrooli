package scenarioprimitives

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

const (
	projectCLIName        = "vrooli"
	projectPrimitiveGroup = "scenario"
	projectDefaultAPIPort = "8092"
)

// Run dispatches the migrated project scenario primitive. The manifest group
// remains a separate governance unit so it can be loaded exclusively through
// cli-core evidence-bearing builders; the invocation is normalized back to the
// public `vrooli scenario` command path.
func Run(root string, args []string, jsonOutput bool, stdout, stderr io.Writer) error {
	manifestPath := filepath.Join(root, "cli", "manifest.json")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read project CLI manifest: %w", err)
	}

	apiBase := strings.TrimSpace(os.Getenv("VROOLI_API_BASE_URL"))
	if apiBase == "" {
		port := strings.TrimSpace(os.Getenv("VROOLI_API_PORT"))
		if port == "" {
			port = projectDefaultAPIPort
		}
		apiBase = "http://127.0.0.1:" + port
	}
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name:           projectCLIName,
		Version:        "2.0.0",
		Description:    "Vrooli control-plane scenario primitives",
		DefaultAPIBase: apiBase,
		APIPrefix:      "/",
		HealthPath:     "/health",
		AllowAnonymous: true,
	})
	if err != nil {
		return fmt.Errorf("configure project CLI primitive: %w", err)
	}
	core.CLI.SetStaleChecker(nil)

	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := cliv1connect.NewScenarioControlPlaneServiceClient(httpClient, baseURL)
	group, err := BuildScenarioPrimitiveGroup(manifestRaw, client)
	if err != nil {
		return fmt.Errorf("load project scenario primitives: %w", err)
	}
	group.Name = "scenario"
	core.SetCommandsWithSubgroups(nil, []cliapp.SubcommandGroup{group})

	primitiveArgs := append([]string{"scenario"}, args...)
	if jsonOutput && !containsFlag(primitiveArgs, "--json") {
		primitiveArgs = append(primitiveArgs, "--json")
	}
	return core.CLI.RunWithWriters(primitiveArgs, stdout, stderr)
}

// BuildScenarioPrimitiveGroup assembles the manifest-bound command tree. It is
// exported for the static evidence test; assembling the tree only creates
// closures and never invokes the supplied client.
func BuildScenarioPrimitiveGroup(manifestRaw []byte, client cliv1connect.ScenarioControlPlaneServiceClient) (cliapp.SubcommandGroup, error) {
	listHandler := cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*cliv1.ScenarioListResponse, error) {
			response, callErr := client.ListScenarios(context.Background(), connect.NewRequest(&cliv1.ListScenariosRequest{
				IncludePorts: ctx.BoolFlag("include-ports"),
			}))
			if callErr != nil {
				return nil, cliapp.WrapAPIError("list scenarios", callErr, nil)
			}
			if response == nil || response.Msg == nil {
				return nil, fmt.Errorf("list scenarios returned an empty response")
			}
			return response.Msg, nil
		},
		projectScenarioListReport,
	)
	return cliapp.LoadFromManifestPrimitives(manifestRaw, projectPrimitiveGroup, map[string]cliapp.PrimitiveHandler{
		"ScenarioControlPlaneService.ListScenarios": listHandler,
	})
}

func containsFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func projectScenarioListReport(_ cliapp.OperationContext, response *cliv1.ScenarioListResponse) cliapp.ListReport {
	if response == nil {
		return cliapp.ListReport{Summary: []string{"No scenario response was returned."}}
	}
	summary := response.GetSummary()
	total, running, available := int32(len(response.GetScenarios())), int32(0), int32(0)
	if summary != nil {
		total = summary.GetTotalScenarios()
		running = summary.GetRunning()
		available = summary.GetAvailable()
	}
	results := make([]string, 0, len(response.GetScenarios())+len(response.GetDiscoveryFailures()))
	for _, scenario := range response.GetScenarios() {
		if scenario == nil {
			continue
		}
		results = append(results, fmt.Sprintf("%s — %s", scenario.GetName(), scenario.GetStatus()))
	}
	for _, failure := range response.GetDiscoveryFailures() {
		if failure == nil {
			continue
		}
		results = append(results, fmt.Sprintf("discovery failure: %s (%s)", failure.GetName(), failure.GetError()))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d scenario(s): %d running, %d available.", total, running, available)},
		ResultsHeading: "Scenarios",
		Results:        results,
		RetrievalHints: []string{"Use `vrooli scenario status <name>` for one scenario's runtime details."},
	}
}
