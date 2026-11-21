package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/pii-detector/cli/internal/client"
)

type fakeClient struct {
	result    client.DetectResult
	detectErr error
	healthErr error
	info      map[string]any
	infoErr   error

	gotText string
}

func (f *fakeClient) Detect(_ context.Context, text string) (client.DetectResult, error) {
	f.gotText = text
	return f.result, f.detectErr
}
func (f *fakeClient) Health(context.Context) error                 { return f.healthErr }
func (f *fakeClient) Info(context.Context) (map[string]any, error) { return f.info, f.infoErr }

func newHandlers(f *fakeClient, stdin string) (*Handlers, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return &Handlers{
		NewClient: func() Client { return f },
		Stdin:     strings.NewReader(stdin),
		Stdout:    &out,
		Stderr:    &errBuf,
	}, &out, &errBuf
}

func TestDetectJSONOutput(t *testing.T) {
	f := &fakeClient{result: client.DetectResult{
		ModelAvailable: true,
		ModelID:        "gravitee-io/bert-small-pii-detection",
		Spans: []client.Span{
			{Category: "EMAIL_ADDRESS", Start: 0, End: 5, Text: "a@b.co", Confidence: 0.99, Source: "regex"},
		},
	}}
	h, out, _ := newHandlers(f, "")
	if err := h.Detect([]string{"--text", "a@b.co here", "--json"}); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if f.gotText != "a@b.co here" {
		t.Errorf("client got text=%q", f.gotText)
	}
	var decoded client.DetectResult
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output not valid json: %v\n%s", err, out.String())
	}
	if len(decoded.Spans) != 1 || decoded.Spans[0].Category != "EMAIL_ADDRESS" {
		t.Errorf("unexpected decoded spans: %+v", decoded.Spans)
	}
}

func TestDetectHumanOutputWarnsOnModelUnavailable(t *testing.T) {
	f := &fakeClient{result: client.DetectResult{ModelAvailable: false, Spans: []client.Span{
		{Category: "US_SSN", Start: 4, End: 15, Text: "123-45-6789", Confidence: 0.95, Source: "regex"},
	}}}
	h, out, errBuf := newHandlers(f, "")
	if err := h.Detect([]string{"--text", "ssn 123-45-6789"}); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if !strings.Contains(out.String(), "US_SSN") {
		t.Errorf("human output missing span: %s", out.String())
	}
	if !strings.Contains(errBuf.String(), "model unavailable") {
		t.Errorf("expected model-unavailable note on stderr, got %s", errBuf.String())
	}
}

func TestDetectStdin(t *testing.T) {
	f := &fakeClient{result: client.DetectResult{ModelAvailable: true}}
	h, _, _ := newHandlers(f, "scan this text\n")
	if err := h.Detect([]string{"--text-stdin"}); err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if f.gotText != "scan this text\n" {
		t.Errorf("stdin text not forwarded: %q", f.gotText)
	}
}

func TestDetectValidation(t *testing.T) {
	f := &fakeClient{}
	h, _, _ := newHandlers(f, "")
	if err := h.Detect([]string{}); err == nil {
		t.Error("expected error when no --text/--text-stdin")
	}
	if err := h.Detect([]string{"--text", "x", "--text-stdin"}); err == nil {
		t.Error("expected error for mutually exclusive sources")
	}
}

func TestHealthJSON(t *testing.T) {
	h, out, _ := newHandlers(&fakeClient{}, "")
	if err := h.Health([]string{"--json"}); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if !strings.Contains(out.String(), `"healthy":true`) {
		t.Errorf("health json wrong: %s", out.String())
	}
}

func TestInfoHumanFiltersKeys(t *testing.T) {
	f := &fakeClient{info: map[string]any{"model_id": "test/model", "regex_pass": true, "noise": "x"}}
	h, out, _ := newHandlers(f, "")
	if err := h.Info(nil); err != nil {
		t.Fatalf("Info() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "test/model") || strings.Contains(got, "noise") {
		t.Errorf("info output wrong: %s", got)
	}
}

func TestCommandsRegistersSubcommands(t *testing.T) {
	g := Commands(nil)
	if g.Name != "gateway" {
		t.Fatalf("group name = %q", g.Name)
	}
	want := map[string]bool{"detect": false, "health": false, "info": false}
	for _, c := range g.Subcommands {
		if _, ok := want[c.Name]; ok {
			want[c.Name] = true
		}
		if c.Run == nil {
			t.Errorf("subcommand %q has nil Run", c.Name)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}
