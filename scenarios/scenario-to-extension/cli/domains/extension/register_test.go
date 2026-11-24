package extension

import (
	"encoding/json"
	"testing"

	"scenario-to-extension/cli/internal/support"
)

func TestManagedSessionTestPayloadUsesBuildRefAndOptionalIDAssertion(t *testing.T) {
	payload := managedSessionTestPayload("build-1", "session-1", "")
	if payload["build_id"] != "build-1" || payload["session_id"] != "session-1" {
		t.Fatalf("payload = %#v", payload)
	}
	if _, ok := payload["extension_id"]; ok {
		t.Fatalf("derived extension id must not be supplied as install authority: %#v", payload)
	}

	payload = managedSessionTestPayload("build-1", "session-1", "expected-id")
	if payload["extension_id"] != "expected-id" {
		t.Fatalf("optional expected extension id missing: %#v", payload)
	}
}

func TestManagedExtensionHeadersRequireAMContextAndStayRequestLocal(t *testing.T) {
	t.Setenv(identityTokenEnv, "")
	t.Setenv(grantReceiptEnv, "trusted-grant")
	if headers, err := managedExtensionHeadersFromEnv(); err == nil || headers != nil {
		t.Fatalf("missing identity accepted: headers=%v err=%v", headers, err)
	}
	t.Setenv(identityTokenEnv, " signed-run ")
	t.Setenv(grantReceiptEnv, " browser:run-1 ")
	headers, err := managedExtensionHeadersFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if headers["X-Agent-Identity-Token"] != "signed-run" || headers["X-Vrooli-Browser-Grant-Receipt"] != "browser:run-1" {
		t.Fatalf("authority headers = %#v", headers)
	}
	delete(headers, "X-Vrooli-Browser-Grant-Receipt")
	if _, err := managedExtensionHeadersFromEnv(); err != nil {
		t.Fatalf("request-local header mutation changed env-derived next request: %v", err)
	}
}

func TestCleanupReceiptSurvivesCLIResponseDecode(t *testing.T) {
	var result support.TestResult
	if err := json.Unmarshal([]byte(`{"success":false,"status":"failed","cleanup_receipt":{"session_id":"session-1","operation":"unload","extension_id":"extension-1","state":"unloaded"}}`), &result); err != nil {
		t.Fatal(err)
	}
	if result.CleanupReceipt == nil || result.CleanupReceipt.SessionID != "session-1" || result.CleanupReceipt.Operation != "unload" || result.CleanupReceipt.State != "unloaded" {
		t.Fatalf("cleanup evidence did not survive the CLI response type: %#v", result.CleanupReceipt)
	}
}
