// Package gocodegraph is the production CodeGraphAdapter for Go
// sources. It is a thin Connect-RPC client wired against the
// `go-code-graph` scenario's GoCodeGraphService.
//
// The adapter translates Connect-RPC transport/protocol errors into
// the typed graph.IntegrationError shape so cartographer's service
// layer can classify them without depending on connect-rpc, and it
// projects the common.v1.CodeGraph envelope onto cartographer's
// language-agnostic RawGraph.
//
// Symbol nodes are emitted by go-code-graph as NODE_KIND_PACKAGE
// with the typed Go kind (go_type/go_func/go_var/go_const/
// go_interface/go_method) recorded in attributes["kind"]; true
// package nodes carry no "kind" attribute. See
// `scenarios/go-code-graph/api/handlers/graph/adapter.go`
// (domainToProtoGraph / nodeKindToProto) for the producer side.
package gocodegraph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/graph"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// ScenarioName is the canonical scenario identifier the discovery
// layer uses to resolve the go-code-graph base URL. It is also the
// value populated into IntegrationError.Scenario so upstream
// classification has a stable key.
const ScenarioName = "go-code-graph"

// URLResolver resolves a scenario's API base URL at call time. The
// production implementation is *discovery.Resolver (api-core), which
// shells out to `vrooli scenario port <slug>` on every call so a
// sibling's dynamic port is always current. Tests inject
// discovery.NewStaticResolver(server.URL).
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// ProjectPathFn resolves a target scenario name to the absolute path of
// its Go project (a directory containing go.mod). found == false means
// the scenario has no Go project; the adapter then contributes nothing
// instead of calling the producer.
type ProjectPathFn = graph.ProjectPathFn

// Config wires a Client. URLResolver and ProjectPath are required in
// production; HTTPClient defaults to http.DefaultClient.
type Config struct {
	URLResolver URLResolver
	ProjectPath ProjectPathFn
	HTTPClient  connect.HTTPClient
	// Profile is the explicit information contract for Go extraction. Zero
	// preserves the producer's full compatibility profile for library users;
	// production wiring selects the cheapest profile that its detectors need.
	Profile graphv1.ExtractionProfile
}

// Client is the production CodeGraphAdapter for Go. It resolves the
// producer URL and the target's Go project path per call (the
// underlying *http.Client default leaves timeouts to context), so it
// tolerates the producer restarting on a new port between extractions.
type Client struct {
	urls       URLResolver
	projectOf  ProjectPathFn
	httpClient connect.HTTPClient
	profile    graphv1.ExtractionProfile
}

// New builds a Client from Config.
func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{urls: cfg.URLResolver, projectOf: cfg.ProjectPath, httpClient: hc, profile: cfg.Profile}
}

var _ graph.CodeGraphAdapter = (*Client)(nil)

// Name returns the adapter identifier.
func (c *Client) Name() string { return "go" }

// SupportedLanguages reports that this adapter returns Go-language
// nodes only.
func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageGo}
}

