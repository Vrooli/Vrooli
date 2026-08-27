package contracthandlers

import (
	"bytes"
	"errors"
	"io"
	"testing"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cli/rootcli/rootclitest"
	"github.com/vrooli/vrooli/internal/cliout"
)

type testContext struct {
	stdout *bytes.Buffer
}

func TestConformance(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}}
	handler := RootHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatHuman, nil
		},
		Service: func(testContext) contractapp.Service { return contractapp.Service{} },
	})

	rootclitest.AssertHelpWithNoArgs(t, func() error { return handler(ctx, nil) }, ctx.stdout, "vrooli contract")
}

func TestValidateHandlerReturnsExitCodeOnFailedValidation(t *testing.T) {
	ctx := testContext{stdout: &bytes.Buffer{}}
	handler := validateHandler(HandlerDeps[testContext]{
		Stdout: func(ctx testContext) io.Writer { return ctx.stdout },
		OutputFormat: func(testContext) (cliout.Format, error) {
			return cliout.FormatJSON, nil
		},
		Service: func(testContext) contractapp.Service {
			return contractapp.Service{}
		},
		Validate: func(testContext) (contractapp.ValidationOutput, error) {
			return contractapp.ValidationOutput{
				Success: false,
				Root:    "/repo",
				Schema:  contractapp.ValidationCheck{Passed: false, Message: "failed"},
			}, nil
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
