package conversations

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

const GroupName = "conversations"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h, base := cliapp.NewConnectHTTPClient(core)
	client := connectv1.NewConversationsServiceClient(h, base)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ConversationsService.Ask": func(ctx cliapp.RunContext) error {
			resp, callErr := client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
				Question:         ctx.Flag("question"),
				Deadline:         ctx.Flag("deadline"),
				SensitivityLabel: ctx.Flag("sensitivity-label"),
				IdempotencyKey:   ctx.Flag("idempotency-key"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("ask notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Created ask %s.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Answer": func(ctx cliapp.RunContext) error {
			resp, callErr := client.Answer(context.Background(), connect.NewRequest(&v1.AnswerRequest{
				AskId:  ctx.Flag("ask-id"),
				Answer: ctx.Flag("answer"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("answer notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Answered ask %s.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Wait": func(ctx cliapp.RunContext) error {
			resp, callErr := client.Wait(context.Background(), connect.NewRequest(&v1.WaitRequest{AskId: ctx.Flag("ask-id"), Deadline: ctx.Flag("deadline")}))
			if callErr != nil {
				return cliapp.WrapAPIError("wait for notification answer", callErr, nil)
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Ask %s: %s", resp.Msg.GetAskId(), resp.Msg.GetState())}, ResultsHeading: "Answer", Results: []string{resp.Msg.GetAnswer(), resp.Msg.GetReason()}})
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conversations: load manifest: %w", err)
	}
	return group, nil
}
