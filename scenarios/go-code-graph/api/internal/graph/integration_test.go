package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go-code-graph/internal/graph"
)

// canonicalGraphBytes serializes g using the same scheme as graph.GraphHash:
// sorted attribute pairs, no HTML escaping, trailing newline. Keeping this
// helper inside the test (and matching hash.go exactly) ensures the
// expected-graph.json bytes the test compares against are exactly what
// GraphHash would consume.
func canonicalGraphBytes(t *testing.T, g graph.Graph) []byte {
	t.Helper()
	// Round-trip through the public GraphHash path indirectly: build the
	// same canonical structure and Encode it identically.
	type canonNode struct {
		ID         string         `json:"id"`
		Kind       graph.NodeKind `json:"kind"`
		Name       string         `json:"name"`
		Path       string         `json:"path"`
		Attributes [][]string     `json:"attributes,omitempty"`
	}
	type canonEdge struct {
		ID         string         `json:"id"`
		Kind       graph.EdgeKind `json:"kind"`
		From       string         `json:"from"`
		To         string         `json:"to"`
		Attributes [][]string     `json:"attributes,omitempty"`
	}
	type canonGraph struct {
		Nodes []canonNode `json:"nodes"`
		Edges []canonEdge `json:"edges"`
	}
	out := canonGraph{Nodes: make([]canonNode, 0, len(g.Nodes)), Edges: make([]canonEdge, 0, len(g.Edges))}
	for _, n := range g.Nodes {
		out.Nodes = append(out.Nodes, canonNode{ID: n.ID, Kind: n.Kind, Name: n.Name, Path: n.Path, Attributes: sortedPairs(n.Attributes)})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, canonEdge{ID: e.ID, Kind: e.Kind, From: e.From, To: e.To, Attributes: sortedPairs(e.Attributes)})
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		t.Fatalf("encode canonical graph: %v", err)
	}
	return buf.Bytes()
}

func sortedPairs(m map[string]string) [][]string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Stable lexicographic order.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	out := make([][]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, []string{k, m[k]})
	}
	return out
}

// resolveFixture returns the absolute path to a fixture directory given
// its path relative to api/.
func resolveFixture(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("resolve fixture %s: %v", rel, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %s missing: %v", abs, err)
	}
	return abs
}

func newRealService() *graph.Service {
	return graph.NewService(graph.NewPackagesLoader(), graph.NewPathMutex())
}

func TestExtractFixtures(t *testing.T) {
	cases := []struct {
		name       string
		fixtureRel string
	}{
		{"go-cycles", "../../../bas/fixtures/go-cycles"},
		{"go-mislocated", "../../../bas/fixtures/go-mislocated"},
		{"go-tests", "../../../bas/fixtures/go-tests"},
		{"go-usage-facts", "../../../bas/fixtures/go-usage-facts"},
	}

	svc := newRealService()
	update := os.Getenv("UPDATE_FIXTURES") == "1"

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			abs := resolveFixture(t, tc.fixtureRel)
			g, _, err := svc.Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}

			actual := canonicalGraphBytes(t, g)
			expectedPath := filepath.Join(abs, "expected-graph.json")

			if update {
				if err := os.WriteFile(expectedPath, actual, 0o644); err != nil {
					t.Fatalf("write expected-graph.json: %v", err)
				}
				t.Logf("wrote %s (%d bytes)", expectedPath, len(actual))
				return
			}

			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected-graph.json: %v (run with UPDATE_FIXTURES=1 to bootstrap)", err)
			}
			if !bytes.Equal(actual, expected) {
				t.Fatalf("canonical graph mismatch for %s\n--- expected (%d bytes) ---\n%s\n--- actual (%d bytes) ---\n%s",
					tc.name, len(expected), string(expected), len(actual), string(actual))
			}
		})
	}
}

