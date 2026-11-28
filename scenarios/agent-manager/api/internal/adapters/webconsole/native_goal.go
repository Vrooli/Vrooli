package webconsole

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

// NativeGoalOrigin records intent, never caller identity or new authority.
type NativeGoalOrigin = sessionsv1.NativeGoalMutationOrigin

const (
	NativeGoalUserDirected = sessionsv1.NativeGoalMutationOrigin_NATIVE_GOAL_MUTATION_ORIGIN_USER_DIRECTED
	NativeGoalAutomatic    = sessionsv1.NativeGoalMutationOrigin_NATIVE_GOAL_MUTATION_ORIGIN_AUTOMATIC
)

// NativeGoalController mutates the provider-owned goal for the exact live
// Web Console session generation. Implementations never send terminal input.
type NativeGoalController interface {
	SetNativeGoalForSession(ctx context.Context, webConsoleSessionID, nativeSessionID, expectedGeneration, objective, status string, origin NativeGoalOrigin, tokenBudget *int64) (string, error)
}

func (c *Client) SetNativeGoalForSession(ctx context.Context, webConsoleSessionID, nativeSessionID, expectedGeneration, objective, status string, origin NativeGoalOrigin, tokenBudget *int64) (string, error) {
	if origin != NativeGoalUserDirected && origin != NativeGoalAutomatic {
		return "", errors.New("native goal mutation origin is required")
	}
	if strings.TrimSpace(objective) == "" && status == "active" {
		return "", errors.New("active native goal objective is required")
	}
	if strings.TrimSpace(expectedGeneration) == "" {
		return "", errors.New("expected native generation is required")
	}
	request := &sessionsv1.SetNativeGoalRequest{
		SessionId: webConsoleSessionID, ExpectedNativeOwner: expectedGeneration,
		ExpectedThreadId: nativeSessionID, Objective: objective, Status: status, Origin: origin,
	}
	if tokenBudget != nil {
		request.TokenBudget = tokenBudget
	}
	response, err := c.sessions.SetNativeGoal(ctx, connect.NewRequest(request))
	if err != nil {
		return "", fmt.Errorf("web-console receiver-bound native goal: %w", err)
	}
	if response.Msg.GetThreadId() != nativeSessionID || response.Msg.GetStatus() != status || strings.TrimSpace(response.Msg.GetObjective()) == "" || (objective != "" && response.Msg.GetObjective() != objective) {
		return "", errors.New("web-console native goal receiver returned a mismatched binding or status")
	}
	return response.Msg.GetStatus(), nil
}
