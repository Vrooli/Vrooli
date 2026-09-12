package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
)

type BridgeClient struct {
	BaseURL string
	Token   func(context.Context) (string, error)
	Client  *http.Client
}

func (c *BridgeClient) clients() (onboardconnect.OnboardServiceClient, machinesconnect.MachineServiceClient, error) {
	if strings.TrimSpace(c.BaseURL) == "" {
		return nil, nil, fmt.Errorf("vrooli-bridge base URL is not configured")
	}
	httpClient := c.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return onboardconnect.NewOnboardServiceClient(httpClient, strings.TrimRight(c.BaseURL, "/")), machinesconnect.NewMachineServiceClient(httpClient, strings.TrimRight(c.BaseURL, "/")), nil
}

func (c *BridgeClient) GetOnboardingPublicKey(ctx context.Context) (string, string, error) {
	onboard, _, err := c.clients()
	if err != nil {
		return "", "", err
	}
	req := connect.NewRequest(&onboardv1.GetOnboardingPublicKeyRequest{})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", "", err
	}
	resp, err := onboard.GetOnboardingPublicKey(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("get bridge onboarding public key: %w", err)
	}
	return resp.Msg.GetPublicKey(), resp.Msg.GetFingerprint(), nil
}

func (c *BridgeClient) CreateMachine(ctx context.Context, host string) (string, error) {
	_, machines, err := c.clients()
	if err != nil {
		return "", err
	}
	req := connect.NewRequest(&machinesv1.CreateMachineRequest{Locators: []*machinesv1.ConnectionLocator{{Kind: "ip", Value: host, Ordinal: 0}}})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", err
	}
	resp, err := machines.CreateMachine(ctx, req)
	if err != nil {
		return "", fmt.Errorf("create bridge machine: %w", err)
	}
	if resp.Msg.GetMachine() == nil {
		return "", fmt.Errorf("bridge returned no machine")
	}
	return resp.Msg.GetMachine().GetId(), nil
}

func (c *BridgeClient) StartOnboarding(ctx context.Context, host, user, machineID string) (string, error) {
	onboard, _, err := c.clients()
	if err != nil {
		return "", err
	}
	req := connect.NewRequest(&onboardv1.StartOnboardingRequest{Host: host, User: user, MachineId: machineID})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", err
	}
	resp, err := onboard.StartOnboarding(ctx, req)
	if err != nil {
		return "", fmt.Errorf("start bridge onboarding: %w", err)
	}
	return resp.Msg.GetOpId(), nil
}

func (c *BridgeClient) authorize(ctx context.Context, headers interface{ Set(string, string) }) error {
	if c.Token == nil {
		return nil
	}
	token, err := c.Token(ctx)
	if err != nil {
		return fmt.Errorf("resolve bridge token: %w", err)
	}
	if strings.TrimSpace(token) != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	return nil
}
