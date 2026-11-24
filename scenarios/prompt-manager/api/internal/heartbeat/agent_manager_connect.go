package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// connectClient resolves the owner for each attempt so a restarted Agent
// Manager cannot leave a long-lived generated client pinned to a stale port.
func (c *AgentManagerClient) connectClient(ctx context.Context, denyRedirect bool) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if denyRedirect {
		copy := *httpClient
		copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		httpClient = &copy
	}
	return apiconnect.NewAgentManagerServiceClient(httpClient, baseURL, connect.WithProtoJSON()), nil
}

func connectRequest[T any](ctx context.Context, message *T, proof createRunCallerProof) *connect.Request[T] {
	req := connect.NewRequest(message)
	if proof.authorization != "" {
		req.Header().Set("Authorization", proof.authorization)
	}
	if proof.runIdentity != "" {
		req.Header().Set("X-Agent-Identity-Token", proof.runIdentity)
	}
	return req
}

func connectError(err error) error {
	if err == nil {
		return nil
	}
	var ce *connect.Error
	if !errors.As(err, &ce) {
		return err
	}
	status := map[connect.Code]int{
		connect.CodeCanceled:           http.StatusRequestTimeout,
		connect.CodeInvalidArgument:    http.StatusBadRequest,
		connect.CodeDeadlineExceeded:   http.StatusGatewayTimeout,
		connect.CodeNotFound:           http.StatusNotFound,
		connect.CodeAlreadyExists:      http.StatusConflict,
		connect.CodePermissionDenied:   http.StatusForbidden,
		connect.CodeResourceExhausted:  http.StatusTooManyRequests,
		connect.CodeFailedPrecondition: http.StatusConflict,
		connect.CodeAborted:            http.StatusConflict,
		connect.CodeUnauthenticated:    http.StatusUnauthorized,
		connect.CodeUnavailable:        http.StatusServiceUnavailable,
		connect.CodeInternal:           http.StatusInternalServerError,
	}[ce.Code()]
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return &AgentManagerHTTPError{StatusCode: status, Code: ce.Code().String(), Message: ce.Message()}
}

func connectApprovalPending(err error) (error, bool) {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeFailedPrecondition {
		return nil, false
	}
	for _, detail := range connectErr.Details() {
		message, decodeErr := detail.Value()
		if decodeErr != nil {
			continue
		}
		approval, ok := message.(*apipb.AuthorizationApprovalPending)
		if !ok {
			continue
		}
		detailText := strings.TrimSpace(approval.GetDetail())
		if detailText == "" {
			detailText = "operator approval pending"
		}
		return errors.New(detailText), true
	}
	return nil, false
}

func statusCodeFromConnectError(err error) (int, bool) {
	var ownerErr *AgentManagerHTTPError
	if errors.As(err, &ownerErr) && ownerErr != nil {
		return ownerErr.StatusCode, true
	}
	return 0, false
}

func isRetryableConnect(err error) bool {
	code := connect.CodeOf(err)
	return code == connect.CodeUnavailable || code == connect.CodeInternal || code == connect.CodeDeadlineExceeded
}

// retryConnect preserves the bounded read recovery policy while constructing
// a newly discovered generated client on each attempt. Mutations opt out.
func (c *AgentManagerClient) retryConnect(ctx context.Context, operation func(apiconnect.AgentManagerServiceClient) error) error {
	backoff := []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second}
	var lastErr error
	for attempt := 0; attempt <= len(backoff); attempt++ {
		client, err := c.connectClient(ctx, false)
		if err == nil {
			err = operation(client)
		}
		if err == nil || !isRetryableConnect(err) {
			return connectError(err)
		}
		lastErr = err
		if attempt < len(backoff) {
			sleep := c.sleep
			if sleep == nil {
				sleep = waitForRetry
			}
			if err := sleep(ctx, backoff[attempt]); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("agent-manager unavailable after %d Connect attempts: %w", len(backoff)+1, connectError(lastErr))
}

func unmarshalConnectProjection[T proto.Message](message T, target any) error {
	raw, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(message)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func unmarshalConnectRequest[T proto.Message](raw []byte, target T) error {
	return (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, target)
}
