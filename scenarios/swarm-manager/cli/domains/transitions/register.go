package transitions

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"swarm-manager/cli/internal/support"
)

func Register(_ support.Dependencies) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "transitions", Description: "Declared agent transition catalog", Subcommands: []cliapp.Command{listCommand(), startCommand(), applyCommand()}}
}

func client(op cliapp.OperationContext) apiconnect.TransitionServiceClient {
	h, base := cliapp.NewConnectHTTPClient(op.Core())
	return apiconnect.NewTransitionServiceClient(h, base)
}

func listCommand() cliapp.Command {
	return cliapp.Command{Name: "list", NeedsAPI: true, Description: "List declared transition capabilities"}.WithPrimitive(cliapp.ProtoList(
		func(op cliapp.OperationContext) (*api.ListTransitionsResponse, error) {
			response, err := client(op).ListTransitions(context.Background(), connect.NewRequest(&api.ListTransitionsRequest{}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *api.ListTransitionsResponse) cliapp.ListReport {
			rows := make([]string, 0, len(response.GetTransitions()))
			for _, transition := range response.GetTransitions() {
				rows = append(rows, fmt.Sprintf("%s — %s / %s", transition.GetKey(), transition.GetSubject(), transition.GetKind().String()))
			}
			return cliapp.ListReport{Summary: []string{fmt.Sprintf("Declared transitions: %d", len(rows))}, ResultsHeading: "Transitions", Results: rows}
		},
	))
}

func startCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "start", NeedsAPI: true, Description: "Start a declared transition (--transition KEY --subject SUBJECT --subject-ref REF) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "transition", Description: "Declared transition key", Required: true},
		{Name: "subject", Description: "Declared subject kind from transitions list", Required: true},
		{Name: "subject-ref", Description: "Typed subject reference", Required: true},
	}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*api.StartTransitionResponse, error) {
			response, err := client(op).StartTransition(context.Background(), connect.NewRequest(&api.StartTransitionRequest{TransitionKey: strings.TrimSpace(op.Flag("transition")), SubjectRef: &api.SubjectReference{Subject: strings.TrimSpace(op.Flag("subject")), Value: strings.TrimSpace(op.Flag("subject-ref"))}}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *api.StartTransitionResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Transition started."}, Changes: []string{fmt.Sprintf("Execution: %s", response.GetExecutionId()), fmt.Sprintf("Entity version: %s", response.GetEntityVersion())}, NextCommand: []string{fmt.Sprintf("swarm-manager transitions apply --transition <key> --execution-id %s", response.GetExecutionId())}}
		},
	))
}

func applyCommand() cliapp.Command {
	cmd := cliapp.Command{Name: "apply", NeedsAPI: true, Description: "Apply a completed declared transition (--transition KEY --execution-id ID) [--json]", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "transition", Description: "Declared transition key", Required: true},
		{Name: "execution-id", Description: "Workflow execution id", Required: true},
	}}}
	return cmd.WithPrimitive(cliapp.ProtoMutation(
		func(op cliapp.OperationContext) (*api.ApplyTransitionResponse, error) {
			response, err := client(op).ApplyTransition(context.Background(), connect.NewRequest(&api.ApplyTransitionRequest{TransitionKey: strings.TrimSpace(op.Flag("transition")), ExecutionId: strings.TrimSpace(op.Flag("execution-id"))}))
			if err != nil {
				return nil, err
			}
			return response.Msg, nil
		},
		func(_ cliapp.OperationContext, response *api.ApplyTransitionResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{"Transition result applied."}, Changes: []string{fmt.Sprintf("Outcome: %s", response.GetOutcome()), fmt.Sprintf("Subject: %s", response.GetSubjectRef())}}
		},
	))
}
