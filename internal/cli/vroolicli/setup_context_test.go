package vroolicli

import (
	"context"
	"io"
	"testing"

	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

func TestTopLevelSetupAndDevelopForwardOperationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var setupContext, developContext context.Context
	app := New(Config{
		HomeDirFn: func() (string, error) { return t.TempDir(), nil },
		RunProjectSetupFn: func(_, _ string, opts projectsetup.Options, _, _ io.Writer) error {
			setupContext = opts.Context
			return nil
		},
		RunProjectDevelopFn: func(_, _ string, opts projectsetup.Options, _, _ io.Writer) error {
			developContext = opts.Context
			return nil
		},
	})
	command := &AppContext{Root: t.TempDir(), Context: ctx, Stdout: io.Discard, Stderr: io.Discard, app: app}
	if err := app.runTopLevelSetup(command, projectsetup.Options{}); err != nil {
		t.Fatal(err)
	}
	if err := app.runTopLevelDevelop(command, projectsetup.Options{}); err != nil {
		t.Fatal(err)
	}
	if setupContext != ctx || developContext != ctx {
		t.Fatalf("operation contexts setup=%v develop=%v, want same command context", setupContext, developContext)
	}
}
