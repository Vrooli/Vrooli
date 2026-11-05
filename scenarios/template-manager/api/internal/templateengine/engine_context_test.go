package templateengine

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

func TestBindSubprocessContextUsesCallerContextWhenUnset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := bindSubprocessContext(ctx, shell.Spec{Name: "go"})
	if got.Context != ctx {
		t.Fatal("subprocess spec did not inherit the caller context")
	}
}

func TestBindSubprocessContextPreservesExplicitContext(t *testing.T) {
	callerCtx, cancelCaller := context.WithCancel(context.Background())
	defer cancelCaller()
	explicitCtx, cancelExplicit := context.WithCancel(context.Background())
	defer cancelExplicit()

	got := bindSubprocessContext(callerCtx, shell.Spec{Context: explicitCtx, Name: "go"})
	if got.Context != explicitCtx {
		t.Fatal("subprocess spec's explicit context was replaced")
	}
}

func TestRunTemplateValidateStopsWhenRequestContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := runTemplateValidate(HandlerDeps[context.Context]{}, ctx, templatecontracts.TemplateValidateRequest{
		Mode: templatecontracts.TemplateValidationModeShallow,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runTemplateValidate error = %v, want context.Canceled", err)
	}
}
