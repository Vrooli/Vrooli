package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestConsumerSigningRotationRejectsSecretMaterialAndRequiresExactVersion(t *testing.T) {
	cases := []string{
		`{"expected_active_version":"abc","private_key_pem":"fixture-secret-must-not-echo"}`,
		`{"expected_active_version":"abc","value":"fixture-secret-must-not-echo"}`,
		`{"expected_active_version":"abc"}{"again":true}`,
		`{}`,
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/consumer-signing/rotate", strings.NewReader(body))
			(&Server{}).adminConsumerSigningRotation(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "fixture-secret-must-not-echo") {
				t.Fatal("rejected secret input echoed in response")
			}
		})
	}
}

func TestConsumerSigningRotationRoutesRequireAdminAndRecentReauth(t *testing.T) {
	contents, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(contents)
	if !strings.Contains(source, `"/api/v1/admin/auth/consumer-signing/status", s.requireAdmin(s.adminConsumerSigningStatus)`) {
		t.Fatal("consumer signing status route is not admin-only")
	}
	if !strings.Contains(source, `"/api/v1/admin/auth/consumer-signing/rotate", s.requireAdmin(s.requireAdminStepUp(s.adminConsumerSigningRotation))`) {
		t.Fatal("consumer signing rotation route lacks admin and recent-reauth gates")
	}
	if !strings.Contains(source, `authadmin.ConsumerSigningJWKSCacheMaxAge.Seconds()`) {
		t.Fatal("published JWKS cache lifetime drifted from signer overlap retention policy")
	}
}

func TestConsumerSigningIssuerUsesRuntimeDefaultWhenUnset(t *testing.T) {
	t.Setenv("JWT_ISSUER", "")
	if got := consumerSigningIssuer(); got != "landing-page-business-suite" {
		t.Fatalf("consumer signing issuer = %q, want the runtime default", got)
	}
}

func TestConsumerSigningRotationAuditsMetadataBeforeAuthorityMutation(t *testing.T) {
	contents, err := os.ReadFile("consumer_signing_rotation_admin.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(contents)
	intent := strings.Index(source, `"consumer_signing_rotation_requested"`)
	mutation := strings.Index(source, `administration.RotateConsumerSigningCredential(authority`)
	if intent < 0 || mutation < 0 || intent > mutation {
		t.Fatal("rotation must persist a metadata-only audit intent before candidate activation")
	}
	for _, forbidden := range []string{"PrivateKeyPEM", `json:"value"`, `"private_key_pem"`} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("admin rotation transport references secret material %q", forbidden)
		}
	}
}
