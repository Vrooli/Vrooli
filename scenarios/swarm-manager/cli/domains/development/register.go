package development

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/cli/internal/support"
)

func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "development", Description: "Retained development contracts and operator decisions (never launches work)", Subcommands: []cliapp.Command{getCommand(), decisionCommand("approve"), decisionCommand("revoke"), decisionCommand("accept"), artifactCommand()}}
}

func client(op cliapp.OperationContext) apiconnect.DevelopmentServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewDevelopmentServiceClient(h, base)
}

func getCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "get", NeedsAPI: true, Description: "Inspect retained approval, limits, goal message and launch blockers", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "item", Required: true, Description: "Canonical kind/name backlog reference"}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*api.DevelopmentResponse, error) {
			response, err := client(op).GetDevelopment(context.Background(), connect.NewRequest(&api.GetDevelopmentRequest{WorkItem: op.Flag("item")}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, value *api.DevelopmentResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: summary(value), ResultsHeading: "Goal message and launch blockers", Results: append([]string{value.GoalMessage}, value.LaunchBlockers...)}
		},
	))
}

func decisionCommand(name string) cliapp.Command {
	cmd := cliapp.Command{Name: name, NeedsAPI: true, Description: name + " a version-bound development decision; does not launch work", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "file", Required: true, Description: "Typed decision request JSON with expected_version"}}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*api.DevelopmentResponse, error) {
			var response *connect.Response[api.DevelopmentResponse]
			var err error
			switch name {
			case "approve":
				request := &api.ApproveDevelopmentRequest{}
				if err := support.ReadProtoFile(op.Flag("file"), request); err != nil {
					return nil, err
				}
				response, err = client(op).ApproveDevelopment(context.Background(), connect.NewRequest(request))
			case "revoke":
				request := &api.RevokeDevelopmentRequest{}
				if err := support.ReadProtoFile(op.Flag("file"), request); err != nil {
					return nil, err
				}
				response, err = client(op).RevokeDevelopment(context.Background(), connect.NewRequest(request))
			case "accept":
				request := &api.AcceptDevelopmentRequest{}
				if err := support.ReadProtoFile(op.Flag("file"), request); err != nil {
					return nil, err
				}
				response, err = client(op).AcceptDevelopment(context.Background(), connect.NewRequest(request))
			default:
				return nil, fmt.Errorf("unsupported decision %q", name)
			}
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, value *api.DevelopmentResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Decision retained. No agent launched."}, Changes: append(summary(value), value.LaunchBlockers...)}
		},
	))
}

func artifactCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "artifact", NeedsAPI: true, Description: "Read exact retained artifact bytes, including a prior approved revision", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "item", Required: true}, {Name: "digest", Required: true}, {Name: "path", Required: true}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*api.GetDevelopmentArtifactResponse, error) {
			r, err := client(op).GetDevelopmentArtifact(context.Background(), connect.NewRequest(&api.GetDevelopmentArtifactRequest{WorkItem: op.Flag("item"), Digest: op.Flag("digest"), Path: op.Flag("path")}))
			if err != nil {
				return nil, err
			}
			return r.Msg, nil
		},
		func(_ cliapp.OperationContext, value *api.GetDevelopmentArtifactResponse) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{value.GetArtifact().GetPath(), "sha256:" + value.GetArtifact().GetSha256()}, ResultsHeading: "Retained content", Results: []string{string(value.Content)}}
		},
	))
}

func summary(value *api.DevelopmentResponse) []string {
	result := []string{
		fmt.Sprintf("%s · %s · %s · version %d", value.WorkItem, value.WorkShape, value.Status, value.Version),
		"Approved target: " + value.Digest,
		fmt.Sprintf("Tokens incurred/reserved: %d/%d; wall seconds incurred/reserved: %d/%d", value.GetUsed().GetTokens(), value.GetReserved().GetTokens(), value.GetUsed().GetWallSeconds(), value.GetReserved().GetWallSeconds()),
	}
	if campaign := value.GetCampaign(); campaign != nil {
		result = append(result, fmt.Sprintf("Campaign approval/owner: %s/%s; remaining outcomes: %s; pending: %t", campaign.GetApprovalDigest(), campaign.GetOwnerExecutionId(), strings.Join(campaign.GetRemainingOutcomeIds(), ", "), campaign.GetPending()))
	}
	if cancellation := value.GetCancellation(); cancellation != nil {
		result = append(result, "Cancellation "+cancellation.GetState()+": "+cancellation.GetReason())
	}
	return result
}
