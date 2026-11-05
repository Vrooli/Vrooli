package development

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"strings"
	"swarm-manager/cli/internal/support"
)

func Register() cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "development", Description: "Retained development contracts and operator decisions (never launches work)", Subcommands: []cliapp.Command{getCommand(), artifactCommand(), approveCommand(), revokeCommand(), acceptCommand()}}
}
func client(op cliapp.OperationContext) apiconnect.DevelopmentServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewDevelopmentServiceClient(h, base)
}
func dataCommand(name, description string) cliapp.Command {
	return cliapp.Command{Name: name, Description: description, NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "data", Description: "Full typed RPC request as JSON; max 128 KiB, no inferred reference or actor", Required: true}}}}
}
func referenceRequired(r *api.DevelopmentReference) error {
	if r == nil || strings.TrimSpace(r.GetEffortId()) == "" {
		return fmt.Errorf("reference.effort_id is required")
	}
	return nil
}
func decisionRequired(r *api.DevelopmentReference, generation *uint64, requestID string) error {
	if err := referenceRequired(r); err != nil {
		return err
	}
	if generation == nil {
		return fmt.Errorf("expected_generation must be explicitly present (zero is preserved)")
	}
	if strings.TrimSpace(requestID) == "" {
		return fmt.Errorf("request_id is required")
	}
	return nil
}
func decisionReport(_ cliapp.OperationContext, r *api.DevelopmentDecisionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Owner development decision retained; no work launched."}, Changes: []string{fmt.Sprintf("Action: %s", r.GetAction()), fmt.Sprintf("Generation: %d", r.GetGeneration()), fmt.Sprintf("Receipt digest: %s", r.GetReceiptDigest())}}
}
func getCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "get", Description: "Inspect retained development and launch blockers (--effort-id ID)", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "effort-id", Required: true, Description: "Exact retained effort identity"}}}}
	return cmd.WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*api.GetDevelopmentResponse, error) {
		id := strings.TrimSpace(op.Flag("effort-id"))
		if id == "" {
			return nil, fmt.Errorf("--effort-id is required")
		}
		r, err := client(op).GetDevelopment(context.Background(), connect.NewRequest(&api.GetDevelopmentRequest{EffortId: id}))
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *api.GetDevelopmentResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Retained development owner view (no launch).", fmt.Sprintf("Effort: %s", r.GetDevelopment().GetReference().GetEffortId()), fmt.Sprintf("Generation: %d", r.GetDevelopment().GetGeneration()), fmt.Sprintf("Approved: %t; revoked: %t; outcome accepted: %t", r.GetDevelopment().GetApproved(), r.GetDevelopment().GetRevoked(), r.GetDevelopment().GetOutcomeAccepted())}, ResultsHeading: "Launch blockers", Results: r.GetDevelopment().GetLaunchBlockers()}
	}))
}
func artifactCommand() cliapp.Command {
	return dataCommand("artifact", "Read exact retained artifact bytes (--data typed JSON)").WithPrimitive(cliapp.ProtoList(func(op cliapp.OperationContext) (*api.GetDevelopmentArtifactResponse, error) {
		req := &api.GetDevelopmentArtifactRequest{}
		if err := support.ReadProtoData(op.Flag("data"), req); err != nil {
			return nil, err
		}
		if err := referenceRequired(req.Reference); err != nil {
			return nil, err
		}
		if strings.TrimSpace(req.ArtifactId) == "" {
			return nil, fmt.Errorf("artifact_id is required")
		}
		r, err := client(op).GetDevelopmentArtifact(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}, func(_ cliapp.OperationContext, r *api.GetDevelopmentArtifactResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("Exact retained artifact: %d bytes (use --json for full typed bytes and metadata).", len(r.GetContent()))}}
	}))
}
func approveCommand() cliapp.Command {
	return dataCommand("approve", "Request owner approve decision without launching work (--data typed JSON)").WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*api.DevelopmentDecisionResponse, error) {
		req := &api.ApproveDevelopmentRequest{}
		if err := support.ReadProtoData(op.Flag("data"), req); err != nil {
			return nil, err
		}
		if err := decisionRequired(req.Reference, req.ExpectedGeneration, req.RequestId); err != nil {
			return nil, err
		}
		r, err := client(op).ApproveDevelopment(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}, decisionReport))
}
func revokeCommand() cliapp.Command {
	return dataCommand("revoke", "Request owner revoke decision without launching work (--data typed JSON)").WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*api.DevelopmentDecisionResponse, error) {
		req := &api.RevokeDevelopmentRequest{}
		if err := support.ReadProtoData(op.Flag("data"), req); err != nil {
			return nil, err
		}
		if err := decisionRequired(req.Reference, req.ExpectedGeneration, req.RequestId); err != nil {
			return nil, err
		}
		r, err := client(op).RevokeDevelopment(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}, decisionReport))
}
func acceptCommand() cliapp.Command {
	return dataCommand("accept", "Request owner accept decision without launching work (--data typed JSON)").WithPrimitive(cliapp.ProtoMutation(func(op cliapp.OperationContext) (*api.DevelopmentDecisionResponse, error) {
		req := &api.AcceptDevelopmentRequest{}
		if err := support.ReadProtoData(op.Flag("data"), req); err != nil {
			return nil, err
		}
		if err := decisionRequired(req.Reference, req.ExpectedGeneration, req.RequestId); err != nil {
			return nil, err
		}
		if req.ExpectedDispositionVersion == nil || strings.TrimSpace(req.TestedProductDigest) == "" {
			return nil, fmt.Errorf("expected_disposition_version and tested_product_digest are required")
		}
		r, err := client(op).AcceptDevelopment(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, err
		}
		return r.Msg, nil
	}, decisionReport))
}
