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
	// Wait is a server-owned long-poll that blocks until an answer or the
	// requested deadline, so the scenario's short default client timeout would
	// abort it long before either. Its context is bounded by that deadline.
	waitHTTP, waitBase := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	waitClient := connectv1.NewConversationsServiceClient(waitHTTP, waitBase)
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
			reassertWithin, err := parseReassertWithin(ctx.Flag("reassert-within"), ctx.Flag("subject-key"))
			if err != nil {
				return err
			}
			askOptions := make([]*v1.AskOption, 0, len(options))
			for _, option := range options {
				askOptions = append(askOptions, &v1.AskOption{Key: option.Key, Label: option.Label})
			}
			resp, callErr := client.Ask(context.Background(), connect.NewRequest(&v1.AskRequest{
				Question:              ctx.Flag("question"),
				Options:               askOptions,
				Recommended:           ctx.Flag("recommend"),
				RecommendationReason:  ctx.Flag("recommend-reason"),
				DefaultAnswer:         ctx.Flag("default"),
				Reversible:            ctx.BoolFlag("reversible"),
				Urgency:               ctx.Flag("urgency"),
				ContextUrl:            ctx.Flag("context-url"),
				BigPicture:            ctx.Flag("big-picture"),
				TechnicalDetails:      ctx.Flag("technical-details"),
				Deadline:              deadline,
				SensitivityLabel:      ctx.Flag("sensitivity-label"),
				IdempotencyKey:        ctx.Flag("idempotency-key"),
				SubjectKey:            ctx.Flag("subject-key"),
				ReassertWithinSeconds: reassertWithin,
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("ask notification question", callErr, nil)
			}
			if err := cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{askOutcomeLine(resp.Msg)}}); err != nil {
				return err
			}
			writeGuideReminder(ctx.Stderr(), checkGuide(context.Background()), askGuideContext{AskID: resp.Msg.GetAskId(), SubjectKey: ctx.Flag("subject-key"), Outcome: resp.Msg.GetOutcome()})
			return nil
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
		"ConversationsService.Dismiss": func(ctx cliapp.RunContext) error {
			resp, callErr := client.Dismiss(context.Background(), connect.NewRequest(&v1.DismissRequest{
				AskId: ctx.Flag("ask-id"),
				Note:  ctx.Flag("note"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("dismiss notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Dismissed ask %s without answering.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Withdraw": func(ctx cliapp.RunContext) error {
			askID, subjectKey := ctx.Flag("ask-id"), ctx.Flag("subject-key")
			if (askID == "") == (subjectKey == "") {
				return fmt.Errorf("give exactly one of --ask-id or --subject-key")
			}
			resp, callErr := client.Withdraw(context.Background(), connect.NewRequest(&v1.WithdrawRequest{
				AskId:      askID,
				SubjectKey: subjectKey,
				Reason:     ctx.Flag("reason"),
			}))
			if callErr != nil {
				return cliapp.WrapAPIError("withdraw notification question", callErr, nil)
			}
			return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Withdrew ask %s.", resp.Msg.GetAskId())}})
		},
		"ConversationsService.Wait": func(ctx cliapp.RunContext) error {
			deadline, err := parseDeadline(ctx.Flag("deadline"), time.Now())
			if err != nil {
				return err
			}
			waitCtx, cancel := waitContext(deadline)
			defer cancel()
			resp, callErr := waitClient.Wait(waitCtx, connect.NewRequest(&v1.WaitRequest{AskId: ctx.Flag("ask-id"), Deadline: deadline}))
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
			resp, callErr := client.ListAsks(context.Background(), connect.NewRequest(&v1.ListAsksRequest{OpenOnly: ctx.BoolFlag("open"), Limit: int32(limit), Cursor: ctx.Flag("cursor"), Query: ctx.Flag("query")}))
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

// askOutcomeLine says what the hub did with an ask. Only an ask with a
// --subject-key can be anything but newly created.
func askOutcomeLine(resp *v1.AskResponse) string {
	id := resp.GetAskId()
	switch resp.GetOutcome() {
	case "reasserted":
		return fmt.Sprintf("Ask %s is already open with these terms; nothing new was sent.", id)
	case "updated":
		return fmt.Sprintf("Updated open ask %s in place; no new notification was sent.", id)
	case "replaced":
		return fmt.Sprintf("Created ask %s; it replaced the open ask for this subject.", id)
	case "suppressed":
		return fmt.Sprintf("Ask %s was dismissed with these exact terms; no new ask was created.", id)
	case "returned":
		return fmt.Sprintf("Ask %s was just answered; returning it instead of creating another.", id)
	}
	return fmt.Sprintf("Created ask %s.", id)
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
