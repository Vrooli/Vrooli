package measures

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	tmmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures"
	tmmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures/measures_v1connect"
)

const GroupName = "measures"

type handlers struct {
	client tmmeasuresconnect.MeasuresServiceClient
}

type declaration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type runResult struct {
	Measure string              `json:"measure"`
	Value   string              `json:"value,omitempty"`
	Buckets []map[string]string `json:"buckets,omitempty"`
}

func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: tmmeasuresconnect.NewMeasuresServiceClient(httpClient, baseURL)}

	list := cliapp.Command{
		Name:        "list",
		Description: "List declared template-manager measures",
		NeedsAPI:    false,
	}.WithPrimitive(cliapp.Action(h.listCall, h.listReport))

	run := cliapp.Command{
		Name:        "run",
		Description: "Run a declared template-manager measure",
		NeedsAPI:    true,
		Args: cliapp.ArgSchema{
			Positionals: []cliapp.Positional{{Name: "measure", Required: true, Description: "Measure name"}},
			Flags:       []cliapp.Flag{{Name: "template", Description: "Template id for template.deep-validate-green-streak", Default: "react-vite"}},
		},
	}.WithPrimitive(cliapp.Action(h.runCall, h.runReport))

	return cliapp.SubcommandGroup{
		Name:        GroupName,
		Description: "List and run template-manager measures",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{list, run},
	}, nil
}

func (h *handlers) listCall(_ cliapp.OperationContext) ([]declaration, error) {
	return []declaration{
		{Name: "template.open-debt-count", Description: "Open inherited-template debt entries this week."},
		{Name: "template.deep-validate-green-streak", Description: "Consecutive passing deep validations for a template."},
		{Name: "template.fleet-standing-distribution", Description: "Current/drift/version-lag/open-debt buckets."},
		{Name: "template.max-version-lag", Description: "Maximum version lag across governed templates."},
	}, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, decls []declaration) cliapp.MutationReport {
	lines := make([]string, 0, len(decls))
	for _, decl := range decls {
		lines = append(lines, fmt.Sprintf("%s - %s", decl.Name, decl.Description))
	}
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Found %d measure(s).", len(decls))},
		Changes: lines,
	}
}

func (h *handlers) runCall(ctx cliapp.OperationContext) (runResult, error) {
	measure := ctx.Positional("measure")
	switch measure {
	case "template.open-debt-count":
		resp, err := h.client.OpenDebtCount(context.Background(), connect.NewRequest(&tmmeasuresv1.OpenDebtCountRequest{}))
		if err != nil {
			return runResult{}, cliapp.WrapAPIError("run open debt count", err, nil)
		}
		return runResult{Measure: measure, Value: fmt.Sprintf("%d", resp.Msg.Count)}, nil
	case "template.deep-validate-green-streak":
		resp, err := h.client.DeepValidateGreenStreak(context.Background(), connect.NewRequest(&tmmeasuresv1.DeepValidateGreenStreakRequest{TemplateId: ctx.Flag("template")}))
		if err != nil {
			return runResult{}, cliapp.WrapAPIError("run deep validate green streak", err, nil)
		}
		return runResult{Measure: measure, Value: fmt.Sprintf("%d", resp.Msg.Streak)}, nil
	case "template.fleet-standing-distribution":
		resp, err := h.client.FleetStandingDistribution(context.Background(), connect.NewRequest(&tmmeasuresv1.FleetStandingDistributionRequest{}))
		if err != nil {
			return runResult{}, cliapp.WrapAPIError("run fleet standing distribution", err, nil)
		}
		buckets := make([]map[string]string, 0, len(resp.Msg.Buckets))
		for _, bucket := range resp.Msg.Buckets {
			buckets = append(buckets, map[string]string{"standing": bucket.Standing, "count": fmt.Sprintf("%d", bucket.Count)})
		}
		return runResult{Measure: measure, Buckets: buckets}, nil
	case "template.max-version-lag":
		resp, err := h.client.MaxVersionLag(context.Background(), connect.NewRequest(&tmmeasuresv1.MaxVersionLagRequest{}))
		if err != nil {
			return runResult{}, cliapp.WrapAPIError("run max version lag", err, nil)
		}
		return runResult{Measure: measure, Value: fmt.Sprintf("%d", resp.Msg.Lag)}, nil
	default:
		return runResult{}, fmt.Errorf("unknown measure %q", measure)
	}
}

func (h *handlers) runReport(_ cliapp.OperationContext, result runResult) cliapp.MutationReport {
	lines := []string{}
	if result.Value != "" {
		lines = append(lines, fmt.Sprintf("value=%s", result.Value))
	}
	for _, bucket := range result.Buckets {
		lines = append(lines, fmt.Sprintf("%s=%s", bucket["standing"], bucket["count"]))
	}
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Ran %s.", result.Measure)},
		Changes: lines,
	}
}
