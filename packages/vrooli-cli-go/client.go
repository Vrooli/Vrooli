package vroolicli

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const defaultBinary = "vrooli"

// defaultTimeout bounds a single CLI operation when the caller's context has no
// deadline. `vrooli scenario list --json` walks the whole tree and can take
// ~10s on a full fleet, so the budget clears that with wide headroom — callers
// should rely on this default rather than passing their own.
const defaultTimeout = 30 * time.Second

// Runner is the process-execution seam for the Vrooli CLI client.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

var _ Runner = execRunner{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Client wraps typed `vrooli ... --json` CLI contracts.
type Client struct {
	bin            string
	runner         Runner
	defaultTimeout time.Duration
	staleCheck     bool
}

// Option customizes a Client.
type Option func(*Client)

// New returns a Vrooli CLI client with production defaults: a 30s per-operation
// timeout and the CLI's stale-binary check disabled (a programmatic caller runs
// the installed binary as-is; freshness is a build concern, and the check walks
// the repo tree on every invocation).
func New(opts ...Option) *Client {
	c := &Client{
		bin:            defaultBinary,
		runner:         execRunner{},
		defaultTimeout: defaultTimeout,
		staleCheck:     false,
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

// WithStaleCheck re-enables the CLI's stale-binary check (disabled by default).
// Pass true only when the caller genuinely wants to be gated on binary freshness.
func WithStaleCheck(enabled bool) Option {
	return func(c *Client) {
		c.staleCheck = enabled
	}
}

var cliJSON = protojson.UnmarshalOptions{DiscardUnknown: true}

// Output runs a raw Vrooli CLI command and returns stdout without injecting
// --json. The default timeout applies when ctx has no deadline.
func (c *Client) Output(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	return c.run(ctx, args...)
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

// run executes one CLI invocation under the already-bounded ctx. It prepends
// --no-stale-check unless the caller opted back into the stale check.
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	if !c.staleCheck {
		args = append([]string{"--no-stale-check"}, args...)
	}
	out, err := c.runner.Run(ctx, c.bin, args...)
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", formatCommand(c.bin, args), err)
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
