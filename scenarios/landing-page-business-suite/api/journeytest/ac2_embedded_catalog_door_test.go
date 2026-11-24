package journeytest

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// This crosses the funded LPBS model-list proxy into a production Gateway
// process built on its embedded supplier catalog, without a synthetic route,
// approval, or agreement overlay. Candidate drafts must remain unavailable to
// the user-facing credits chooser. It does not prove LPBS J13 hosted serving.
func TestAC2EmbeddedCandidateCatalogStaysFailClosedThroughLPBSCreditsModelList(t *testing.T) {
	t.Setenv("VROOLI_JOURNEY_EMBEDDED_CATALOG_ONLY", "1")
	s := newSpine(t)

	request, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/v1/ai/models?harness=opencode", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+s.key.Token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request account-scoped model chooser through the LPBS Door: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("LPBS model chooser status = %d, want 200", response.StatusCode)
	}
	if got := response.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("LPBS model chooser cache policy = %q, want no-store", got)
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode LPBS model chooser response: %v", err)
	}
	if len(result.Data) != 0 || result.Reason != "no_eligible_route" {
		t.Fatalf("embedded proposal catalog leaked through LPBS chooser: data=%+v reason=%q", result.Data, result.Reason)
	}
}

// This test is invoked by scenario-to-cloud after its production bundle builder
// creates an explicit LPBS/Gateway artifact and extracts the catalog bytes
// from the tarball. The Gateway journey helper loads those exact artifact bytes
// into its production policy source; LPBS then serves its authenticated credits
// model list through the normal Door proxy. The proposal catalog must stay
// unavailable without any route or approval overlay.
func TestAC2ProductionArtifactCatalogReachesLPBSCreditsChooser(t *testing.T) {
	catalogPath := os.Getenv("VROOLI_JOURNEY_PACKAGED_CATALOG_FILE")
	if catalogPath == "" {
		t.Fatal("scenario-to-cloud must provide catalog bytes extracted from the deterministic production bundle")
	}
	t.Setenv("VROOLI_JOURNEY_EMBEDDED_CATALOG_ONLY", "1")
	s := newSpine(t)

	request, err := http.NewRequest(http.MethodGet, s.server.URL+"/api/v1/ai/models?harness=opencode", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+s.key.Token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request LPBS model chooser over Gateway loaded from production artifact: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("artifact-backed LPBS model chooser status=%d cache=%q, want 200/no-store", response.StatusCode, response.Header.Get("Cache-Control"))
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode artifact-backed LPBS model chooser: %v", err)
	}
	if len(result.Data) != 0 || result.Reason != "no_eligible_route" {
		t.Fatalf("artifact proposal catalog was not fail-closed at LPBS chooser: data=%+v reason=%q", result.Data, result.Reason)
	}
}
