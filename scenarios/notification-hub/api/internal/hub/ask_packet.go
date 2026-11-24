package hub

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// The decision packet of an ask: attachments, the answer form, and the advice
// about both. AskDecision prepares the attachments (attachments.go), then the
// form, and checks the packet once the ask is stored.

// prepareForm validates the answer form against the attachments the Hub
// accepted, because choices and sections may point at them by name. What the
// Hub cannot accept is left out and reported.
func prepareForm(spec AskSpec) (AskSpec, []PacketWarning) {
	attached := make(map[string]bool, len(spec.Attachments))
	for _, attachment := range spec.Attachments {
		attached[attachment.Name] = true
	}
	var warnings []PacketWarning
	spec.Questions, warnings = normalizeQuestions(spec.Questions, attached)
	if spec.AcceptsContextRequests && spec.SubjectKey == "" {
		// The answer to a question is a revision under the same subject key.
		spec.AcceptsContextRequests = false
		warnings = append(warnings, PacketWarning{
			Code: WarnContextRequestsNeedSubjectKey, Field: "accepts_context_requests", Severity: WarningDropped,
			Message: "accepts_context_requests needs a subject_key, because the answer to a question is a revision under the same key; the operator will not be offered a question button",
		})
	}
	return spec, warnings
}

// checkPacket returns the packet check's advice for a normalized request. The
// explainer page is read from the Hub's own copy; a page the Hub cannot read
// gets no content advice (the attachment warnings already say why).
//
// except is the ask itself when it is stored or revised (so it is not its own
// history), and sender the name the operator sees on it; the recurring-subject
// rule reads the sender's recent asks.
func (s *Service) checkPacket(ctx context.Context, spec AskSpec, read BlobReader, except, sender string) []PacketWarning {
	attachments := make([]PacketAttachment, 0, len(spec.Attachments))
	var explainer *PacketExplainer
	for index, attachment := range spec.Attachments {
		attachments = append(attachments, PacketAttachment{Name: attachment.Name, SourcePath: attachment.SourcePath})
		if attachment.Role != AttachmentRoleExplainer || explainer != nil {
			continue
		}
		explainer = &PacketExplainer{
			Field: fmt.Sprintf("attachments[%d]", index), Name: attachment.Name, SourcePath: attachment.SourcePath,
			MediaType: attachment.MediaType, IsFolder: attachment.Kind == AttachmentFolder, Size: attachment.SizeBytes,
		}
		if attachment.Kind == AttachmentFile && attachment.SizeBytes <= MaxExplainerBytes {
			page, err := read(ctx, attachment.SHA256, MaxExplainerBytes)
			if err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrBlobTooLarge) {
				s.log.Printf("packet check: read explainer %s: %v", attachment.Name, err)
			}
			explainer.HTML = page
		}
	}
	warnings := CheckPacket(spec, attachments, explainer)
	warnings = append(warnings, checkFeatureDecisionCards(ctx, spec, explainer, read)...)
	warnings = append(warnings, checkRegionFeedbackBundles(ctx, spec, read)...)
	warnings = append(warnings, s.checkRecurring(ctx, spec, sender, except)...)
	warnings = append(warnings, s.checkRelatedOpen(ctx, spec, except)...)
	return warnings
}

// attachmentNotesFor validates the operator's attachment notes against the
// attachments the ask holds now.
func (s *Service) attachmentNotesFor(ctx context.Context, askID string, notes []AskAttachmentNote) ([]AskAttachmentNote, error) {
	if len(notes) == 0 {
		return nil, nil
	}
	attachments, err := s.loadAttachments(ctx, askID)
	if err != nil {
		return nil, err
	}
	return normalizeAttachmentNotes(notes, attachments)
}

// WarnContextRequestsNeedSubjectKey reports a declaration the Hub dropped.
const WarnContextRequestsNeedSubjectKey = "context_requests_need_subject_key"

// WarnContextResponseUnused reports a context_response the Hub dropped because
// the ask had no open context question.
const WarnContextResponseUnused = "context_response_unused"

// openSubjectAsk is the open ask a request with a subject key would revise,
// read outside any transaction: for a preview, never for a write.
func (s *Service) openSubjectAsk(ctx context.Context, spec AskSpec) (AskRecord, bool, error) {
	if spec.SubjectKey == "" {
		return AskRecord{}, false, nil
	}
	open, err := scanAsk(s.db.QueryRowContext(ctx, askSelect+askSubjectWhere+` AND a.state IN ('pending','escalated') LIMIT 1`, spec.Source, spec.SourceEventType, spec.SubjectKey))
	if errors.Is(err, ErrNotFound) {
		return AskRecord{}, false, nil
	}
	if err != nil {
		return AskRecord{}, false, err
	}
	return open, true, nil
}

