// Package graph is the domain-scoped home for deterministic Go module
// extraction. It owns the in-process representation of a Go module's
// package/file/symbol graph and the seams that produce it.
//
// Layering follows the canonical Vrooli pattern: handler → Service →
// seam (PackagesLoader). The PackagesLoader seam is the only edge that
// touches golang.org/x/tools/go/packages.
//
// The domain types in this file deliberately mirror the relevant subset
// of scenarios/architecture-cartographer/api/internal/graph/types.go so
// cartographer can translate ExtractResponse → its own RawGraph in a
// single mechanical step.
package graph

import (
	"fmt"
)

// ExtractionProfile controls the amount of analysis performed for one
// request. Full is the compatibility profile and preserves the original
// graph contract. The lighter profiles are explicit opt-ins for consumers
// that do not need semantic usage facts.
type ExtractionProfile string

// InformationOmission describes a capability intentionally excluded by the
// selected extraction profile. It is informational, not an extraction error.
type InformationOmission struct {
	Capability string
	Reason     string
}

const (
	ExtractionProfileFull       ExtractionProfile = "full"
	ExtractionProfileSemantic   ExtractionProfile = "semantic"
	ExtractionProfileStructural ExtractionProfile = "structural"
)

func (p ExtractionProfile) valid() bool {
	switch p {
	case ExtractionProfileFull, ExtractionProfileSemantic, ExtractionProfileStructural:
		return true
	default:
		return false
	}
}

func (p ExtractionProfile) normalized() ExtractionProfile {
	if !p.valid() {
		return ExtractionProfileFull
	}
	return p
}

// OmittedInformation returns the explicit capability contract for a profile.
// Keep these identifiers stable so consumers can explain absent fields.
func (p ExtractionProfile) OmittedInformation() []InformationOmission {
	switch p.normalized() {
	case ExtractionProfileSemantic:
		return []InformationOmission{
			{Capability: "test_variants", Reason: "semantic profile excludes test packages"},
			{Capability: "test_only_relationships", Reason: "test packages were not loaded"},
		}
	case ExtractionProfileStructural:
		return []InformationOmission{
			{Capability: "resolved_type_information", Reason: "structural profile does not run Go type checking"},
			{Capability: "resolved_usage_facts", Reason: "structural profile cannot resolve references, calls, or type usages"},
			{Capability: "test_variants", Reason: "structural profile excludes test packages"},
			{Capability: "test_only_relationships", Reason: "test packages were not loaded"},
		}
	default:
		return nil
	}
}

// Language is the source language a node/edge belongs to.
type Language string

const (
	// LanguageGo is the only producer language this scenario ships.
	LanguageGo Language = "go"
)

// NodeKind enumerates the structural node families this scenario emits.
// Values are stable strings so canonical JSON serialization (used by
// GraphHash) is stable across Go upgrades.
type NodeKind string

const (
	NodeKindFile              NodeKind = "file"
	NodeKindPackage           NodeKind = "package"
	NodeKindModule            NodeKind = "module"
	NodeKindType              NodeKind = "go_type"
	NodeKindFunc              NodeKind = "go_func"
	NodeKindVar               NodeKind = "go_var"
	NodeKindConst             NodeKind = "go_const"
	NodeKindInterface         NodeKind = "go_interface"
	NodeKindMethod            NodeKind = "go_method"
	NodeKindImportSpec        NodeKind = "go_import_spec"
	NodeKindReference         NodeKind = "go_reference"
	NodeKindCall              NodeKind = "go_call"
	NodeKindTypeUsage         NodeKind = "go_type_usage"
	NodeKindRouteRegistration NodeKind = "go_route_registration"
)

// EdgeKind enumerates the directed-edge families this scenario emits.
type EdgeKind string

const (
	EdgeKindImport          EdgeKind = "import"
	EdgeKindIntraPackageRef EdgeKind = "intra_package_ref"
)

// FileNode is one Go source file. ID format: "file:<path>".
type FileNode struct {
	ID        string
	Path      string
	PackageID string
	IsTest    bool
}

