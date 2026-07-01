package authoring

import (
	"context"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func (s *service) runAutofill(ctx context.Context, sess *Session, src AutofillSource) AutofillResult {
	var (
		key     SectionKey
		content string
	)
	switch src {
	case AutofillRegressionAnchor:
		key = SectionRegressionAnchor
		if s.anchor == nil {
			return degraded(src, key, "anchor intent deriver unavailable")
		}
		boundary := planmodel.ParseBoundarySection(contentOf(sess.Sections, SectionAcceptanceBoundary))
		content = s.anchor.DeriveAnchorIntent(ctx, sess.Title, sess.Slug, boundary)
	default:
		return AutofillResult{Source: src, Degraded: true, Detail: "unknown autofill source"}
	}
	if strings.TrimSpace(content) == "" {
		return degraded(src, key, "source returned no content")
	}
	idx := indexOf(sess.Sections, key)
	if idx < 0 {
		return degraded(src, key, "section not present in this session")
	}
	sess.Sections[idx].Content = content
	sess.Sections[idx].Filled = true
	sess.Sections[idx].Autofilled = true
	return AutofillResult{Source: src, SectionKey: key, Filled: true, Detail: "autofilled"}
}

// SuggestReferences queries search-hub's Answer projection from the session's
// title + scope + technical approach and stores reviewable reference candidates
// (routed by locator shape) on the session. It NEVER writes the references
// section — only Accept finalizes a reviewed candidate. A nil seam / error /
// empty result degrades honestly to no candidates (the references step still
// offers manual entry and NO_CODE_REFS).
