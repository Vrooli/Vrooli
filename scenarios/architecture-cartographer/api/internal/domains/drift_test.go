package domains

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchDocCodeDriftJoinsSourceAndAbstains(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	doc := "# Test\n\n## Domain Inventory\n| Domain | Responsibility | Source Paths |\n|---|---|---|\n| billing | Own invoice decisions. | `api/internal/billing/` |\n| absent | Own missing code. | `api/internal/absent/` |\n\n## Other\n"
	write("docs/concepts/DOMAINS.md", doc)
	write("api/internal/billing/service.go", "package billing\n\nfunc Charge() {}\n")
	write("api/internal/orphan/worker.go", "package orphan\n\nfunc Run() {}\n")
	mapData := DerivedDomainMap{Scenario: "fixture", Authority: SourceDomainsDoc, AuthorityConfidence: ConfidenceHigh, Domains: []DerivedDomain{{Name: "billing", Paths: []string{"api/internal/billing/"}, Responsibility: "Own invoice decisions.", Provenance: []Source{SourceDomainsDoc}}, {Name: "absent", Paths: []string{"api/internal/absent/"}, Responsibility: "Own missing code.", Provenance: []Source{SourceDomainsDoc}}}}
	matched, err := SearchDocCodeDrift(root, mapData, "billing", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(matched.Results) == 0 || matched.Results[0].Classification != "STRUCTURAL_PATH_MATCH" || len(matched.Results[0].Code) == 0 || matched.Results[0].Documentation.SHA256 == "" {
		t.Fatalf("missing source-backed match: %#v", matched)
	}
	missing, err := SearchDocCodeDrift(root, mapData, "absent", 10)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range missing.Results {
		if row.Classification == "MISSING_SOURCE_CANDIDATE" && row.Status == "CONTRADICTION_CANDIDATE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing documented path not surfaced: %#v", missing.Results)
	}
	negative, err := SearchDocCodeDrift(root, mapData, "underwater penguin", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(negative.Results) != 0 {
		t.Fatalf("unrelated query should abstain: %#v", negative.Results)
	}
}

func TestSearchDocCodeDriftKeepsFallbackAuthorityUnknown(t *testing.T) {
	got, err := SearchDocCodeDrift(t.TempDir(), DerivedDomainMap{Scenario: "fixture", Authority: SourceAPIFolders, AuthorityConfidence: ConfidenceLow}, "billing", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 || len(got.Degradations) == 0 {
		t.Fatalf("fallback authority must remain unknown: %#v", got)
	}
}

func TestSearchDocCodeDriftLinksLiteralRouteHandlerAndInternalDeclaration(t *testing.T) {
	root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, `package billing
func Charge() {}`)
	writeDriftFixtureFile(t, root, "api/handlers/graph/module.go", `package graph
import "fixture/internal/billing"
type Handler struct{}
func Mount(r *Router, h *Handler) { r.Handle("/api/v1/billing/charge", h.ChargeHTTP()).Methods("POST") }
func (h *Handler) ChargeHTTP() HandlerFunc { return func() { billing.Charge() } }
`)
	got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing api route direct handler", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 {
		t.Fatalf("missing source path result: %#v", got)
	}
	row := got.Results[0]
	if row.APIStatus != "ROUTE_HANDLER_DIRECT_CALL" || len(row.API) != 4 {
		t.Fatalf("expected exact literal route/handler/call/target chain: %+v", row)
	}
	if !strings.Contains(row.API[0].Basis, "POST /api/v1/billing/charge") {
		t.Fatalf("route literal absent from evidence: %+v", row.API)
	}
	for _, edge := range row.API {
		if edge.Path == "" || edge.Line <= 0 || edge.SHA256 == "" {
			t.Fatalf("missing path/line/hash: %+v", edge)
		}
	}
}

