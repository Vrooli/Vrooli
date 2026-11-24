package credentials

import (
	"context"
	"errors"
	"testing"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
)

func TestProvisionValidatesIdentityBeforeCallingProvider(t *testing.T) {
	calls := 0
	service := Service{ProvisionFn: func(context.Context, string, string, string) (credentialclient.ProvisionResponse, error) {
		calls++
		return credentialclient.ProvisionResponse{Version: "v2"}, nil
	}}

	if _, err := service.Provision(context.Background(), "not a credential id", "api-key", "secret"); !errors.Is(err, ErrInvalidProvisionRequest) {
		t.Fatalf("invalid identity error = %v, want ErrInvalidProvisionRequest", err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want zero for invalid identity", calls)
	}
}

func TestProvisionTrimsAddressDefaultsFieldAndPreservesSecretValue(t *testing.T) {
	var gotID, gotField, gotValue string
	service := Service{ProvisionFn: func(_ context.Context, logicalID, field, value string) (credentialclient.ProvisionResponse, error) {
		gotID, gotField, gotValue = logicalID, field, value
		return credentialclient.ProvisionResponse{Version: "v2"}, nil
	}}

	response, err := service.Provision(context.Background(), " vrooli/demo ", " ", " secret with spaces ")
	if err != nil {
		t.Fatal(err)
	}
	if gotID != "vrooli/demo" || gotField != "value" || gotValue != " secret with spaces " {
		t.Fatalf("provider request = (%q, %q, %q), want normalized address/default field and exact value", gotID, gotField, gotValue)
	}
	if response.Version != "v2" {
		t.Fatalf("response version = %q, want v2", response.Version)
	}
}
