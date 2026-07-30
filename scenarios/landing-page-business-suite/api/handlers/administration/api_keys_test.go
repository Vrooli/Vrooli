package administration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	admin "landing-page-business-suite-api/internal/administration"
)

func TestCreateAPIKeyRejectsMalformedJSONBeforeServiceAccess(t *testing.T) {
	w := httptest.NewRecorder()
	CreateAPIKey(APIKeyDependencies{
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	}).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", strings.NewReader("{")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAPIKeyConnectListReturnsOnlyDisplaySafeMetadata(t *testing.T) {
	now := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)
	handler := NewAPIKeyConnectHandler(apiKeyConnectFake{keys: []admin.APIKey{{ID: "key-1", Provider: "openai", KeyHint: "****1234", IsActive: true, CreatedAt: now, UpdatedAt: now}}})

	response, err := handler.ListAPIKeys(context.Background(), connect.NewRequest(&lpbsv1.ListAPIKeysRequest{}))
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if got := response.Msg.GetKeys(); len(got) != 1 || got[0].GetId() != "key-1" || got[0].GetKeyHint() != "****1234" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestAPIKeyConnectDeleteMapsMissingProviderToInvalidArgument(t *testing.T) {
	handler := NewAPIKeyConnectHandler(apiKeyConnectFake{})
	_, err := handler.DeleteAPIKey(context.Background(), connect.NewRequest(&lpbsv1.DeleteAPIKeyRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code=%s, want invalid argument (err=%v)", connect.CodeOf(err), err)
	}
}

func TestAPIKeyConnectDeleteMapsMissingKeyToNotFound(t *testing.T) {
	handler := NewAPIKeyConnectHandler(apiKeyConnectFake{deleteErr: errors.New("api key not found: openai")})
	_, err := handler.DeleteAPIKey(context.Background(), connect.NewRequest(&lpbsv1.DeleteAPIKeyRequest{Provider: "openai"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("code=%s, want not found (err=%v)", connect.CodeOf(err), err)
	}
}

type apiKeyConnectFake struct {
	keys      []admin.APIKey
	deleteErr error
}

func (f apiKeyConnectFake) List(context.Context) ([]admin.APIKey, error) { return f.keys, nil }
func (f apiKeyConnectFake) Store(context.Context, string, string) (*admin.APIKey, error) {
	return nil, errors.New("not implemented")
}
func (f apiKeyConnectFake) Delete(context.Context, string) error { return f.deleteErr }
func (f apiKeyConnectFake) Test(context.Context, string) (bool, string, error) {
	return false, "", nil
}
func (f apiKeyConnectFake) SetActive(context.Context, string, bool) error { return nil }
