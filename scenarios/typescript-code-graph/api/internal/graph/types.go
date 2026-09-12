// Package graph is the domain-scoped home for deterministic TypeScript
// project extraction. It owns the in-process representation of a TS
// project's file/module/symbol graph and the seams that produce it.
//
// Layering follows the canonical Vrooli pattern: handler → Service →
// seam (sidecar.SidecarClient). The SidecarClient seam is the only edge
// that touches the Node sidecar child process; every other operation in
// this package is pure data shaping.
//
// The domain types here deliberately mirror the relevant subset of
// scenarios/go-code-graph/api/internal/graph/types.go so the proto
// adapter in handlers/graph/ can fold either language onto the shared
// common.v1.CodeGraph envelope with one mechanical step.
//
// Substrate boundary: this package MUST NOT import time, os, net/http,
// or os/exec. The sidecar package is the substrate that owns those
// concerns. See no_prod_import_test.go for the enforcement gate.
package graph

import (
	"fmt"
)

// Language is the source language a node/edge belongs to.
type Language string

const (
	// LanguageTypeScript is the only producer language this scenario
	// ships in v1.
	LanguageTypeScript Language = "typescript"
)

// NodeKind enumerates the structural node families this scenario emits.
// Values are stable strings so canonical JSON serialization (used by
// GraphHash) is stable across TypeScript / ts-morph upgrades.
//
// The string values are deliberately the lower-case form of the proto
// TsNodeKind enum names — adapter.go translates them to the int32 enum
// values when projecting onto common.v1.CodeGraphNode.
type NodeKind string

const (
	NodeKindFile          NodeKind = "file"
	NodeKindModule        NodeKind = "ts_module"
	NodeKindComponent     NodeKind = "ts_component"
	NodeKindHook          NodeKind = "ts_hook"
	NodeKindClass         NodeKind = "ts_class"
	NodeKindInterface     NodeKind = "ts_interface"
	NodeKindType          NodeKind = "ts_type"
	NodeKindFunction      NodeKind = "ts_function"
	NodeKindVar           NodeKind = "ts_var"
	NodeKindConst         NodeKind = "ts_const"
	NodeKindReExport      NodeKind = "ts_re_export"
	NodeKindImportBinding NodeKind = "ts_import_binding"
	NodeKindReference     NodeKind = "ts_reference"
	NodeKindCall          NodeKind = "ts_call"
	NodeKindJsxUsage      NodeKind = "ts_jsx_usage"
	NodeKindExport        NodeKind = "ts_export"
	NodeKindRoute         NodeKind = "ts_route_registration"
)

// EdgeKind enumerates the directed-edge families this scenario emits.
type EdgeKind string

const (
	EdgeKindImport   EdgeKind = "import"
	EdgeKindReExport EdgeKind = "re_export"
)

// FileNode is one TypeScript source file. ID format: "file:<path>".
type FileNode struct {
	ID              string
	Path            string
	ModuleID        string
	LeadingComments []string
}

// PackageNode is one TypeScript module (the project / tsconfig root or
// a top-level npm package). The name mirrors go-code-graph's Package
// for envelope symmetry; "module" is the TS-native term and the
// "ts_module" NodeKind reflects that.
type PackageNode struct {
	ID         string
	ImportPath string
	Name       string
	Directory  string
}

// SymbolNode is one declared identifier (component/hook/class/...).
// ID format: "<kind>:<module_id>:<name>".
type SymbolNode struct {
	ID              string
	Name            string
	ModuleID        string
	FileID          string
	Kind            NodeKind
	Exported        bool
	TsKind          string // verbatim sidecar kind string (e.g. "TS_NODE_KIND_COMPONENT")
	LeadingComments []string
}

// ImportEdge is a directed dependency between modules. ID format:
// "import:<from>:<to>".
type ImportEdge struct {
	ID         string
	Kind       EdgeKind
	FromNodeID string
	ToNodeID   string
}

// Node is the union row emitted on the proto wire. Domain code keeps
// the kind-specific structs (FileNode, PackageNode, SymbolNode) and the
// adapter projects them onto Node for serialization. Carried here so
// hash.go's canonical JSON has exactly one shape.
type Node struct {
	ID              string            `json:"id"`
	Kind            NodeKind          `json:"kind"`
	Name            string            `json:"name"`
	Path            string            `json:"path"`
	Attributes      map[string]string `json:"attributes,omitempty"`
	IsTest          bool              `json:"is_test,omitempty"`
	Lines           int32             `json:"lines,omitempty"`
	LeadingComments []string          `json:"leading_comments,omitempty"`
}