func TestExtractGoTestsFixtureMarksTestOnlyEdges(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-tests")
	g, warnings, err := newRealService().Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
	if err != nil {
		t.Fatalf("Extract go-tests: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("go-tests fixture should extract without warnings, got %+v", warnings)
	}

	edges := map[string]graph.Edge{}
	for _, edge := range g.Edges {
		edges[edge.From+"->"+edge.To] = edge
	}

	liveFmt := edges["package:github.com/vrooli/fixtures/go-tests/lib->package:fmt"]
	if liveFmt.ID == "" {
		t.Fatalf("missing live fmt import edge: %+v", g.Edges)
	}
	if got := liveFmt.Attributes["test_only"]; got != "false" {
		t.Fatalf("live fmt import test_only = %q, want false", got)
	}

	internalHelper := edges["package:github.com/vrooli/fixtures/go-tests/lib->package:github.com/vrooli/fixtures/go-tests/helper"]
	if internalHelper.ID == "" {
		t.Fatalf("missing internal test helper import edge: %+v", g.Edges)
	}
	if got := internalHelper.Attributes["test_only"]; got != "true" {
		t.Fatalf("internal helper import test_only = %q, want true", got)
	}

	externalLib := edges["package:github.com/vrooli/fixtures/go-tests/lib_test->package:github.com/vrooli/fixtures/go-tests/lib"]
	if externalLib.ID == "" {
		t.Fatalf("missing external test package import edge: %+v", g.Edges)
	}
	if got := externalLib.Attributes["test_only"]; got != "true" {
		t.Fatalf("external test package import test_only = %q, want true", got)
	}
}

func TestExtractGenericNonScenarioModulePath(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-mislocated")
	if _, err := os.Stat(filepath.Join(abs, ".vrooli", "service.json")); err == nil {
		t.Fatalf("fixture should remain a generic module, not a Vrooli scenario: %s", abs)
	}

	g, warnings, err := newRealService().Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
	if err != nil {
		t.Fatalf("Extract generic module: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("generic module should extract without warnings, got %+v", warnings)
	}
	if len(g.Nodes) == 0 {
		t.Fatalf("generic module produced no graph nodes")
	}
}

func TestExtractGenericUsageFacts(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-usage-facts")
	g, warnings, err := newRealService().Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
	if err != nil {
		t.Fatalf("Extract usage facts: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("usage facts fixture should extract without warnings, got %+v", warnings)
	}

	nodes := map[graph.NodeKind][]graph.Node{}
	for _, node := range g.Nodes {
		nodes[node.Kind] = append(nodes[node.Kind], node)
	}
	for _, kind := range []graph.NodeKind{
		graph.NodeKindImportSpec,
		graph.NodeKindReference,
		graph.NodeKindCall,
		graph.NodeKindTypeUsage,
	} {
		if len(nodes[kind]) == 0 {
			t.Fatalf("missing %s nodes; graph has %d nodes", kind, len(g.Nodes))
		}
	}

	if !hasNodeAttr(nodes[graph.NodeKindImportSpec], "alias", "prod") {
		t.Fatalf("missing aliased import fact: %+v", nodes[graph.NodeKindImportSpec])
	}
	if !hasNodeAttr(nodes[graph.NodeKindImportSpec], "is_blank", "true") {
		t.Fatalf("missing blank import fact: %+v", nodes[graph.NodeKindImportSpec])
	}
	if !hasNodeAttr(nodes[graph.NodeKindImportSpec], "is_dot", "true") {
		t.Fatalf("missing dot import fact: %+v", nodes[graph.NodeKindImportSpec])
	}
	if !hasNodeAttr(nodes[graph.NodeKindCall], "callee", "writer.WriteThing") {
		t.Fatalf("missing interface selector call fact: %+v", nodes[graph.NodeKindCall])
	}
	if !hasNodeAttr(nodes[graph.NodeKindCall], "callee", "service.WriteThing") {
		t.Fatalf("missing concrete selector call fact: %+v", nodes[graph.NodeKindCall])
	}
	if !hasNodeAttr(nodes[graph.NodeKindTypeUsage], "address_of", "true") {
		t.Fatalf("missing address-of composite literal type usage fact: %+v", nodes[graph.NodeKindTypeUsage])
	}
}

func TestExtractRouteRegistrationFacts(t *testing.T) {
	root := t.TempDir()
	writeRouteFixtureFile(t, filepath.Join(root, "go.mod"), "module example.com/routes\n\ngo 1.25\n")
	writeRouteFixtureFile(t, filepath.Join(root, "muxstub", "mux.go"), `package mux

import "net/http"

type Router struct{}
type Route struct{}

func (r *Router) HandleFunc(path string, f func(http.ResponseWriter, *http.Request)) *Route { return &Route{} }
func (r *Router) Handle(path string, h http.Handler) *Route { return &Route{} }
func (r *Route) Methods(methods ...string) *Route { return r }
`)
	writeRouteFixtureFile(t, filepath.Join(root, "routes.go"), `package routes

import (
	"net/http"

	mux "example.com/routes/muxstub"
)

const methodPatch = http.MethodPatch

func Register(r *mux.Router, path string, h http.Handler) {
	r.HandleFunc("/health", health).Methods(http.MethodGet)
	r.Handle("/api/v1/notes/{id}/attachments", h).Methods("POST", methodPatch)
	r.HandleFunc(path, health).Methods(http.MethodDelete)
	http.HandleFunc("/legacy", health)
}

func health(http.ResponseWriter, *http.Request) {}
`)

	g, warnings, err := newRealService().Extract(context.Background(), graph.ExtractInput{ModulePath: root})
	if err != nil {
		t.Fatalf("Extract route fixture: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("route fixture should extract without warnings, got %+v", warnings)
	}

	var routes []graph.Node
	for _, node := range g.Nodes {
		if node.Kind == graph.NodeKindRouteRegistration {
			routes = append(routes, node)
		}
	}
	if len(routes) != 5 {
		t.Fatalf("route facts = %d, want 5: %+v", len(routes), routes)
	}
	requireRouteFact(t, routes, "/health", "GET", "gorilla/mux")
	requireRouteFact(t, routes, "/api/v1/notes/{id}/attachments", "POST", "gorilla/mux")
	requireRouteFact(t, routes, "/api/v1/notes/{id}/attachments", "PATCH", "gorilla/mux")
	requireRouteFact(t, routes, "/legacy", "", "net/http")
	if !hasRouteWithStatus(routes, "route_path_status", "unknown") {
		t.Fatalf("missing dynamic path route with unknown status: %+v", routes)
	}
	if hasNodeAttr(routes, "route_path", "path") {
		t.Fatalf("dynamic route path was treated as a proven literal: %+v", routes)
	}
}

func TestExtractScenarioAPIModulePath(t *testing.T) {
	abs := resolveFixture(t, "../..")
	if _, err := os.Stat(filepath.Join(filepath.Dir(abs), ".vrooli", "service.json")); err != nil {
		t.Fatalf("expected scenario metadata next to API module: %v", err)
	}

	g, _, err := newRealService().Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
	if err != nil {
		t.Fatalf("Extract scenario API module: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatalf("scenario API module produced no graph nodes")
	}
}

func hasNodeAttr(nodes []graph.Node, key, value string) bool {
	for _, node := range nodes {
		if node.Attributes[key] == value {
			return true
		}
	}
	return false
}

func requireRouteFact(t *testing.T, nodes []graph.Node, path, method, framework string) {
	t.Helper()
	for _, node := range nodes {
		attrs := node.Attributes
		if attrs["route_path"] == path && attrs["http_method"] == method && attrs["router_framework"] == framework {
			if attrs["handler_expr"] == "" || attrs["route_source"] == "" || attrs["start_line"] == "" {
				t.Fatalf("route fact missing expected detail: %+v", node)
			}
			return
		}
	}
	t.Fatalf("missing route fact path=%q method=%q framework=%q in %+v", path, method, framework, nodes)
}

func hasRouteWithStatus(nodes []graph.Node, key, value string) bool {
	for _, node := range nodes {
		if node.Attributes[key] == value {
			return true
		}
	}
	return false
}

func writeRouteFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestConcurrentExtractSamePath(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-cycles")
	svc := newRealService()

	const N = 5
	hashes := make([]string, N)
	var wg sync.WaitGroup
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g, _, err := svc.Extract(context.Background(), graph.ExtractInput{ModulePath: abs})
			if err != nil {
				errs[i] = err
				return
			}
			hashes[i] = graph.GraphHash(g)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i := 1; i < N; i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("hash mismatch: hashes[0]=%s hashes[%d]=%s", hashes[0], i, hashes[i])
		}
	}
}

func TestConcurrentExtractDifferentPaths(t *testing.T) {
	a := resolveFixture(t, "../../../bas/fixtures/go-cycles")
	b := resolveFixture(t, "../../../bas/fixtures/go-mislocated")
	svc := newRealService()

	var wg sync.WaitGroup
	var errA, errB error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _, errA = svc.Extract(context.Background(), graph.ExtractInput{ModulePath: a})
	}()
	go func() {
		defer wg.Done()
		_, _, errB = svc.Extract(context.Background(), graph.ExtractInput{ModulePath: b})
	}()
	wg.Wait()
	if errA != nil {
		t.Fatalf("extract go-cycles: %v", errA)
	}
	if errB != nil {
		t.Fatalf("extract go-mislocated: %v", errB)
	}
}
