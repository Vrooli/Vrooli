// Package gatewayreq is the only place document-manager constructs an
// ai-gateway GatewayRequest.  Keeping this boundary narrow makes the
// residency guarantee auditable and testable.
package gatewayreq

import (
	"context"
	"fmt"
	"time"

	"document-manager/internal/sensitivity"

	gatewaypb "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

// DocumentClass is the minimum information needed by the routing policy.
type DocumentClass struct {
	PrivacyClass documentpb.PrivacyClass
}

type Options struct {
	Profile        gatewaypb.Profile
	Kind           gatewaypb.RequestKind
	Role           string
	Operation      string
	Timeout        time.Duration
	RequestID      string
	Metadata       map[string]string
	MaxOutputToken int32
}

type Builder interface {
	For(context.Context, DocumentClass, Options) (*gatewaypb.GatewayRequest, error)
}

type builder struct {
	scenario string
}

func New(scenario string) Builder { return builder{scenario: scenario} }

// For is the sole GatewayRequest construction site in the scenario.
func (b builder) For(_ context.Context, doc DocumentClass, opts Options) (*gatewaypb.GatewayRequest, error) {
	policy, err := sensitivity.PolicyFor(doc.PrivacyClass)
	if err != nil {
		return nil, err
	}
	if !policy.Allows(opts.Profile) {
		return nil, fmt.Errorf("profile %s is weaker than privacy class %s", opts.Profile, doc.PrivacyClass)
	}
	if opts.Role == "" || opts.Kind == gatewaypb.RequestKind_REQUEST_KIND_UNSPECIFIED {
		return nil, fmt.Errorf("gateway role and request kind are required")
	}
	timeoutMs := opts.Timeout / time.Millisecond
	if timeoutMs < 0 || timeoutMs > time.Duration(2_147_483_647) {
		return nil, fmt.Errorf("gateway timeout exceeds protobuf int32 range: %s", opts.Timeout)
	}
	request := &gatewaypb.GatewayRequest{
		Kind:            opts.Kind,
		Role:            opts.Role,
		Profile:         opts.Profile,
		PrivacyClass:    gatewayPrivacy(doc.PrivacyClass),
		Operation:       opts.Operation,
		Scenario:        b.scenario,
		TimeoutMs:       int32(timeoutMs), // #nosec G115 -- timeout is range-checked above.
		RequestId:       opts.RequestID,
		Metadata:        cloneMetadata(opts.Metadata),
		MaxOutputTokens: opts.MaxOutputToken,
	}
	return request, nil
}

func gatewayPrivacy(class documentpb.PrivacyClass) gatewaypb.PrivacyClass {
	return gatewaypb.PrivacyClass(class)
}

func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
