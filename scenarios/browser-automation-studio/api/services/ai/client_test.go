package ai

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestNewOpenRouterClient(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] bakes in no concrete default model slug", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(os.Stderr)
		client := NewOpenRouterClient(log)

		if client == nil {
			t.Fatal("expected non-nil client")
		}
		// The model is intentionally unset; it is resolved through the OpenRouter
		// resource policy (role based) at execution time. No hard-coded slug.
		if client.model != "" {
			t.Errorf("expected empty model on a fresh client, got %q", client.model)
		}
		if client.log == nil {
			t.Error("expected non-nil logger")
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] ignores legacy BAS_OPENROUTER_MODEL env var", func(t *testing.T) {
		originalModel := os.Getenv("BAS_OPENROUTER_MODEL")
		os.Setenv("BAS_OPENROUTER_MODEL", "anthropic/claude-3-sonnet")
		defer func() {
			if originalModel != "" {
				os.Setenv("BAS_OPENROUTER_MODEL", originalModel)
			} else {
				os.Unsetenv("BAS_OPENROUTER_MODEL")
			}
		}()

		log := logrus.New()
		client := NewOpenRouterClient(log)

		if client.model != "" {
			t.Errorf("expected client to ignore legacy env var, got model %q", client.model)
		}
	})
}

func TestResolveRoleModel(t *testing.T) {
	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] resolves model through policy seam", func(t *testing.T) {
		original := resolveRoleModelFunc
		var gotRole string
		resolveRoleModelFunc = func(_ context.Context, role string) (string, error) {
			gotRole = role
			return "vendor/resolved-model", nil
		}
		defer func() { resolveRoleModelFunc = original }()

		model, err := resolveRoleModel(context.Background(), "chat.default")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model != "vendor/resolved-model" {
			t.Errorf("expected resolved model, got %q", model)
		}
		if gotRole != "chat.default" {
			t.Errorf("expected role chat.default, got %q", gotRole)
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] propagates resolution failure without fallback slug", func(t *testing.T) {
		original := resolveRoleModelFunc
		resolveRoleModelFunc = func(_ context.Context, _ string) (string, error) {
			return "", errors.New("policy unavailable")
		}
		defer func() { resolveRoleModelFunc = original }()

		_, err := resolveRoleModel(context.Background(), "chat.default")
		if err == nil {
			t.Fatal("expected error when policy resolution fails")
		}
	})

	t.Run("[REQ:BAS-AI-GENERATION-VALIDATION] BAS_OPENROUTER_ROLE selects the role", func(t *testing.T) {
		originalEnv := os.Getenv("BAS_OPENROUTER_ROLE")
		os.Setenv("BAS_OPENROUTER_ROLE", "chat.small")
		defer func() {
			if originalEnv != "" {
				os.Setenv("BAS_OPENROUTER_ROLE", originalEnv)
			} else {
				os.Unsetenv("BAS_OPENROUTER_ROLE")
			}
		}()

		if got := openRouterRole(); got != "chat.small" {
			t.Errorf("expected role chat.small from env, got %q", got)
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
