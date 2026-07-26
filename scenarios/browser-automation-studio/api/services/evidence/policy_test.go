package evidence

import (
	"strings"
	"testing"

	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

func TestDescribeHarIsProtectedAndHasIntegrity(t *testing.T) {
	descriptor := Describe("har", "application/json", []byte(`{"log":{}}`), DefaultPolicy())
	if descriptor.Kind != basevidence.ArtifactKind_ARTIFACT_KIND_HAR || descriptor.Access != basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY {
		t.Fatalf("HAR descriptor policy = %#v", descriptor)
	}
	if descriptor.Retention != basevidence.RetentionClass_RETENTION_CLASS_PROTECTED || !descriptor.Redacted || len(descriptor.SHA256) != 64 {
		t.Fatalf("HAR descriptor metadata = %#v", descriptor)
	}
}

func TestSanitizeHARRedactsHeadersQueriesAndBodies(t *testing.T) {
	// enforces invariant: harDerivativeIsRedacted
	raw := []byte(`{"log":{"entries":[{"request":{"url":"https://example.test/?token=secret&ok=yes","headers":[{"name":"Authorization","value":"Bearer secret"}],"postData":{"text":"secret"}},"response":{"content":{"text":"secret"}}}]}}`)
	sanitized, err := SanitizeHAR(raw, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	got := string(sanitized)
	if strings.Contains(got, "Bearer secret") || strings.Contains(got, "token=secret") || strings.Contains(got, `\"text\":\"secret\"`) {
		t.Fatalf("secret leaked from sanitized HAR: %s", got)
	}
	if !strings.Contains(got, redactedValue) {
		t.Fatalf("redaction marker missing: %s", got)
	}
}
