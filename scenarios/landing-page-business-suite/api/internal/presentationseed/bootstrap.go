package presentationseed

import (
	"context"
	"errors"
	"strings"

	"landing-page-business-suite-api/internal/experimentation"
)

// EnsureDraft is idempotent across restarts. It never updates a populated head,
// even when the shipped recommendation changes, and it never publishes.
// The composition root supplies the actual commerce/delivery bundle identity;
// the design template's illustrative key is never treated as a second catalog.
func EnsureDraft(ctx context.Context, store *experimentation.ConfigStore, variant, bundleKey string) (*experimentation.PresentationState, bool, error) {
	if strings.TrimSpace(bundleKey) == "" || strings.TrimSpace(bundleKey) != bundleKey {
		return nil, false, errors.New("presentation bootstrap requires an exact configured bundle key")
	}
	state, err := store.GetPresentationState(ctx, variant)
	if err != nil && !errors.Is(err, experimentation.ErrPresentationNotFound) {
		return nil, false, err
	}
	if err == nil && (state.Generation != 0 || state.DraftRevision != "" || state.ActiveRevision != "") {
		return state, false, nil
	}
	document, err := Recommended()
	if err != nil {
		return nil, false, err
	}
	document.Bundle.Key = bundleKey
	if err := document.Validate(); err != nil {
		return nil, false, err
	}
	state, err = store.SavePresentationDraft(ctx, variant, document, 0)
	if errors.Is(err, experimentation.ErrPresentationConflict) {
		// Another startup/editor may already have initialized it. Never replace
		// that work with our seed, and never report an empty head as initialized.
		current, readErr := store.GetPresentationState(ctx, variant)
		if readErr == nil && current.Generation > 0 {
			return current, false, nil
		}
	}
	return state, err == nil, err
}