// sealAsk makes the ask a request is stored as: what the request says laid over
// what the open ask of its subject says, validated whole, with the answer form
// checked against the merged attachments. open is nil, and current empty, for a
// request that revises nothing.
//
// A revision says only what changes. keepOpenTerms keeps the terms it leaves out.
// For the packet, what it sends replaces the part with the same name or question
// id, or is added; everything else stays. The request's ClearAttachments and
// ClearQuestions drop every current part first, and RemoveAttachments drops the
// named attachments. A question that follows a file (SectionsFrom) takes its
// sections from the file as the merged packet holds it, so a revision that
// replaces the file changes the review with it.
//
// Sealing reads nothing from the database, so a storing request runs it inside
// the write transaction that reads open and current: two concurrent revisions
// then merge one after the other, and neither loses what the other changed. The
// slow work on the request's own parts (hashing, blob lookups) is done before, in
// prepareAttachments; the only read left is of the small Markdown file a review
// follows, through read.
func sealAsk(ctx context.Context, spec AskSpec, open *AskRecord, current []AskAttachment, read BlobReader) (AskSpec, []PacketWarning, error) {
	var warnings []PacketWarning
	completing := spec.Keep.Any()
	if open != nil {
		spec = keepOpenTerms(spec, *open)
		if !spec.ClearAttachments {
			kept, removal := removeAttachments(current, spec.RemoveAttachments)
			warnings = append(warnings, removal...)
			var capped []PacketWarning
			spec.Attachments, capped = capAttachments(mergeAttachments(kept, spec.Attachments))
			warnings = append(warnings, capped...)
		}
		if !spec.ClearQuestions {
			kept, removal := removeQuestions(open.Questions, spec.RemoveQuestions)
			warnings = append(warnings, removal...)
			spec.Questions = mergeQuestions(kept, spec.Questions)
		}
	} else if spec.Keep.Question || spec.Keep.Options {
		return spec, nil, missingTermsError(spec)
	}
	spec.Keep = AskKeep{}
	spec.RemoveAttachments, spec.RemoveQuestions = nil, nil
	if spec.ContextResponse != "" && (open == nil || open.OpenContextRequestCount == 0) {
		spec.ContextResponse = ""
		warnings = append(warnings, PacketWarning{
			Code: WarnContextResponseUnused, Field: "context_response", Severity: WarningDropped,
			Message: "context_response was not stored: the operator has no open context question on this ask. A response answers a question the operator asked; the rest of the request was applied.",
		})
	}
	if completing {
		var err error
		if spec, err = normalizeAskSpec(spec); err != nil {
			return spec, nil, err
		}
	}
	var followed []PacketWarning
	spec.Questions, followed = followSections(ctx, spec.Questions, spec.Attachments, read)
	spec, formWarnings := prepareForm(spec)
	return spec, append(append(warnings, followed...), formWarnings...), nil
}

// removeAttachments drops the named attachments from the current ones. A name the
// ask has no attachment of changes nothing and is reported, because it is
// usually a misspelling the sender would otherwise never learn of.
func removeAttachments(current []AskAttachment, names []string) ([]AskAttachment, []PacketWarning) {
	if len(names) == 0 {
		return current, nil
	}
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[strings.TrimSpace(name)] = true
	}
	kept := make([]AskAttachment, 0, len(current))
	for _, attachment := range current {
		if drop[attachment.Name] {
			delete(drop, attachment.Name)
			continue
		}
		kept = append(kept, attachment)
	}
	var warnings []PacketWarning
	for _, name := range names {
		if name = strings.TrimSpace(name); drop[name] {
			delete(drop, name)
			warnings = append(warnings, attachmentWarning(WarnAttachmentRemoveUnknown, "remove_attachments",
				fmt.Sprintf("the ask has no attachment named %q to remove; its attachments are named by file name (see `conversations get`).", name), WarningAdvice))
		}
	}
	return kept, warnings
}

// removeQuestions drops the questions with the named ids from the current ones.
// An id the ask has no question of changes nothing and is reported: it is usually
// a misspelling.
func removeQuestions(current []AskQuestion, ids []string) ([]AskQuestion, []PacketWarning) {
	if len(ids) == 0 {
		return current, nil
	}
	drop := make(map[string]bool, len(ids))
	for _, id := range ids {
		drop[strings.TrimSpace(id)] = true
	}
	kept := make([]AskQuestion, 0, len(current))
	for _, question := range current {
		if drop[question.ID] {
			delete(drop, question.ID)
			continue
		}
		kept = append(kept, question)
	}
	var warnings []PacketWarning
	for _, id := range ids {
		if id = strings.TrimSpace(id); drop[id] {
			delete(drop, id)
			warnings = append(warnings, PacketWarning{
				Code: WarnQuestionRemoveUnknown, Field: "remove_questions", Severity: WarningAdvice,
				Message: fmt.Sprintf("the ask has no form question with the id %q to remove; its questions are named by id (see `conversations get`).", id),
			})
		}
	}
	return kept, warnings
}

