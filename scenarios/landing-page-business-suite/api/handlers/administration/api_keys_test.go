package administration

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	admin "landing-page-business-suite-api/internal/administration"
)

func TestAPIKeyConnectListReturnsOnlyDisplaySafeMetadata(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	handler := NewAPIKeyConnectHandler(&apiKeyConnectFake{keys: []admin.APIKey{{ID: "key-1", Provider: "openai", KeyHint: "****1234", IsActive: true, CreatedAt: now, UpdatedAt: now}}})

	response, err := handler.ListAPIKeys(context.Background(), connect.NewRequest(&lpbsv1.ListAPIKeysRequest{}))
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if got := response.Msg.GetKeys(); len(got) != 1 || got[0].GetId() != "key-1" || got[0].GetKeyHint() != "****1234" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestAPIKeyConnectCreateForwardsCredentialOnlyToService(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	fake := &apiKeyConnectFake{stored: &admin.APIKey{ID: "key-2", Provider: "openai", KeyHint: "****abcd", CreatedAt: now, UpdatedAt: now}}
	handler := NewAPIKeyConnectHandler(fake)

	response, err := handler.CreateAPIKey(context.Background(), connect.NewRequest(&lpbsv1.CreateAPIKeyRequest{Provider: "openai", Key: "raw-secret"}))
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if fake.provider != "openai" || fake.rawKey != "raw-secret" {
		t.Fatalf("service received provider=%q key=%q", fake.provider, fake.rawKey)
	}
	if got := response.Msg.GetKey(); got.GetKeyHint() != "****abcd" || got.GetProvider() != "openai" {
		t.Fatalf("unexpected safe response: %+v", got)
	}
}

func TestAPIKeyConnectTestAndSetActiveForwardRequests(t *testing.T) {
	fake := &apiKeyConnectFake{testSuccess: true, testMessage: "valid"}
	handler := NewAPIKeyConnectHandler(fake)

	testResponse, err := handler.TestAPIKey(context.Background(), connect.NewRequest(&lpbsv1.TestAPIKeyRequest{Provider: "anthropic"}))
	if err != nil {
		t.Fatalf("TestAPIKey: %v", err)
	}
	if !testResponse.Msg.GetSuccess() || testResponse.Msg.GetMessage() != "valid" || fake.testProvider != "anthropic" {
		t.Fatalf("unexpected test response=%+v provider=%q", testResponse.Msg, fake.testProvider)
	}
	if _, err := handler.SetAPIKeyActive(context.Background(), connect.NewRequest(&lpbsv1.SetAPIKeyActiveRequest{Provider: "anthropic", Active: false})); err != nil {
		t.Fatalf("SetAPIKeyActive: %v", err)
	}
	if fake.activeProvider != "anthropic" || fake.active {
		t.Fatalf("active request provider=%q active=%v", fake.activeProvider, fake.active)
	}
}

func TestAPIKeyConnectDeleteMapsMissingProviderToInvalidArgument(t *testing.T) {
	handler := NewAPIKeyConnectHandler(&apiKeyConnectFake{})
	_, err := handler.DeleteAPIKey(context.Background(), connect.NewRequest(&lpbsv1.DeleteAPIKeyRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code=%s, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestAPIKeyConnectDeleteMapsMissingKeyToNotFound(t *testing.T) {
	handler := NewAPIKeyConnectHandler(&apiKeyConnectFake{deleteErr: errors.New("api key not found: openai")})
	_, err := handler.DeleteAPIKey(context.Background(), connect.NewRequest(&lpbsv1.DeleteAPIKeyRequest{Provider: "openai"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code=%s, want not found (err=%v)", connect.CodeOf(err), err)
	}
}

type apiKeyConnectFake struct {
	keys           []admin.APIKey
	stored         *admin.APIKey
	deleteErr      error
	provider       string
	rawKey         string
	testProvider   string
	testSuccess    bool
	testMessage    string
	activeProvider string
	active         bool
}

func (f apiKeyConnectFake) List(context.Context) ([]admin.APIKey, error) { return f.keys, nil }
func (f *apiKeyConnectFake) Store(_ context.Context, provider, key string) (*admin.APIKey, error) {
	f.provider, f.rawKey = provider, key
	if f.stored == nil {
		return nil, errors.New("not implemented")
	}
	return f.stored, nil
}
func (f apiKeyConnectFake) Delete(context.Context, string) error { return f.deleteErr }
func (f *apiKeyConnectFake) Test(_ context.Context, provider string) (bool, string, error) {
	f.testProvider = provider
	return f.testSuccess, f.testMessage, nil
}

func (f *apiKeyConnectFake) SetActive(_ context.Context, provider string, active bool) error {
	f.activeProvider, f.active = provider, active
	return nil
}