// Edge is the union row emitted on the proto wire.
type Edge struct {
	ID          string            `json:"id"`
	Kind        EdgeKind          `json:"kind"`
	From        string            `json:"from"`
	To          string            `json:"to"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	TestOnly    bool              `json:"test_only,omitempty"`
	SymbolIDs   []string          `json:"symbol_ids,omitempty"`
	SymbolKinds []string          `json:"symbol_kinds,omitempty"`
}

// WarningKind classifies non-fatal extraction issues.
type WarningKind string

const (
	WarningKindParseError       WarningKind = "parse_error"
	WarningKindUnresolvedImport WarningKind = "unresolved_import"
	WarningKindTypeCheckFailure WarningKind = "type_check_failure"
)

// Warning is a non-fatal issue surfaced alongside an otherwise-valid
// Graph. Catastrophic project errors are returned as ExtractError, not
// Warning.
type Warning struct {
	Kind    WarningKind
	File    string
	Message string
}

// Graph is the normalized, immutable result of extracting one TS
// project. All collections are sorted by stable key before the value is
// constructed; downstream serializers can rely on byte stability.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// ExtractInput is the validated request payload threaded from handler
// to Service. ProjectPath is the absolute path to the project root or a
// concrete tsconfig.json file.
type ExtractInput struct {
	ProjectPath string
}

// ExtractOutput bundles everything the handler needs to project onto
// the proto ExtractResponse. Kept as a struct (rather than a multi-
// return) so adding fields (e.g. cache hint, content hash) does not
// churn every caller.
type ExtractOutput struct {
	Graph            Graph
	Warnings         []Warning
	ExtractionMs     int64
	GraphHash        string
	SidecarRequestID string
}

// ExtractErrorKind names the catastrophic conditions that prevent a
// graph from being produced at all (as opposed to partial-success
// Warnings).
type ExtractErrorKind string

const (
	// ExtractErrorNoTsConfig means no tsconfig.json was found at
	// ProjectPath.
	ExtractErrorNoTsConfig ExtractErrorKind = "no_tsconfig_found"
	// ExtractErrorMultipleTsConfig means more than one tsconfig.json
	// exists under ProjectPath; the project boundary is ambiguous.
	ExtractErrorMultipleTsConfig ExtractErrorKind = "multiple_tsconfig_files"
	// ExtractErrorWorkspaceUnsupported means a pnpm/yarn workspace was
	// detected; multi-project workspaces are out of scope for v1.
	ExtractErrorWorkspaceUnsupported ExtractErrorKind = "workspace_unsupported"
	// ExtractErrorPathUnreadable means the project path could not be
	// stat'd or read.
	ExtractErrorPathUnreadable ExtractErrorKind = "path_unreadable"
	// ExtractErrorInvalidInput means the request payload itself was
	// malformed (empty ProjectPath, etc.).
	ExtractErrorInvalidInput ExtractErrorKind = "invalid_input"
	// ExtractErrorSidecarUnavailable means the Node sidecar is not in a
	// state to accept requests.
	ExtractErrorSidecarUnavailable ExtractErrorKind = "sidecar_unavailable"
	// ExtractErrorSidecarTimeout means a sidecar call exceeded its
	// deadline.
	ExtractErrorSidecarTimeout ExtractErrorKind = "sidecar_timeout"
	// ExtractErrorInternal means the sidecar (or this Service) returned
	// an unexpected error; the caller sees CodeInternal.
	ExtractErrorInternal ExtractErrorKind = "internal"
)

// ExtractError is the typed sentinel the Service returns when graph
// extraction cannot proceed. Handlers translate the Kind to a Connect
// error code via ErrorToConnectCode.
type ExtractError struct {
	Kind    ExtractErrorKind
	Path    string
	Message string
	Cause   error
}

func (e ExtractError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("extract %s (%s): %s", e.Kind, e.Path, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("extract %s (%s): %v", e.Kind, e.Path, e.Cause)
	}
	return fmt.Sprintf("extract %s (%s)", e.Kind, e.Path)
}

func (e ExtractError) Unwrap() error { return e.Cause }
