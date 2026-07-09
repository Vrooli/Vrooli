package resourcetemplate

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	resourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template"
	resourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template/resource_template_v1connect"
)

type handlers struct {
	client resourceconnect.ResourceTemplateServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: resourceconnect.NewResourceTemplateServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(_ cliapp.OperationContext) (*resourcev1.ListResourceTemplatesResponse, error) {
	resp, err := h.client.ListResourceTemplates(context.Background(), connect.NewRequest(&resourcev1.ListResourceTemplatesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list resource templates", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *resourcev1.ListResourceTemplatesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Templates))
	for _, item := range msg.Templates {
		results = append(results, formatTemplate(item))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d resource template(s).", len(msg.Templates))}, ResultsHeading: "Resource templates", Results: results}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*resourcev1.GetResourceTemplateResponse, error) {
	resp, err := h.client.GetResourceTemplate(context.Background(), connect.NewRequest(&resourcev1.GetResourceTemplateRequest{Name: ctx.Positional("name")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("show resource template", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *resourcev1.GetResourceTemplateResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Fetched resource template."}, ResultsHeading: "Resource template", Results: []string{formatTemplate(msg.Template)}}
}

func (h *handlers) validateCall(_ cliapp.OperationContext) (*resourcev1.ValidateResourceTemplatesResponse, error) {
	resp, err := h.client.ValidateResourceTemplates(context.Background(), connect.NewRequest(&resourcev1.ValidateResourceTemplatesRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("validate resource templates", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) validateReport(_ cliapp.OperationContext, msg *resourcev1.ValidateResourceTemplatesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Templates))
	for _, item := range msg.Templates {
		results = append(results, fmt.Sprintf("%s driver=%s transitional=%t", item.Name, item.Driver, item.Transitional))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Validated %d resource template(s).", msg.Count)}, ResultsHeading: "Templates", Results: results}
}

func (h *handlers) generateCall(ctx cliapp.OperationContext) (*resourcev1.GenerateResourceTemplateResponse, error) {
	values, err := parseVars(ctx.FlagValues("var"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.GenerateResourceTemplate(context.Background(), connect.NewRequest(&resourcev1.GenerateResourceTemplateRequest{
		Template:      ctx.Positional("template"),
		FromBlueprint: ctx.Flag("from-blueprint"),
		Destination:   ctx.Flag("dest"),
		Force:         ctx.BoolFlag("force"),
		DryRun:        ctx.BoolFlag("dry-run"),
		Values:        values,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("generate resource template", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) generateReport(_ cliapp.OperationContext, msg *resourcev1.GenerateResourceTemplateResponse) cliapp.MutationReport {
	action := "Generated"
	if msg.DryRun {
		action = "Would generate"
	}
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s resource template %s at %s.", action, msg.Template.Name, msg.Destination)},
		Changes: []string{fmt.Sprintf("blueprint=%s files=%d dry_run=%t", msg.BlueprintName, len(msg.Files), msg.DryRun)},
	}
}

func formatTemplate(item *resourcev1.ResourceTemplateInfo) string {
	if item == nil || item.Manifest == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s display=%q driver=%s required=%d optional=%d", item.Name, item.Manifest.DisplayName, item.Manifest.Driver, len(item.Manifest.RequiredVars), len(item.Manifest.OptionalVars))
}

func parseVars(values []string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--var must be KEY=VALUE")
		}
		out[key] = val
	}
	return out, nil
}
