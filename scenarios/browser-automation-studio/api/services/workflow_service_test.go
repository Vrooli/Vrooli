package services

import (
	"errors"
	"testing"
)

func TestExtractAIErrorMessage(t *testing.T) {
	msg := extractAIErrorMessage(map[string]interface{}{
		"error": "Cannot generate workflow using placeholder domain",
	})

	if msg != "Cannot generate workflow using placeholder domain" {
		t.Fatalf("expected message to be preserved, got %q", msg)
	}

	nested := extractAIErrorMessage(map[string]interface{}{
		"workflow": map[string]interface{}{
			"error": "Nested error detected",
		},
	})

	if nested != "Nested error detected" {
		t.Fatalf("expected nested message, got %q", nested)
	}
}

func TestNormalizeFlowDefinitionReturnsAIWorkflowError(t *testing.T) {
	definition := map[string]interface{}{
		"error": "Cannot generate workflow using placeholder domain",
	}

	_, err := normalizeFlowDefinition(definition)
	if err == nil {
		t.Fatalf("expected error for AI failure payload")
	}

	var aiErr *AIWorkflowError
	if !errors.As(err, &aiErr) {
		t.Fatalf("expected AIWorkflowError, got %T", err)
	}

	if aiErr.Reason == "" {
		t.Fatalf("expected AIWorkflowError to include a reason")
	}
}
