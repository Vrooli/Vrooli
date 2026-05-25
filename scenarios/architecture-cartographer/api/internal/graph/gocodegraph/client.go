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
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	"architecture-cartographer/internal/graph"

	"github.com/vrooli/api-core/discovery"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// atoiAttr decodes an integer-valued attribute string, returning 0 when
// the attribute is absent or unparseable.
func atoiAttr(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

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
type ProjectPathFn func(scenarioName string) (path string, found bool, err error)

// Config wires a Client. URLResolver and ProjectPath are required in
// production; HTTPClient defaults to http.DefaultClient.
type Config struct {
	URLResolver URLResolver
	ProjectPath ProjectPathFn
	HTTPClient  connect.HTTPClient
}

// Client is the production CodeGraphAdapter for Go. It resolves the
// producer URL and the target's Go project path per call (the
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
	if c.projectOf == nil || c.urls == nil {
		return graph.RawGraph{}, graph.IntegrationError{
			Kind:     "internal",
			Scenario: ScenarioName,
			Cause:    errors.New("gocodegraph adapter not fully configured (missing URLResolver or ProjectPath)"),
		}
	}
	projectPath, found, err := c.projectOf(scenario)
	if err != nil {
		return graph.RawGraph{}, graph.IntegrationError{
			Kind:     "internal",
			Scenario: ScenarioName,
			Cause:    fmt.Errorf("resolve Go project path for %q: %w", scenario, err),
		}
	}
	if !found {
		// Scenario has no Go project; contribute nothing.
		return graph.RawGraph{}, nil
	}
	baseURL, err := c.urls.ResolveScenarioURLDefault(ctx, ScenarioName)
	if err != nil {
		return graph.RawGraph{}, classifyResolveError(err)
	}
	rpc := graph_v1connect.NewGoCodeGraphServiceClient(c.httpClient, baseURL)
	resp, err := rpc.Extract(ctx, connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: projectPath,
	}))
	if err != nil {
		return graph.RawGraph{}, classifyConnectError(err)
	}
	return protoToRawGraph(resp.Msg), nil
}

// classifyResolveError maps an api-core discovery failure onto the typed
// graph.IntegrationError kinds cartographer's service understands. A
// not-running / unreachable producer becomes "scenario_unreachable" so
// the service can skip this language rather than failing the whole
// cross-language extract.
func classifyResolveError(err error) error {
	kind := "internal"
	var de *discovery.Error
	if errors.As(err, &de) {
		switch de.Kind {
		case discovery.ErrScenarioNotRunning, discovery.ErrVrooliNotFound, discovery.ErrCommandFailed, discovery.ErrInvalidPort:
			kind = "scenario_unreachable"
		case discovery.ErrTimeout:
			kind = "scenario_timeout"
		default:
			kind = "internal"
		}
	}
	return graph.IntegrationError{Kind: kind, Scenario: ScenarioName, Cause: err}
}

// classifyConnectError maps a connect.Error code to the typed
// graph.IntegrationError.Kind cartographer's service layer
// understands. Kept in sync with tscodegraph.classifyConnectError so
// upstream classification is language-agnostic.
func classifyConnectError(err error) error {
	if err == nil {
		return nil
	}
	kind := "internal"
	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeUnavailable:
			kind = "scenario_unreachable"
		case connect.CodeDeadlineExceeded:
			kind = "scenario_timeout"
		case connect.CodeInvalidArgument:
			kind = "invalid_argument"
		case connect.CodeNotFound:
			kind = "not_found"
		case connect.CodeUnimplemented:
			kind = "unimplemented"
		default:
			kind = "internal"
		}
	}
	return graph.IntegrationError{
		Kind:     kind,
		Scenario: ScenarioName,
		Cause:    err,
	}
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
//   - PackageNode.Internal is not yet emitted; left false.
//   - ImportEdge.TestOnly / ImportEdge.SymbolIDs are not yet emitted;
//     left as zero values.
func protoToRawGraph(resp *graphv1.ExtractResponse) graph.RawGraph {
	out := graph.RawGraph{Languages: []graph.Language{graph.LanguageGo}}
	if resp == nil {
		return out
	}
	out.ExtractionMS = resp.GetExtractionMs()
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
			out.Files = append(out.Files, graph.FileNode{
				ID:        n.GetId(),
				Path:      n.GetPath(),
				PackageID: attrs["package_id"],
				Language:  graph.LanguageGo,
				Lines:     atoiAttr(attrs["lines"]),
				IsTest:    attrs["is_test"] == "true",
			})
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
					Directory:  n.GetPath(),
					Language:   graph.LanguageGo,
					Internal:   attrs["internal"] == "true",
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
				Directory:  n.GetPath(),
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
		out.Imports = append(out.Imports, graph.ImportEdge{
			From:        e.GetFromNodeId(),
			ToPackageID: e.GetToNodeId(),
			TestOnly:    e.GetAttributes()["test_only"] == "true",
		})
	}
	return out
}
