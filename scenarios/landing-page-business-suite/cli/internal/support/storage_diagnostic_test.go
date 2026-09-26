package support

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestStorageDiagnosticFromError(t *testing.T) {
	raw := []byte(`{"error":"HeadBucket denied","error_type":"head_bucket_denied","validation":{"ready":false,"diagnostic":{"code":"head_bucket_denied","summary":"HeadBucket was denied for bucket \"b\"","operation":"HeadBucket","bucket":"b","remediation":"attach VrooliDeliveryBucketAccess","http_status":403,"request_id":"req-1"}}}`)
	err := &cliutil.APIError{StatusCode: 403, Message: "HeadBucket denied", RawResponse: raw}

	diagnostic, ok := StorageDiagnosticFromError(err)
	if !ok {
		t.Fatal("expected a structured diagnostic")
	}
	if diagnostic.Code != "head_bucket_denied" || diagnostic.Operation != "HeadBucket" || diagnostic.RequestID != "req-1" {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}

	described := DescribeStorageFailure(err)
	for _, fragment := range []string{"head_bucket_denied", "operation=HeadBucket", "http_status=403", "request_id=req-1", "attach VrooliDeliveryBucketAccess"} {
		if !strings.Contains(described, fragment) {
			t.Fatalf("DescribeStorageFailure() = %q, missing %q", described, fragment)
		}
	}
}

func TestDescribeStorageFailureFallsBackToMessage(t *testing.T) {
	if _, ok := StorageDiagnosticFromError(errors.New("plain error")); ok {
		t.Fatal("plain error must not carry a diagnostic")
	}
	if got := DescribeStorageFailure(errors.New("plain error")); got != "plain error" {
		t.Fatalf("DescribeStorageFailure() = %q", got)
	}
}
