// Package gates provides a read-model over existing swarm state that
// enumerates decision points ("gates") requiring a human or agent action
// before dependent work can proceed: unanswered workshop decisions,
// review-pending items and runs, captures awaiting classification, and
// items whose workshop maturity is not yet execution-ready.
//
// The package owns no storage and no policy — it only projects sources
// that already exist. Sources sit behind the Source interface so a future
// gate pre-approval / autonomy-policy layer has a single seam to plug into.
package gates

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// Kind classifies what sort of action resolves a gate.
type Kind string

const (
	// KindDecide — unanswered workshop decisions on a backlog item.
	KindDecide Kind = "decide"
	// KindReview — a human review is pending (review_pending item or a
	// needs_review / needs_fixup execution).
	KindReview Kind = "review"
	// KindClassify — a capture is awaiting classification review.
	KindClassify Kind = "classify"
	// KindWorkshop — a queueable item whose plan maturity is not yet
	// execution-ready (agent-actionable, not a human gate).
	KindWorkshop Kind = "workshop"
)

// Gate is one enumerated decision point.
type Gate struct {
	// ID is stable per (kind, owner): e.g. "decide:fix/foo".
	ID         string
	Kind       Kind
	OwnerType  string // "backlog" | "execution" | "capture"
	OwnerKind  string // backlog kind when OwnerType == "backlog"
	OwnerName  string
	OwnerTitle string
	// Count of pending questions / unreviewed units behind the gate.
	Count int
	// Blocks lists item keys (kind/name) directly blocked while this gate
	// is open — the owner item's direct dependents.
	Blocks []string
	// DecidableSince is the RFC3339 timestamp the gate became answerable,
	// when the source knows it.
	DecidableSince string
	// Suggested is a workshop-gate hint: "workshop" or "finalize".
	Suggested string
}

// GateID builds the canonical gate identifier.
func GateID(kind Kind, ownerType, ownerRef string) string {
	return fmt.Sprintf("%s:%s/%s", kind, ownerType, ownerRef)
}

// Source enumerates gates from one underlying data source.
type Source interface {
	// Name identifies the source in degradation logs.
	Name() string
	Enumerate(ctx context.Context) ([]Gate, error)
}

// Service concatenates all sources into one gate list. A failing source
// degrades gracefully (logged, skipped) — the board stays useful with the
// remaining sources.
type Service struct {
	sources []Source
}

// NewService builds a Service over the given sources.
func NewService(sources ...Source) *Service {
	return &Service{sources: sources}
}

// Enumerate returns all gates from all sources, deterministically sorted
// by (kind, owner type, owner name).
func (s *Service) Enumerate(ctx context.Context) []Gate {
	var all []Gate
	for _, src := range s.sources {
		gs, err := src.Enumerate(ctx)
		if err != nil {
			slog.Warn("gates: source enumeration failed; omitting", "source", src.Name(), "error", err)
			continue
		}
		all = append(all, gs...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Kind != all[j].Kind {
			return all[i].Kind < all[j].Kind
		}
		if all[i].OwnerType != all[j].OwnerType {
			return all[i].OwnerType < all[j].OwnerType
		}
		if all[i].OwnerKind != all[j].OwnerKind {
			return all[i].OwnerKind < all[j].OwnerKind
		}
		return all[i].OwnerName < all[j].OwnerName
	})
	return all
}
