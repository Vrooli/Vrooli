// Package auth implements the host-level "am I signed in to <tool>?"
// surface backing the `vrooli auth status` CLI command. It is a single
// extensible registry of probes that all share the SignInProbe contract.
//
// New tools (claude, codex, gh, cloudflared, stripe, ...) get added by
// implementing SignInProbe and registering an instance in DefaultProbes.
// The CLI surface is fixed; only the probe set grows.
package auth

import (
	"context"
	"sort"
)

// State is the externally visible sign-in state of one tool.
type State string

const (
	StateSignedIn  State = "signed_in"
	StateSignedOut State = "signed_out"
	StateExpired   State = "expired"
	StateUnknown   State = "unknown"
)

// ProbeResult is what a SignInProbe.Probe call returns.
type ProbeResult struct {
	State         State    `json:"state"`
	Detail        string   `json:"detail,omitempty"`
	SignInCommand []string `json:"sign_in_command,omitempty"`
}

// ProbeOptions controls optional behaviors common to all probes.
type ProbeOptions struct {
	// CheckExpiry asks the probe to perform an authenticated test call when
	// it is otherwise unable to distinguish "signed in" from "expired". The
	// flag is opt-in because the test calls usually generate network traffic
	// against the upstream service.
	CheckExpiry bool
}

// SignInProbe is the contract every tool's auth probe implements.
type SignInProbe interface {
	Name() string
	Probe(ctx context.Context, opts ProbeOptions) ProbeResult
}

// Status pairs a probe's name with its result. Used as the row shape in
// the renderer.
type Status struct {
	Name   string      `json:"name"`
	Result ProbeResult `json:"result"`
}

// Report is the full output shape for a `vrooli auth status` invocation.
type Report struct {
	Statuses []Status `json:"statuses"`
}

// Run probes every supplied probe in name order and returns the report.
// Probes are invoked sequentially; their per-probe context is shared (so
// callers can cancel the whole pass with a single ctx.Done()).
func Run(ctx context.Context, probes []SignInProbe, opts ProbeOptions) Report {
	sort.Slice(probes, func(i, j int) bool { return probes[i].Name() < probes[j].Name() })
	report := Report{Statuses: make([]Status, 0, len(probes))}
	for _, probe := range probes {
		result := probe.Probe(ctx, opts)
		report.Statuses = append(report.Statuses, Status{Name: probe.Name(), Result: result})
	}
	return report
}
