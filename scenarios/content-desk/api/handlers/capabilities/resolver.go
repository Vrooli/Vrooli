package capabilities

import (
	"context"
	"database/sql"
	"errors"

	"content-desk/internal/artifacts"
	internalcapabilities "content-desk/internal/capabilities"
)

// artifactOutputResolver reads current output state from the Content Desk
// artifact owner. It is read-only: it observes draft lifecycle state and never
// mutates the draft or copies its state into the capability catalog.
type artifactOutputResolver struct {
	drafts artifacts.Repository
}

func newArtifactOutputResolver(db artifacts.SQLExecutor) internalcapabilities.EvidenceResolver {
	return artifactOutputResolver{drafts: artifacts.NewSQLiteRepository(db)}
}

// CurrentOutputState maps a linked evidence id to the current draft lifecycle
// state. An id the artifact owner does not know resolves as not-artifact so the
// caller skips it instead of failing the read.
func (r artifactOutputResolver) CurrentOutputState(ctx context.Context, artifactID string) (internalcapabilities.OutputState, error) {
	draft, err := r.drafts.Get(ctx, artifactID)
	if errors.Is(err, sql.ErrNoRows) {
		return internalcapabilities.OutputState{ArtifactID: artifactID, Resolvable: false}, nil
	}
	if err != nil {
		return internalcapabilities.OutputState{}, err
	}
	return internalcapabilities.OutputState{
		ArtifactID: artifactID,
		Resolvable: true,
		Accepted:   draftAccepted(draft.Status),
	}, nil
}

// draftAccepted reports whether a draft's current lifecycle state is accepted
// output. Reaching reviewed means a review passed; approved and published come
// after the operator approval gate, which itself requires a passing review.
func draftAccepted(status artifacts.DraftStatus) bool {
	switch status {
	case artifacts.DraftReviewed, artifacts.DraftApproved, artifacts.DraftPublished:
		return true
	}
	return false
}
