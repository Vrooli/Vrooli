package decisionpermissions

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	conversationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

type ownerProofKey struct{}
type ownerProof struct {
	authorization string
	accessJWT     string
	cookie        string
	invalid       bool
}

// WithIncomingOwnerProof attaches only the original owner-authentication
// headers needed for Notification Hub to independently authenticate the same
// human. It never mints or widens authority. The proof lives only in this
// request context and is not logged or serialized into permission values.
func WithIncomingOwnerProof(ctx context.Context, headers http.Header) context.Context {
	proof := ownerProof{}
	for _, name := range []string{"Authorization", "Cf-Access-Jwt-Assertion"} {
		values := headers.Values(name)
		if len(values) > 1 {
			proof.invalid = true
			continue
		}
		if len(values) == 1 {
			if len(values[0]) == 0 || len(values[0]) > 16384 || strings.ContainsAny(values[0], "\r\n") {
				proof.invalid = true
				continue
			}
			if name == "Authorization" {
				proof.authorization = values[0]
			} else {
				proof.accessJWT = values[0]
			}
		}
	}
	if values := headers.Values("Cookie"); len(values) > 1 {
		proof.invalid = true
	} else if len(values) == 1 {
		request := &http.Request{Header: http.Header{"Cookie": append([]string(nil), values...)}}
		if cookie, err := request.Cookie("notification_hub_owner_token"); err == nil && cookie.Value != "" && len(cookie.Value) <= 16384 {
			proof.cookie = "notification_hub_owner_token=" + cookie.Value
		}
	}
	return context.WithValue(ctx, ownerProofKey{}, proof)
}

type Client struct {
	BaseURL func(context.Context) (string, error)
	HTTP    *http.Client
}

func NewDiscoveredClient() *Client {
	return &Client{BaseURL: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
	}}
}

func (c *Client) ListPending(ctx context.Context, limit int, cursor string) (Page, error) {
	proof, client, err := c.request(ctx)
	if err != nil {
		return Page{}, err
	}
	response, err := client.ListPendingDecisionReferences(ctx, withOwnerProof(proof, connect.NewRequest(&conversationsv1.ListPendingDecisionReferencesRequest{Limit: int32(limit), Cursor: cursor})))
	if err != nil {
		return Page{}, mapOwnerTransportError(err)
	}
	page := Page{NextCursor: response.Msg.GetNextCursor(), Decisions: make([]Reference, 0, len(response.Msg.GetDecisions()))}
	for _, value := range response.Msg.GetDecisions() {
		if value == nil {
			return Page{}, connect.NewError(connect.CodeDataLoss, errors.New("Notification Hub returned an empty decision reference"))
		}
		page.Decisions = append(page.Decisions, Reference{
			DecisionRef: value.GetDecisionRef(), OwnerID: value.GetOwnerId(), Source: value.GetSource(),
			SourceEventType: value.GetSourceEventType(), Revision: value.GetRevision(), State: value.GetState(),
			CreatedAt: value.GetCreatedAt(), UpdatedAt: value.GetUpdatedAt(), Deadline: value.GetDeadline(), Urgency: value.GetUrgency(),
		})
	}
	return page, nil
}

func (c *Client) ReadPending(ctx context.Context, decisionRef, expectedRevision string) (Reference, error) {
	proof, client, err := c.request(ctx)
	if err != nil {
		return Reference{}, err
	}
	response, err := client.ReadPendingDecisionReference(ctx, withOwnerProof(proof, connect.NewRequest(&conversationsv1.ReadPendingDecisionReferenceRequest{DecisionRef: decisionRef, ExpectedRevision: expectedRevision})))
	if err != nil {
		return Reference{}, mapOwnerTransportError(err)
	}
	value := response.Msg.GetDecision()
	if value == nil {
		return Reference{}, connect.NewError(connect.CodeDataLoss, errors.New("Notification Hub returned an empty decision reference"))
	}
	return Reference{
		DecisionRef: value.GetDecisionRef(), OwnerID: value.GetOwnerId(), Source: value.GetSource(),
		SourceEventType: value.GetSourceEventType(), Revision: value.GetRevision(), State: value.GetState(),
		CreatedAt: value.GetCreatedAt(), UpdatedAt: value.GetUpdatedAt(), Deadline: value.GetDeadline(), Urgency: value.GetUrgency(),
	}, nil
}

func (c *Client) request(ctx context.Context) (ownerProof, conversationsconnect.ConversationsServiceClient, error) {
	proof, ok := ctx.Value(ownerProofKey{}).(ownerProof)
	if !ok || proof.invalid || (proof.authorization == "" && proof.accessJWT == "" && proof.cookie == "") {
		return ownerProof{}, nil, connect.NewError(connect.CodeUnauthenticated, errors.New("forwarded Notification Hub owner proof is unavailable"))
	}
	baseURL := c.BaseURL
	if baseURL == nil {
		baseURL = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "notification-hub")
		}
	}
	base, err := baseURL(ctx)
	if err != nil {
		return ownerProof{}, nil, connect.NewError(connect.CodeUnavailable, errors.New("Notification Hub is unavailable"))
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return proof, conversationsconnect.NewConversationsServiceClient(httpClient, strings.TrimRight(base, "/")), nil
}

func withOwnerProof[T any](proof ownerProof, request *connect.Request[T]) *connect.Request[T] {
	if proof.authorization != "" {
		request.Header().Set("Authorization", proof.authorization)
	}
	if proof.accessJWT != "" {
		request.Header().Set("Cf-Access-Jwt-Assertion", proof.accessJWT)
	}
	if proof.cookie != "" {
		request.Header().Set("Cookie", proof.cookie)
	}
	return request
}

func mapOwnerTransportError(err error) error {
	if connect.CodeOf(err) == connect.CodeUnauthenticated || connect.CodeOf(err) == connect.CodePermissionDenied || connect.CodeOf(err) == connect.CodeNotFound || connect.CodeOf(err) == connect.CodeFailedPrecondition || connect.CodeOf(err) == connect.CodeInvalidArgument || connect.CodeOf(err) == connect.CodeResourceExhausted || connect.CodeOf(err) == connect.CodeDataLoss {
		return err
	}
	return connect.NewError(connect.CodeUnavailable, errors.New("Notification Hub decision owner read failed"))
}
