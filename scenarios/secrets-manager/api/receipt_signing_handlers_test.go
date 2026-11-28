package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/receiptsigning"
)

type fakeRotationSigner struct{ called bool }

func (s *fakeRotationSigner) Rotate(context.Context) (receiptsigning.Health, error) {
	s.called = true
	return receiptsigning.Health{Ready: true, Provider: "credential-authority-ed25519", KeyID: "key:v2", Production: true, RotationOK: true}, nil
}

func TestReceiptSigningStatusReportsExplicitDevelopmentModeWithoutSecrets(t *testing.T) {
	handler := NewReceiptSigningHandlers()
	handler.configPath = func() (string, error) { return t.TempDir() + "/missing.json", nil }
	request := httptest.NewRequest(http.MethodGet, "/receipt-signing/status", nil)
	response := httptest.NewRecorder()
	handler.Status(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("Status() status = %d", response.Code)
	}
	body := response.Body.String()
	if !(strings.Contains(body, `"configured":true`) && strings.Contains(body, `"mode":"development"`) && strings.Contains(body, `"production":false`)) {
		t.Fatalf("Status() = %s", body)
	}
}

func TestReceiptSigningStatusHidesInvalidConfigurationDetail(t *testing.T) {
	handler := NewReceiptSigningHandlers()
	path := filepath.Join(t.TempDir(), "invalid.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler.configPath = func() (string, error) { return path, nil }
	request := httptest.NewRequest(http.MethodGet, "/receipt-signing/status", nil)
	response := httptest.NewRecorder()
	handler.Status(response, request)
	if !strings.Contains(response.Body.String(), `"error":"invalid receipt signing configuration"`) || strings.Contains(response.Body.String(), "invalid.json") {
		t.Fatalf("Status() leaked runtime path: %s", response.Body.String())
	}
}

func TestReceiptSigningRotationRequiresAuthorizedMTLSOperator(t *testing.T) {
	signer := &fakeRotationSigner{}
	handler := NewReceiptSigningHandlers()
	handler.operatorSigner = func() (rotationSigner, []string, error) { return signer, []string{"vrooli-operator"}, nil }
	request := httptest.NewRequest(http.MethodPost, "/receipt-signing/rotate", nil)
	request.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{{Subject: pkix.Name{CommonName: "vrooli-operator"}}}, VerifiedChains: [][]*x509.Certificate{{{Subject: pkix.Name{CommonName: "vrooli-operator"}}}}}
	response := httptest.NewRecorder()
	handler.Rotate(response, request)
	if response.Code != http.StatusOK || !signer.called {
		t.Fatalf("Rotate() status=%d called=%t body=%s", response.Code, signer.called, response.Body.String())
	}
}

func TestReceiptSigningRotationRejectsUnverifiedRequest(t *testing.T) {
	handler := NewReceiptSigningHandlers()
	handler.operatorSigner = func() (rotationSigner, []string, error) {
		return &fakeRotationSigner{}, []string{"vrooli-operator"}, nil
	}
	response := httptest.NewRecorder()
	handler.Rotate(response, httptest.NewRequest(http.MethodPost, "/receipt-signing/rotate", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("Rotate() status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
