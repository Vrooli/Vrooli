package resources

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	resourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/resources"
	"google.golang.org/protobuf/proto"
	"vrooli-onboarding/cli/internal/support"
)

const (
	listProcedure   = "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/ListResources"
	getProcedure    = "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/GetResource"
	healthProcedure = "/vrooli.vrooli_onboarding.v1.resources.ResourcesService/GetResourceHealth"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "resources", Description: "List Vrooli resources and inspect their onboarding health", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "list", Aliases: []string{"ls"}, Description: "List available resources", Run: func(args []string) error { return runList(core, args) }},
		{Name: "get", Aliases: []string{"show"}, Description: "Show one resource by name", Run: func(args []string) error { return getResource(core, args) }},
		{Name: "health", Description: "Show onboarding health for all resources", Run: func(args []string) error { return runHealth(core, args) }},
	}}
}

func request(core *cliapp.ScenarioApp, procedure string, message, response proto.Message) error {
	return support.RequestProto(core, procedure, message, response, "resources")
}

func targetFlags(name string, args []string) (*string, *bool, error) {
	fs := support.NewFlagSet(name)
	target := fs.String("target", "local", "Onboarding target scenario")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(*target) == "" {
		return nil, nil, fmt.Errorf("--target is required")
	}
	return target, jsonOutput, nil
}

func printJSON(message proto.Message) error {
	return support.PrintProto(os.Stdout, message, "resources")
}

func runList(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("resources list", args)
	if err != nil {
		return err
	}
	response := new(resourcesv1.ListResourcesResponse)
	if err := request(core, listProcedure, &resourcesv1.ListResourcesRequest{Target: *target}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: []string{fmt.Sprintf("Resources: %d", response.GetCount())}, ResultsHeading: "Resources", Results: resourceRows(response.GetResources()), RetrievalHints: []string{fmt.Sprintf("%s resources get <name>", support.CLIName), fmt.Sprintf("%s resources health", support.CLIName)}})
}

func getResource(core *cliapp.ScenarioApp, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: resources get <name>")
	}
	name := args[0]
	target, jsonOutput, err := targetFlags("resources get", args[1:])
	if err != nil {
		return err
	}
	response := new(resourcesv1.GetResourceResponse)
	if err := request(core, getProcedure, &resourcesv1.GetResourceRequest{Target: *target, Name: name}, response); err != nil {
		return err
	}
	resource := response.GetResource()
	if resource == nil {
		return fmt.Errorf("resource response was empty")
	}
	if *jsonOutput {
		return printJSON(resource)
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: []string{fmt.Sprintf("Resource: %s (%s)", resource.GetName(), resource.GetStatus())}, ResultsHeading: "Details", Results: []string{fmt.Sprintf("Name: %s", resource.GetName()), fmt.Sprintf("Status: %s", resource.GetStatus()), fmt.Sprintf("Category: %s", resource.GetCategory()), fmt.Sprintf("Installed: %t", resource.GetInstalled())}})
}

func runHealth(core *cliapp.ScenarioApp, args []string) error {
	target, jsonOutput, err := targetFlags("resources health", args)
	if err != nil {
		return err
	}
	response := new(resourcesv1.GetResourceHealthResponse)
	if err := request(core, healthProcedure, &resourcesv1.GetResourceHealthRequest{Target: *target}, response); err != nil {
		return err
	}
	if *jsonOutput {
		return printJSON(response)
	}
	return cliapp.RenderListReport(os.Stdout, cliapp.ListReport{Summary: []string{fmt.Sprintf("Resource health: %d/%d healthy", response.GetHealthyCount(), response.GetTotal()), fmt.Sprintf("Checked at: %s", support.FormatTime(response.GetCheckedAt()))}, ResultsHeading: "Statuses", Results: healthRows(response.GetResources()), RetrievalHints: []string{fmt.Sprintf("%s resources list", support.CLIName)}})
}

func resourceRows(resources []*resourcesv1.Resource) []string {
	if len(resources) == 0 {
		return []string{"No resources available"}
	}
	rows := make([]string, 0, len(resources))
	for _, r := range resources {
		rows = append(rows, fmt.Sprintf("%s | status=%s | category=%s | installed=%t", r.GetName(), r.GetStatus(), r.GetCategory(), r.GetInstalled()))
	}
	return rows
}
func healthRows(statuses []*resourcesv1.ResourceHealth) []string {
	if len(statuses) == 0 {
		return []string{"No resources reporting health"}
	}
	rows := make([]string, 0, len(statuses))
	for _, s := range statuses {
		rows = append(rows, fmt.Sprintf("%s | status=%s | category=%s | available=%t", s.GetName(), s.GetStatus(), s.GetCategory(), s.GetAvailable()))
	}
	return rows
}
