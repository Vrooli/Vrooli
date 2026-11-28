// Package gateway provides the `resource-pii-detector gateway ...` subcommand
// group — the canonical entrypoint scenarios use to talk to the shared PII
// detector service (never raw HTTP). The detector returns PII spans over a text
// input, merging a deterministic regex/checksum pass with an NER model pass.
package gateway

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/resources/pii-detector/cli/internal/client"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/cliout"
)

// Client is the upstream-facing surface used by the gateway handlers. It is
// satisfied by *client.Client in production and by fakes in tests.
type Client interface {
	Detect(ctx context.Context, text string) (client.DetectResult, error)
	Health(ctx context.Context) error
	Info(ctx context.Context) (map[string]any, error)
}

// Handlers owns the runtime dependencies for the gateway subcommand group.
type Handlers struct {
	NewClient func() Client
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
}

// Default returns Handlers wired to the real detector client.
func Default() *Handlers {
	return &Handlers{
		NewClient: func() Client { return client.NewClient() },
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
	}
}

// Commands returns the `gateway` subcommand group for registration.
func Commands(h *Handlers) cliapp.SubcommandGroup {
	if h == nil {
		h = Default()
	}
	return cliapp.SubcommandGroup{
		Name:        "gateway",
		Description: "Detect PII spans in text via the shared detector",
		Subcommands: []cliapp.Command{
			{
				Name:        "detect",
				Description: "Detect PII spans in --text (or --text-stdin)",
				Usage:       "resource-pii-detector gateway detect (--text <text> | --text-stdin) [--json]",
				Run:         h.Detect,
			},
			{
				Name:        "health",
				Description: "Probe the detector readiness endpoint (GET /health)",
				Usage:       "resource-pii-detector gateway health [--json]",
				Run:         h.Health,
			},
			{
				Name:        "info",
				Description: "Show the served model info (GET /info)",
				Usage:       "resource-pii-detector gateway info [--json]",
				Run:         h.Info,
			},
		},
	}
}

// --- detect -------------------------------------------------------------------

func (h *Handlers) Detect(args []string) error {
	fs := flag.NewFlagSet("gateway detect", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	text := fs.String("text", "", "Text to scan for PII")
	fromStdin := fs.Bool("text-stdin", false, "Read the text to scan from stdin")
	asJSON := fs.Bool("json", false, "Emit the detect result as a JSON object on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	input, err := h.resolveText(*text, *fromStdin)
	if err != nil {
		return err
	}
	result, err := h.NewClient().Detect(context.Background(), input)
	if err != nil {
		return err
	}
	if *asJSON {
		return cliout.NewEncoder(h.Stdout).Encode(result)
	}
	if !result.ModelAvailable {
		fmt.Fprintln(h.Stderr, "note: NER model unavailable; regex/checksum spans only")
	}
	for _, s := range result.Spans {
		if _, err := fmt.Fprintf(h.Stdout, "%s\t%d:%d\t%.3f\t%s\n", s.Category, s.Start, s.End, s.Confidence, truncate(s.Text, 80)); err != nil {
			return err
		}
	}
	return nil
}

// --- health -------------------------------------------------------------------

func (h *Handlers) Health(args []string) error {
	fs := flag.NewFlagSet("gateway health", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit {\"healthy\":bool} on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	err := h.NewClient().Health(context.Background())
	if *asJSON {
		_ = cliout.NewCompactEncoder(h.Stdout).Encode(struct {
			Healthy bool   `json:"healthy"`
			Error   string `json:"error,omitempty"`
		}{Healthy: err == nil, Error: errString(err)})
		return err
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(h.Stdout, "healthy")
	return err
}

// --- info ---------------------------------------------------------------------

func (h *Handlers) Info(args []string) error {
	fs := flag.NewFlagSet("gateway info", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	asJSON := fs.Bool("json", false, "Emit the raw /info JSON object on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	info, err := h.NewClient().Info(context.Background())
	if err != nil {
		return err
	}
	if *asJSON {
		return cliout.NewEncoder(h.Stdout).Encode(info)
	}
	for _, key := range []string{"model_id", "regex_pass", "categories"} {
		if v, ok := info[key]; ok {
			if _, err := fmt.Fprintf(h.Stdout, "%s: %v\n", key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- shared -------------------------------------------------------------------

func (h *Handlers) resolveText(inline string, fromStdin bool) (string, error) {
	if strings.TrimSpace(inline) != "" && fromStdin {
		return "", fmt.Errorf("--text and --text-stdin are mutually exclusive")
	}
	if fromStdin {
		buf, err := io.ReadAll(h.Stdin)
		if err != nil {
			return "", fmt.Errorf("read text from stdin: %w", err)
		}
		if strings.TrimSpace(string(buf)) == "" {
			return "", fmt.Errorf("no text read from stdin")
		}
		return string(buf), nil
	}
	if strings.TrimSpace(inline) == "" {
		return "", fmt.Errorf("--text or --text-stdin is required")
	}
	return inline, nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\t", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
