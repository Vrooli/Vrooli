package auth

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// bufNetrcMatch matches `machine buf.build` lines (with optional whitespace
// and a word boundary). Buf v1.37 stores the token as a `machine buf.build`
// entry in $HOME/.netrc; presence of this line is the deterministic probe.
var bufNetrcMatch = regexp.MustCompile(`(?m)^\s*machine\s+buf\.build\b`)

// BufProbe implements the BSR sign-in probe. See
// docs/configuration/integrations/buf-bsr.md for the contract.
type BufProbe struct {
	// HomeDir overrides $HOME for tests. Production code leaves it empty
	// and falls back to os.UserHomeDir().
	HomeDir func() (string, error)
	// ExpiryProbe is the optional authenticated test call used when
	// ProbeOptions.CheckExpiry is true. Returning a non-nil error is
	// interpreted as "token expired or revoked"; nil means "still valid".
	// Production wires this to a `buf curl --schema buf.build/...` call;
	// tests can stub it.
	ExpiryProbe func(ctx context.Context) error
}

func (p BufProbe) Name() string { return "buf" }

func (p BufProbe) Probe(ctx context.Context, opts ProbeOptions) ProbeResult {
	signIn := []string{"buf", "registry", "login"}
	home, err := p.resolveHome()
	if err != nil {
		return ProbeResult{
			State:         StateUnknown,
			Detail:        "could not resolve home dir: " + err.Error(),
			SignInCommand: signIn,
		}
	}
	netrcPath := filepath.Join(home, ".netrc")
	data, err := os.ReadFile(netrcPath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ProbeResult{
			State:         StateSignedOut,
			Detail:        "no ~/.netrc file",
			SignInCommand: signIn,
		}
	case err != nil:
		return ProbeResult{
			State:         StateUnknown,
			Detail:        "read ~/.netrc: " + err.Error(),
			SignInCommand: signIn,
		}
	}
	if !bufNetrcMatch.Match(data) {
		return ProbeResult{
			State:         StateSignedOut,
			Detail:        "no `machine buf.build` line in ~/.netrc",
			SignInCommand: signIn,
		}
	}
	detail := "token in ~/.netrc; expiry not checked"
	if opts.CheckExpiry {
		if p.ExpiryProbe == nil {
			detail = "token in ~/.netrc; --check-expiry requested but no expiry probe configured"
		} else if err := p.ExpiryProbe(ctx); err != nil {
			return ProbeResult{
				State:         StateExpired,
				Detail:        "authenticated BSR call failed: " + strings.TrimSpace(err.Error()),
				SignInCommand: signIn,
			}
		} else {
			detail = "token in ~/.netrc; expiry probe returned 2xx"
		}
	}
	return ProbeResult{State: StateSignedIn, Detail: detail, SignInCommand: signIn}
}

func (p BufProbe) resolveHome() (string, error) {
	if p.HomeDir != nil {
		return p.HomeDir()
	}
	return os.UserHomeDir()
}

// DefaultProbes returns the canonical probe set. Today this is just buf;
// future probes (claude, codex, gh, cloudflared, stripe) plug in here.
func DefaultProbes() []SignInProbe {
	return []SignInProbe{BufProbe{}}
}
