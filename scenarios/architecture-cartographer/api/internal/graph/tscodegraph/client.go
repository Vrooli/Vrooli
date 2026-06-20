// Package tscodegraph is the production CodeGraphAdapter for
// TypeScript sources. It is a thin Connect-RPC client wired against
// the `typescript-code-graph` scenario's TypeScriptCodeGraphService.
//
// The adapter translates Connect-RPC transport/protocol errors into
// the typed graph.IntegrationError shape so cartographer's service
// layer can classify them without depending on connect-rpc.
package tscodegraph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/graph"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// ScenarioName is the canonical scenario identifier the discovery
// layer uses to resolve the typescript-code-graph base URL. It is
// also the value populated into IntegrationError.Scenario so
// upstream classification has a stable key.
const ScenarioName = "typescript-code-graph"

// URLResolver resolves a scenario's API base URL at call time. The
// production implementation is *discovery.Resolver (api-core), which
// shells out to `vrooli scenario port <slug>` on every call so a
// sibling's dynamic port is always current. Tests inject
// discovery.NewStaticResolver(server.URL).
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// ProjectPathFn resolves a target scenario name to the absolute path of
// its TypeScript project (a directory containing tsconfig.json).
// found == false means the scenario has no TypeScript project; the
// adapter then contributes nothing instead of calling the producer.
type ProjectPathFn = graph.ProjectPathFn

// Config wires a Client. URLResolver and ProjectPath are required in
// production; HTTPClient defaults to http.DefaultClient.
type Config struct {
	URLResolver URLResolver
	ProjectPath ProjectPathFn
	HTTPClient  connect.HTTPClient
}

// Client is the production CodeGraphAdapter for TypeScript. It resolves
// the producer URL and the target's TS project path per call (the
// underlying *http.Client default leaves timeouts to context), so it
// tolerates the producer restarting on a new port between extractions.
type Client struct {
	urls       URLResolver
	projectOf  ProjectPathFn
	httpClient connect.HTTPClient
}

// New builds a Client from Config.
func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{urls: cfg.URLResolver, projectOf: cfg.ProjectPath, httpClient: hc}
}

var _ graph.CodeGraphAdapter = (*Client)(nil)

// Name returns the adapter identifier.
func (c *Client) Name() string { return "typescript" }

// SupportedLanguages reports that this adapter returns TypeScript
// nodes only.
func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageTypeScript}
}

// Extract resolves the target scenario's TypeScript project directory,
// resolves the typescript-code-graph producer URL via discovery, calls
// its Extract RPC, and translates the proto response into the adapter's
// language-agnostic RawGraph. A scenario with no TS project yields an
// empty graph (not an error). Discovery and Connect failures are
// classified into graph.IntegrationError.
func (c *Client) Extract(ctx context.Context, scenario string) (graph.RawGraph, error) {
	return graph.ExtractFromProject(
		ctx,
		c.urls,
		c.projectOf,
		scenario,
		ScenarioName,
		"TypeScript",
		func(ctx context.Context, baseURL string, projectPath string) (graph.RawGraph, error) {
			rpc := graph_v1connect.NewTypeScriptCodeGraphServiceClient(c.httpClient, baseURL)
			resp, err := rpc.Extract(ctx, connect.NewRequest(&graphv1.ExtractRequest{
				ProjectPath: projectPath,
			}))
			if err != nil {
				return graph.RawGraph{}, graph.ClassifyConnectError(err, ScenarioName)
			}
			return protoToRawGraph(resp.Msg), nil
		},
	)
}

