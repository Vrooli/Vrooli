package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeNetrc(t *testing.T, home, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".netrc"), []byte(content), 0o600); err != nil {
		t.Fatalf("write netrc: %v", err)
	}
}

func TestBufProbeNoNetrcReportsSignedOut(t *testing.T) {
	tmp := t.TempDir()
	probe := BufProbe{HomeDir: func() (string, error) { return tmp, nil }}
	result := probe.Probe(context.Background(), ProbeOptions{})
	if result.State != StateSignedOut {
		t.Fatalf("State = %q (want %q)", result.State, StateSignedOut)
	}
}

func TestBufProbeNetrcWithoutBufBuildIsSignedOut(t *testing.T) {
	tmp := t.TempDir()
	writeNetrc(t, tmp, "machine github.com login user password tok\n")
	probe := BufProbe{HomeDir: func() (string, error) { return tmp, nil }}
	result := probe.Probe(context.Background(), ProbeOptions{})
	if result.State != StateSignedOut {
		t.Fatalf("State = %q", result.State)
	}
}

func TestBufProbeNetrcWithBufBuildIsSignedIn(t *testing.T) {
	tmp := t.TempDir()
	writeNetrc(t, tmp, "machine buf.build login user password tok\n")
	probe := BufProbe{HomeDir: func() (string, error) { return tmp, nil }}
	result := probe.Probe(context.Background(), ProbeOptions{})
	if result.State != StateSignedIn {
		t.Fatalf("State = %q (detail=%q)", result.State, result.Detail)
	}
}

func TestBufProbeCheckExpiryDelegatesToExpiryProbeFailure(t *testing.T) {
	tmp := t.TempDir()
	writeNetrc(t, tmp, "machine buf.build login user password tok\n")
	probe := BufProbe{
		HomeDir:     func() (string, error) { return tmp, nil },
		ExpiryProbe: func(context.Context) error { return errors.New("HTTP 401 from buf.build") },
	}
	result := probe.Probe(context.Background(), ProbeOptions{CheckExpiry: true})
	if result.State != StateExpired {
		t.Fatalf("State = %q (detail=%q)", result.State, result.Detail)
	}
}

func TestBufProbeCheckExpirySucceeds(t *testing.T) {
	tmp := t.TempDir()
	writeNetrc(t, tmp, "machine buf.build login user password tok\n")
	probe := BufProbe{
		HomeDir:     func() (string, error) { return tmp, nil },
		ExpiryProbe: func(context.Context) error { return nil },
	}
	result := probe.Probe(context.Background(), ProbeOptions{CheckExpiry: true})
	if result.State != StateSignedIn {
		t.Fatalf("State = %q", result.State)
	}
}

func TestRunOrdersProbesByName(t *testing.T) {
	probes := []SignInProbe{stubProbe{name: "z-tool"}, stubProbe{name: "a-tool"}}
	report := Run(context.Background(), probes, ProbeOptions{})
	if len(report.Statuses) != 2 || report.Statuses[0].Name != "a-tool" || report.Statuses[1].Name != "z-tool" {
		t.Fatalf("not sorted by name: %+v", report)
	}
}

type stubProbe struct{ name string }

func (s stubProbe) Name() string { return s.name }
func (s stubProbe) Probe(context.Context, ProbeOptions) ProbeResult {
	return ProbeResult{State: StateUnknown}
}

func TestSignInProbeContractIsImplementedByBufProbe(t *testing.T) {
	var _ SignInProbe = BufProbe{}
}
