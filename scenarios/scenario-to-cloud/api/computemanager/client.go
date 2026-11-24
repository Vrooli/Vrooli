// Package computemanager is Scenario-to-Cloud's typed client for the
// Compute Manager instance-reference API. Compute acquisition remains owned
// by Compute Manager; this package only requests and resolves references.
package computemanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instancev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"scenario-to-cloud/identity"
)

const (
	ScenarioName  = "compute-manager"
	QEMUProvider  = "local-qemu"
	QEMURequester = "scenario-to-cloud:qemu-certification"
)

// URLResolver resolves the current scenario URL on every call. This keeps
// service routing owned by Vrooli discovery instead of baking an address into
// the Scenario-to-Cloud configuration.
type URLResolver func(context.Context) (string, error)

type Client struct {
	HTTP       *http.Client
	ResolveURL URLResolver
}

// New creates a client that uses the canonical discovery resolver. Compute
// Manager's current Connect instance service does not require a service token;
// if that transport contract changes, auth must be wired through credential
// authority rather than an environment variable or caller-supplied secret.
func New() *Client {
	return NewWithResolver(&http.Client{}, func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, ScenarioName)
	})
}

// NewWithResolver injects HTTP and service discovery for transport and
// consumer-boundary tests.
func NewWithResolver(httpClient *http.Client, resolver URLResolver) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTP: httpClient, ResolveURL: resolver}
}

func (c *Client) instanceService(ctx context.Context) (instancev1connect.InstanceServiceClient, error) {
	if c == nil {
		return nil, fmt.Errorf("compute-manager client is not configured")
	}
	if c.ResolveURL == nil {
		return nil, fmt.Errorf("compute-manager URL resolver is not configured")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve compute-manager URL: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("resolved compute-manager URL is empty")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return instancev1connect.NewInstanceServiceClient(httpClient, baseURL), nil
}

// QEMURequest contains the caller-owned identity and resource request for one
// local QEMU guest. The idempotency key must be stable for the lane target.
type QEMURequest struct {
	IdempotencyKey   string
	Region           string
	Size             string
	Image            string
	LifetimeSeconds  int64
	Memory           string
	CPUs             int32
	SSHUser          string
	SSHPort          int32
	Profile          string
	SSHAuthorizedKey string
}

// InstanceReference keeps Compute Manager's stable instance identity
// separate from a Bridge machine identity. Target contains only the locator
// and transport used by Scenario-to-Cloud to reach the instance.
type InstanceReference struct {
	ManagerInstanceID  string
	Provider           string
	ProviderInstanceID string
	State              instancev1.InstanceState
	Region             string
	Size               string
	Image              string
	Address            string
	BridgeMachineID    string
	SSHUser            string
	SSHPort            int32
	ExpiresAt          time.Time
	Target             identity.TargetRef
}

// RequestQEMU acquires a local test guest through Compute Manager's generated
// Connect API, then returns its typed target reference. This method makes no
// local QEMU calls and does not create a second provider interface.
func (c *Client) RequestQEMU(ctx context.Context, request QEMURequest) (InstanceReference, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" {
		return InstanceReference{}, fmt.Errorf("QEMU acquisition requires a stable idempotency key")
	}
	if request.Region != "local" || (request.Profile != "headless-linux" && request.Profile != "desktop-linux") || request.Size != request.Profile ||
		strings.TrimSpace(request.Image) == "" || strings.TrimSpace(request.SSHAuthorizedKey) == "" || request.LifetimeSeconds <= 0 {
		return InstanceReference{}, fmt.Errorf("QEMU acquisition requires region=local, size equal to a supported profile, image, public key, and positive lifetime")
	}
	service, err := c.instanceService(ctx)
	if err != nil {
		return InstanceReference{}, err
	}
	response, err := service.RequestInstance(ctx, connect.NewRequest(&instancev1.RequestInstanceRequest{
		IdempotencyKey:   request.IdempotencyKey,
		Provider:         QEMUProvider,
		Region:           request.Region,
		Size:             request.Size,
		Image:            request.Image,
		LifetimeSeconds:  request.LifetimeSeconds,
		RequestedBy:      QEMURequester,
		Memory:           request.Memory,
		Cpus:             request.CPUs,
		SshUser:          request.SSHUser,
		SshPort:          request.SSHPort,
		Profile:          request.Profile,
		SshAuthorizedKey: request.SSHAuthorizedKey,
	}))
	if err != nil {
		return InstanceReference{}, fmt.Errorf("request local QEMU instance from compute-manager: %w", err)
	}
	return referenceFromProto(response.Msg.GetInstance())
}