func TestSearchDocCodeDriftLinksLiteralHandleFuncRouteToMethodReference(t *testing.T) {
	root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
	writeDriftFixtureFile(t, root, "api/handlers/graph/module.go", `package graph
import "fixture/internal/billing"
type Handler struct{}
func Mount(r *Router, h *Handler) { r.HandleFunc("/api/v1/billing/charge", h.ChargeHTTP).Methods("POST") }
func (h *Handler) ChargeHTTP(w Writer, r *Request) { billing.Charge() }
`)
	got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing api route direct handler", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 || got.Results[0].APIStatus != "ROUTE_HANDLER_DIRECT_CALL" || len(got.Results[0].API) != 4 {
		t.Fatalf("HandleFunc method reference should join route, handler, call and declaration: %+v", got)
	}
	if !strings.Contains(got.Results[0].API[0].Basis, "POST /api/v1/billing/charge") {
		t.Fatalf("literal HandleFunc route missing from evidence: %+v", got.Results[0].API)
	}
}

func TestSearchDocCodeDriftLinksTypedStdlibServeMuxMethodPattern(t *testing.T) {
	root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
	writeDriftFixtureFile(t, root, "api/handlers/billing/routes.go", `package billinghandler
import (
	"net/http"
	billing "fixture/internal/billing"
)
type Handler struct{}
func Mount(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /api/v1/billing/charge", h.ChargeHTTP)
}
func (h *Handler) ChargeHTTP(w http.ResponseWriter, r *http.Request) {
	billing.Charge()
}
`)
	got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing API route", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 || got.Results[0].APIStatus != "ROUTE_HANDLER_DIRECT_CALL" || len(got.Results[0].API) != 4 {
		t.Fatalf("typed stdlib ServeMux route should join to the direct handler chain: %+v", got)
	}
	if !strings.Contains(got.Results[0].API[0].Basis, "POST /api/v1/billing/charge") {
		t.Fatalf("method pattern missing from route evidence: %+v", got.Results[0].API)
	}
}

func TestSearchDocCodeDriftIgnoresCustomHandleFuncLookalike(t *testing.T) {
	root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
	writeDriftFixtureFile(t, root, "api/handlers/billing/routes.go", `package billinghandler
import "fixture/internal/billing"
type Router struct{}
type Handler struct{}
func Mount(mux *Router, h *Handler) {
	mux.HandleFunc("POST /api/v1/billing/charge", h.ChargeHTTP)
}
func (h *Handler) ChargeHTTP() { billing.Charge() }
`)
	got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing API route", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 || got.Results[0].APIStatus != "UNKNOWN_NO_SUPPORTED_LITERAL_ROUTE" {
		t.Fatalf("custom HandleFunc-shaped router must not be treated as net/http ServeMux: %+v", got)
	}
}

