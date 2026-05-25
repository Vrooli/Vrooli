// Package sidecar is the Go-side substrate that owns the Node sidecar
// child process hosting ts-morph.
//
// Substrate rules:
//   - This package is the ONLY place in the api/ tree allowed to import
//     os/exec, manipulate process pipes, or call cmd.Process.Kill.
//   - The package exposes a single seam — SidecarClient — that domain
//     packages (internal/graph, internal/rewrite) depend on. Domain
//     packages must never spawn or speak IPC directly.
//   - JSON shapes here mirror the wire protocol verbatim (snake_case);
//     translation into the proto envelope is the graph/rewrite domain's
//     job, not this package's.
package sidecar

// Status describes the supervisor's view of the child process. The string
// values match the proto SidecarStatus enum names so downstream handlers
// can pass them through without remapping.
type Status string

const (
	// StatusReady means the most recent handshake succeeded and no
	// terminal error has occurred since.
	StatusReady Status = "STATUS_READY"
	// StatusUnhealthy means the child is currently not answering; the
	// supervisor goroutine is either waiting on backoff or attempting a
	// respawn. Requests in this state fail fast with SidecarUnavailable.
	StatusUnhealthy Status = "STATUS_UNHEALTHY"
	// StatusRestarting means the supervisor is actively spawning a
	// replacement child and has not yet completed handshake.
	StatusRestarting Status = "STATUS_RESTARTING"
	// StatusPermanentlyUnhealthy means the restart budget was exhausted
	// (5 restarts in 60s); no further restart attempts will be made.
	StatusPermanentlyUnhealthy Status = "STATUS_PERMANENTLY_UNHEALTHY"
)

// RawGraph is the wire-shape graph the sidecar emits. The graph domain
// is responsible for normalizing and converting to the proto envelope.
type RawGraph struct {
	Nodes []RawNode `json:"nodes"`
	Edges []RawEdge `json:"edges"`
}

// ExtractResult bundles everything a single Extract call yields. The
// RequestID is the supervisor-minted UUID that scoped the underlying
// IPC request — domain code threads it onto ExtractResponse.sidecar_request_id
// so an API caller can correlate a graph with the exact sidecar request
// (and its stderr log lines) that produced it.
type ExtractResult struct {
	Graph     RawGraph
	Warnings  []Warning
	RequestID string
}

// RawNode is a single graph node as emitted by the sidecar. Kind is the
// numeric common.v1.NodeKind enum value (1=FILE, 2=PACKAGE, 3=MODULE,
// 200..209 for TS-specific). The TS-specific enum *name* (e.g.
// "TS_NODE_KIND_COMPONENT") rides on Attributes["kind"] per the proto
// envelope contract (see common/v1/code_graph.proto NodeKind comment).
type RawNode struct {
	ID              string            `json:"id"`
	Kind            int32             `json:"kind"`
	Name            string            `json:"name"`
	Path            string            `json:"path"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	LeadingComments []string          `json:"leading_comments,omitempty"`
}

// RawEdge is a single graph edge as emitted by the sidecar. Kind is the
// numeric common.v1.EdgeKind enum value (1=IMPORT, 2=INTRA_PACKAGE_REF,
// 3=RE_EXPORT).
type RawEdge struct {
	ID         string            `json:"id"`
	Kind       int32             `json:"kind"`
	FromNodeID string            `json:"from_node_id"`
	ToNodeID   string            `json:"to_node_id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Warning is a non-fatal diagnostic surfaced during extraction. The
// wire shape mirrors common.v1.CodeGraphWarning exactly: a numeric
// CodeGraphWarningKind (1=PARSE_ERROR, 2=UNRESOLVED_IMPORT,
// 3=TYPE_CHECK_FAILURE, 4=AMBIGUOUS_DECLARATION), the file the warning
// applies to, and a human-readable message. The graph domain decodes
// Kind into its typed WarningKind.
type Warning struct {
	Kind    int32  `json:"kind"`
	File    string `json:"file,omitempty"`
	Message string `json:"message"`
}

// Operation is a discriminated union — exactly one of the embedded
// pointers must be non-nil. The sidecar package owns its own Go type
// here independent of proto; Phase 5 (rewrite domain) translates
// between proto Operation and sidecar.Operation.
type Operation struct {
	FileMove      *FileMove      `json:"file_move,omitempty"`
	ImportRewrite *ImportRewrite `json:"import_rewrite,omitempty"`
}

// FileMove relocates a source file inside the project.
type FileMove struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ImportRewrite changes every import / export specifier referring to
// OldPath so it refers to NewPath instead.
type ImportRewrite struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

// OperationResult mirrors the per-operation outcome the sidecar reports
// in a rewrite_apply response. The wire shape is exactly {status,
// message}: Status is the canonical proto OperationStatus enum name
// ("OPERATION_STATUS_OK" / "OPERATION_STATUS_FAILED"). The sidecar does
// not echo the operation — the rewrite domain zips results back onto the
// request operations by index.
type OperationResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
