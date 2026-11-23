package models

import (
	"context"
	"fmt"
	"time"

	"image-tools/internal/catalogstate"
	internalmodels "image-tools/internal/models"
	"image-tools/internal/verdicts"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// viewState is everything a response needs, read once: the effective enabled
// overlay, the stored overrides with their provenance, the install records and
// the verdict board. Handlers build every model view from one viewState so a
// single response cannot show two different states of the same catalog.
type viewState struct {
	overlay   map[string]bool
	overrides map[string]catalogstate.Override
	installs  map[string]internalmodels.InstallRecord
	board     *verdicts.Board // nil when no verdict source is wired
}

func (h *connectHandler) loadViewState(ctx context.Context) (*viewState, error) {
	st := &viewState{}
	var err error
	if st.overlay, err = h.overlay(ctx); err != nil {
		return nil, fmt.Errorf("load model state: %w", err)
	}
	if h.deps.Store != nil {
		if st.overrides, err = h.deps.Store.LoadOverrides(ctx); err != nil {
			return nil, fmt.Errorf("load model overrides: %w", err)
		}
	}
	if st.installs, err = h.installStates(ctx); err != nil {
		return nil, fmt.Errorf("load install state: %w", err)
	}
	if h.deps.Verdicts != nil {
		if st.board, err = h.deps.Verdicts.Load(ctx); err != nil {
			return nil, fmt.Errorf("load verdicts: %w", err)
		}
	}
	return st, nil
}

// held applies the catalog's own "why it is off" to a verdict when the model is
// not enabled right now. An operator-enabled model is not held: the reason
// describes the seed's state, which the override replaced.
func (st *viewState) held(m internalmodels.Model, v verdicts.Verdict) verdicts.Verdict {
	if m.Disabled == nil || internalmodels.EffectiveEnabled(m, st.overlay) {
		return v
	}
	return verdicts.ApplyHold(v, verdicts.Hold{Reason: m.Disabled.Reason, Breaks: m.Disabled.Breaks(), Ref: "registry.seed.json " + m.ID + ".disabled"})
}

// verdictFor is a model's verdict for one operation; untested without a board.
func (st *viewState) verdictFor(m internalmodels.Model, op string) verdicts.Verdict {
	if st == nil || st.board == nil {
		return verdicts.UntestedVerdict(op)
	}
	return st.held(m, st.board.For(m.ID, op))
}

func (st *viewState) modelVerdicts(m internalmodels.Model) ([]verdicts.Verdict, verdicts.Verdict) {
	if st == nil || st.board == nil {
		per := make([]verdicts.Verdict, 0, len(m.Operations))
		for _, op := range m.Operations {
			per = append(per, verdicts.UntestedVerdict(op))
		}
		return per, verdicts.Rollup(per)
	}
	per, _ := st.board.ForModel(m.ID, m.Operations)
	for i := range per {
		per[i] = st.held(m, per[i])
	}
	return per, verdicts.Rollup(per)
}

// overrideToProto renders a stored override against the seed's current flag.
// A nil override (the model follows the seed) renders as nil.
func overrideToProto(o *catalogstate.Override, seedEnabled bool) *modelsv1.StateOverride {
	if o == nil {
		return nil
	}
	out := &modelsv1.StateOverride{
		Enabled: o.Enabled, SetBy: o.SetBy, Reason: o.Reason, Pinned: o.Pinned, SeedEnabled: seedEnabled,
	}
	if out.SetBy == "" {
		out.SetBy = catalogstate.UnknownActor
	}
	if !o.SetAt.IsZero() {
		out.SetAt = o.SetAt.UTC().Format(time.RFC3339)
	}
	switch {
	case !o.Active():
		out.Relation, out.Note = "superseded", o.SupersededNote
	case o.Enabled != seedEnabled:
		out.Relation = "contradicts_seed"
		out.Note = fmt.Sprintf("the seed says enabled=%t; this override says enabled=%t and wins", seedEnabled, o.Enabled)
		if !o.Pinned {
			out.Note += " until the seed's flag changes"
		}
	default:
		out.Relation = "agrees"
	}
	return out
}