func TestSearchDocCodeDriftAPICompatibilityMatrixKeepsUncertainty(t *testing.T) {
	t.Run("missing declaration", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/absent/"}, nil, "")
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/absent/"}), "absent API", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) == 0 || got.Results[0].Classification != "MISSING_SOURCE_CANDIDATE" || !strings.HasPrefix(got.Results[0].APIStatus, "UNKNOWN") {
			t.Fatalf("missing claim must stay uncertain: %+v", got)
		}
	})
	t.Run("untracked source with direct route", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go", "api/internal/orphan/worker.go"}, "package billing\nfunc Charge() {}")
		writeDriftFixtureFile(t, root, "api/internal/orphan/worker.go", "package orphan\nfunc Run() {}")
		writeDriftFixtureFile(t, root, "api/handlers/graph/module.go", `package graph
import "fixture/internal/orphan"
type Handler struct{}
func Mount(r *Router, h *Handler) { r.Handle("/api/v1/orphan/run", h.OrphanHTTP()).Methods("POST") }
func (h *Handler) OrphanHTTP() HandlerFunc { return func() { orphan.Run() } }
`)
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "orphan API source", 10)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, row := range got.Results {
			if row.Classification == "UNDOCUMENTED_SOURCE_CANDIDATE" && row.Domain == "orphan" && row.APIStatus == "ROUTE_HANDLER_DIRECT_CALL" {
				found = true
			}
		}
		if !found {
			t.Fatalf("untracked source/API relation was lost: %+v", got)
		}
	})
	t.Run("duplicate route is ambiguous", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
		writeDriftFixtureFile(t, root, "api/handlers/graph/module.go", `package graph
import "fixture/internal/billing"
type Handler struct{}
func Mount(r *Router, h *Handler) { r.Handle("/api/v1/billing/charge", h.ChargeHTTP()).Methods("POST"); r.Handle("/api/v1/billing/charge", h.ChargeHTTP()).Methods("POST") }
func (h *Handler) ChargeHTTP() HandlerFunc { return func() { billing.Charge() } }
`)
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing api route", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) == 0 || got.Results[0].APIStatus != "UNKNOWN_AMBIGUOUS_ROUTE" {
			t.Fatalf("duplicate registration must be unknown: %+v", got)
		}
	})
	t.Run("dynamic route stays unknown", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
		writeDriftFixtureFile(t, root, "api/handlers/graph/module.go", `package graph
import "fixture/internal/billing"
type Handler struct{}
func Mount(r *Router, h *Handler, routePrefix string) { r.Handle(routePrefix+"/billing/charge", h.ChargeHTTP()).Methods("POST") }
func (h *Handler) ChargeHTTP() HandlerFunc { return func() { billing.Charge() } }
`)
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing API route", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) == 0 || got.Results[0].APIStatus != "UNKNOWN_DYNAMIC_ROUTE" {
			t.Fatalf("dynamic route must not look proven: %+v", got)
		}
	})
	t.Run("unsupported path syntax", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/billing/[x]"}, nil, "")
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/[x]"}), "billing unsupported", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) == 0 || got.Results[0].APIStatus != "UNKNOWN_UNSUPPORTED_PATH" {
			t.Fatalf("unsupported docs pattern must remain unknown: %+v", got)
		}
	})
	t.Run("handler scan degradation", func(t *testing.T) {
		root := writeDriftFixture(t, []string{"api/internal/billing/"}, []string{"api/internal/billing/service.go"}, "package billing\nfunc Charge() {}")
		for i := 0; i <= maxDriftHandlerFiles; i++ {
			writeDriftFixtureFile(t, root, fmt.Sprintf("api/handlers/generated/file-%03d.go", i), "package generated\nfunc F() {}\n")
		}
		got, err := SearchDocCodeDrift(root, driftFixtureMap("fixture", []string{"api/internal/billing/"}), "billing API", 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Results) == 0 || got.Results[0].APIStatus != "UNKNOWN_SOURCE_SCAN_CAPPED" || len(got.Degradations) == 0 {
			t.Fatalf("source cap must degrade API linkage: %+v", got)
		}
	})
}

func driftFixtureMap(scenario string, paths []string) DerivedDomainMap {
	return DerivedDomainMap{Scenario: scenario, Authority: SourceDomainsDoc, AuthorityConfidence: ConfidenceHigh, Domains: []DerivedDomain{{Name: "billing", Paths: paths, Responsibility: "Own billing behavior.", Provenance: []Source{SourceDomainsDoc}}}}
}

func writeDriftFixture(t *testing.T, paths, codeFiles []string, code string) string {
	t.Helper()
	root := t.TempDir()
	writeDriftFixtureFile(t, root, "docs/concepts/DOMAINS.md", "## Domain Inventory\n| Domain | Responsibility | Source Paths |\n|---|---|---|\n| billing | Own billing behavior. | "+strings.Join(paths, ", ")+" |\n")
	for _, path := range codeFiles {
		writeDriftFixtureFile(t, root, path, code)
	}
	return root
}

func writeDriftFixtureFile(t *testing.T, root, path, body string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
