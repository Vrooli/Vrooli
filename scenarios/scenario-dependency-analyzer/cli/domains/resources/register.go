package resources

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	resourceusagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/resource_usage"
	resourceusageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/resource_usage/resource_usage_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

// Register exposes the resources command group: the federated resource-usage
// search leaf ("which scenarios use postgres").
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Resources",
		Commands: []cliapp.Command{
			{
				Name:        "resources",
				Description: "Search local resource usage across the fleet (which scenarios use a resource)",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	if len(args) > 0 && strings.EqualFold(strings.TrimSpace(args[0]), "search") {
		args = args[1:]
	}
	return runSearch(core, args)
}

// runSearch queries the .resources federated leaf (SearchResourceUsage).
func runSearch(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("resources search")
	var limit int
	var jsonOutput bool
	fs.IntVar(&limit, "limit", 10, "Maximum hits to return")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: %s resources search <query> [--limit n] [--json]", support.AppName)
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 45*time.Second)
	client := resourceusageconnect.NewResourceUsageServiceClient(httpClient, baseURL)
	resp, err := client.SearchResourceUsage(context.Background(), connect.NewRequest(&resourceusagev1.SearchResourceUsageRequest{
		Query: query,
		Limit: int32(limit),
	}))
	if err != nil {
		return cliapp.WrapAPIError("search resource usage", err, nil)
	}
	if jsonOutput {
		body, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp.Msg)
		if err != nil {
			return fmt.Errorf("render search JSON: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(body))
		return nil
	}

	hits := resp.Msg.GetResults()
	results := make([]string, 0, len(hits))
	for _, h := range hits {
		results = append(results, fmt.Sprintf("%s (%.3f) — %s", h.GetResource(), h.GetRelevanceScore(), h.GetSummary()))
	}
	report := cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Query: %s", query), fmt.Sprintf("Hits: %d", len(hits))},
		ResultsHeading: "Resource Usage Hits",
		Results:        results,
		RetrievalHints: []string{fmt.Sprintf("%s resources search %q --json", support.AppName, query)},
	}
	return support.PrintList(false, report, nil)
}
