package validation

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	internalvalidation "plan-manager/internal/validation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type testGenieReceiptClient struct {
	resolver interface {
		ResolveScenarioURLDefault(context.Context, string) (string, error)
	}
	http connect.HTTPClient
}

func newTestGenieReceiptClient() internalvalidation.ReceiptClient {
	return testGenieReceiptClient{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: http.DefaultClient}
}

func (c testGenieReceiptClient) client(ctx context.Context) (validationconnect.ValidationServiceClient, error) {
	baseURL, err := c.resolver.ResolveScenarioURLDefault(ctx, "test-genie")
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie URL: %w", err)
	}
	return validationconnect.NewValidationServiceClient(c.http, baseURL), nil
}

func (c testGenieReceiptClient) CreateValidation(ctx context.Context, intent *validationv1.ValidationIntent) (*validationv1.ValidationReceipt, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.CreateValidation(ctx, connect.NewRequest(&validationv1.CreateValidationRequest{Intent: intent}))
	if err != nil {
		return nil, fmt.Errorf("create test-genie validation receipt: %w", err)
	}
	return response.Msg.GetReceipt(), nil
}

func (c testGenieReceiptClient) GetValidation(ctx context.Context, receiptID string) (*validationv1.ValidationReceipt, error) {
	client, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetValidation(ctx, connect.NewRequest(&validationv1.GetValidationRequest{ReceiptId: receiptID}))
	if err != nil {
		return nil, fmt.Errorf("get test-genie validation receipt: %w", err)
	}
	return response.Msg.GetReceipt(), nil
}