// protoToRawGraph translates the proto envelope into the
// language-agnostic RawGraph cartographer normalizes downstream.
// Mapping:
//   - NODE_KIND_FILE                                          → FileNode
//   - NODE_KIND_MODULE                                        → PackageNode
//   - NODE_KIND_PACKAGE without a symbol kind in attributes   → PackageNode
//   - NODE_KIND_PACKAGE with a TS-specific kind in attributes → SymbolNode
//   - anything else (including TS-specific kinds 200..299)    → SymbolNode
//   - every edge → ImportEdge (cartographer's Normalize() reclassifies)
//
// The NODE_KIND_PACKAGE discrimination mirrors the Go adapter
// (gocodegraph): the typescript-code-graph producer rides every
// symbol-level node (calls, references, functions, components, …)
// under NODE_KIND_PACKAGE with the real kind in attributes["kind"]
// (TS_NODE_KIND_CALL/REFERENCE/…). Only nodes whose kind attribute is
// empty or literally "package" are true packages. Without this gate a
// single UI extraction injects thousands of fake packages into the
// import-cluster graph, exploding its O(N²) modularity pass (see
// scenarios/architecture-cartographer/docs/internal/DECISIONS.md and
// the import-cluster signal). True modules arrive as NODE_KIND_MODULE
// and are always packages.
//
// Producers are expected to emit stable ids; this translator does
// not re-sort. graph.Normalize() handles canonicalization.
func protoToRawGraph(resp *graphv1.ExtractResponse) graph.RawGraph {
	if resp == nil {
		return graph.RawGraph{Languages: []graph.Language{graph.LanguageTypeScript}}
	}
	out := graph.RawGraph{
		Languages:    []graph.Language{graph.LanguageTypeScript},
		ExtractionMS: resp.GetExtractionMs(),
	}
	g := resp.GetGraph()
	if g == nil {
		return out
	}
	for _, n := range g.GetNodes() {
		attrs := n.GetAttributes()
		switch {
		case n.GetKind() == commonv1.NodeKind_NODE_KIND_FILE:
			out.Files = append(out.Files, graph.FileNodeFromProto(n, graph.LanguageTypeScript))
		case n.GetKind() == commonv1.NodeKind_NODE_KIND_MODULE:
			out.Packages = append(out.Packages, graph.PackageNode{
				ID:         n.GetId(),
				ImportPath: n.GetName(),
				Language:   graph.LanguageTypeScript,
			})
		case n.GetKind() == commonv1.NodeKind_NODE_KIND_PACKAGE && isTruePackage(attrs):
			out.Packages = append(out.Packages, graph.PackageNode{
				ID:         n.GetId(),
				ImportPath: n.GetName(),
				Language:   graph.LanguageTypeScript,
			})
		default:
			// Everything else — NODE_KIND_PACKAGE carrying a TS-specific
			// symbol kind (calls/references/functions/…), the reserved
			// TS kinds (200..299), and any future producer kinds — surfaces
			// as a SymbolNode with the kind preserved in the
			// attributes-derived Kind string. Cartographer's Normalize()
			// does not care about the specific enum value.
			out.Symbols = append(out.Symbols, graph.SymbolNode{
				ID:        n.GetId(),
				Name:      n.GetName(),
				PackageID: attrs["package_id"],
				FileID:    attrs["file_id"],
				Kind:      symbolKind(n.GetKind().String(), attrs),
				Exported:  attrs["exported"] == "true",
			})
		}
	}
	for _, e := range g.GetEdges() {
		out.Imports = append(out.Imports, graph.ImportEdgeFromProto(e))
	}
	return out
}

func symbolKind(fallback string, attrs map[string]string) string {
	if attrs["kind"] != "" {
		return attrs["kind"]
	}
	return fallback
}

// isTruePackage reports whether a NODE_KIND_PACKAGE node is a genuine
// package rather than a symbol-level node the producer rode under the
// PACKAGE enum. The typescript-code-graph producer tags real packages
// with an empty or "package" kind attribute and every symbol with a
// concrete TS_NODE_KIND_* value, mirroring how go-code-graph
// distinguishes package nodes from go_func/go_type/… symbol nodes.
func isTruePackage(attrs map[string]string) bool {
	kind := attrs["kind"]
	return kind == "" || kind == "package"
}
