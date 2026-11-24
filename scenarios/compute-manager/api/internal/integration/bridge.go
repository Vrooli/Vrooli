package integration

import (
	"context"
	"fmt"
	"net"
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

type OnboardingEvidence struct {
	OperationID, MachineID, Host, User, State string
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
	kind := "hostname"
	if net.ParseIP(strings.TrimSpace(host)) != nil {
		kind = "ip"
	}
	req := connect.NewRequest(&machinesv1.CreateMachineRequest{Locators: []*machinesv1.ConnectionLocator{{Kind: kind, Value: strings.TrimSpace(host), Ordinal: 0}}})
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

// FindMachine reattaches to a machine created before a lost Bridge response.
// It requires one exact hostname/IP locator match and refuses ambiguity.
func (c *BridgeClient) FindMachine(ctx context.Context, host string) (string, bool, error) {
	_, machines, err := c.clients()
	if err != nil {
		return "", false, err
	}
	req := connect.NewRequest(&machinesv1.ListMachinesRequest{})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", false, err
	}
	resp, err := machines.ListMachines(ctx, req)
	if err != nil {
		return "", false, fmt.Errorf("read bridge machine inventory failed")
	}
	wantHost := strings.TrimSpace(host)
	wantKind := "hostname"
	if net.ParseIP(wantHost) != nil {
		wantKind = "ip"
	}
	match := ""
	for _, machine := range resp.Msg.GetMachines() {
		if machine.GetLifecycle() == "MACHINE_LIFECYCLE_ARCHIVED" || machine.GetLifecycle() == "MACHINE_LIFECYCLE_REMOVED" {
			continue
		}
		for _, locator := range machine.GetLocators() {
			if locator.GetKind() != wantKind || strings.TrimSpace(locator.GetValue()) != wantHost {
				continue
			}
			if match != "" && match != machine.GetId() {
				return "", false, fmt.Errorf("bridge inventory has multiple exact host matches")
			}
			match = machine.GetId()
		}
	}
	return match, match != "", nil
}

// ArchiveMachine releases the Bridge attachment while preserving its durable
// history. It never contacts the adopted host or removes the Bridge key.
func (c *BridgeClient) ArchiveMachine(ctx context.Context, machineID string) error {
	_, machines, err := c.clients()
	if err != nil {
		return err
	}
	getReq := connect.NewRequest(&machinesv1.GetMachineRequest{Id: strings.TrimSpace(machineID)})
	if err := c.authorize(ctx, getReq.Header()); err != nil {
		return err
	}
	got, err := machines.GetMachine(ctx, getReq)
	if err != nil {
		return fmt.Errorf("read bridge machine before release: %w", err)
	}
	if got.Msg.GetMachine() == nil {
		return fmt.Errorf("bridge returned no machine for release")
	}
	archiveReq := connect.NewRequest(&machinesv1.ArchiveMachineRequest{Id: got.Msg.GetMachine().GetId(), Version: got.Msg.GetMachine().GetVersion()})
	if err := c.authorize(ctx, archiveReq.Header()); err != nil {
		return err
	}
	if _, err := machines.ArchiveMachine(ctx, archiveReq); err != nil {
		return fmt.Errorf("archive bridge machine attachment: %w", err)
	}
	return nil
}

// MachineLifecycle reads the exact persisted Bridge machine identifier. A
// release retry may use this only to confirm an archive; it must not infer
// archive success from a locator or mutable host address.
func (c *BridgeClient) MachineLifecycle(ctx context.Context, machineID string) (string, error) {
	_, machines, err := c.clients()
	if err != nil {
		return "", err
	}
	req := connect.NewRequest(&machinesv1.GetMachineRequest{Id: strings.TrimSpace(machineID)})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", err
	}
	resp, err := machines.GetMachine(ctx, req)
	if err != nil {
		return "", fmt.Errorf("read exact bridge machine lifecycle: %w", err)
	}
	if resp.Msg.GetMachine() == nil || resp.Msg.GetMachine().GetId() != strings.TrimSpace(machineID) {
		return "", fmt.Errorf("bridge returned a different machine identity")
	}
	return resp.Msg.GetMachine().GetLifecycle(), nil
}

