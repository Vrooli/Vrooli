// Package gateway provides the `resource-reranker gateway ...` subcommand group.
//
// It is the canonical entrypoint scenarios use to talk to the shared reranker
// (TEI) container — never raw HTTP. The reranker scores a query against a set of
// candidate passages and returns them ordered by relevance; it is the
// second-stage reranker for aisearch-go / search-hub hybrid retrieval.
package gateway

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"resource-reranker/cli/internal/client"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Client is the upstream-facing surface used by the gateway handlers. It is
// satisfied by *client.Client in production and by fakes in tests.
type Client interface {
	Rerank(ctx context.Context, query string, documents []string, returnText bool) ([]client.RankResult, error)
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

// Default returns Handlers wired to the real TEI client.
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
		Description: "Score query/passage relevance against the shared reranker",
		Subcommands: []cliapp.Command{
			{
				Name:        "rerank",
				Description: "Rank --document passages by relevance to --query (cross-encoder scoring)",
				Usage:       "resource-reranker gateway rerank --query <text> (--document <text> ... | --documents-stdin) [--top-k N] [--return-text] [--json]",
				Run:         h.Rerank,
			},
			{
				Name:        "health",
				Description: "Probe the reranker readiness endpoint (GET /health)",
				Usage:       "resource-reranker gateway health [--json]",
				Run:         h.Health,
			},
			{
				Name:        "info",
				Description: "Show the served model info (GET /info)",
				Usage:       "resource-reranker gateway info [--json]",
				Run:         h.Info,
			},
		},
	}
}

type docList []string

func (d *docList) String() string { return strings.Join(*d, ", ") }

func (d *docList) Set(v string) error {
	*d = append(*d, v)
	return nil
}

// --- rerank -------------------------------------------------------------------

func (h *Handlers) Rerank(args []string) error {
	fs := flag.NewFlagSet("gateway rerank", flag.ContinueOnError)
	fs.SetOutput(h.Stderr)
	query := fs.String("query", "", "Query text to score passages against")
	var docs docList
	fs.Var(&docs, "document", "A candidate passage; repeat the flag for multiple")
	fromStdin := fs.Bool("documents-stdin", false, "Read passages from stdin, one per line")
	topK := fs.Int("top-k", 0, "Return only the top K results; 0 = all")
	returnText := fs.Bool("return-text", false, "Include each passage's text in the output")
	asJSON := fs.Bool("json", false, "Emit the ranked results as a JSON array on stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*query) == "" {
		return fmt.Errorf("--query is required")
	}
	documents, err := h.resolveDocuments(docs, *fromStdin)
	if err != nil {
		return err
	}

	results, err := h.NewClient().Rerank(context.Background(), *query, documents, *returnText || !*asJSON)
	if err != nil {
		return err
	}
	if *topK > 0 && *topK < len(results) {
		results = results[:*topK]
	}

	if *asJSON {
		if !*returnText {
			for i := range results {
				results[i].Text = ""
			}
		}
		enc := json.NewEncoder(h.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	for rank, r := range results {
		text := r.Text
		if text == "" && r.Index >= 0 && r.Index < len(documents) {
			text = documents[r.Index]
		}
		if _, err := fmt.Fprintf(h.Stdout, "%d\t%.6f\t[doc %d]\t%s\n", rank+1, r.Score, r.Index, truncate(text, 100)); err != nil {
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
		_ = json.NewEncoder(h.Stdout).Encode(struct {
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
		enc := json.NewEncoder(h.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}
	for _, key := range []string{"model_id", "model_type", "model_dtype", "version"} {
		if v, ok := info[key]; ok {
			if _, err := fmt.Fprintf(h.Stdout, "%s: %v\n", key, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- shared -------------------------------------------------------------------

func (h *Handlers) resolveDocuments(inline docList, fromStdin bool) ([]string, error) {
	if len(inline) > 0 && fromStdin {
		return nil, fmt.Errorf("--document and --documents-stdin are mutually exclusive")
	}
	if fromStdin {
		buf, err := io.ReadAll(h.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read documents from stdin: %w", err)
		}
		var out []string
		for _, line := range strings.Split(string(buf), "\n") {
			if s := strings.TrimSpace(line); s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("no documents read from stdin")
		}
		return out, nil
	}
	if len(inline) == 0 {
		return nil, fmt.Errorf("at least one --document or --documents-stdin is required")
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
