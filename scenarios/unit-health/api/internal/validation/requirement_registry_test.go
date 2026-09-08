package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type registryURL string

func (r registryURL) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

func TestOwnerRequirementRegistryPreservesExactIDsAndResponsibilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scenarios/demo/requirements" || r.URL.Query().Get("view") != "registry" {
			t.Errorf("wrong owner request: %s", r.URL)
		}
		w.Write([]byte(`{"schemaVersion":"requirement-registry/v1","requirements":[{"id":"UH-CORE-001"},{"id":"UH-CORE-010","validation":[{"type":"test","phase":"integration","ref":"test/integration.sh"}]}]}`))
	}))
	defer server.Close()
	registry, err := (testGenieRegistryReader{Resolver: registryURL(server.URL)}).Read(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Requirements) != 2 || registry.Requirements[0].ID != "UH-CORE-001" || registry.Requirements[1].ID != "UH-CORE-010" || registry.Requirements[1].Validations[0].Phase != "integration" {
		t.Fatalf("owner declarations changed: %+v", registry)
	}
}

func TestOwnerRequirementRegistryRejectsUnknownAndIncompleteResponses(t *testing.T) {
	for _, payload := range []string{`{`, `{}`, `{"schemaVersion":"future","requirements":[]}`, `{"schemaVersion":"requirement-registry/v1"}`, `{"schemaVersion":"requirement-registry/v1","requirements":[{"id":"UH-CORE-001"},{"id":"UH-CORE-001"}]}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(payload)) }))
		registry, err := (testGenieRegistryReader{Resolver: registryURL(server.URL)}).Read(context.Background(), "demo")
		server.Close()
		if err == nil || registry.SchemaVersion != "" {
			t.Fatalf("incomplete response accepted: %s: %+v %v", payload, registry, err)
		}
	}
}
