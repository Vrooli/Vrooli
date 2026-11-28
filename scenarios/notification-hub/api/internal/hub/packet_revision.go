package hub

import (
	"context"
	"strings"
)

// A revision repeats the packet check. Every finding that was true of the open
// ask is true of the revised one too, unless the revision touched the part it is
// about, so an agent that revises a long ask would read the same advice on every
// call and learn to skip the block. The Hub marks such a finding carried over; a
// caller shows it as a count, and what is news stays in full.

// revisionFindings is what a revision needs to know about the ask it revises.
type revisionFindings struct {
	// said is the request as it came: what it says is what it changes.
	said AskSpec
	// before are the packet check's findings on the open ask as it stands.
	before []PacketWarning
	// revising is false when the request revises nothing: all findings are news.
	revising bool
}

// beginRevision reads the findings of the ask spec revises, before the revision
// changes it. It is best effort: without the open ask every finding is news.
func (s *Service) beginRevision(ctx context.Context, spec AskSpec, read BlobReader) revisionFindings {
	if spec.SubjectKey == "" {
		return revisionFindings{}
	}
	open, found, err := s.openSubjectAsk(ctx, spec)
	if err != nil || !found {
		return revisionFindings{}
	}
	attachments, err := s.loadAttachments(ctx, open.ID)
	if err != nil {
		return revisionFindings{}
	}
	before := AskSpec{
		Question: open.Question, BigPicture: open.BigPicture, TechnicalDetails: open.TechnicalDetails, ContextURL: open.ContextURL,
		Recommended: open.Recommended, RecommendationReason: open.RecommendationReason,
		Options: open.Options, Source: open.Source, SourceEventType: open.SourceEventType, SubjectKey: open.SubjectKey,
		Attachments: attachments, Questions: open.Questions,
	}
	return revisionFindings{said: spec, before: s.checkPacket(ctx, before, read, open.ID, open.Sender), revising: true}
}

// markCarriedOver marks every finding that the open ask already had, word for
// word, about a part the request did not say.
func (r revisionFindings) markCarriedOver(findings []PacketWarning) []PacketWarning {
	if !r.revising {
		return findings
	}
	had := make(map[PacketWarning]bool, len(r.before))
	for _, finding := range r.before {
		had[finding] = true
	}
	for index, finding := range findings {
		if had[finding] && !r.saidPart(finding.Field) {
			findings[index].CarriedOver = true
		}
	}
	return findings
}

// saidPart reports whether the request touched the part a finding's field names.
func (r revisionFindings) saidPart(field string) bool {
	base, _, _ := strings.Cut(field, "[")
	base, _, _ = strings.Cut(base, ".")
	keep := r.said.Keep
	switch base {
	case "question":
		return !keep.Question
	case "big_picture":
		return !keep.BigPicture
	case "technical_details":
		return !keep.TechnicalDetails
	case "context_url":
		return !keep.ContextURL
	case "recommendation_reason":
		return !keep.RecommendationReason || !keep.Recommended
	case "attachments":
		// The packet check names the explainer page; a request that sent an
		// explainer, or took attachments away, touched it.
		return r.touchedPage()
	case "options":
		return !keep.Options
	case "explainer":
		// Whether the page shows a command depends on the page and on the
		// cover that asks for one.
		return r.touchedPage() || !keep.Question || !keep.Options || !keep.BigPicture
	}
	return true
}

// touchedPage reports a request that sent an explainer or took attachments away.
func (r revisionFindings) touchedPage() bool {
	for _, attachment := range r.said.Attachments {
		if attachment.Role == AttachmentRoleExplainer {
			return true
		}
	}
	return r.said.ClearAttachments || len(r.said.RemoveAttachments) > 0
}
