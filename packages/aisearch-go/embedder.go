package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// EmbedRunner runs the embedder subprocess. Injectable so tests substitute a
// fake without shelling out.
type EmbedRunner func(ctx context.Context, args []string, stdin []byte) ([]byte, error)

type cliEmbedder struct {
	bin   string
	model string
	role  string
	run   EmbedRunner
	// queryPrefix / docPrefix are the asymmetric task-instruction prefixes some
	// embedding models require (see TaskPrefixesFor). Empty means symmetric.
	queryPrefix string
	docPrefix   string
}

const defaultEmbedderBin = "resource-ollama"

// TaskPrefixesFor returns the (query, document) task-instruction prefixes an
// asymmetric embedding model is trained to expect, or ("","") for a symmetric
// model. nomic-embed-text — the substrate default — is the salient case: it was
// trained with "search_query:" on the query side and "search_document:" on the
// indexed-passage side, and retrieval quality drops markedly when neither is
// applied (it silently falls back to the model's clustering behavior). The drop
// is worst for a short, terse corpus matched against long natural-language
// queries (exactly cli-health's command index). Only models we can attribute
// with confidence get a prefix; everything else stays symmetric so we never
// mis-prefix an unknown model.
func TaskPrefixesFor(model string) (query, document string) {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.Contains(m, "nomic-embed"):
		return "search_query: ", "search_document: "
	default:
		return "", ""
	}
}

// NewEmbedder returns the production CLI-backed Embedder. It shells out to
// `resource-ollama gateway embed` (lifted verbatim from cli-health) and is
// SYMMETRIC (no task prefix) — the legacy behavior — so existing collections are
// never silently invalidated. Opt into asymmetric task prefixes via config
// (Config.EmbedTaskPrefix → NewEmbedderForConfig) or explicitly with
// NewEmbedderWithPrefixes; an adopter that flips it on must reindex (the
// recipe-aware drift hash detects the change and re-embeds automatically).
func NewEmbedder(model string) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	return &cliEmbedder{
		bin:   defaultEmbedderBin,
		model: model,
		role:  DefaultEmbedRole,
		run:   runLocalCLI,
	}
}

// NewEmbedderForConfig returns the embedder a Config asks for: prefixed
// (asymmetric, via TaskPrefixesFor) when cfg.EmbedTaskPrefix is set AND the
// model has known prefixes, symmetric otherwise. This is the constructor the
// engine assemblers (NewDenseEngine / NewHybridEngine) use so the prefix is a
// per-adopter, env-gated decision rather than a silent global default.
func NewEmbedderForConfig(cfg Config) Embedder {
	model := cfg.EmbedModel
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	role := strings.TrimSpace(cfg.EmbedRole)
	if role == "" {
		role = DefaultEmbedRole
	}
	if cfg.EmbedTaskPrefix {
		if qp, dp := TaskPrefixesFor(model); qp != "" || dp != "" {
			return newEmbedderWithRoleAndPrefixes(model, role, qp, dp, runLocalCLI)
		}
	}
	return newEmbedderWithRoleAndPrefixes(model, role, "", "", runLocalCLI)
}

// NewEmbedderWithPrefixes returns the production CLI-backed Embedder with the
// task prefixes supplied explicitly rather than derived from the model name.
// Pass ("","") to force symmetric embedding (e.g. to A/B the prefix effect, or
// for a model whose prefixes TaskPrefixesFor does not yet know).
func NewEmbedderWithPrefixes(model, queryPrefix, docPrefix string) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	return &cliEmbedder{
		bin:         defaultEmbedderBin,
		model:       model,
		role:        DefaultEmbedRole,
		run:         runLocalCLI,
		queryPrefix: queryPrefix,
		docPrefix:   docPrefix,
	}
}

// NewEmbedderWithRunner returns a SYMMETRIC Embedder with an injected runner
// (tests). Use NewEmbedderWithRunnerPrefixed to exercise the task-prefix path.
func NewEmbedderWithRunner(model string, run EmbedRunner) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	return &cliEmbedder{bin: defaultEmbedderBin, model: model, role: DefaultEmbedRole, run: run}
}

// NewEmbedderWithRunnerPrefixed returns an Embedder with an injected runner that
// applies the model's task prefixes (tests).
func NewEmbedderWithRunnerPrefixed(model string, run EmbedRunner) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	qp, dp := TaskPrefixesFor(model)
	return &cliEmbedder{bin: defaultEmbedderBin, model: model, role: DefaultEmbedRole, run: run, queryPrefix: qp, docPrefix: dp}
}

func newEmbedderWithRoleAndPrefixes(model, role, queryPrefix, docPrefix string, run EmbedRunner) Embedder {
	if strings.TrimSpace(model) == "" {
		model = DefaultEmbedModel
	}
	if strings.TrimSpace(role) == "" {
		role = DefaultEmbedRole
	}
	return &cliEmbedder{bin: defaultEmbedderBin, model: model, role: role, run: run, queryPrefix: queryPrefix, docPrefix: docPrefix}
}

// runLocalCLI is the shared default subprocess runner behind both the embedder
// (EmbedRunner) and the LLM reranker (GenerateRunner) — they shell out the same
// way (resource-ollama gateway {embed,generate}), so the plumbing lives here
// once. It runs args[0] with args[1:], pipes stdin when present, and surfaces
// stderr in the error so a failed gateway call is diagnosable.
func runLocalCLI(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	return stdout.Bytes(), nil
}

type cliEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed embeds text symmetrically (no task prefix). It is kept for the generic
// Embedder contract and the availability ping; the read path and reconciler use
// the role-aware EmbedQuery / EmbedDocument so an asymmetric model embeds in its
// trained role.
func (e *cliEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, "", text)
}

// EmbedQuery embeds text in the query role (TaskEmbedder).
func (e *cliEmbedder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, e.queryPrefix, text)
}

// EmbedDocument embeds text in the indexed-passage role (TaskEmbedder).
func (e *cliEmbedder) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, e.docPrefix, text)
}

func (e *cliEmbedder) embed(ctx context.Context, prefix, text string) ([]float64, error) {
	if e.run == nil {
		return nil, errors.New("embedder runner is not configured")
	}
	role := strings.TrimSpace(e.role)
	if role == "" {
		role = DefaultEmbedRole
	}
	args := []string{e.bin, "gateway", "embed", "--role", role, "--json", "--input-stdin"}
	out, err := e.run(ctx, args, []byte(prefix+text))
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway embed: %w", err)
	}
	var decoded cliEmbedResponse
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	if len(decoded.Embedding) == 0 {
		return nil, errors.New("embed response contained no vector")
	}
	return decoded.Embedding, nil
}

// EmbedRecipe reports the embedding recipe identity (RecipeEmbedder). It is
// empty for a symmetric (no-prefix) embedder so legacy collections keep their
// existing drift hashes; when task prefixes are active it folds the model + both
// prefixes in, so adding prefixes forces a one-time corpus re-embed.
func (e *cliEmbedder) EmbedRecipe() string {
	if e.queryPrefix == "" && e.docPrefix == "" {
		return ""
	}
	return "model=" + e.model + "|q=" + e.queryPrefix + "|d=" + e.docPrefix
}

func (e *cliEmbedder) Available(ctx context.Context) bool {
	if e.run == nil {
		return false
	}
	_, err := e.Embed(ctx, "ping")
	return err == nil
}
