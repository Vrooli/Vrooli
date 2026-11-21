// Package deploytarget provides generated-Connect commands for release targets.
package deploytarget

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

type deployTargetRPC interface {
	ListDeployTargets(context.Context, *connect.Request[domainv1.ListDeployTargetsRequest]) (*connect.Response[domainv1.ListDeployTargetsResponse], error)
	GetDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.GetDeployTargetResponse], error)
	SaveDeployTarget(context.Context, *connect.Request[domainv1.SaveDeployTargetRequest]) (*connect.Response[domainv1.SaveDeployTargetResponse], error)
	DeleteDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DeleteDeployTargetResponse], error)
	TestDeployTarget(context.Context, *connect.Request[domainv1.TestDeployTargetRequest]) (*connect.Response[domainv1.TestDeployTargetResponse], error)
	DiagnoseDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DiagnoseDeployTargetResponse], error)
}

type Commands struct{ rpc deployTargetRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewDeployTargetServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	nameArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true, Description: "Deploy target name"}}}
	return cliapp.SubcommandGroup{Name: "deploy-target", Description: "Manage durable desktop deployment targets", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List saved deployment targets"}).WithPrimitive(c.listPrimitive()),
		(cliapp.Command{Name: "get", Description: "Get a deployment target", Args: nameArgs}).WithPrimitive(c.getPrimitive()),
		(cliapp.Command{Name: "add", Description: "Create or update a deployment target", Args: cliapp.ArgSchema{Positionals: nameArgs.Positionals, Flags: []cliapp.Flag{{Name: "scenario", Required: true}, {Name: "profile", Required: true}, {Name: "label"}, {Name: "deployment-manager-profile-id"}}}}).WithPrimitive(c.savePrimitive()),
		(cliapp.Command{Name: "remove", Description: "Remove a deployment target", Args: nameArgs}).WithPrimitive(c.deletePrimitive()),
		(cliapp.Command{Name: "test", Description: "Verify deployment-target connectivity", Args: cliapp.ArgSchema{Positionals: nameArgs.Positionals, Flags: []cliapp.Flag{{Name: "require-service-auth", Bool: true}}}}).WithPrimitive(c.testPrimitive()),
		(cliapp.Command{Name: "doctor", Description: "Diagnose deployment-target readiness", Args: nameArgs}).WithPrimitive(c.doctorPrimitive()),
	}}
}

func (c *Commands) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.GetDeployTargetResponse, error) {
		response, err := c.rpc.GetDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: ctx.Positional("name")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get deployment target", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.GetDeployTargetResponse) cliapp.ListReport {
		target := response.GetTarget()
		return cliapp.ListReport{Summary: []string{"Deployment target: " + target.GetName()}, ResultsHeading: "Configuration", Results: []string{fmt.Sprintf("Scenario: %s", target.GetScenarioName()), fmt.Sprintf("Profile: %s", target.GetRemoteProfile())}}
	})
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(cliapp.OperationContext) (*domainv1.ListDeployTargetsResponse, error) {
		response, err := c.rpc.ListDeployTargets(context.Background(), connect.NewRequest(&domainv1.ListDeployTargetsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list deployment targets", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListDeployTargetsResponse) cliapp.ListReport {
		report := cliapp.ListReport{Summary: []string{fmt.Sprintf("Deployment targets: %d", len(response.GetTargets()))}, ResultsHeading: "Targets", RetrievalHints: []string{"Add one with `scenario-to-desktop deploy-target add <name> --scenario <scenario> --profile <profile>`."}}
		for _, target := range response.GetTargets() {
			label := target.GetLabel()
			if label == "" {
				label = target.GetName()
			}
			report.Results = append(report.Results, fmt.Sprintf("%s | label=%q | scenario=%s | profile=%s", target.GetName(), label, target.GetScenarioName(), target.GetRemoteProfile()))
		}
		return report
	})
}

func (c *Commands) savePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.SaveDeployTargetResponse, error) {
		target := &domainv1.DeployTarget{Name: strings.TrimSpace(ctx.Positional("name")), Label: strings.TrimSpace(ctx.Flag("label")), ScenarioName: strings.TrimSpace(ctx.Flag("scenario")), RemoteProfile: strings.TrimSpace(ctx.Flag("profile"))}
		if id := strings.TrimSpace(ctx.Flag("deployment-manager-profile-id")); id != "" {
			target.DeploymentManagerProfileId = &id
		}
		response, err := c.rpc.SaveDeployTarget(context.Background(), connect.NewRequest(&domainv1.SaveDeployTargetRequest{Target: target}))
		if err != nil {
			return nil, cliapp.WrapAPIError("save deployment target", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.SaveDeployTargetResponse) cliapp.MutationReport {
		target := response.GetTarget()
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deployment target %q saved.", target.GetName())}, Changes: []string{"Scenario: " + target.GetScenarioName(), "Profile: " + target.GetRemoteProfile()}, NextCommand: []string{"scenario-to-desktop deploy-target doctor " + target.GetName()}}
	})
}

func (c *Commands) deletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.DeleteDeployTargetResponse, error) {
		response, err := c.rpc.DeleteDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: ctx.Positional("name")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete deployment target", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.DeleteDeployTargetResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deployment target %q removed.", response.GetName())}, NextCommand: []string{"scenario-to-desktop deploy-target list"}}
	})
}

func (c *Commands) testPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoOperational(func(ctx cliapp.OperationContext) (*domainv1.TestDeployTargetResponse, error) {
		response, err := c.rpc.TestDeployTarget(context.Background(), connect.NewRequest(&domainv1.TestDeployTargetRequest{Name: ctx.Positional("name"), RequireServiceAuth: ctx.BoolFlag("require-service-auth")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("test deployment target", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.TestDeployTargetResponse) cliapp.OperationalReport {
		return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Deployment target %q connectivity verified.", response.GetTarget().GetName())}, NextSteps: []string{"scenario-to-desktop deploy-target doctor " + response.GetTarget().GetName()}}
	})
}

func (c *Commands) doctorPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoOperational(func(ctx cliapp.OperationContext) (*domainv1.DiagnoseDeployTargetResponse, error) {
		response, err := c.rpc.DiagnoseDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: ctx.Positional("name")}))
		if err != nil {
			return nil, cliapp.WrapAPIError("diagnose deployment target", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.DiagnoseDeployTargetResponse) cliapp.OperationalReport {
		status := "NOT READY"
		if response.GetReady() {
			status = "READY"
		}
		report := cliapp.OperationalReport{Status: []string{"Status: " + status, fmt.Sprintf("Target: %s (scenario=%s profile=%s)", response.GetTarget().GetName(), response.GetTarget().GetScenarioName(), response.GetTarget().GetRemoteProfile())}, NextSteps: response.GetNextSteps()}
		for _, check := range response.GetChecks() {
			state := "PASS"
			if check.GetBlocked() {
				state = "BLOCKED"
			} else if !check.GetPassed() {
				state = "FAIL"
			}
			report.Triage = append(report.Triage, cliapp.TriageGroup{Heading: check.GetName(), Items: []string{fmt.Sprintf("[%s] %s", state, check.GetDetail())}})
		}
		return report
	})
}
