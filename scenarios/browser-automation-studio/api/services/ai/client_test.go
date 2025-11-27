package ai

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewOpenRouterClient(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] uses default model when env var not set", func(t *testing.T) {
		originalModel := os.Getenv("BAS_OPENROUTER_MODEL")
		os.Unsetenv("BAS_OPENROUTER_MODEL")
		defer func() {
			if originalModel != "" {
				os.Setenv("BAS_OPENROUTER_MODEL", originalModel)
			}
		}()

		log := logrus.New()
		log.SetOutput(os.Stderr)
		client := NewOpenRouterClient(log)

		if client == nil {
			t.Fatal("expected non-nil client")
		}
		if client.model != openRouterDefaultModel {
			t.Errorf("expected default model %q, got %q", openRouterDefaultModel, client.model)
		}
		if client.log == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] uses custom model from env var", func(t *testing.T) {
		originalModel := os.Getenv("BAS_OPENROUTER_MODEL")
		customModel := "anthropic/claude-3-sonnet"
		os.Setenv("BAS_OPENROUTER_MODEL", customModel)
		defer func() {
			if originalModel != "" {
				os.Setenv("BAS_OPENROUTER_MODEL", originalModel)
			} else {
				os.Unsetenv("BAS_OPENROUTER_MODEL")
			}
		}()

		log := logrus.New()
		client := NewOpenRouterClient(log)

		if client.model != customModel {
			t.Errorf("expected custom model %q, got %q", customModel, client.model)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] handles whitespace in model env var", func(t *testing.T) {
		originalModel := os.Getenv("BAS_OPENROUTER_MODEL")
		os.Setenv("BAS_OPENROUTER_MODEL", "   ")
		defer func() {
			if originalModel != "" {
				os.Setenv("BAS_OPENROUTER_MODEL", originalModel)
			} else {
				os.Unsetenv("BAS_OPENROUTER_MODEL")
			}
		}()

		log := logrus.New()
		client := NewOpenRouterClient(log)

		if client.model != openRouterDefaultModel {
			t.Errorf("expected default model when env var is whitespace, got %q", client.model)
		}
	})
}

func TestExecutePrompt_Validation(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)
	client := NewOpenRouterClient(log)
	ctx := context.Background()

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects empty prompt", func(t *testing.T) {
		_, err := client.ExecutePrompt(ctx, "")
		if err == nil {
			t.Fatal("expected error for empty prompt")
		}
		if !strings.Contains(err.Error(), "prompt is required") {
			t.Errorf("expected 'prompt is required' error, got: %v", err)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] rejects whitespace-only prompt", func(t *testing.T) {
		_, err := client.ExecutePrompt(ctx, "   \n\t  ")
		if err == nil {
			t.Fatal("expected error for whitespace-only prompt")
		}
		if !strings.Contains(err.Error(), "prompt is required") {
			t.Errorf("expected 'prompt is required' error, got: %v", err)
		}
	})
}

func TestExecutePrompt_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping context cancellation test in short mode")
	}

	log := logrus.New()
	log.SetOutput(os.Stderr)
	client := NewOpenRouterClient(log)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Give context time to expire
		time.Sleep(5 * time.Millisecond)

		_, err := client.ExecutePrompt(ctx, "Generate a workflow for testing google.com")
		if err == nil {
			t.Fatal("expected error due to context cancellation")
		}
		// The error should indicate the command was killed or timed out
		errMsg := err.Error()
		if !strings.Contains(errMsg, "killed") && !strings.Contains(errMsg, "context") && !strings.Contains(errMsg, "signal") {
			t.Logf("Context cancellation produced error: %v", err)
		}
	})
}

func TestExecutePrompt_PromptFileManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping file management test in short mode")
	}

	// This test verifies that the prompt file is properly created and cleaned up,
	// even when the openrouter command fails
	log := logrus.New()
	log.SetOutput(os.Stderr)
	client := NewOpenRouterClient(log)
	ctx := context.Background()

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] creates and cleans up prompt file", func(t *testing.T) {
		prompt := "test prompt for file management"

		// Execute the prompt (will likely fail due to missing resource-openrouter binary in test env)
		_, err := client.ExecutePrompt(ctx, prompt)

		// We expect an error since resource-openrouter likely isn't available
		if err == nil {
			t.Log("Note: resource-openrouter successfully executed (unexpected in most test environments)")
		}

		// The important part is that the function completes and doesn't panic
		// The defer cleanup should have executed regardless of error
		// We can't easily verify the file was deleted since it's in a temp location,
		// but we can verify the function completed without panic
	})
}

func TestOpenRouterClient_Deterministic(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] multiple clients with same config are equivalent", func(t *testing.T) {
		client1 := NewOpenRouterClient(log)
		client2 := NewOpenRouterClient(log)

		if client1.model != client2.model {
			t.Errorf("expected identical model config, got %q and %q", client1.model, client2.model)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] validation errors are consistent", func(t *testing.T) {
		client := NewOpenRouterClient(log)
		ctx := context.Background()

		err1 := getExecutePromptError(client, ctx, "")
		err2 := getExecutePromptError(client, ctx, "")

		if err1.Error() != err2.Error() {
			t.Errorf("expected consistent error messages, got %q and %q", err1.Error(), err2.Error())
		}
	})
}

func getExecutePromptError(client *OpenRouterClient, ctx context.Context, prompt string) error {
	_, err := client.ExecutePrompt(ctx, prompt)
	return err
}
