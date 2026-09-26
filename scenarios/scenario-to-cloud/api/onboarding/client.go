// Package onboarding is the narrow scenario-to-cloud client for the
// onboarding-owned handoff contract. It carries no credential values and does
// not duplicate onboarding state.
package onboarding

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
)

type Request struct {
	Target               string
	MachineID            string
	NodeID               string
	NodeKind             string
	DeploymentID         string
	EnrollmentGeneration uint64
	DesiredRevision      uint64
	SelectionDigest      string
	RequestKey           string
	Missing              []string
	Selection            *setupv1.Selection
}

type Issuer interface {
	Issue(context.Context, Request) (*selectionv1.Handoff, error)
}

type Client struct {
	HTTP     *http.Client
	Resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	Token string
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{}, Resolver: discovery.DefaultResolver(), Token: strings.TrimSpace(os.Getenv("VROOLI_ONBOARDING_API_TOKEN"))}
}

func (c *Client) Issue(ctx context.Context, request Request) (*selectionv1.Handoff, error) {
	if c == nil {
		return nil, fmt.Errorf("onboarding handoff client is not configured")
	}
	resolver := c.Resolver
	if resolver == nil {
		resolver = discovery.DefaultResolver()
	}
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, "vrooli-onboarding")
	if err != nil {
		return nil, fmt.Errorf("resolve onboarding URL: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("onboarding URL is empty")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if strings.TrimSpace(c.Token) == "" {
		return nil, fmt.Errorf("onboarding handoff service token is not configured")
	}
	client := selectionconnect.NewSelectionServiceClient(withBearer(httpClient, c.Token), baseURL)
	response, err := client.CreateHandoff(ctx, connect.NewRequest(&selectionv1.CreateHandoffRequest{
		Target: request.Target, MachineId: request.MachineID, NodeId: request.NodeID, NodeKind: request.NodeKind,
		DeploymentId: request.DeploymentID, EnrollmentGeneration: request.EnrollmentGeneration, DesiredRevision: request.DesiredRevision,
		RequestKey: request.RequestKey, Missing: append([]string(nil), request.Missing...), DesiredSelection: request.Selection,
	}))
	if err != nil {
		return nil, fmt.Errorf("create onboarding handoff: %w", err)
	}
	handoff := response.Msg.GetHandoff()
	if handoff == nil || strings.TrimSpace(handoff.GetReference()) == "" {
		return nil, fmt.Errorf("onboarding returned no durable handoff identity")
	}
	if handoff.GetDeploymentId() != request.DeploymentID || handoff.GetTarget() != request.Target ||
		handoff.GetMachineId() != request.MachineID || handoff.GetNodeId() != request.NodeID ||
		handoff.GetNodeKind() != request.NodeKind || handoff.GetEnrollmentGeneration() != request.EnrollmentGeneration ||
		handoff.GetDesiredRevision() != request.DesiredRevision ||
		(strings.TrimSpace(request.SelectionDigest) != "" && handoff.GetSelectionDigest() != request.SelectionDigest) {
		return nil, fmt.Errorf("onboarding returned a handoff with mismatched target or revision fences")
	}
	return handoff, nil
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func withBearer(client *http.Client, token string) *http.Client {
	clone := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	clone.Transport = bearerTransport{base: base, token: token}
	return &clone
}

func (t bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}
