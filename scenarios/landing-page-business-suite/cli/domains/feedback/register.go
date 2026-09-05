// Package feedback exposes the feedback lifecycle through the generated
// FeedbackService contract. It deliberately keeps the established command
// names and JSON field values so automation does not need to understand
// protobuf enum identifiers.
package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite/cli/internal/support"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Engagement - Feedback", Commands: []cliapp.Command{
		createCommand(deps), listCommand(deps), bulkDeleteCommand(deps), getCommand(deps), deleteCommand(deps), updateStatusCommand(deps),
	}}
}

func publicClient(deps support.Dependencies) (lpbsconnect.FeedbackServiceClient, error) {
	core := deps.ScenarioApp()
	if core == nil {
		return nil, fmt.Errorf("scenario app is not initialized")
	}
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return lpbsconnect.NewFeedbackServiceClient(httpClient, baseURL), nil
}

func adminClient(deps support.Dependencies) (lpbsconnect.FeedbackServiceClient, error) {
	httpClient, baseURL, err := deps.AdminConnectHTTPClient()
	if err != nil {
		return nil, err
	}
	return lpbsconnect.NewFeedbackServiceClient(httpClient, baseURL), nil
}

func action(deps support.Dependencies, admin bool, description string, call func(context.Context, lpbsconnect.FeedbackServiceClient, cliapp.OperationContext) (proto.Message, error)) cliapp.Command {
	op := cliapp.Action(func(ctx cliapp.OperationContext) (json.RawMessage, error) {
		var service lpbsconnect.FeedbackServiceClient
		var err error
		if admin {
			service, err = adminClient(deps)
		} else {
			service, err = publicClient(deps)
		}
		if err != nil {
			return nil, err
		}
		response, err := call(context.Background(), service, ctx)
		if err != nil {
			return nil, cliapp.WrapAPIError(description, err, nil)
		}
		payload, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(response)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", description, err)
		}
		return json.RawMessage(payload), nil
	}, func(cliapp.OperationContext, json.RawMessage) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{description}}
	})
	return (cliapp.Command{NeedsAPI: true, Description: description, Architecture: cliapp.CommandArchitecture{Primitive: cliapp.PrimitiveAction}}).WithPrimitive(op)
}

func createCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, false, "Submit feedback through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		request := &lpbsv1.FeedbackCreateRequest{}
		if err := decodeCreateBody(op.Flag("body"), request); err != nil {
			return nil, err
		}
		response, err := service.CreateFeedback(ctx, connect.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "feedback-create"
	command.Args = cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "Feedback JSON payload or @file.json", Required: true}}}
	return command
}

func listCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, true, "List feedback through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		status, err := statusEnum(op.Flag("status"), true)
		if err != nil {
			return nil, err
		}
		request := &lpbsv1.ListFeedbackRequest{Status: status}
		response, err := service.ListFeedback(ctx, connect.NewRequest(request))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "admin-feedback-list"
	command.Args = cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "status", Description: "Optional status: pending, in_progress, resolved, or rejected"}}}
	return command
}

func getCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, true, "Get feedback through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		id, err := feedbackID(op)
		if err != nil {
			return nil, err
		}
		response, err := service.GetFeedback(ctx, connect.NewRequest(&lpbsv1.GetFeedbackRequest{Id: id}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "admin-feedback-get"
	command.Args = feedbackIDArgs()
	return command
}

func deleteCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, true, "Delete feedback through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		id, err := feedbackID(op)
		if err != nil {
			return nil, err
		}
		response, err := service.DeleteFeedback(ctx, connect.NewRequest(&lpbsv1.DeleteFeedbackRequest{Id: id}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "admin-feedback-delete"
	command.Args = feedbackIDArgs()
	return command
}

func bulkDeleteCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, true, "Bulk delete feedback through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		payload, err := support.ParseBody(op.Flag("body"))
		if err != nil {
			return nil, err
		}
		var input struct {
			IDs []int64 `json:"ids"`
		}
		if err := json.Unmarshal(payload, &input); err != nil {
			return nil, fmt.Errorf("decode feedback IDs: %w", err)
		}
		if len(input.IDs) == 0 {
			return nil, fmt.Errorf("at least one feedback ID is required")
		}
		response, err := service.DeleteFeedbackBulk(ctx, connect.NewRequest(&lpbsv1.DeleteFeedbackBulkRequest{Ids: input.IDs}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "admin-feedback-bulk-delete"
	command.Args = cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body", Description: "JSON payload containing ids, e.g. {\"ids\":[1,2]}", Required: true}}}
	return command
}

func updateStatusCommand(deps support.Dependencies) cliapp.Command {
	command := action(deps, true, "Update feedback status through the generated Connect contract.", func(ctx context.Context, service lpbsconnect.FeedbackServiceClient, op cliapp.OperationContext) (proto.Message, error) {
		id, err := feedbackID(op)
		if err != nil {
			return nil, err
		}
		payload, err := support.ParseBody(op.Flag("body"))
		if err != nil {
			return nil, err
		}
		var input struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(payload, &input); err != nil {
			return nil, fmt.Errorf("decode feedback status: %w", err)
		}
		status, err := statusEnum(input.Status, false)
		if err != nil {
			return nil, err
		}
		response, err := service.UpdateFeedbackStatus(ctx, connect.NewRequest(&lpbsv1.UpdateFeedbackStatusRequest{Id: id, Status: *status}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	})
	command.Name = "admin-feedback-status-update"
	command.Args = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "feedback_id", Required: true}}, Flags: []cliapp.Flag{{Name: "body", Description: "JSON payload containing status", Required: true}}}
	return command
}

func decodeCreateBody(raw string, request *lpbsv1.FeedbackCreateRequest) error {
	payload, err := support.ParseBody(raw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payload, request); err != nil {
		return fmt.Errorf("decode feedback creation: %w", err)
	}
	return nil
}

func feedbackID(ctx cliapp.OperationContext) (int64, error) {
	raw := strings.TrimSpace(ctx.Positional("feedback_id"))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("feedback_id must be a positive integer")
	}
	return id, nil
}

func feedbackIDArgs() cliapp.ArgSchema {
	return cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "feedback_id", Required: true}}}
}

func statusEnum(raw string, optional bool) (*lpbsv1.FeedbackStatus, error) {
	switch strings.TrimSpace(raw) {
	case "":
		if optional {
			return nil, nil
		}
	case "pending":
		value := lpbsv1.FeedbackStatus_FEEDBACK_STATUS_PENDING
		return &value, nil
	case "in_progress":
		value := lpbsv1.FeedbackStatus_FEEDBACK_STATUS_IN_PROGRESS
		return &value, nil
	case "resolved":
		value := lpbsv1.FeedbackStatus_FEEDBACK_STATUS_RESOLVED
		return &value, nil
	case "rejected":
		value := lpbsv1.FeedbackStatus_FEEDBACK_STATUS_REJECTED
		return &value, nil
	}
	return nil, fmt.Errorf("status must be pending, in_progress, resolved, or rejected")
}
