// Package controlplane is the single audio-tools seam for invoking the Vrooli
// control plane. It resolves the installed binary once and keeps callers from
// duplicating PATH lookup, process execution, and missing-binary behavior.
package controlplane

import (
	"context"
	"fmt"
	"os/exec"
)

// Client invokes the installed vrooli control-plane binary.
type Client struct {
	bin string
	run func(context.Context, string, ...string) ([]byte, error)
}

// New resolves vrooli once at process startup. A missing binary is retained as
// an unavailable client so capability probes can degrade honestly.
func New() *Client {
	bin, _ := exec.LookPath("vrooli")
	return &Client{bin: bin}
}

// NewForTest builds an injectable client without performing PATH discovery.
func NewForTest(bin string, run func(context.Context, string, ...string) ([]byte, error)) *Client {
	return &Client{bin: bin, run: run}
}

// Available reports whether the control-plane binary was found.
func (c *Client) Available() bool { return c != nil && c.bin != "" }

// Run invokes vrooli with bounded context ownership supplied by the caller.
func (c *Client) Run(ctx context.Context, args ...string) ([]byte, error) {
	if c == nil || c.bin == "" {
		return nil, fmt.Errorf("vrooli binary not found on PATH")
	}
	if c.run != nil {
		return c.run(ctx, c.bin, args...)
	}
	return exec.CommandContext(ctx, c.bin, args...).Output()
}

// Command returns a configured command for callers that must stream stdout
// instead of buffering it. Binary discovery remains owned by this package.
func (c *Client) Command(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if c == nil || c.bin == "" {
		return nil, fmt.Errorf("vrooli binary not found on PATH")
	}
	return exec.CommandContext(ctx, c.bin, args...), nil
}
