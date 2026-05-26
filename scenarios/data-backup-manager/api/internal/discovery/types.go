// Package discovery is the onboarding-suggestions domain. It scans the local
// environment read-only and derives two kinds of one-click suggestions —
// targets worth protecting and destinations worth backing up to — filtering
// against the live catalog and a small dismissals table.
//
// Suggestions are NEVER persisted (only dismissals are): the service recomputes
// each call, so an accepted suggestion disappears naturally once it exists in
// the targets/destinations catalog. "Accepting" is not a method here — the UI
// and CLI call the existing RegisterTarget / CreateDestination, preserving a
// single source of truth and reusing their validation. The scanners are
// strictly read-only and never read the contents of any scanned path.
//
// Layering mirrors the canonical per-domain pattern, but the "repository" here
// is just the DismissalStore — there is no suggestions table.
//
//	Connect handler → Service (scan, filter, rank) → seams (scanners, catalogs, dismissals)
//
// The proto wire types live one floor up (packages/proto/...) and never import
// this package; the handler is the only translation point.
package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/sysmounts"
)

// DriveClass is the volume classification, re-exported from sysmounts so the
// handler and tests share one name. The handler maps it to the proto enum.
type DriveClass = sysmounts.DriveClass

// suggestionKind tags a stable suggestion id so the target and destination
// namespaces never collide even if a path happened to coincide.
const (
	kindTarget      = "target"
	kindDestination = "dest"
)

// TargetCandidate is a discovered source worth protecting, before any catalog
// or dismissal filtering. Produced by a TargetSourceScanner.
type TargetCandidate struct {
	// Owner/Name are the (owner, name) key a registered target would carry.
	Owner string
	Name  string
	// SourceKind selects the capture strategy on registration.
	SourceKind sources.SourceKind
	// Locator is the absolute source locator (a path for filesystem, a db file
	// for sqlite). Stable input to the suggestion id.
	Locator string
	// Rationale is the human-facing reason this is worth protecting.
	Rationale string
	// ApproxBytes is a bounded best-effort size estimate; 0 = unknown.
	ApproxBytes int64
}

// TargetSuggestion is a candidate that survived filtering, with its stable id.
type TargetSuggestion struct {
	ID string
	TargetCandidate
}

// DestinationSuggestion is a mounted volume offered as a backup destination.
type DestinationSuggestion struct {
	ID             string
	Label          string
	Location       string
	Class          DriveClass
	FreeBytes      int64
	TotalBytes     int64
	Removable      bool
	SeparateRootOK bool
	Rationale      string
}

// ExistingTarget is the minimal projection of a registered target the
// TargetCatalog returns for filtering.
type ExistingTarget struct {
	Owner   string
	Name    string
	Locator string
}

// ExistingDestination is the minimal projection of a registered destination the
// DestinationCatalog returns for filtering.
type ExistingDestination struct {
	Location string
}

// ErrInvalidDiscovery is the typed validation sentinel. Handlers translate it
// into connect.CodeInvalidArgument carrying "<field>: <reason>".
type ErrInvalidDiscovery struct {
	Field  string
	Reason string
}

func (e ErrInvalidDiscovery) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// targetSuggestionID / destSuggestionID derive stable ids from the locator so a
// dismissal survives rescans and process restarts (idempotency).
func targetSuggestionID(locator string) string { return suggestionID(kindTarget, locator) }
func destSuggestionID(location string) string  { return suggestionID(kindDestination, location) }

func suggestionID(kind, locator string) string {
	sum := sha256.Sum256([]byte(kind + "|" + locator))
	return hex.EncodeToString(sum[:])[:16]
}

// Volume is re-exported so callers that wire the seam don't import sysmounts.
type Volume = sysmounts.Volume
