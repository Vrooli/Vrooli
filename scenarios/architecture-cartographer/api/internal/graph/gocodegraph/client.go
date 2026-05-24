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

// Client is the production CodeGraphAdapter for Go. It holds a single
// Connect-RPC client constructed in New and reused for the lifetime
// of the adapter; the underlying *http.Client is the stdlib default
// so callers control timeouts via context.
type Client struct {
	baseURL string
	rpc     graph_v1connect.GoCodeGraphServiceClient
}

// New returns a Client wired against baseURL (resolved by the
// scenario-discovery layer). The Connect client is constructed
// eagerly so per-call hot path stays allocation-free. An empty
// baseURL is allowed at construction; Extract returns
// IntegrationError{Kind:"scenario_unreachable"} on first use.
func New(baseURL string) *Client {
	return NewWithHTTPClient(baseURL, http.DefaultClient)
}

// NewWithHTTPClient is the seam tests use to drive the client against
// an in-process httptest.Server (or any custom connect.HTTPClient).
func NewWithHTTPClient(baseURL string, httpClient connect.HTTPClient) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL: baseURL,
		rpc:     graph_v1connect.NewGoCodeGraphServiceClient(httpClient, baseURL),
	}
}

var _ graph.CodeGraphAdapter = (*Client)(nil)

// Name returns the adapter identifier.
func (c *Client) Name() string { return "go" }

// SupportedLanguages reports that this adapter returns Go-language
// nodes only.
func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageGo}
}

// Extract performs an Extract call against the go-code-graph scenario
// and translates the proto response into the adapter's language-
// agnostic RawGraph shape. Connect errors are classified into
// graph.IntegrationError per classifyConnectError.
func (c *Client) Extract(ctx context.Context, scenario string) (graph.RawGraph, error) {
	if c.baseURL == "" {
		return graph.RawGraph{}, graph.IntegrationError{
			Kind:     "scenario_unreachable",
			Scenario: ScenarioName,
			Cause:    errors.New("empty baseURL: discovery layer did not resolve go-code-graph"),
		}
	}
	resp, err := c.rpc.Extract(ctx, connect.NewRequest(&graphv1.ExtractRequest{
		ScenarioPath: scenario,
	}))
	if err != nil {
		return graph.RawGraph{}, classifyConnectError(err)
	}
	return protoToRawGraph(resp.Msg), nil
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
