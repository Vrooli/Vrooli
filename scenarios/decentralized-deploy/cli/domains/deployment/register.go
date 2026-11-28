package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	deploymentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment"
	deploymentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment/deployment_v1connect"
)

const GroupName = "deployment"

type handlers struct {
	client deploymentconnect.DeploymentServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: deploymentconnect.NewDeploymentServiceClient(httpClient, baseURL)}
	service := deploymentconnect.DeploymentServiceName + "."
	bindings := map[string]cliapp.PrimitiveHandler{
		service + "Deploy":   cliapp.ProtoMutation(h.deployCall, h.deployReport),
		service + "Get":      cliapp.ProtoList(h.getCall, h.getReport),
		service + "Resolve":  cliapp.ProtoList(h.resolveCall, h.resolveReport),
		service + "Renew":    cliapp.ProtoMutation(h.renewCall, h.renewReport),
		service + "Teardown": cliapp.ProtoMutation(h.teardownCall, h.teardownReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("deployment: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) deployCall(ctx cliapp.OperationContext) (*deploymentv1.DeployResponse, error) {
	resp, err := h.client.Deploy(context.Background(), connect.NewRequest(&deploymentv1.DeployRequest{Id: ctx.Flag("id"), Name: ctx.Flag("name"), ComputeReference: ctx.Flag("compute-reference"), StorageProvider: ctx.Flag("storage-provider")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("deploy reference bundle", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Deployment == nil {
		return nil, fmt.Errorf("server returned no deployment receipt")
	}
	return resp.Msg, nil
}

func (h *handlers) deployReport(_ cliapp.OperationContext, msg *deploymentv1.DeployResponse) cliapp.MutationReport {
	d := msg.GetDeployment()
	if d.GetStorageProvider() == "turbo-sandbox" {
		return cliapp.MutationReport{
			Result:  []string{fmt.Sprintf("Uploaded synthetic reference content to the free ar.io sandbox as %s.", d.GetId())},
			Changes: []string{formatReceipt(d), "Sandbox content is public and ephemeral; this scenario cannot guarantee remote deletion."},
			NextCommand: []string{
				fmt.Sprintf("`decentralized-deploy deployment resolve %s` — verify exact content retrieval", d.GetId()),
				fmt.Sprintf("`decentralized-deploy deployment teardown %s` — remove the local lifecycle receipt (the sandbox copy may remain until purged)", d.GetId()),
			},
		}
	}
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Deployed reference %s to %s.", d.GetId(), d.GetContentUri())},
		Changes: []string{formatReceipt(d)},
		NextCommand: []string{
			fmt.Sprintf("`decentralized-deploy deployment resolve %s` — verify exact content retrieval", d.GetId()),
			fmt.Sprintf("`decentralized-deploy deployment renew %s` — renew the active references", d.GetId()),
		},
	}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*deploymentv1.GetResponse, error) {
	resp, err := h.client.Get(context.Background(), connect.NewRequest(&deploymentv1.GetRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get deployment", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Deployment == nil {
		return nil, fmt.Errorf("server returned no deployment receipt")
	}
	return resp.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, msg *deploymentv1.GetResponse) cliapp.ListReport {
	d := msg.GetDeployment()
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Deployment %s.", d.GetId())}, ResultsHeading: "Lifecycle receipt", Results: []string{formatReceipt(d)}}
}

func (h *handlers) resolveCall(ctx cliapp.OperationContext) (*deploymentv1.ResolveResponse, error) {
	resp, err := h.client.Resolve(context.Background(), connect.NewRequest(&deploymentv1.ResolveRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("resolve deployment", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Deployment == nil {
		return nil, fmt.Errorf("server returned no resolved deployment")
	}
	return resp.Msg, nil
}

func (h *handlers) resolveReport(_ cliapp.OperationContext, msg *deploymentv1.ResolveResponse) cliapp.ListReport {
	d := msg.GetDeployment()
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Resolved %d exact bytes for %s (%s).", len(msg.GetContent()), d.GetId(), d.GetContentUri())},
		ResultsHeading: "Resolved content",
		Results:        []string{string(msg.GetContent())},
	}
}

func (h *handlers) renewCall(ctx cliapp.OperationContext) (*deploymentv1.RenewResponse, error) {
	resp, err := h.client.Renew(context.Background(), connect.NewRequest(&deploymentv1.RenewRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("renew deployment", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Deployment == nil {
		return nil, fmt.Errorf("server returned no renewed deployment receipt")
	}
	return resp.Msg, nil
}

func (h *handlers) renewReport(_ cliapp.OperationContext, msg *deploymentv1.RenewResponse) cliapp.MutationReport {
	d := msg.GetDeployment()
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Renewed deployment %s.", d.GetId())}, Changes: []string{formatReceipt(d)}, NextCommand: []string{fmt.Sprintf("`decentralized-deploy deployment get %s` — inspect the receipt", d.GetId())}}
}

func (h *handlers) teardownCall(ctx cliapp.OperationContext) (*deploymentv1.TeardownResponse, error) {
	resp, err := h.client.Teardown(context.Background(), connect.NewRequest(&deploymentv1.TeardownRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("teardown deployment", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Deployment == nil {
		return nil, fmt.Errorf("server returned no torn-down deployment receipt")
	}
	return resp.Msg, nil
}

func (h *handlers) teardownReport(_ cliapp.OperationContext, msg *deploymentv1.TeardownResponse) cliapp.MutationReport {
	d := msg.GetDeployment()
	if d.GetStorageProvider() == "turbo-sandbox" {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Removed local lifecycle state for %s; the ar.io sandbox copy may remain until its ephemeral retention window expires.", d.GetId())}, Changes: []string{formatReceipt(d)}}
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Tore down deployment %s.", d.GetId())}, Changes: []string{formatReceipt(d)}, NextCommand: []string{"`decentralized-deploy deployment deploy --id <stable-id> --name <ens.eth-or-arns-name>` — create another deployment"}}
}

func formatReceipt(d *deploymentv1.DeploymentReceipt) string {
	result := fmt.Sprintf("id=%s state=%s compute=%s storage=%s uri=%s cid=%s", d.GetId(), d.GetState(), d.GetComputeReference(), d.GetStorageProvider(), d.GetContentUri(), d.GetContentId())
	for _, name := range d.GetNames() {
		result += fmt.Sprintf("\n%s name=%s target=%s", name.GetProvider(), name.GetName(), name.GetTarget())
	}
	if len(d.GetNames()) == 0 {
		result += "\nnaming adapters: none configured"
	}
	return result
}
