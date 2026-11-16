package ai

import (
	"context"
	"encoding/json"

	"image-tools/internal/models"
)

// allowModelRightsForHandlerTest keeps admission tests focused on their HTTP
// contracts. Model-rights behavior has separate registry and evaluator tests.
func allowModelRightsForHandlerTest(context.Context, json.RawMessage, string) (models.RightsDecision, error) {
	return models.RightsDecision{Status: "allowed", DecisionSHA256: "test"}, nil
}
