package classification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const decisionRole = "decide.memory-observation"

type DecideClient interface {
	DecideBatch(context.Context, *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error)
}

func classify(ctx context.Context, client DecideClient, category Category, items []observation) ([]decision, error) {
	answers := make([]decision, len(items))
	var states []*inferencev1.DecideBatchItem
	var indexes []int
	for i, item := range items {
		if matchesSource(category.Sources, item) {
			answers[i] = decision{match: true, method: "ledger-source-rule"}
			continue
		}
		states = append(states, &inferencev1.DecideBatchItem{StateJson: stateJSON(item)})
		indexes = append(indexes, i)
	}
	if len(states) == 0 {
		return answers, nil
	}
	response, err := client.DecideBatch(ctx, connect.NewRequest(&inferencev1.DecideBatchRequest{
		Role: decisionRole,
		Questions: []*inferencev1.DecisionQuestion{{
			Id:           category.ID,
			Type:         inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE,
			Instructions: instructions(category),
			Options: []*inferencev1.DecisionOption{
				{Label: "matches", Description: "The observation fits the category definition."},
				{Label: "unrelated", Description: "The observation does not fit the category definition."},
			},
		}},
		Items:             states,
		Profile:           sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:      sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacyProvenance: "source_ledger.journal_observation",
	}))
	var routeErr error
	switch {
	case err != nil:
		routeErr = fmt.Errorf("AI Gateway DecideBatch: %s", connect.CodeOf(err))
	case response == nil || len(response.Msg.GetResults()) != len(states):
		routeErr = errors.New("AI Gateway returned an incomplete classification batch")
	}
	if routeErr != nil {
		// Leave the cursor in place on a wholly failed model batch. When
		// typed source rules resolved some items, publish those decisions and
		// durably defer the items that still need the model.
		if len(indexes) == len(answers) {
			return nil, routeErr
		}
		for _, index := range indexes {
			answers[index].failure = routeErr.Error()
		}
		return answers, routeErr
	}
	for i, result := range response.Msg.GetResults() {
		answers[indexes[i]] = decodeDecision(category.ID, result)
	}
	return answers, nil
}

func matchesSource(rules []SourceRule, item observation) bool {
	for _, rule := range rules {
		if item.kind == rule.Kind && strings.HasPrefix(item.importKey, "request:"+rule.RequestKeyPrefix) {
			return true
		}
	}
	return false
}

func decodeDecision(categoryID string, result *inferencev1.DecideResponse) decision {
	if failure := result.GetError(); failure != nil && failure.GetCode() != inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNSPECIFIED {
		return decision{failure: failure.GetCode().String()}
	}
	answers := result.GetAnswers()
	if len(answers) != 1 || answers[0].GetQuestionId() != categoryID ||
		answers[0].GetType() != inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE ||
		result.GetMethod() == inferencev1.DecisionMethod_DECISION_METHOD_UNSPECIFIED {
		return decision{failure: "invalid model answer"}
	}
	switch answers[0].GetChoice() {
	case "matches":
		return decision{match: true, method: result.GetMethod().String(), provider: result.GetProvider(), model: result.GetModel()}
	case "unrelated":
		return decision{}
	default:
		return decision{failure: "invalid model choice"}
	}
}

func stateJSON(item observation) string {
	body, _ := json.Marshal(struct {
		Observation string `json:"observation"`
		Kind        string `json:"kind,omitempty"`
		Evidence    string `json:"evidence,omitempty"`
		RoutingHint string `json:"routing_hint,omitempty"`
	}{
		Observation: bound(item.body, 2200),
		Kind:        bound(item.kind, 100),
		Evidence:    bound(item.evidence, 400),
		RoutingHint: bound(item.hint, 200),
	})
	return string(body)
}

func bound(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	const marker = " … "
	half := (limit - len(marker)) / 2
	return strings.ToValidUTF8(text[:half], "") + marker + strings.ToValidUTF8(text[len(text)-half:], "")
}
