// Package cliruntime holds the binary-resolution and `--help`-tree primitives
// shared by cli-health's AI search index builder (internal/aisearch) and its
// manifest-validation runtime probe (internal/services/manifestvalidation).
//
// It is deliberately a leaf package: it imports nothing from aisearch or
// manifestvalidation, so both can depend inward on it without an import cycle.
// Its job is narrow — resolve a CLI binary, exec `<bin> [subcmd…] --help` with a
// timeout, and walk the resulting help tree into a flat list of leaf commands.
// Everything richer (embeddings, measures, findings) is layered on top by the
// consumers.
package cliruntime

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Source enumerates how a Command was discovered. Values match the string the
// consumers persist, so a converted record keeps a stable provenance label.
const (
	SourceHelp       = "help"
	SourceHelpFailed = "help-failed"
)

// DefaultHelpMaxDepth bounds the help-tree recursion. depth=1 walks
// `<bin> --help` only; depth=3 covers `vrooli scenario start`.
const DefaultHelpMaxDepth = 3

// defaultHelpTimeout bounds a single `--help` invocation when ExecRunner is
// constructed without an explicit timeout.
const defaultHelpTimeout = 5 * time.Second

// Command is one leaf of a CLI's `--help` tree. It carries only the fields the
// help walk can observe; consumers project it onto their own richer record.
type Command struct {
	Origin           string // binary/scenario name placed on every record
	Group            string // first path segment for depth>=2 leaves; empty otherwise
	Name             string // the leaf command name
	FullPath         string // "<origin> <group> <name>" canonical command
	Description      string
	GroupDescription string // nearest enclosing group's one-line summary
	Source           string // SourceHelp | SourceHelpFailed
}

// Runner invokes `<bin> args... --help` and returns combined stdout.
// seam: Runner — production wires ExecRunner (exec.CommandContext); tests inject
// a static map so the recursive parser can be exercised against captured help
// corpora without spawning processes.
type Runner func(ctx context.Context, bin string, args []string) ([]byte, error)

// ExecRunner returns the production Runner that shells out to the binary,
// applying timeout per invocation. A non-positive timeout defaults to 5s.
func ExecRunner(timeout time.Duration) Runner {
	if timeout <= 0 {
		timeout = defaultHelpTimeout
	}
	return func(ctx context.Context, bin string, args []string) ([]byte, error) {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		argv := append(append([]string{}, args...), "--help")
		cmd := exec.CommandContext(cctx, bin, argv...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			if msg := strings.TrimSpace(stderr.String()); msg != "" {
				return nil, &runErr{msg: msg, err: err}
			}
			return nil, err
		}
		return stdout.Bytes(), nil
	}
}

// runErr wraps a help invocation failure with the binary's stderr so callers
// surface the real diagnostic rather than a bare "exit status 1".
type runErr struct {
	msg string
	err error
}

func (e *runErr) Error() string { return e.msg + ": " + e.err.Error() }
func (e *runErr) Unwrap() error { return e.err }

// ResolveBinary resolves a CLI binary by name. When envVar is non-empty and that
// environment variable holds a path, it wins (an explicit override). Otherwise
// the canonical runtime-home install is preferred over PATH. This matters on
// developer machines that still have an older same-named CLI in ~/.local/bin:
// validation must exercise the artifact the Vrooli lifecycle installed, not an
// ambient legacy copy selected by a runner's PATH ordering. PATH remains the
// fallback for tools that are not Vrooli-installed. Returns "" when nothing
// resolves so callers can degrade gracefully rather than hard-fail.
func ResolveBinary(name, envVar string) string {
	if envVar != "" {
		if p := strings.TrimSpace(os.Getenv(envVar)); p != "" {
			return p
		}
	}
	name = strings.TrimSpace(name)
	if canonical := canonicalRuntimeHomeBinary(name); canonical != "" {
		return canonical
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func canonicalRuntimeHomeBinary(name string) string {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return ""
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		if path := executableFile(filepath.Join(home, ".vrooli", "bin", name)); path != "" {
			return path
		}
	}

	// Test-genie and other governed runners may sandbox HOME while preserving
	// the operator's canonical bin directory on PATH. Prefer that explicit
	// runtime-home entry over a stale same-named binary in another PATH entry.
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Base(filepath.Clean(dir)) != "bin" || filepath.Base(filepath.Dir(filepath.Clean(dir))) != ".vrooli" {
			continue
		}
		if path := executableFile(filepath.Join(dir, name)); path != "" {
			return path
		}
	}
	return ""
}

func executableFile(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return ""
	}
	return path
}

func truncateForEmbedding(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
