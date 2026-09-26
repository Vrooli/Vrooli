package registry

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry/registry_v1connect"
)

type handlers struct {
	client registryconnect.RegistryServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: registryconnect.NewRegistryServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*registryv1.ListTemplatesResponse, error) {
	kind, err := parseKind(ctx.Flag("kind"))
	if err != nil {
		return nil, err
	}
	resp, err := h.client.ListTemplates(context.Background(), connect.NewRequest(&registryv1.ListTemplatesRequest{Kind: kind}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list templates", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *registryv1.ListTemplatesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Templates))
	for _, record := range msg.Templates {
		results = append(results, formatTemplate(record))
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d template record(s).", len(msg.Templates))},
		ResultsHeading: "Templates",
		Results:        results,
		RetrievalHints: []string{"`registry show <id>` - show one template record"},
	}
}

func (h *handlers) showCall(ctx cliapp.OperationContext) (*registryv1.GetTemplateResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetTemplate(context.Background(), connect.NewRequest(&registryv1.GetTemplateRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("show template %q", id), err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) showReport(_ cliapp.OperationContext, msg *registryv1.GetTemplateResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched template %s.", msg.Template.Id)},
		ResultsHeading: "Template",
		Results:        []string{formatTemplate(msg.Template)},
	}
}

func parseKind(raw string) (registryv1.TemplateKind, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "":
		return registryv1.TemplateKind_TEMPLATE_KIND_UNSPECIFIED, nil
	case "scenario":
		return registryv1.TemplateKind_TEMPLATE_KIND_SCENARIO, nil
	case "design":
		return registryv1.TemplateKind_TEMPLATE_KIND_DESIGN, nil
	case "resource":
		return registryv1.TemplateKind_TEMPLATE_KIND_RESOURCE, nil
	default:
		return registryv1.TemplateKind_TEMPLATE_KIND_UNSPECIFIED, fmt.Errorf("unknown template kind %q (use scenario, design, or resource)", raw)
	}
}

func formatTemplate(record *registryv1.TemplateRecord) string {
	if record == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s [%s] version=%s status=%s source=%s", record.Id, strings.TrimPrefix(record.Kind.String(), "TEMPLATE_KIND_"), record.Version, record.Status, record.SourcePath)
}
