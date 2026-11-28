package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"deployment-manager/releases"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const deploymentManagerComputeLifecyclePrincipal = "deployment-manager-compute-lifecycle"

type cloudComputeReleaseAdapter struct {
	resolve func(context.Context) (string, error)
	client  *http.Client
	tokens  *servicetoken.Tokens
}

func newCloudComputeReleaseAdapter() *cloudComputeReleaseAdapter {
	a := &cloudComputeReleaseAdapter{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "compute-manager")
		},
		client: &http.Client{Timeout: 30 * time.Second},
	}
	a.tokens = servicetoken.New(servicetoken.Config{
		Name:   "deployment-manager Compute Manager lifecycle",
		Expect: servicetoken.ExpectIdentity("deployment-manager-compute-lifecycle", "scenario-authenticator:compute-manager-deployment-lifecycle", "compute-manager:deployment-lifecycle"),
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			id, err := credentialauthority.ParseIdentity("scenario-authenticator/service-principal/" + deploymentManagerComputeLifecyclePrincipal + "/compute-manager")
			if err != nil {
				return "", err
			}
			return authority.Require(id, "credential")
		},
	}, servicetoken.WithExchanger(servicetoken.AuthenticatorExchanger(&http.Client{Timeout: 5 * time.Second}, func(ctx context.Context) (string, error) {
		if base := strings.TrimSpace(httpEnv("SCENARIO_AUTHENTICATOR_API_URL")); base != "" {
			return strings.TrimRight(base, "/"), nil
		}
		return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	})))
	return a
}

func httpEnv(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func (a *cloudComputeReleaseAdapter) cmClient(ctx context.Context) (instanceconnect.InstanceServiceClient, error) {
	if a == nil || a.resolve == nil || a.tokens == nil {
		return nil, fmt.Errorf("Compute Manager lifecycle identity is unavailable")
	}
	base, err := a.resolve(ctx)
	if err != nil || strings.TrimSpace(base) == "" {
		return nil, fmt.Errorf("Compute Manager is unavailable: %v", err)
	}
	client := a.client
	if client == nil {
		client = http.DefaultClient
	}
	transport := authTransport{next: client.Transport, tokens: a.tokens}
	clientCopy := *client
	clientCopy.Transport = transport
	return instanceconnect.NewInstanceServiceClient(&clientCopy, strings.TrimRight(base, "/")), nil
}

type authTransport struct {
	next   http.RoundTripper
	tokens *servicetoken.Tokens
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		auth, err := t.tokens.Authorization(req.Context())
		if err != nil {
			return nil, fmt.Errorf("Compute Manager lifecycle identity: %w", err)
		}
		clone := req.Clone(req.Context())
		clone.Header = req.Header.Clone()
		clone.Header.Set("Authorization", auth)
		req = clone
	}
	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(req)
}

func (a *cloudComputeReleaseAdapter) ReleaseCloudCompute(ctx context.Context, request releases.CloudReleaseRequest) (*releases.CloudReleaseResult, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(request.InstanceID) == "" || len(request.IdempotencyKey) < 16 || len(request.IdempotencyKey) > 128 {
		return nil, fmt.Errorf("exact CM release key and instance are required")
	}
	service, err := a.cmClient(ctx)
	if err != nil {
		return nil, err
	}
	var response *instancev1.GetDestroyReceiptResponse
	prior, readErr := service.GetDestroyReceipt(ctx, connect.NewRequest(&instancev1.GetDestroyReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
	if readErr == nil && prior != nil {
		response = prior.Msg
	} else if connect.CodeOf(readErr) == connect.CodeNotFound {
		_, destroyErr := service.DestroyInstance(ctx, connect.NewRequest(&instancev1.DestroyInstanceRequest{Id: request.InstanceID, IdempotencyKey: request.IdempotencyKey}))
		// The immediate response is not proof. Always reconcile the exact key.
		prior, readErr = service.GetDestroyReceipt(ctx, connect.NewRequest(&instancev1.GetDestroyReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
		if readErr != nil {
			if destroyErr != nil {
				return nil, fmt.Errorf("destroy reply uncertain (%v) and exact CM release readback unavailable: %w", destroyErr, readErr)
			}
			return nil, fmt.Errorf("exact CM release readback unavailable: %w", readErr)
		}
		if prior != nil {
			response = prior.Msg
		}
	} else {
		return nil, fmt.Errorf("exact CM release receipt lookup failed: %w", readErr)
	}
	if response == nil || response.GetReceipt() == nil {
		return nil, fmt.Errorf("CM release readback returned no receipt")
	}
	r := response.GetReceipt()
	if r.GetIdempotencyKey() != request.IdempotencyKey || r.GetOwner() != "deployment-manager" || r.GetInstanceId() != request.InstanceID || r.GetProvider() != request.Provider || r.GetProviderInstanceId() != request.ProviderInstanceID || r.GetAction() != "release" {
		return nil, fmt.Errorf("CM release receipt does not match exact operation owner and target")
	}
	encoded, err := protojson.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode exact CM release receipt: %w", err)
	}
	return &releases.CloudReleaseResult{Receipt: encoded}, nil
}

func (a *cloudComputeReleaseAdapter) RenewCloudLease(ctx context.Context, request releases.CloudLeaseRenewalRequest) (*releases.CloudLeaseRenewalResult, error) {
	if strings.TrimSpace(request.IdempotencyKey) == "" || strings.TrimSpace(request.InstanceID) == "" || request.RequestedUntil.IsZero() {
		return nil, fmt.Errorf("exact CM renewal key, instance and requested expiry are required")
	}
	service, err := a.cmClient(ctx)
	if err != nil {
		return nil, err
	}
	read := func() (*instancev1.RenewalReceipt, error) {
		response, err := service.GetRenewalReceipt(ctx, connect.NewRequest(&instancev1.GetRenewalReceiptRequest{IdempotencyKey: request.IdempotencyKey}))
		if err != nil {
			return nil, err
		}
		if response == nil || response.Msg.GetReceipt() == nil {
			return nil, fmt.Errorf("CM renewal readback returned no receipt")
		}
		return response.Msg.GetReceipt(), nil
	}
	receipt, readErr := read()
	if connect.CodeOf(readErr) == connect.CodeNotFound {
		_, requestErr := service.RequestRenewal(ctx, connect.NewRequest(&instancev1.RequestRenewalRequest{IdempotencyKey: request.IdempotencyKey, InstanceId: request.InstanceID, RequestedUntil: timestamppb.New(request.RequestedUntil.UTC())}))
		// Regardless of the immediate reply, only exact keyed readback is evidence.
		receipt, readErr = read()
		if readErr != nil && requestErr != nil {
			return nil, fmt.Errorf("renewal reply uncertain (%v); exact CM readback unavailable: %w", requestErr, readErr)
		}
	}
	if readErr != nil {
		return nil, fmt.Errorf("exact CM renewal receipt lookup failed: %w", readErr)
	}
	encoded, err := protojson.Marshal(receipt)
	if err != nil {
		return nil, fmt.Errorf("encode exact CM renewal receipt: %w", err)
	}
	return &releases.CloudLeaseRenewalResult{Receipt: encoded}, nil
}
