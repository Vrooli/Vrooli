package hub

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path"
	"time"

	"github.com/vrooli/api-core/filekind"
	"github.com/vrooli/api-core/operatordecision"
)

// No decision reaches the operator as bare text. An ask that holds no page of
// its own is read with the page operatordecision.CoverPage makes from its
// cover: the question, the big picture, the options with the recommendation,
// what happens on approval and with no answer. The Hub builds it when the ask
// is read, from the ask as it is now, so it is always current, it covers asks
// stored before it existed, and a page the sender sends takes its place with
// nothing to replace. It is a floor: the packet check still advises the sender
// to make a real page (explainer_missing).

// CoverPageName is the attachment name under which a reader finds the Hub's
// page for an ask that has none of its own.
const CoverPageName = "decision-summary.html"

// The entry is shaped like a stored explainer: the Hub serves every attached
// page as text, and the explainer frame renders it sealed.

// coverPage is the Hub's page for ask and its attachment entry; ok is false
// when one of the ask's attachments is its own page.
func coverPage(ask AskRecord, attachments []AskAttachment) (entry AskAttachment, page []byte, ok bool) {
	brief := ""
	for _, attachment := range attachments {
		if attachment.Role == AttachmentRoleExplainer {
			return AskAttachment{}, nil, false
		}
		if ask.Packet != "" && attachment.Kind == AttachmentFile && (attachment.SourcePath == ask.Packet || attachment.SourcePath == path.Join(ask.Packet, "BRIEF.md")) {
			brief = attachment.Name
		}
	}
	facts := operatordecision.Facts{
		Question: ask.Question, BigPicture: ask.BigPicture, Recommended: ask.Recommended,
		RecommendationReason: ask.RecommendationReason, DefaultAnswer: ask.DefaultAnswer,
		Reversible: ask.Reversible, Packet: ask.Packet,
	}
	for _, option := range ask.Options {
		facts.Options = append(facts.Options, operatordecision.Option{Key: option.Key, Label: option.Label})
	}
	deadline, _ := time.Parse(time.RFC3339Nano, ask.Deadline)
	page = []byte(operatordecision.CoverPage(facts, brief, deadline))
	sum := sha256.Sum256(page)
	return AskAttachment{
		Name: CoverPageName, Label: "The Hub's page, made from the question and options",
		Kind: AttachmentFile, Role: AttachmentRoleExplainer, SHA256: hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(page)), MediaType: "text/plain; charset=utf-8", PreviewKind: string(filekind.KindCode), SourceStatus: SourceUnknown,
	}, page, true
}

// WithCoverPage is the attachments a reader of ask sees: its own, led by the
// Hub's page when it has no page of its own.
func WithCoverPage(ask AskRecord, attachments []AskAttachment) []AskAttachment {
	entry, _, ok := coverPage(ask, attachments)
	if !ok {
		return attachments
	}
	return append([]AskAttachment{entry}, attachments...)
}

// openCoverPage serves the Hub's page of an ask that has none of its own.
func (s *Service) openCoverPage(ctx context.Context, ask AskRecord) (AttachmentContent, error) {
	attachments, err := s.loadAttachments(ctx, ask.ID)
	if err != nil {
		return AttachmentContent{}, err
	}
	entry, page, ok := coverPage(ask, attachments)
	if !ok {
		return AttachmentContent{}, ErrNotFound
	}
	return AttachmentContent{FileName: CoverPageName, SHA256: entry.SHA256, SizeBytes: entry.SizeBytes, MediaType: entry.MediaType, Content: bytesSeeker{bytes.NewReader(page)}}, nil
}
