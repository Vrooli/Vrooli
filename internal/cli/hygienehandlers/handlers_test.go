package hygienehandlers

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/cliout"
)

func TestHandlerConstructsServiceBeforeParsing(t *testing.T) {
	wantErr := errors.New("output format unavailable")
	handler := Handler(HandlerDeps[string]{
		Stdout: func(string) io.Writer { return &bytes.Buffer{} },
		OutputFormat: func(string) (cliout.Format, error) {
			return "", wantErr
		},
		Home: func(string) (string, error) {
			t.Fatal("home lookup ran after output format failed")
			return "", nil
		},
	})

	if err := handler("ctx", []string{"--invalid"}); !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}
