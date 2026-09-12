package identities

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

// Register exposes the minimum agent-facing ingress: inspect identities,
// author a product, and import canon. Richer render/conformance verbs retain
// their generated Connect contracts and can be added without changing this API.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := studioconnect.NewStudioServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{Name: "identities", Description: "Author and inspect reusable visual identities", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List identity versions", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoList}}).WithPrimitive(cliapp.ProtoList(
			func(_ cliapp.OperationContext) (*studiov1.ListIdentitiesResponse, error) {
				resp, err := client.ListIdentities(context.Background(), connect.NewRequest(&studiov1.ListIdentitiesRequest{}))
				if err != nil {
					return nil, cliapp.WrapAPIError("list identities", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.ListIdentitiesResponse) cliapp.ListReport {
				rows := make([]string, 0, len(msg.Identities))
				for _, i := range msg.Identities {
					rows = append(rows, fmt.Sprintf("%s — %s v%d", i.Id, i.Kind, i.Version))
				}
				return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d identity version(s).", len(rows))}, ResultsHeading: "Identities", Results: rows}
			},
		)),
		(cliapp.Command{Name: "create-product", Description: "Create a product identity", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "name", Required: true, Description: "Product identity name"}, {Name: "form", Required: true, Description: "Physical or UI form"}, {Name: "finish", Required: true, Description: "Material or visual finish"}}}}).WithPrimitive(cliapp.ProtoMutation(
			func(ctx cliapp.OperationContext) (*studiov1.CreateIdentityResponse, error) {
				resp, err := client.CreateIdentity(context.Background(), connect.NewRequest(&studiov1.CreateIdentityRequest{Identity: &studiov1.Identity{Name: ctx.Flag("name"), Kind: "product", Traits: map[string]string{"form": ctx.Flag("form"), "finish": ctx.Flag("finish")}, CredentialClaims: ""}, ActorId: "operator-cli", ActorKind: "operator"}))
				if err != nil {
					return nil, cliapp.WrapAPIError("create product identity", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.CreateIdentityResponse) cliapp.MutationReport {
				return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created product identity %s.", msg.Identity.Id)}, Changes: []string{fmt.Sprintf("%s v%d", msg.Identity.Name, msg.Identity.Version)}}
			},
		)),
		(cliapp.Command{Name: "import", Description: "Import authored rich-media canon", Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveProtoMutation}, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "root", Required: true, Description: "Path to rich-media catalogue root"}}}}).WithPrimitive(cliapp.ProtoMutation(
			func(ctx cliapp.OperationContext) (*studiov1.ImportCanonResponse, error) {
				root := strings.TrimSpace(ctx.Flag("root"))
				resp, err := client.ImportCanon(context.Background(), connect.NewRequest(&studiov1.ImportCanonRequest{Root: root}))
				if err != nil {
					return nil, cliapp.WrapAPIError("import canon", err, nil)
				}
				return resp.Msg, nil
			},
			func(_ cliapp.OperationContext, msg *studiov1.ImportCanonResponse) cliapp.MutationReport {
				return cliapp.MutationReport{Result: []string{fmt.Sprintf("Imported %d, revised %d identity records.", msg.Created, msg.Revised)}, Changes: msg.Errors}
			},
		)),
	}}
}
