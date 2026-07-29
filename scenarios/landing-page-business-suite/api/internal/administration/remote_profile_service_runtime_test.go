package administration

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNewRemoteProfileServiceWithRuntimeUsesInjectedSecretResolver(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	lookups := []string{}
	service, err := NewRemoteProfileServiceWithRuntime(
		nil,
		nil,
		func(name string) string {
			lookups = append(lookups, name)
			if name == "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY" {
				return key
			}
			return ""
		},
		func() bool { return true },
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRemoteProfileServiceWithRuntime returned error: %v", err)
	}
	if len(service.EncryptionKey) != 32 {
		t.Fatalf("expected decoded 32-byte encryption key, got %d bytes", len(service.EncryptionKey))
	}
	if len(lookups) != 1 || lookups[0] != "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY" {
		t.Fatalf("expected injected resolver to provide preferred key, got %v", lookups)
	}
}

func TestNewRemoteProfileServiceWithRuntimeRejectsMissingProductionKey(t *testing.T) {
	_, err := NewRemoteProfileServiceWithRuntime(nil, nil, func(string) string { return "" }, func() bool { return true }, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "required in production") {
		t.Fatalf("expected production encryption-key error, got %v", err)
	}
}

func TestRemoteProfileServiceUsesInjectedProductionPolicy(t *testing.T) {
	service := &RemoteProfileService{IsProduction: func() bool { return true }}
	_, err := service.normalizeAPIBase("http://example.com/api/v1")
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected injected production policy to require HTTPS, got %v", err)
	}
}
