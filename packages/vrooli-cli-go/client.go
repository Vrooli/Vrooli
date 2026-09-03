package vroolicli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/repo-contract-go/cliinvoke"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const defaultBinary = "vrooli"

// defaultTimeout bounds a single CLI operation when the caller's context has no
// deadline. `vrooli scenario list --json` walks the whole tree and can take
// ~10s on a full fleet, so the budget clears that with wide headroom — callers
// should rely on this default rather than passing their own.
const defaultTimeout = 30 * time.Second

// Runner is the process-execution seam for the Vrooli CLI client. It exposes
// the two execution modes the client needs: Run returns stdout only (for JSON
// commands, where stderr may carry non-JSON warnings), and RunCombined returns
// interleaved stdout+stderr (for human-readable text commands whose meaningful
// output may go to stderr). A single exec invocation cannot honestly produce
// both representations, so they are distinct methods rather than one with a
// flag.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	RunCombined(ctx context.Context, name string, args ...string) ([]byte, error)
}

// execRunner is the production Runner. It delegates resolution and execution
// to cli-invoke-go so the typed client shares the supervisors' invocation
// discipline (deadline, WaitDelay, failure classes) instead of carrying its
// own exec calls.
type execRunner struct{}

var _ Runner = execRunner{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	res := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: resolveBinary(name), Args: args, Timeout: timeoutFrom(ctx)})
	return res.Stdout, res.Error()
}

func (execRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	var combined bytes.Buffer
	res := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: resolveBinary(name), Args: args, Timeout: timeoutFrom(ctx), Stdout: &combined, Stderr: &combined})
	return combined.Bytes(), res.Error()
}

// resolveBinary turns the client's configured name into a path through the
// one invoker seam. A configured path is explicit; the default name resolves
// through VROOLI_BIN, the runtime home, then PATH. A miss returns the name so
// the exec error names what was looked for.
func resolveBinary(name string) string {
	explicit := ""
	if name != defaultBinary {
		explicit = name
	}
	home, _ := os.UserHomeDir()
	path, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{Explicit: explicit, RuntimeHome: home})
	if err != nil {
		return name
	}
	return path
}

// timeoutFrom keeps cli-invoke-go's default backstop unless the caller's
// context already carries a tighter deadline.
func timeoutFrom(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			return remaining
		}
	}
	return cliinvoke.DefaultTimeout
}

// Client wraps typed `vrooli ... --json` CLI contracts.
type Client struct {
	bin            string
	runner         Runner
	defaultTimeout time.Duration
}

// Option customizes a Client.
type Option func(*Client)

// New returns a Vrooli CLI client with production defaults: a 30s per-operation
// timeout and no global flags on the argv (a programmatic caller runs the
// installed binary as-is; freshness is the lifecycle engine's concern).
func New(opts ...Option) *Client {
	c := &Client{
		bin:            defaultBinary,
		runner:         execRunner{},
		defaultTimeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithRunner replaces the command runner. It is primarily used by tests.
func WithRunner(runner Runner) Option {
	return func(c *Client) {
		if runner != nil {
			c.runner = runner
		}
	}
}

// WithTimeout overrides the default per-operation timeout applied when the
// caller's context has no deadline. Most callers should not need this — the
// default is sized for a full-fleet sweep.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.defaultTimeout = timeout
	}
}

// WithBinary sets the executable name or path used for the Vrooli CLI.
func WithBinary(bin string) Option {
	return func(c *Client) {
		if strings.TrimSpace(bin) != "" {
			c.bin = bin
		}
	}
}

var cliJSON = protojson.UnmarshalOptions{DiscardUnknown: true}

// Output runs a raw Vrooli CLI command and returns stdout without injecting
// --json. The default timeout applies when ctx has no deadline. Use it for
// fire-and-forget commands and for commands whose machine output is on stdout.
func (c *Client) Output(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.run(ctx, args...)
}

// OutputCombined runs a raw Vrooli CLI command and returns combined
// stdout+stderr. Use it for human-readable text commands where the CLI may
// write meaningful content (e.g. "no lifecycle log found") to stderr that
// should be surfaced as output rather than swallowed as an error.
func (c *Client) OutputCombined(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.runCombined(ctx, args...)
}

// withTimeout derives a deadline-bound context once per operation. Applying it
// at the operation boundary (rather than per CLI invocation) keeps a fallback
// sequence within a single budget instead of granting each attempt a fresh one.
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); !ok && c.defaultTimeout > 0 {
		return context.WithTimeout(ctx, c.defaultTimeout)
	}
	return ctx, func() {}
}

// run executes one CLI invocation under the already-bounded ctx.
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	out, err := c.runner.Run(ctx, c.bin, args...)
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", formatCommand(c.bin, args), err)
	}
	return out, nil
}

// runCombined executes one CLI invocation returning combined stdout+stderr. It
// prefers a CombinedRunner; runners that do not implement it fall back to
// stdout-only Run.
func (c *Client) runCombined(ctx context.Context, args ...string) ([]byte, error) {
	out, err := c.runner.RunCombined(ctx, c.bin, args...)
	if err != nil {
		// Return the captured output alongside the error (matching
		// exec.CombinedOutput) so callers can surface what the command printed
		// to stderr — e.g. "no lifecycle log found" — as content.
		return out, fmt.Errorf("run %s: %w", formatCommand(c.bin, args), err)
	}
	return out, nil
}

func decode[T proto.Message](out []byte, msg T) (T, error) {
	if err := cliJSON.Unmarshal(out, msg); err != nil {
		return msg, fmt.Errorf("decode %T: %w", msg, err)
	}
	return msg, nil
}

func formatCommand(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
