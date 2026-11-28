package conversations

import (
	"context"

	"connectrpc.com/connect"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
)

// The answer form and context requests: the proto mapping of questions and
// responses, and the RequestContext procedure. Rules:
// docs/concepts/DECISION_PACKETS.md ("Answer form", "Context requests").

// RequestContext records the recipient's question about an open ask. Like
// Answer and Dismiss it is recipient-only.
func (h *handler) RequestContext(ctx context.Context, req *connect.Request[v1.RequestContextRequest]) (*connect.Response[v1.RequestContextResponse], error) {
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	request, err := h.service.RequestContext(ctx, req.Msg.GetAskId(), actor, req.Msg.GetQuestion())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.RequestContextResponse{AskId: req.Msg.GetAskId(), Request: contextRequestToProto(request)}), nil
}

func questionsFromProto(in []*v1.AskQuestion) []hub.AskQuestion {
	if len(in) == 0 {
		return nil
	}
	out := make([]hub.AskQuestion, 0, len(in))
	for _, question := range in {
		item := hub.AskQuestion{
			ID: question.GetId(), Type: question.GetType(), Prompt: question.GetPrompt(), Help: question.GetHelp(),
			Required: question.GetRequired(), RecommendationReason: question.GetRecommendationReason(),
			UnansweredVerdict: question.GetUnansweredVerdict(), SectionsFrom: question.GetSectionsFrom(),
		}
		for _, choice := range question.GetChoices() {
			item.Choices = append(item.Choices, hub.AskQuestionChoice{Key: choice.GetKey(), Label: choice.GetLabel(), Detail: choice.GetDetail(), Attachment: choice.GetAttachment()})
		}
		for _, section := range question.GetSections() {
			item.Sections = append(item.Sections, hub.AskQuestionSection{Key: section.GetKey(), Title: section.GetTitle(), Summary: section.GetSummary(), Attachment: section.GetAttachment(), Anchor: section.GetAnchor()})
		}
		if question.GetRecommended() != nil {
			recommended := responseFromProto(question.GetRecommended())
			item.Recommended = &recommended
		}
		out = append(out, item)
	}
	return out
}

func questionsToProto(in []hub.AskQuestion) []*v1.AskQuestion {
	if len(in) == 0 {
		return nil
	}
	out := make([]*v1.AskQuestion, 0, len(in))
	for _, question := range in {
		item := &v1.AskQuestion{
			Id: question.ID, Type: question.Type, Prompt: question.Prompt, Help: question.Help,
			Required: question.Required, RecommendationReason: question.RecommendationReason,
			UnansweredVerdict: question.UnansweredVerdict, SectionsFrom: question.SectionsFrom,
		}
		for _, choice := range question.Choices {
			item.Choices = append(item.Choices, &v1.AskQuestionChoice{Key: choice.Key, Label: choice.Label, Detail: choice.Detail, Attachment: choice.Attachment})
		}
		for _, section := range question.Sections {
			item.Sections = append(item.Sections, &v1.AskQuestionSection{Key: section.Key, Title: section.Title, Summary: section.Summary, Attachment: section.Attachment, Anchor: section.Anchor})
		}
		if question.Recommended != nil {
			item.Recommended = responseToProto(*question.Recommended)
		}
		out = append(out, item)
	}
	return out
}

func responseFromProto(in *v1.AskQuestionResponse) hub.AskQuestionResponse {
	out := hub.AskQuestionResponse{QuestionID: in.GetQuestionId(), Selected: in.GetSelected(), Text: in.GetText()}
	for _, verdict := range in.GetSections() {
		out.Sections = append(out.Sections, hub.AskSectionVerdict{SectionKey: verdict.GetSectionKey(), Verdict: verdict.GetVerdict(), Note: verdict.GetNote()})
	}
	return out
}

func responsesFromProto(in []*v1.AskQuestionResponse) []hub.AskQuestionResponse {
	out := make([]hub.AskQuestionResponse, 0, len(in))
	for _, response := range in {
		out = append(out, responseFromProto(response))
	}
	return out
}

func responseToProto(in hub.AskQuestionResponse) *v1.AskQuestionResponse {
	out := &v1.AskQuestionResponse{QuestionId: in.QuestionID, Selected: in.Selected, Text: in.Text}
	for _, verdict := range in.Sections {
		out.Sections = append(out.Sections, &v1.AskSectionVerdict{SectionKey: verdict.SectionKey, Verdict: verdict.Verdict, Note: verdict.Note})
	}
	return out
}

func responsesToProto(in []hub.AskQuestionResponse) []*v1.AskQuestionResponse {
	if len(in) == 0 {
		return nil
	}
	out := make([]*v1.AskQuestionResponse, 0, len(in))
	for _, response := range in {
		out = append(out, responseToProto(response))
	}
	return out
}

func contextRequestToProto(in hub.AskContextRequest) *v1.AskContextRequest {
	return &v1.AskContextRequest{
		Id: in.ID, Question: in.Question, RequestedBy: in.RequestedBy, RequestedAt: in.RequestedAt,
		AddressedAt: in.AddressedAt, AddressedRevision: int32(in.AddressedRevision), Response: in.Response,
	}
}

func contextRequestsToProto(in []hub.AskContextRequest) []*v1.AskContextRequest {
	if len(in) == 0 {
		return nil
	}
	out := make([]*v1.AskContextRequest, 0, len(in))
	for _, request := range in {
		out = append(out, contextRequestToProto(request))
	}
	return out
}

func attachmentNotesFromProto(in []*v1.AskAttachmentNote) []hub.AskAttachmentNote {
	out := make([]hub.AskAttachmentNote, 0, len(in))
	for _, note := range in {
		out = append(out, hub.AskAttachmentNote{Attachment: note.GetAttachment(), Path: note.GetPath(), Note: note.GetNote()})
	}
	return out
}

func attachmentNotesToProto(in []hub.AskAttachmentNote) []*v1.AskAttachmentNote {
	if len(in) == 0 {
		return nil
	}
	out := make([]*v1.AskAttachmentNote, 0, len(in))
	for _, note := range in {
		out = append(out, &v1.AskAttachmentNote{Attachment: note.Attachment, Path: note.Path, Note: note.Note})
	}
	return out
}
