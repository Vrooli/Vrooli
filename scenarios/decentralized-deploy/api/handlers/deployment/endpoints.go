package deployment

import (
	"decentralized-deploy/internal/module"
	deploymentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/deployment/deployment_v1connect"
)

var ConnectEndpoints = []module.EndpointDescriptor{
	{
		ID: "deployment_rpc_deploy", Path: deploymentconnect.DeploymentServiceDeployProcedure, Method: "POST",
		Summary: "Deploy the scenario reference bundle to the configured local storage and naming adapters.", Category: "deployment",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string", "name": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deployment": "DeploymentReceipt"}},
	},
	{
		ID: "deployment_rpc_get", Path: deploymentconnect.DeploymentServiceGetProcedure, Method: "POST",
		Summary: "Read a persisted deployment lifecycle receipt.", Category: "deployment",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deployment": "DeploymentReceipt"}},
	},
	{
		ID: "deployment_rpc_resolve", Path: deploymentconnect.DeploymentServiceResolveProcedure, Method: "POST",
		Summary: "Resolve a deployment's content URI and return the exact stored bytes.", Category: "deployment",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deployment": "DeploymentReceipt", "content": "bytes"}},
	},
	{
		ID: "deployment_rpc_renew", Path: deploymentconnect.DeploymentServiceRenewProcedure, Method: "POST",
		Summary: "Renew the configured storage and naming adapter records.", Category: "deployment",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deployment": "DeploymentReceipt"}},
	},
	{
		ID: "deployment_rpc_teardown", Path: deploymentconnect.DeploymentServiceTeardownProcedure, Method: "POST",
		Summary: "Remove deployment names and owned storage state.", Category: "deployment",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"deployment": "DeploymentReceipt"}},
	},
}

