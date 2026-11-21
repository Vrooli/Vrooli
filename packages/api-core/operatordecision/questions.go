package operatordecision

import (
	"fmt"
	"strings"

	"github.com/vrooli/api-core/askreader"
)

// Question is one typed question of a decision's answer form, shown under the
// main options: the event form of an agent's `ask --form`. A page's Groups
// block fills a multi_choice question whose id it names, and the operator's
// answer comes back on DecisionResolution.Responses.
//
// Event decisions carry no attachments, so a question names no file: choice
// and text questions only.
type Question struct {
	ID       string
	Type     string // QuestionSingleChoice, QuestionMultiChoice or QuestionText
	Prompt   string
	Help     string
	Required bool
	Choices  []Choice
	// Recommended is the recommended choice key(s); RecommendedText the
	// recommended text of a text question.
	Recommended          []string
	RecommendedText      string
	RecommendationReason string
}

// Choice is one selectable answer of a choice question.
type Choice struct {
	Key    string
	Label  string
	Detail string
}

// Question types an event decision can carry.
const (
	QuestionSingleChoice = "single_choice"
	QuestionMultiChoice  = "multi_choice"
	QuestionText         = "text"
)

// Form limits, as the Hub enforces them.
const (
	MaxQuestions       = 12
	MaxQuestionChoices = 12
	MaxQuestionPrompt  = 500
)

// validateQuestions checks the form against the Hub's rules, so a decision
// this package accepts never loses a question at the Hub.
func validateQuestions(questions []Question) error {
	if len(questions) > MaxQuestions {
		return fmt.Errorf("a decision carries at most %d form questions", MaxQuestions)
	}
	ids := map[string]bool{"answer": true}
	for _, question := range questions {
		switch {
		case !factsKeyRule.MatchString(question.ID):
			return fmt.Errorf("form question id %q must be 1-64 letters, digits, '.', '_', ':' or '-'", question.ID)
		case ids[question.ID]:
			return fmt.Errorf("form question id %q is repeated or reserved", question.ID)
		case strings.TrimSpace(question.Prompt) == "" || len(question.Prompt) > MaxQuestionPrompt:
			return fmt.Errorf("form question %q needs a prompt of at most %d bytes", question.ID, MaxQuestionPrompt)
		}
		ids[question.ID] = true
		switch question.Type {
		case QuestionText:
			if len(question.Choices) > 0 || len(question.Recommended) > 0 {
				return fmt.Errorf("form question %q is text: it has no choices", question.ID)
			}
		case QuestionSingleChoice, QuestionMultiChoice:
			if len(question.Choices) < 2 || len(question.Choices) > MaxQuestionChoices {
				return fmt.Errorf("form question %q needs 2-%d choices", question.ID, MaxQuestionChoices)
			}
			keys := map[string]bool{}
			for _, choice := range question.Choices {
				if !factsKeyRule.MatchString(choice.Key) || keys[choice.Key] {
					return fmt.Errorf("form question %q has a missing, invalid or repeated choice key %q", question.ID, choice.Key)
				}
				keys[choice.Key] = true
			}
			if question.Type == QuestionSingleChoice && len(question.Recommended) > 1 {
				return fmt.Errorf("form question %q is single_choice: recommend one choice", question.ID)
			}
			for _, key := range question.Recommended {
				if !keys[key] {
					return fmt.Errorf("form question %q recommends %q, which is not one of its choices", question.ID, key)
				}
			}
		default:
			return fmt.Errorf("form question %q has type %q; an event decision carries single_choice, multi_choice or text", question.ID, question.Type)
		}
	}
	return nil
}

// questionsPayload is the form as the Hub's decision intake reads it.
func questionsPayload(questions []Question) []any {
	out := make([]any, 0, len(questions))
	for _, question := range questions {
		entry := map[string]any{"id": question.ID, "type": question.Type, "prompt": question.Prompt}
		if question.Help != "" {
			entry["help"] = question.Help
		}
		if question.Required {
			entry["required"] = true
		}
		if len(question.Choices) > 0 {
			choices := make([]any, 0, len(question.Choices))
			for _, choice := range question.Choices {
				item := map[string]any{"key": choice.Key, "label": choice.Label}
				if choice.Detail != "" {
					item["detail"] = choice.Detail
				}
				choices = append(choices, item)
			}
			entry["choices"] = choices
		}
		switch {
		case len(question.Recommended) > 0:
			// The payload travels as a protobuf Struct, which takes []any, never []string.
			selected := make([]any, 0, len(question.Recommended))
			for _, key := range question.Recommended {
				selected = append(selected, key)
			}
			entry["recommended"] = map[string]any{"question_id": question.ID, "selected": selected}
		case question.RecommendedText != "":
			entry["recommended"] = map[string]any{"question_id": question.ID, "text": question.RecommendedText}
		}
		if question.RecommendationReason != "" {
			entry["recommendation_reason"] = question.RecommendationReason
		}
		out = append(out, entry)
	}
	return out
}

// Response is the operator's answer to one form question, as the resolution
// and the Hub's record of the ask carry it (askreader.Response).
type Response = askreader.Response

// SectionVerdict is the operator's verdict on one section of a review.
type SectionVerdict = askreader.SectionVerdict

// Response returns the operator's answer to one form question, and whether
// they answered it.
func (r DecisionResolution) Response(questionID string) (Response, bool) {
	for _, response := range r.Responses {
		if response.QuestionID == questionID {
			return response, true
		}
	}
	return Response{}, false
}
