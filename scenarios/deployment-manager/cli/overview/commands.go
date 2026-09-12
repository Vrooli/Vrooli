package overview

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"deployment-manager/cli/cmdutil"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	dependenciesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/dependencies/dependenciesv1connect"
	fitnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/fitness/fitnessv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Commands struct {
	api          *cliutil.APIClient
	dependencies dependenciesconnect.DependenciesServiceClient
	fitness      fitnessconnect.FitnessServiceClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

// NewWithConnectClients binds overview operations to the generated services.
// New remains available for focused compatibility fixtures.
func NewWithConnectClients(api *cliutil.APIClient, dependencies dependenciesconnect.DependenciesServiceClient, fitness fitnessconnect.FitnessServiceClient) *Commands {
	return &Commands{api: api, dependencies: dependencies, fitness: fitness}
}

func typedOverviewValue(payload map[string]interface{}, call func(context.Context, *connect.Request[structpb.Value]) (*connect.Response[structpb.Value], error)) ([]byte, error) {
	request, err := structpb.NewValue(payload)
	if err != nil {
		return nil, fmt.Errorf("encode overview request: %w", err)
	}
	response, err := call(context.Background(), connect.NewRequest(request))
	if err != nil {
		return nil, cliapp.WrapAPIError("overview operation", err, nil)
	}
	if response == nil || response.Msg == nil {
		return nil, errors.New("typed overview response was empty")
	}
	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(response.Msg)
}

func (c *Commands) Analyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	var body []byte
	var err error
	if c.dependencies != nil {
		body, err = typedOverviewValue(map[string]interface{}{"scenario": scenario}, c.dependencies.Analyze)
	} else {
		body, err = c.api.Get("/api/v1/dependencies/analyze/"+scenario, nil)
	}
	if err != nil {
		return err
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) != "json" {
		return renderAnalyzeReport(scenario, body)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Fitness(args []string) error {
	fs := flag.NewFlagSet("fitness", flag.ContinueOnError)
	tier := fs.String("tier", "2", "deployment tier")
	format := fs.String("format", "", "output format (json|table)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
	if len(remaining) == 0 {
		return errors.New("scenario is required")
	}
	scenario := remaining[0]
	tierNum := cmdutil.TierToNumber(*tier)
	payload := map[string]interface{}{
		"scenario": scenario,
		"tiers":    []interface{}{tierNum},
	}
	var body []byte
	var err error
	if c.fitness != nil {
		body, err = typedOverviewValue(payload, c.fitness.Score)
	} else {
		body, err = c.api.Request("POST", "/api/v1/fitness/score", nil, payload)
	}
	if err != nil {
		return err
	}
	if strings.ToLower(cmdutil.ResolveFormat(*format)) != "json" {
		return renderFitnessReport(scenario, tierNum, body)
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func renderAnalyzeReport(scenario string, body []byte) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println(string(body))
		return nil
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Scenario: %s", scenario),
		},
		ResultsHeading: "Dependency Analysis",
		RetrievalHints: []string{
			fmt.Sprintf("deployment-manager fitness %s --tier 2", scenario),
		},
	}
	for _, key := range sortedKeys(parsed) {
		report.Results = append(report.Results, fmt.Sprintf("%s: %v", key, parsed[key]))
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func renderFitnessReport(scenario string, tier int, body []byte) error {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		fmt.Println(string(body))
		return nil
	}

	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Scenario: %s", scenario),
			fmt.Sprintf("Tier: %d", tier),
		},
	}
	if score, ok := parsed["score"]; ok {
		report.Status = append(report.Status, fmt.Sprintf("Score: %v", score))
	}
	triage := cliapp.TriageGroup{Heading: "Fitness Factors"}
	for _, key := range sortedKeys(parsed) {
		if key == "score" {
			continue
		}
		triage.Items = append(triage.Items, fmt.Sprintf("%s: %v", key, parsed[key]))
	}
	if len(triage.Items) > 0 {
		report.Triage = append(report.Triage, triage)
	}
	report.NextSteps = []string{
		fmt.Sprintf("deployment-manager analyze %s", scenario),
	}
	return cliapp.RenderOperationalReport(os.Stdout, report)
}

func sortedKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
