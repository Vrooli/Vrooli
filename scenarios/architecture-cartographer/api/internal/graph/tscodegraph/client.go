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
	"errors"
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

// Client is the production CodeGraphAdapter for TypeScript. It holds
// a single Connect-RPC client constructed in New and reused for the
// lifetime of the adapter; the underlying *http.Client is the
// stdlib default so callers control timeouts via context.
type Client struct {
	baseURL string
	rpc     graph_v1connect.TypeScriptCodeGraphServiceClient
}

// New returns a Client wired against baseURL (resolved by the
// scenario-discovery layer). The Connect client is constructed
// eagerly so per-call hot path stays allocation-free.
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		rpc:     graph_v1connect.NewTypeScriptCodeGraphServiceClient(http.DefaultClient, baseURL),
	}
}

var _ graph.CodeGraphAdapter = (*Client)(nil)

// Name returns the adapter identifier.
func (c *Client) Name() string { return "typescript" }

// SupportedLanguages reports that this adapter returns TypeScript
// nodes only.
func (c *Client) SupportedLanguages() []graph.Language {
	return []graph.Language{graph.LanguageTypeScript}
}

// Extract performs an Extract call against the typescript-code-graph
// scenario and translates the proto response into the adapter's
// language-agnostic RawGraph shape. Connect errors are classified
// into graph.IntegrationError per classifyConnectError.
func (c *Client) Extract(ctx context.Context, scenario string) (graph.RawGraph, error) {
	if c.baseURL == "" {
		return graph.RawGraph{}, graph.IntegrationError{
			Kind:     "scenario_unreachable",
			Scenario: ScenarioName,
			Cause:    errors.New("empty baseURL: discovery layer did not resolve typescript-code-graph"),
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
// understands. Non-connect errors (notably context.Canceled wrapped
// before transport, or local marshaling failures) fall through to
// "internal" so they're at least surfaced consistently.
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
// Mapping:
//   - NODE_KIND_FILE   → FileNode
//   - NODE_KIND_PACKAGE or NODE_KIND_MODULE → PackageNode
//   - anything else (including TS-specific kinds 200..299) → SymbolNode
//   - every edge → ImportEdge (cartographer's Normalize() reclassifies)
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
		switch n.GetKind() {
		case commonv1.NodeKind_NODE_KIND_FILE:
			out.Files = append(out.Files, graph.FileNode{
				ID:       n.GetId(),
				Path:     n.GetPath(),
				Language: graph.LanguageTypeScript,
			})
		case commonv1.NodeKind_NODE_KIND_PACKAGE, commonv1.NodeKind_NODE_KIND_MODULE:
			out.Packages = append(out.Packages, graph.PackageNode{
				ID:         n.GetId(),
				ImportPath: n.GetName(),
				Directory:  n.GetPath(),
				Language:   graph.LanguageTypeScript,
			})
		default:
			// TS-specific kinds (200..299) and any future producer kinds
			// surface as SymbolNodes with the kind preserved in the
			// attributes-derived Kind string. Cartographer's Normalize()
			// does not care about the specific enum value.
			out.Symbols = append(out.Symbols, graph.SymbolNode{
				ID:   n.GetId(),
				Name: n.GetName(),
				Kind: n.GetKind().String(),
			})
		}
	}
	for _, e := range g.GetEdges() {
		out.Imports = append(out.Imports, graph.ImportEdge{
			From:        e.GetFromNodeId(),
			ToPackageID: e.GetToNodeId(),
		})
	}
	return out
}
