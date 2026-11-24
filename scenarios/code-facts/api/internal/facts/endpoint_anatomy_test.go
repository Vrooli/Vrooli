package facts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndpointAnatomyJoinsDeclaredRouteHandlerAndAbstains(t *testing.T) {
	root, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatal(err)
	}
	positive, err := scanEndpointAnatomy(root, "code-facts", "persistence patterns", 10)
	if err != nil {
		t.Fatal(err)
	}
	var joined *EndpointAnatomy
	for i := range positive.Results {
		if positive.Results[i].ID == "code-facts:endpoint:facts_persistence_pattern_search" {
			joined = &positive.Results[i]
			break
		}
	}
	if joined == nil || joined.State != "DECLARATION_ROUTE_HANDLER_JOINED" || joined.HandlerSource.Line == 0 || joined.HandlerSource.SHA256 == "" || joined.Registration.Line == 0 {
		t.Fatalf("expected declaration/route/handler join, got %#v", positive.Results)
	}
	negative, err := scanEndpointAnatomy(root, "code-facts", "underwater penguin telescope", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(negative.Results) != 0 {
		t.Fatalf("unrelated endpoint query should abstain: %#v", negative.Results)
	}
}

func TestEndpointAnatomyKeepsDuplicateRouteHandlersAmbiguous(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "api", "handlers", "facts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var module strings.Builder
	module.WriteString("package facts\nvar _ = module.EndpointDescriptor{ID: \"duplicate\", Path: \"/api/v1/duplicate\", Method: \"POST\", Summary: \"duplicate route\"}\nfunc mount(r *mux.Router) {\n")
	var handlers strings.Builder
	handlers.WriteString("package facts\n")
	for i := 0; i < maxEndpointRouteCandidates+2; i++ {
		fmt.Fprintf(&module, "r.Handle(\"/api/v1/duplicate\", handler%d()).Methods(http.MethodPost)\n", i)
		fmt.Fprintf(&handlers, "func handler%d() {}\n", i)
	}
	module.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "module.go"), []byte(module.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "handlers.go"), []byte(handlers.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	response, err := scanEndpointAnatomy(root, "code-facts", "duplicate route", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 {
		t.Fatalf("expected one declared endpoint row, got %#v", response.Results)
	}
	row := response.Results[0]
	if row.State != "ROUTE_AMBIGUOUS" || row.Handler != "" || row.Registration.Path != "" || len(row.RouteCandidates) != maxEndpointRouteCandidates || row.RouteCandidateTotal != maxEndpointRouteCandidates+2 || row.RouteCandidatesOmitted != 2 {
		t.Fatalf("duplicate registrations must remain ambiguous without a selected handler: %#v", row)
	}
	if row.RouteCandidates[0].Registration.Line == 0 || row.RouteCandidates[0].Registration.SHA256 == "" || row.RouteCandidates[1].HandlerSource.Path == "" {
		t.Fatalf("candidate source provenance was not retained: %#v", row.RouteCandidates)
	}
	limits := strings.Join(row.Limits, " ")
	if !strings.Contains(limits, "runtime routing precedence is unknown") || !strings.Contains(limits, "2 candidates were omitted") {
		t.Fatalf("ambiguity boundary was not explicit: %#v", row.Limits)
	}
}