// Extract resolves the target scenario's Go project directory, resolves
// the go-code-graph producer URL via discovery, calls its Extract RPC,
// and translates the proto response into the adapter's language-agnostic
// RawGraph. A scenario with no Go project yields an empty graph (not an
// error). Discovery and Connect failures are classified into
// graph.IntegrationError.
func (c *Client) Extract(ctx context.Context, scenario string) (graph.RawGraph, error) {
	return graph.ExtractFromProject(
		ctx,
		c.urls,
		c.projectOf,
		scenario,
		ScenarioName,
		"go",
		func(ctx context.Context, baseURL string, projectPath string) (graph.RawGraph, error) {
			rpc := graph_v1connect.NewGoCodeGraphServiceClient(c.httpClient, baseURL)
			resp, err := rpc.Extract(ctx, connect.NewRequest(&graphv1.ExtractRequest{
				ModulePath: projectPath,
				Profile:    c.profile,
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
//
// Mapping (per go-code-graph's domainToProtoGraph):
//   - NODE_KIND_FILE                                            → FileNode
//   - NODE_KIND_PACKAGE without attributes["kind"]              → PackageNode
//   - NODE_KIND_PACKAGE with a Go-typed kind in attributes      → SymbolNode
//   - NODE_KIND_MODULE                                          → PackageNode
//   - EDGE_KIND_IMPORT                                          → ImportEdge
//
// Producers are expected to emit stable ids; this translator does
// not re-sort. graph.Normalize() handles canonicalization.
//
// Data-loss notes (tracked in scenarios/go-code-graph/docs/internal/PROBLEMS.md
// entry 2026-05-23):
//   - FileNode.Lines / FileNode.IsTest are not yet emitted by go-code-graph;
//     left as Go zero values here.
//   - ImportEdge.TestOnly / ImportEdge.SymbolIDs are not yet emitted;
//     left as zero values.
//
// PackageNode.RepoPath is derived from the package's file nodes (the
// common directory of every file's repo-relative path) because
// go-code-graph only emits the import path on PACKAGE nodes. A package
// with no files has RepoPath="" and will not map to any domain.
func protoToRawGraph(resp *graphv1.ExtractResponse) graph.RawGraph {
	out := graph.RawGraph{Languages: []graph.Language{graph.LanguageGo}}
	if resp == nil {
		return out
	}
	out.ExtractionMS = resp.GetExtractionMs()
	profile := resp.GetProfile()
	out.ExtractionProfiles = []string{profileName(profile)}
	for _, omission := range resp.GetOmittedInformation() {
		if omission == nil {
			continue
		}
		out.OmittedInformation = append(out.OmittedInformation, graph.InformationOmission{
			Capability: omission.GetCapability(),
			Reason:     omission.GetReason(),
		})
	}
	if len(out.OmittedInformation) == 0 {
		out.OmittedInformation = omissionsForProfile(profile)
	}
	g := resp.GetGraph()
	if g == nil {
		return out
	}
	for _, n := range g.GetNodes() {
		if n == nil {
			continue
		}
		attrs := n.GetAttributes()
		switch n.GetKind() {
		case commonv1.NodeKind_NODE_KIND_FILE:
			out.Files = append(out.Files, graph.FileNodeFromProto(n, graph.LanguageGo))
		case commonv1.NodeKind_NODE_KIND_PACKAGE:
			// Go-specific symbol kinds (go_type, go_func, …) ride
			// under NODE_KIND_PACKAGE with attributes["kind"] set;
			// true packages have no "kind" attribute.
			kindAttr := attrs["kind"]
			if kindAttr == "" || kindAttr == "package" {
				importPath := attrs["import_path"]
				if importPath == "" {
					importPath = n.GetPath()
				}
				out.Packages = append(out.Packages, graph.PackageNode{
					ID:         n.GetId(),
					ImportPath: importPath,
					Language:   graph.LanguageGo,
				})
			} else {
				out.Symbols = append(out.Symbols, graph.SymbolNode{
					ID:        n.GetId(),
					Name:      n.GetName(),
					PackageID: attrs["package_id"],
					FileID:    attrs["file_id"],
					Kind:      kindAttr,
					Exported:  attrs["exported"] == "true",
				})
			}
		case commonv1.NodeKind_NODE_KIND_MODULE:
			out.Packages = append(out.Packages, graph.PackageNode{
				ID:         n.GetId(),
				ImportPath: n.GetName(),
				Language:   graph.LanguageGo,
			})
		}
	}
	for _, e := range g.GetEdges() {
		if e == nil {
			continue
		}
		if e.GetKind() != commonv1.EdgeKind_EDGE_KIND_IMPORT {
			continue
		}
		out.Imports = append(out.Imports, graph.ImportEdgeFromProto(e))
	}
	return out
}

func profileName(profile graphv1.ExtractionProfile) string {
	switch profile {
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC:
		return "semantic"
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL:
		return "structural"
	default:
		return "full"
	}
}

func omissionsForProfile(profile graphv1.ExtractionProfile) []graph.InformationOmission {
	switch profile {
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC:
		return []graph.InformationOmission{
			{Capability: "test_variants", Reason: "semantic profile excludes test packages"},
			{Capability: "test_only_relationships", Reason: "test packages were not loaded"},
		}
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL:
		return []graph.InformationOmission{
			{Capability: "resolved_type_information", Reason: "structural profile does not run Go type checking"},
			{Capability: "resolved_usage_facts", Reason: "structural profile cannot resolve references, calls, or type usages"},
			{Capability: "test_variants", Reason: "structural profile excludes test packages"},
			{Capability: "test_only_relationships", Reason: "test packages were not loaded"},
		}
	default:
		return nil
	}
}
