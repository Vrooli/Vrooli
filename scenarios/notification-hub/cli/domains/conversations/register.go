package conversations

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
			options, err := parseOptions(ctx.FlagValues("option"))
			if err != nil {
				return err
			}
			deadline, err := parseDeadline(ctx.Flag("deadline"), time.Now())
			if err != nil {
				return err
			}
			askOptions := make([]*v1.AskOption, 0, len(options))
			for _, option := range options {
				askOptions = append(askOptions, &v1.AskOption{Key: option.Key, Label: option.Label})
			}
			resp, callErr := client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
				Question:             ctx.Flag("question"),
				Options:              askOptions,
				Recommended:          ctx.Flag("recommend"),
				RecommendationReason: ctx.Flag("recommend-reason"),
				DefaultAnswer:        ctx.Flag("default"),
				Reversible:           ctx.BoolFlag("reversible"),
				Urgency:              ctx.Flag("urgency"),
				ContextUrl:           ctx.Flag("context-url"),
				Deadline:             deadline,
				SensitivityLabel:     ctx.Flag("sensitivity-label"),
				IdempotencyKey:       ctx.Flag("idempotency-key"),
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
				Note:   ctx.Flag("note"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("answer notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Answered ask %s.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Wait": func(ctx cliapp.RunContext) error {
			deadline, err := parseDeadline(ctx.Flag("deadline"), time.Now())
			if err != nil {
				return err
			}
			resp, callErr := client.Wait(context.Background(), connect.NewRequest(&v1.WaitRequest{AskId: ctx.Flag("ask-id"), Deadline: deadline}))
			if callErr != nil {
				return cliapp.WrapAPIError("wait for notification answer", callErr, nil)
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Ask %s: %s", resp.Msg.GetAskId(), resp.Msg.GetState())}, ResultsHeading: "Answer", Results: []string{resp.Msg.GetAnswer(), resp.Msg.GetReason()}})
		},
		"ConversationsService.GetAsk": func(ctx cliapp.RunContext) error {
			resp, callErr := client.GetAsk(context.Background(), connect.NewRequest(&v1.GetAskRequest{AskId: ctx.Positional("ask-id")}))
			if callErr != nil {
				return cliapp.WrapAPIError("read ask", callErr, nil)
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{askSummary(resp.Msg.GetAsk())}, ResultsHeading: "Options", Results: askOptionLines(resp.Msg.GetAsk())})
		},
		"ConversationsService.ListAsks": func(ctx cliapp.RunContext) error {
			limit, _ := strconv.Atoi(ctx.Flag("limit"))
			resp, callErr := client.ListAsks(context.Background(), connect.NewRequest(&v1.ListAsksRequest{OpenOnly: ctx.BoolFlag("open"), Limit: int32(limit)}))
			if callErr != nil {
				return cliapp.WrapAPIError("list asks", callErr, nil)
			}
			lines := make([]string, 0, len(resp.Msg.GetAsks()))
			for _, ask := range resp.Msg.GetAsks() {
				lines = append(lines, askSummary(ask))
			}
			return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("%d asks", len(lines))}, ResultsHeading: "Asks", Results: lines})
		},
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("conversations: load manifest: %w", err)
	}
	return group, nil
}

func askSummary(ask *v1.Ask) string {
	summary := fmt.Sprintf("%s [%s] %s", ask.GetId(), ask.GetState(), ask.GetQuestion())
	if ask.GetAnswer() != "" {
		summary += " -> " + ask.GetAnswer()
	}
	if ask.GetReason() != "" {
		summary += " (" + ask.GetReason() + ")"
	}
	return summary
}

func askOptionLines(ask *v1.Ask) []string {
	lines := make([]string, 0, len(ask.GetOptions()))
	for _, option := range ask.GetOptions() {
		var marks []string
		if option.GetKey() == ask.GetRecommended() {
			marks = append(marks, "recommended")
		}
		if option.GetKey() == ask.GetDefaultAnswer() {
			marks = append(marks, "default")
		}
		line := option.GetKey() + " = " + option.GetLabel()
		if len(marks) > 0 {
			line += " (" + strings.Join(marks, ", ") + ")"
		}
		lines = append(lines, line)
	}
	return lines
}
