// Package workflows wires the BAS `workflows` CLI group to the proto
// WorkflowsService via the generic protodispatch dispatcher.
//
// The manifest (cli/manifest.json) is the single source of truth for
// command shape. WorkflowsService lives in the
// browser_automation_studio.v1 namespace alongside ExecutionsService;
// both share the same proto file (api/service.proto).
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc): the prior hand-coded
// REST-driven workflow handlers (execute.go, list.go, etc.) targeted
// chi routes that have been deleted from the API; this register
// supersedes them.
//
// step_parser.go and workflow_builder.go remain in this package as pure
// helpers used by the deprecated `execute.go` path. They are kept while
// any embedder still imports them; their unit tests still pass.
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	bascompat "github.com/vrooli/browser-automation-studio/compat"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/encoding/protojson"
)

// GroupName is the manifest group this register package owns.
const GroupName = "workflows"

// Register builds the workflows subcommand group from the embedded
// manifest, binding every command to a generic protodispatch handler
// resolved against the generated WorkflowsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := apiv1.File_browser_automation_studio_v1_api_service_proto.Services().ByName("WorkflowsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: WorkflowsService descriptor not found", GroupName)
	}
	bindings, err := cliapp.ProtoBindings(core, svc.FullName(), protoBindingOptions(string(svc.Name())))
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	for i := range group.Subcommands {
		if group.Subcommands[i].Name != "execute-adhoc" || group.Subcommands[i].RunCtx == nil {
			continue
		}
		group.Subcommands[i].RunCtx = runAdhoc
	}
	return group, nil
}

func protoBindingOptions(serviceName string) cliapp.ProtoBindingOptions {
	return cliapp.ProtoBindingOptions{Normalize: map[string]func([]byte) ([]byte, error){
		serviceName + ".ExecuteAdhocWorkflow": normalizeAdhocFlowFile,
	}}
}

func normalizeAdhocFlowFile(body []byte) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	for _, key := range []string{"flow_definition", "flowDefinition"} {
		if flow, ok := envelope[key]; ok && len(flow) > 0 && string(flow) != "null" {
			body = flow
			break
		}
	}
	return bascompat.NormalizeWorkflowDefinitionV2Bytes(body)
}

func runAdhoc(ctx cliapp.RunContext) error {
	flowBody, envelope, err := readAdhocFlowFile(ctx.Flag("flow-file"))
	if err != nil {
		return fmt.Errorf("read --flow-file: %w", err)
	}
	normalized, err := normalizeAdhocFlowFile(flowBody)
	if err != nil {
		return fmt.Errorf("normalize --flow-file: %w", err)
	}
	flow := &basworkflows.WorkflowDefinitionV2{}
	if err := protojson.Unmarshal(normalized, flow); err != nil {
		return fmt.Errorf("decode --flow-file: %w", err)
	}

	req := &basexecution.ExecuteAdhocRequest{
		FlowDefinition:    flow,
		WaitForCompletion: ctx.BoolFlag("wait"),
		Options: &basexecution.ExecuteWorkflowOptions{
			RequiresVideo:  ctx.BoolFlag("requires-video"),
			RequiresTrace:  ctx.BoolFlag("requires-trace"),
			RequiresHar:    ctx.BoolFlag("requires-har"),
			FrameStreaming: ctx.BoolFlag("frame-streaming"),
			SeedMode:       ctx.Flag("seed-mode"),
			SeedScenario:   ctx.Flag("seed-scenario"),
		},
	}
	name, description := adhocMetadata(ctx, envelope)
	if name != "" || description != "" {
		req.Metadata = &basexecution.ExecutionMetadata{Name: name, Description: description}
	}
	if parametersFile := ctx.Flag("parameters-file"); parametersFile != "" {
		body, err := os.ReadFile(parametersFile)
		if err != nil {
			return fmt.Errorf("read --parameters-file: %w", err)
		}
		req.Parameters = &basexecution.ExecutionParameters{}
		if err := protojson.Unmarshal(body, req.Parameters); err != nil {
			return fmt.Errorf("decode --parameters-file: %w", err)
		}
	}
	if req.Parameters == nil {
		req.Parameters = &basexecution.ExecutionParameters{}
	}
	if req.Parameters.ProjectRoot == nil {
		if projectRoot := findAdhocProjectRoot(ctx.Flag("flow-file")); projectRoot != "" {
			req.Parameters.ProjectRoot = &projectRoot
		}
	}

	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(ctx.Core(), 0)
	client := apiconnect.NewWorkflowsServiceClient(httpClient, baseURL)
	resp, err := client.ExecuteAdhocWorkflow(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("WorkflowsService.ExecuteAdhocWorkflow", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("WorkflowsService.ExecuteAdhocWorkflow: server returned no response")
	}
	return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
}

func findAdhocProjectRoot(flowFile string) string {
	current, err := filepath.Abs(filepath.Dir(flowFile))
	if err != nil {
		return ""
	}
	for {
		for _, relative := range []string{
			filepath.Join("ui", "src", "consts", "selectors.manifest.json"),
			filepath.Join("ui", "src", "constants", "selectors.manifest.json"),
		} {
			if info, err := os.Stat(filepath.Join(current, relative)); err == nil && !info.IsDir() {
				return current
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

func readAdhocFlowFile(path string) ([]byte, map[string]json.RawMessage, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, nil, err
	}
	for _, key := range []string{"flow_definition", "flowDefinition"} {
		if flow, ok := envelope[key]; ok && len(flow) > 0 && string(flow) != "null" {
			return flow, envelope, nil
		}
	}
	return body, envelope, nil
}

func adhocMetadata(ctx cliapp.RunContext, envelope map[string]json.RawMessage) (string, string) {
	name := ctx.Flag("name")
	description := ctx.Flag("description")
	var metadata struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if raw, ok := envelope["metadata"]; ok {
		_ = json.Unmarshal(raw, &metadata)
	}
	if name == "" {
		name = metadata.Name
	}
	if description == "" {
		description = metadata.Description
	}
	return name, description
}
