package recoveryhandlers

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestCommandHandlersConstructServiceBeforeParsing(t *testing.T) {
	wantErr := errors.New("output format unavailable")
	deps := HandlerDeps[string]{
		Stdout: func(string) io.Writer { return &bytes.Buffer{} },
		Root:   func(string) string { return t.TempDir() },
		OutputFormat: func(string) (cliout.Format, error) {
			return "", wantErr
		},
		Clock: func() time.Time { return time.Time{} },
	}

	for name, handler := range commandtree.BuildHandlerMap(buildCommandTable(deps)) {
		t.Run(name, func(t *testing.T) {
			if err := handler("ctx", []string{"--invalid"}); !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want %v", err, wantErr)
			}
		})
	}
}