// Package is one Go package. ID format: "package:<import_path>".
type Package struct {
	ID         string
	ImportPath string
	Name       string
	Directory  string
	// Module reports the module path the package belongs to, empty for
	// standard-library imports.
	Module string
}

// SymbolNode is one declared identifier (type/func/var/const/method).
// ID format: "<kind>:<package_id>:<name>".
type SymbolNode struct {
	ID        string
	Name      string
	PackageID string
	FileID    string
	Kind      NodeKind
	Exported  bool
}

// ImportEdge is a directed dependency between packages. ID format:
// "import:<from_package_id>:<to_package_id>".
type ImportEdge struct {
	ID            string
	FromPackageID string
	ToPackageID   string
}

// Node is the union row emitted on the proto wire. Domain code keeps
// the kind-specific structs (FileNode, Package, SymbolNode) and the
// adapter projects them onto Node for serialization. Carried here so
// hash.go's canonical JSON has exactly one shape.
type Node struct {
	ID         string            `json:"id"`
	Kind       NodeKind          `json:"kind"`
	Name       string            `json:"name"`
	Path       string            `json:"path"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Edge is the union row emitted on the proto wire.
type Edge struct {
	ID         string            `json:"id"`
	Kind       EdgeKind          `json:"kind"`
	From       string            `json:"from"`
	To         string            `json:"to"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// WarningKind classifies non-fatal extraction issues.
type WarningKind string

const (
	WarningKindParseError       WarningKind = "parse_error"
	WarningKindUnresolvedImport WarningKind = "unresolved_import"
	WarningKindTypeCheckFailure WarningKind = "type_check_failure"
	WarningKindUnresolvedSymbol WarningKind = "unresolved_symbol"
)

// Warning is a non-fatal issue surfaced alongside an otherwise-valid
// Graph. Catastrophic project errors are returned as ExtractError, not
// Warning.
type Warning struct {
	Kind    WarningKind
	File    string
	Message string
}

// Graph is the normalized, immutable result of extracting one Go
// module. All collections are sorted by stable key before the value is
// constructed; downstream serializers can rely on byte stability.
type Graph struct {
	// Nodes is the union list — every FileNode/Package/SymbolNode
	// projected onto Node and sorted by ID.
	Nodes []Node `json:"nodes"`
	// Edges is every ImportEdge (and intra-package ref edges, if any)
	// projected onto Edge and sorted by ID.
	Edges []Edge `json:"edges"`
}

// ExtractInput is the validated request payload threaded from handler
// to Service. ModulePath is the absolute path to the module root.
type ExtractInput struct {
	ModulePath      string
	IncludeVendor   bool
	Profile         ExtractionProfile
	PackagePatterns []string
}

// ExtractErrorKind names the catastrophic conditions that prevent a
// graph from being produced at all (as opposed to partial-success
// Warnings).
type ExtractErrorKind string

const (
	// ExtractErrorNoGoMod means no go.mod was found at ModulePath.
	ExtractErrorNoGoMod ExtractErrorKind = "no_go_mod_found"
	// ExtractErrorMultipleGoMod means more than one go.mod exists
	// under ModulePath; the scenario boundary is ambiguous.
	ExtractErrorMultipleGoMod ExtractErrorKind = "multiple_go_mod_files"
	// ExtractErrorWorkspaceUnsupported means a go.work file was
	// detected; multi-module workspaces are out of scope for v1.
	ExtractErrorWorkspaceUnsupported ExtractErrorKind = "workspace_unsupported"
	// ExtractErrorPathUnreadable means the module path could not be
	// stat'd or read.
	ExtractErrorPathUnreadable ExtractErrorKind = "path_unreadable"
	// ExtractErrorInvalidInput means the request payload itself was
	// malformed (empty ModulePath, etc.).
	ExtractErrorInvalidInput ExtractErrorKind = "invalid_input"
	// ExtractErrorInternal means the loader returned an unexpected
	// error; the caller sees CodeInternal.
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
