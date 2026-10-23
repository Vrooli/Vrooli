package surfaces

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
)

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := surfacesconnect.NewSurfaceCatalogServiceClient(httpClient, baseURL)
	apps := surfacesconnect.NewApplicationCatalogServiceClient(httpClient, baseURL)
	return cliapp.LoadFromManifest(manifest, "surfaces", map[string]func(cliapp.RunContext) error{
		"ApplicationCatalogService.ListApplications": func(ctx cliapp.RunContext) error {
			response, err := apps.ListApplications(context.Background(), connect.NewRequest(&surfacesv1.ListApplicationsRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("list applications", err, nil)
			}
			report := cliapp.ListReport{ResultsHeading: "Applications", Summary: []string{"Runtime availability does not establish TV compatibility."}}
			for _, app := range response.Msg.Applications {
				report.Results = append(report.Results, fmt.Sprintf("%s · %s · TV %s", app.Scenario, app.RuntimeState, app.TvQualification))
			}
			return cliapp.RenderProtoList(ctx, response.Msg, report)
		},
		"ApplicationCatalogService.ResolveApplication": func(ctx cliapp.RunContext) error {
			response, err := apps.ResolveApplication(context.Background(), connect.NewRequest(&surfacesv1.ResolveApplicationRequest{Scenario: ctx.Positional("scenario"), UseGateway: ctx.BoolFlag("gateway")}))
			if err != nil {
				return cliapp.WrapAPIError("resolve application", err, nil)
			}
			return cliapp.RenderProtoList(ctx, response.Msg, cliapp.ListReport{Results: []string{response.Msg.Url}})
		},
		"SurfaceCatalogService.List": func(ctx cliapp.RunContext) error {
			response, err := client.List(context.Background(), connect.NewRequest(&surfacesv1.ListRequest{}))
			if err != nil {
				return cliapp.WrapAPIError("list surfaces", err, nil)
			}
			return cliapp.RenderProtoList(ctx, response.Msg, report(response.Msg.Surfaces, response.Msg.Sources))
		},
		"SurfaceCatalogService.Resolve": func(ctx cliapp.RunContext) error {
			request := &surfacesv1.ResolveRequest{DisplayLabel: ctx.Positional("label")}
			if ctx.Flag("target-owner") != "" || ctx.Flag("target-id") != "" || ctx.Flag("host-node-id") != "" || ctx.Flag("surface-owner") != "" || ctx.Flag("surface-id") != "" {
				request.ExactRef = &commonv1.SurfaceRef{Target: &commonv1.TargetRef{OwnerScenario: ctx.Flag("target-owner"), ResourceId: ctx.Flag("target-id"), HostNodeId: ctx.Flag("host-node-id")}, OwnerScenario: ctx.Flag("surface-owner"), SurfaceId: ctx.Flag("surface-id")}
			}
			response, err := client.Resolve(context.Background(), connect.NewRequest(request))
			if err != nil {
				return cliapp.WrapAPIError("resolve surface", err, nil)
			}
			result := report(response.Msg.Matches, response.Msg.Sources)
			if len(response.Msg.Matches) > 1 {
				result.Summary = append(result.Summary, "Ambiguous name: select an exact owner reference before opening a session.")
			}
			return cliapp.RenderProtoList(ctx, response.Msg, result)
		},
	})
}

func report(descriptors []*commonv1.SurfaceDescriptor, sources []*surfacesv1.SourceStatus) cliapp.ListReport {
	result := cliapp.ListReport{Summary: []string{fmt.Sprintf("%d offered surface(s); owner admission is required for access.", len(descriptors))}, ResultsHeading: "Surfaces"}
	for _, source := range sources {
		result.Summary = append(result.Summary, fmt.Sprintf("%s: %s %s", source.OwnerScenario, source.State, source.ReasonCode))
	}
	for _, surface := range descriptors {
		ref := surface.GetRef()
		target := ref.GetTarget()
		result.Results = append(result.Results, fmt.Sprintf("%s — %s target=%s/%s surface=%s/%s", surface.DisplayLabel, surface.Kind.String(), target.GetOwnerScenario(), target.GetResourceId(), ref.GetOwnerScenario(), ref.GetSurfaceId()))
	}
	return result
}
