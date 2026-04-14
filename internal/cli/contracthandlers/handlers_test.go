package contracthandlers

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cliout"
)

type testContext struct {
	stdout *bytes.Buffer
}

func TestRootHandlerRendersHelpWithNoArgs(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
		Service: func(testContext) contractapp.Service { return contractapp.Service{} },
	})

	if err := handler(ctx, nil); err != nil {
		t.Fatalf("RootHandler() error = %v", err)
	}
	if got := ctx.stdout.String(); !strings.Contains(got, "vrooli contract") {
		t.Fatalf("RootHandler() help missing contract usage: %q", got)
	}
}

func TestValidateHandlerReturnsExitCodeOnFailedValidation(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}}
	handler := validateHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatJSON, nil
		},
		Service: func(testContext) contractapp.Service {
			return contractapp.Service{
				ResolveRootFn: func() (string, error) { return "/repo", nil },
				ValidateFn: func(string) (contractapp.ValidationOutput, error) {
					return contractapp.ValidationOutput{
						Success: false,
						Root:    "/repo",
						Schema:  contractapp.ValidationCheck{Passed: false, Message: "failed"},
					}, nil
				},
			}
		},
	})

	err := handler(ctx, nil)
	if err == nil {
		t.Fatal("validateHandler() expected exit-code error")
	}
	var codeErr interface{ ExitCode() int }
	if !errors.As(err, &codeErr) {
		t.Fatalf("validateHandler() error does not expose ExitCode: %T", err)
	}
	if codeErr.ExitCode() != 1 {
		t.Fatalf("validateHandler() exit code = %d, want 1", codeErr.ExitCode())
	}
}
