package inference

import (
	"strings"
	"testing"
)

func TestInferenceEndpointDescriptorsExposeProfileAndPrivacyContract(t *testing.T) {
	want := map[string][]string{
		"inference_run":          {"profile", "privacy_class"},
		"inference_run_batch":    {"profile", "privacy_class"},
		"inference_decide":       {"profile", "privacy_class"},
		"inference_decide_batch": {"profile", "privacy_class"},
		"inference_embed":        {"profile", "privacy_class", "privacy_provenance"},
	}
	for _, endpoint := range Endpoints {
		properties, required := want[endpoint.ID]
		if !required {
			continue
		}
		for _, property := range properties {
			if _, ok := endpoint.Request.Properties[property]; !ok {
				t.Errorf("%s request descriptor omits %q", endpoint.ID, property)
			}
		}
		delete(want, endpoint.ID)
	}
	for id := range want {
		t.Errorf("endpoint descriptor %q is missing", id)
	}
}

func TestInferenceEmbedDescriptorDocumentsLocalOnlyBoundary(t *testing.T) {
	for _, endpoint := range Endpoints {
		if endpoint.ID == "inference_embed" {
			if endpoint.Request.Properties["profile"] != "Profile" {
				t.Fatal("embedding request must expose its explicit profile")
			}
			if !strings.Contains(endpoint.Description, "local-only profile") {
				t.Fatalf("embedding descriptor must explain its local-only boundary: %q", endpoint.Description)
			}
			return
		}
	}
	t.Fatal("embedding endpoint descriptor is missing")
}