// missingTermsError refuses a request that leaves out the question or the answers
// and has no open ask to take them from.
func missingTermsError(spec AskSpec) error {
	if spec.SubjectKey == "" {
		return fmt.Errorf("%w: question, deadline, and allowed answers are required", ErrInvalidArgument)
	}
	return fmt.Errorf("%w: a new ask needs a question and answers; no open ask has the subject key %q to revise", ErrInvalidArgument, spec.SubjectKey)
}

// keepOpenTerms gives the request the open ask's value for every field it left
// unsaid. The recommendation and the default answer name answers of the menu, so
// they are kept only with the menu, and the recommendation's reason only with its
// recommendation: a reason belongs to the answer it explains.
func keepOpenTerms(spec AskSpec, open AskRecord) AskSpec {
	keep := spec.Keep
	if keep.Question {
		spec.Question = open.Question
	}
	if keep.Options {
		spec.Options = append([]AskOption(nil), open.Options...)
	}
	if keep.Recommended && keep.Options {
		spec.Recommended = open.Recommended
	}
	if keep.RecommendationReason && keep.Recommended && keep.Options {
		spec.RecommendationReason = open.RecommendationReason
	}
	if keep.DefaultAnswer && keep.Options {
		spec.DefaultAnswer = open.DefaultAnswer
	}
	if keep.Reversible {
		spec.Reversible = open.Reversible
	}
	if keep.Urgency {
		spec.Urgency = open.Urgency
	}
	if keep.ContextURL {
		spec.ContextURL = open.ContextURL
	}
	if keep.BigPicture {
		spec.BigPicture = open.BigPicture
	}
	if keep.TechnicalDetails {
		spec.TechnicalDetails = open.TechnicalDetails
	}
	if keep.ReassertWithin {
		spec.ReassertWithin = time.Duration(open.ReassertWithinSeconds) * time.Second
	}
	if keep.AcceptsContextRequests {
		spec.AcceptsContextRequests = open.AcceptsContextRequests
	}
	if keep.Packet {
		spec.Packet = open.Packet
	}
	if keep.DecisionFamily {
		spec.DecisionFamily = open.DecisionFamily
	}
	if keep.DecisionCause {
		spec.DecisionCause = open.DecisionCause
	}
	if keep.DecisionKind {
		spec.DecisionKind = open.DecisionKind
	}
	return spec
}

// capAttachments applies the count limit to a merged packet. The explainer comes
// first, so the limit never takes it.
func capAttachments(list []AskAttachment) ([]AskAttachment, []PacketWarning) {
	if len(list) <= maxAttachmentsPerAsk {
		return list, nil
	}
	return list[:maxAttachmentsPerAsk], []PacketWarning{attachmentWarning(WarnAttachmentLimit, "attachments",
		fmt.Sprintf("%d attachments were over the limit of %d and were left out; attach fewer files or put them in a folder.", len(list)-maxAttachmentsPerAsk, maxAttachmentsPerAsk), WarningDropped)}
}

// mergeAttachments lays the sent attachments over the current ones. A sent
// attachment replaces the current one of the same name, and a sent explainer
// replaces the current explainer; the explainer goes first, as a producer lists
// it, so the attachment limit never takes it. sent is already shaped, so its
// names are final.
func mergeAttachments(current, sent []AskAttachment) []AskAttachment {
	if len(sent) == 0 {
		return current
	}
	replaced := make(map[string]bool, len(sent))
	sentExplainer := false
	for _, attachment := range sent {
		replaced[attachment.Name] = true
		sentExplainer = sentExplainer || attachment.Role == AttachmentRoleExplainer
	}
	merged := make([]AskAttachment, 0, len(current)+len(sent))
	for _, attachment := range sent {
		if attachment.Role == AttachmentRoleExplainer {
			merged = append(merged, attachment)
		}
	}
	for _, attachment := range current {
		if replaced[attachment.Name] || (sentExplainer && attachment.Role == AttachmentRoleExplainer) {
			continue
		}
		merged = append(merged, attachment)
	}
	for _, attachment := range sent {
		if attachment.Role != AttachmentRoleExplainer {
			merged = append(merged, attachment)
		}
	}
	return merged
}

// mergeQuestions lays the sent questions over the current ones by question id:
// a sent question takes the place of the current one with its id, and a new one
// goes last.
func mergeQuestions(current, sent []AskQuestion) []AskQuestion {
	if len(sent) == 0 {
		return current
	}
	merged := make([]AskQuestion, len(current), len(current)+len(sent))
	copy(merged, current)
	position := make(map[string]int, len(current))
	for index, question := range merged {
		position[question.ID] = index
	}
	for _, question := range sent {
		if index, exists := position[strings.TrimSpace(question.ID)]; exists {
			merged[index] = question
			continue
		}
		position[strings.TrimSpace(question.ID)] = len(merged)
		merged = append(merged, question)
	}
	return merged
}
