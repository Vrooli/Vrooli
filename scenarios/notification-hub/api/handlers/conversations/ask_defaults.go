package conversations

import (
	"strings"
	"time"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
	"notification-hub/internal/identity"
)

// askDeadline reads an ask's deadline. A request that gives none gets the Hub's
// default (hub.DefaultAskDeadline from now) and is not "specified": its deadline
// is not part of the ask's identity, so sending the same request again returns
// the same ask instead of conflicting over a deadline that moved with the clock.
func askDeadline(value string, now time.Time) (deadline time.Time, specified bool, err error) {
	if strings.TrimSpace(value) == "" {
		return now.Add(hub.DefaultAskDeadline), false, nil
	}
	deadline, err = time.Parse(time.RFC3339, value)
	return deadline, true, err
}

// askSensitivity is the label of an ask: the caller's, or the Hub's default.
func askSensitivity(label string) string {
	if strings.TrimSpace(label) == "" {
		return hub.DefaultAskSensitivity
	}
	return label
}

// askSpecFromProto is the Hub's ask for a request from a verified producer. Ask
// and CheckAsk build it the same way, so a dry run checks what a send would store.
func askSpecFromProto(msg *v1.AskRequest, producer identity.Producer, now time.Time) (hub.AskSpec, error) {
	deadline, deadlineSpecified, err := askDeadline(msg.GetDeadline(), now)
	if err != nil {
		return hub.AskSpec{}, err
	}
	if err := hub.ValidateAskContextURL(msg.GetContextUrl()); err != nil {
		return hub.AskSpec{}, err
	}
	options := make([]hub.AskOption, 0, len(msg.GetOptions())+len(msg.GetAllowedAnswers()))
	for _, option := range msg.GetOptions() {
		options = append(options, hub.AskOption{Key: option.GetKey(), Label: option.GetLabel()})
	}
	if len(options) == 0 {
		for _, key := range msg.GetAllowedAnswers() {
			options = append(options, hub.AskOption{Key: key, Label: key})
		}
	}
	kind := strings.ToLower(strings.TrimSpace(msg.GetKind()))
	if kind == "status" && strings.TrimSpace(msg.GetDeadline()) == "" {
		deadline, deadlineSpecified = time.Time{}, false
	}
	var templateData map[string]any
	if msg.GetTemplateData() != nil {
		templateData = msg.GetTemplateData().AsMap()
	}
	var explainerHTML string
	if msg.GetTemplateName() != "" {
		explainerHTML, err = hub.RenderKitTemplate(strings.TrimSpace(msg.GetTemplateName()), templateData)
		if err != nil {
			return hub.AskSpec{}, err
		}
	}
	keep := askKeepFromProto(msg)
	if kind == "status" {
		keep = hub.AskKeep{}
	}
	audience := producer.Audience()
	return hub.AskSpec{
		RevisionAudience:        &audience,
		AllowRelated:            msg.GetAllowRelated(),
		Keep:                    keep,
		Kind:                    kind,
		TemplateName:            msg.GetTemplateName(),
		TemplateData:            templateData,
		ExplainerHTML:           explainerHTML,
		Recipient:               producer.Recipient,
		Source:                  producer.Source,
		Question:                msg.GetQuestion(),
		Options:                 options,
		Recommended:             msg.GetRecommended(),
		RecommendationReason:    msg.GetRecommendationReason(),
		DefaultAnswer:           msg.GetDefaultAnswer(),
		Reversible:              msg.GetReversible(),
		Urgency:                 msg.GetUrgency(),
		ContextURL:              msg.GetContextUrl(),
		BigPicture:              msg.GetBigPicture(),
		TechnicalDetails:        msg.GetTechnicalDetails(),
		ProviderDecisionContext: providerDecisionContextFromProto(msg.GetProviderDecisionContext()),
		Deadline:                deadline,
		DeadlineSpecified:       deadlineSpecified,
		SensitivityLabel:        askSensitivity(msg.GetSensitivityLabel()),
		IdempotencyKey:          msg.GetIdempotencyKey(),
		SubjectKey:              msg.GetSubjectKey(),
		ReassertWithin:          time.Duration(msg.GetReassertWithinSeconds()) * time.Second,
		Attachments:             attachmentsFromProto(msg.GetAttachments()),
		VerifiedProducer:        !producer.Anonymous(),
		AgentSent:               producer.Source == "",
		CreatedBy:               producer.Creator(),
		CreatedConversation:     producer.Conversation,
		CreatedSubject:          producer.Subject,
		Sender:                  msg.GetSender(),
		DefaultSender:           msg.GetDefaultSender(),
		Questions:               questionsFromProto(msg.GetQuestions()),
		AcceptsContextRequests:  msg.GetAcceptsContextRequests(),
		ClearAttachments:        msg.GetClearAttachments(),
		ClearQuestions:          msg.GetClearQuestions(),
		RemoveAttachments:       msg.GetRemoveAttachments(),
		RemoveQuestions:         msg.GetRemoveQuestions(),
		ContextResponse:         msg.GetContextResponse(),
		Packet:                  msg.GetPacket(),
	}, nil
}

// askKeepFromProto lists the fields a request left unsaid. A field marked
// optional is unsaid when unset (an explicit empty value says "clear it"); the
// question and the answers are unsaid when empty, because clearing either would
// leave nothing to ask.
func askKeepFromProto(msg *v1.AskRequest) hub.AskKeep {
	return hub.AskKeep{
		Question:               strings.TrimSpace(msg.GetQuestion()) == "",
		Options:                len(msg.GetOptions()) == 0 && len(msg.GetAllowedAnswers()) == 0,
		Recommended:            msg.Recommended == nil,
		RecommendationReason:   msg.RecommendationReason == nil,
		DefaultAnswer:          msg.DefaultAnswer == nil,
		Reversible:             msg.Reversible == nil,
		Urgency:                msg.Urgency == nil,
		ContextURL:             msg.ContextUrl == nil,
		BigPicture:             msg.BigPicture == nil,
		TechnicalDetails:       msg.TechnicalDetails == nil,
		ReassertWithin:         msg.ReassertWithinSeconds == nil,
		AcceptsContextRequests: msg.AcceptsContextRequests == nil,
		Packet:                 msg.Packet == nil,
	}
}
