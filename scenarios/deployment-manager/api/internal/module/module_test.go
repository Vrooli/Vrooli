package module

import (
	"testing"

	"github.com/vrooli/api-core/endpoints"
)

func TestEndpointAliasesAndReasonsRemainAligned(t *testing.T) {
	var descriptor EndpointDescriptor
	var exception RESTException
	var payload RESTPayload
	var protoPayloads RESTProtoPayloads
	var schema Schema
	if _, _, _, _, _ = descriptor, exception, payload, protoPayloads, schema; false {
		t.Fatal("unreachable")
	}

	want := []endpoints.RESTReason{
		endpoints.RESTReasonMultipartUpload,
		endpoints.RESTReasonWebhookReceiver,
		endpoints.RESTReasonThirdPartyShape,
		endpoints.RESTReasonOpsProbe,
	}
	got := []endpoints.RESTReason{
		RESTReasonMultipartUpload,
		RESTReasonWebhookReceiver,
		RESTReasonThirdPartyShape,
		RESTReasonOpsProbe,
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reason %d = %q, want %q", i, got[i], want[i])
		}
	}
}