func (c *Client) GetInstance(ctx context.Context, id string) (InstanceReference, error) {
	if strings.TrimSpace(id) == "" {
		return InstanceReference{}, fmt.Errorf("compute-manager instance ID is required")
	}
	service, err := c.instanceService(ctx)
	if err != nil {
		return InstanceReference{}, err
	}
	response, err := service.GetInstance(ctx, connect.NewRequest(&instancev1.GetInstanceRequest{Id: id}))
	if err != nil {
		return InstanceReference{}, fmt.Errorf("get compute-manager instance %q: %w", id, err)
	}
	return referenceFromProto(response.Msg.GetInstance())
}

func (c *Client) ListInstances(ctx context.Context) ([]InstanceReference, error) {
	service, err := c.instanceService(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.ListInstances(ctx, connect.NewRequest(&instancev1.ListInstancesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list compute-manager instances: %w", err)
	}
	instances := response.Msg.GetInstances()
	refs := make([]InstanceReference, 0, len(instances))
	for _, item := range instances {
		ref, err := referenceFromProto(item)
		if err != nil {
			return nil, fmt.Errorf("convert compute-manager instance reference: %w", err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (c *Client) DestroyInstance(ctx context.Context, id string) (InstanceReference, error) {
	if strings.TrimSpace(id) == "" {
		return InstanceReference{}, fmt.Errorf("compute-manager instance ID is required")
	}
	service, err := c.instanceService(ctx)
	if err != nil {
		return InstanceReference{}, err
	}
	response, err := service.DestroyInstance(ctx, connect.NewRequest(&instancev1.DestroyInstanceRequest{Id: id}))
	if err != nil {
		return InstanceReference{}, fmt.Errorf("destroy compute-manager instance %q: %w", id, err)
	}
	return referenceFromProto(response.Msg.GetInstance())
}

func referenceFromProto(item *instancev1.Instance) (InstanceReference, error) {
	if item == nil || strings.TrimSpace(item.GetId()) == "" {
		return InstanceReference{}, fmt.Errorf("compute-manager returned an instance without an ID")
	}
	ref := InstanceReference{
		ManagerInstanceID:  item.GetId(),
		Provider:           item.GetProvider(),
		ProviderInstanceID: item.GetProviderInstanceId(),
		State:              item.GetState(),
		Region:             item.GetRegion(),
		Size:               item.GetSize(),
		Image:              item.GetImage(),
		Address:            item.GetAddress(),
		BridgeMachineID:    item.GetBridgeMachineId(),
		SSHUser:            item.GetSshUser(),
		SSHPort:            item.GetSshPort(),
	}
	if item.GetExpiresAt() != nil {
		ref.ExpiresAt = item.GetExpiresAt().AsTime()
	}
	return ref, nil
}

// TargetRef returns the reach identity for this Compute Manager reference.
// The manager instance ID is never overloaded into MachineID: that field is
// reserved for Bridge's own machine identity.
func (r InstanceReference) TargetRef() (identity.TargetRef, error) {
	if strings.TrimSpace(r.ManagerInstanceID) == "" {
		return identity.TargetRef{}, fmt.Errorf("compute-manager instance identity is required")
	}
	if strings.TrimSpace(r.Address) == "" && strings.TrimSpace(r.BridgeMachineID) == "" {
		return identity.TargetRef{}, fmt.Errorf("compute-manager instance %q has no address", r.ManagerInstanceID)
	}
	user := strings.TrimSpace(r.SSHUser)
	if user == "" {
		user = "root"
	}
	transport := identity.TransportSSH
	target := identity.TargetRef{Transport: transport, Locator: identity.TargetLocator{
		Host: r.Address, Port: int(r.SSHPort), User: user,
	}}
	target.ComputeManagerID = r.ManagerInstanceID
	if strings.TrimSpace(r.BridgeMachineID) != "" {
		target.MachineID = r.BridgeMachineID
		target.Transport = identity.TransportBridge
	}
	if target.Transport == identity.TransportSSH && (r.SSHPort < 1 || r.SSHPort > 65535) {
		return identity.TargetRef{}, fmt.Errorf("compute-manager instance %q has invalid SSH port %d", r.ManagerInstanceID, r.SSHPort)
	}
	return target, nil
}
