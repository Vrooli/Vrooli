package module

import "testing"

func TestHealthEndpointStandardShapeAndOwnerPayloads(t *testing.T) {
	endpoint := HealthEndpoint(HealthEndpointConfig{
		ResponseProtoFullName: "vrooli.example.v1.health.Response",
		ErrorProtoFullName:    "vrooli.example.v1.errors.ErrorEnvelope",
	})
	if endpoint.ID != "health" || endpoint.Path != "/health" || endpoint.Method != "GET" {
		t.Fatalf("endpoint identity = %+v", endpoint)
	}
	if endpoint.Response.Type != "object" || endpoint.Response.Properties["dependencies"] != "object" {
		t.Fatalf("response schema = %+v", endpoint.Response)
	}
	payloads := endpoint.RESTException.ProtoPayloads
	if endpoint.RESTException.Reason != RESTReasonOpsProbe || payloads.Request.Transport != "none" {
		t.Fatalf("REST exception = %+v", endpoint.RESTException)
	}
	if payloads.Response.ProtoFullName != "vrooli.example.v1.health.Response" || payloads.Error.ProtoFullName != "vrooli.example.v1.errors.ErrorEnvelope" {
		t.Fatalf("owner payloads = %+v", payloads)
	}
}

func TestHealthEndpointSupportsOwnerResponseExtensions(t *testing.T) {
	endpoint := HealthEndpoint(HealthEndpointConfig{
		ResponseProtoFullName:        "vrooli.code_facts.v1.health.Response",
		ErrorProtoFullName:           "vrooli.code_facts.v1.shared.ErrorEnvelope",
		ResponseSchemaType:           "Response",
		AdditionalResponseProperties: map[string]string{"metrics": "object"},
	})
	if endpoint.Response.Type != "Response" || endpoint.Response.Properties["metrics"] != "object" {
		t.Fatalf("extended response schema = %+v", endpoint.Response)
	}
	if _, exists := endpoint.Response.Properties["readiness"]; !exists {
		t.Fatal("owner extension removed the common health response fields")
	}
}