func (c *BridgeClient) StartOnboarding(ctx context.Context, host, user, machineID string) (string, error) {
	return c.startOnboarding(ctx, host, user, machineID, "")
}

// StartOnboardingWithPassword is the request-scoped secure first-touch path.
// The byte slice is consumed and cleared here; Bridge owns the transient SSH
// use and zeroes its own copy. Callers must not persist or log the value.
func (c *BridgeClient) StartOnboardingWithPassword(ctx context.Context, host, user, machineID string, password []byte) (string, error) {
	defer clear(password)
	secret := string(password)
	defer func() { secret = "" }()
	return c.startOnboarding(ctx, host, user, machineID, secret)
}

func (c *BridgeClient) GetOnboarding(ctx context.Context, operationID string) (OnboardingEvidence, error) {
	onboard, _, err := c.clients()
	if err != nil {
		return OnboardingEvidence{}, err
	}
	req := connect.NewRequest(&onboardv1.GetOnboardingRequest{Id: strings.TrimSpace(operationID)})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return OnboardingEvidence{}, err
	}
	resp, err := onboard.GetOnboarding(ctx, req)
	if err != nil {
		return OnboardingEvidence{}, fmt.Errorf("read bridge onboarding evidence failed")
	}
	op := resp.Msg.GetOp()
	if op == nil {
		return OnboardingEvidence{}, fmt.Errorf("bridge returned no onboarding operation")
	}
	return OnboardingEvidence{OperationID: op.GetId(), MachineID: op.GetMachineId(), Host: op.GetHost(), User: op.GetUser(), State: op.GetState().String()}, nil
}

// FindOnboarding reattaches to a durable operation after a lost response. It
// only returns an exact host and MachineID match; callers must reject any
// ambiguous result before advancing their local state.
func (c *BridgeClient) FindOnboarding(ctx context.Context, host, machineID string) (OnboardingEvidence, bool, error) {
	onboard, _, err := c.clients()
	if err != nil {
		return OnboardingEvidence{}, false, err
	}
	req := connect.NewRequest(&onboardv1.ListOnboardingsRequest{Host: strings.TrimSpace(host), Limit: 100})
	if err := c.authorize(ctx, req.Header()); err != nil {
		return OnboardingEvidence{}, false, err
	}
	resp, err := onboard.ListOnboardings(ctx, req)
	if err != nil {
		return OnboardingEvidence{}, false, fmt.Errorf("read bridge onboarding history failed")
	}
	var match OnboardingEvidence
	for _, op := range resp.Msg.GetOps() {
		if op.GetMachineId() != machineID || op.GetHost() != host {
			continue
		}
		candidate := OnboardingEvidence{OperationID: op.GetId(), MachineID: op.GetMachineId(), Host: op.GetHost(), User: op.GetUser(), State: op.GetState().String()}
		if match.OperationID != "" && match.OperationID != candidate.OperationID {
			return OnboardingEvidence{}, false, fmt.Errorf("bridge onboarding history has multiple matching operations")
		}
		match = candidate
	}
	return match, match.OperationID != "", nil
}

func (c *BridgeClient) startOnboarding(ctx context.Context, host, user, machineID, password string) (string, error) {
	onboard, _, err := c.clients()
	if err != nil {
		return "", err
	}
	req := connect.NewRequest(&onboardv1.StartOnboardingRequest{Host: host, User: user, MachineId: machineID, SshPassword: password})
	defer func() { req.Msg.SshPassword = "" }()
	if err := c.authorize(ctx, req.Header()); err != nil {
		return "", err
	}
	resp, err := onboard.StartOnboarding(ctx, req)
	if err != nil {
		// Do not propagate transport details that could contain request data.
		return "", fmt.Errorf("start bridge onboarding failed")
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
