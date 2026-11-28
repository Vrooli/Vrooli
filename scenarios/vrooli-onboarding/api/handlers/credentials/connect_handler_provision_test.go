package credentials

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	credentialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials"
	internalcredentials "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestProvisionCredentialRejectsInvalidIdentityBeforeProviderCall(t *testing.T) {
	calls := 0
	handler := NewConnectHandler(internalcredentials.Service{ProvisionFn: func(context.Context, string, string, string) (credentialclient.ProvisionResponse, error) {
		calls++
		return credentialclient.ProvisionResponse{}, nil
	}})

	_, err := handler.ProvisionCredential(context.Background(), connect.NewRequest(&credentialsv1.ProvisionCredentialRequest{
		LogicalId: "not a credential id", Field: "token", Value: "secret",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, err = %v; want invalid argument", connect.CodeOf(err), err)
	}
	if calls != 0 {
		t.Fatalf("provider calls = %d, want zero", calls)
	}
}

func TestProvisionCredentialDoesNotMisclassifyProviderErrorText(t *testing.T) {
	const privateDetail = "remote provider missing entry secret-value"
	handler := NewConnectHandler(internalcredentials.Service{ProvisionFn: func(context.Context, string, string, string) (credentialclient.ProvisionResponse, error) {
		return credentialclient.ProvisionResponse{}, errors.New(privateDetail)
	}})

	_, err := handler.ProvisionCredential(context.Background(), connect.NewRequest(&credentialsv1.ProvisionCredentialRequest{
		LogicalId: "vrooli/demo", Field: "token", Value: "secret-value",
	}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v, err = %v; want unavailable provider signal", connect.CodeOf(err), err)
	}
	if strings.Contains(err.Error(), privateDetail) || strings.Contains(err.Error(), "secret-value") {
		t.Fatalf("provider details leaked in response: %v", err)
	}
}

func TestProvisionCredentialReturnsOnlyMetadataOnSuccess(t *testing.T) {
	const secret = "not-returned-secret"
	var gotID, gotField, gotValue string
	handler := NewConnectHandler(internalcredentials.Service{ProvisionFn: func(_ context.Context, logicalID, field, value string) (credentialclient.ProvisionResponse, error) {
		gotID, gotField, gotValue = logicalID, field, value
		return credentialclient.ProvisionResponse{Version: "v3"}, nil
	}})

	response, err := handler.ProvisionCredential(context.Background(), connect.NewRequest(&credentialsv1.ProvisionCredentialRequest{
		LogicalId: " vrooli/demo ", Value: secret,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if gotID != "vrooli/demo" || gotField != "value" || gotValue != secret {
		t.Fatalf("provider request = (%q, %q, %q)", gotID, gotField, gotValue)
	}
	if response.Msg.GetLogicalId() != gotID || response.Msg.GetField() != "value" || response.Msg.GetVersion() != "v3" || response.Msg.GetStatus() != "provisioned" {
		t.Fatalf("provision response = %+v, want metadata", response.Msg)
	}
	encoded, err := protojson.Marshal(response.Msg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), secret) {
		t.Fatalf("provision response leaked credential value: %s", encoded)
	}
}
