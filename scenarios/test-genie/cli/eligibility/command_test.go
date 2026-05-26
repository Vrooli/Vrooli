package eligibility

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
)

func withStub(t *testing.T, resp *eligpb.CheckResponse, callErr error) {
	t.Helper()
	prev := callCheck
	callCheck = func(_ context.Context, _ *cliutil.APIClient, _ string) (*connect.Response[eligpb.CheckResponse], error) {
		if callErr != nil {
			return nil, callErr
		}
		return connect.NewResponse(resp), nil
	}
	t.Cleanup(func() { callCheck = prev })
}

func TestCheck_HumanOutput_Routed(t *testing.T) {
	withStub(t, &eligpb.CheckResponse{Routed: true}, nil)

	var buf bytes.Buffer
	err := runCheck(nil, []string{"demo"}, &buf)
	if err != nil {
		t.Fatalf("expected nil error (exit 0); got %v", err)
	}
	if !strings.Contains(buf.String(), "Routed: yes") {
		t.Errorf("expected 'Routed: yes' in output; got %q", buf.String())
	}
}

func TestCheck_HumanOutput_NotRouted_Exit1(t *testing.T) {
	withStub(t, &eligpb.CheckResponse{
		Routed:               false,
		DisqualifyingReasons: []string{"raw sql.Open"},
		Violations: []*eligpb.Violation{
			{RuleId: "routed_database_drivers", Severity: "high", File: "db.go", Line: 7},
		},
	}, nil)

	var buf bytes.Buffer
	err := runCheck(nil, []string{"demo"}, &buf)
	if err == nil {
		t.Fatal("expected non-nil error to drive exit 1")
	}
	var ec *exitErr
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitErr; got %T", err)
	}
	if ec.ExitCode() != int(ExitNotRouted) {
		t.Errorf("expected exit code %d; got %d", ExitNotRouted, ec.ExitCode())
	}
	out := buf.String()
	if !strings.Contains(out, "Routed: no") || !strings.Contains(out, "raw sql.Open") || !strings.Contains(out, "db.go:7") {
		t.Errorf("missing expected lines; got %q", out)
	}
}

func TestCheck_Unreachable_Exit2(t *testing.T) {
	withStub(t, nil, errors.New("connection refused"))

	var buf bytes.Buffer
	err := runCheck(nil, []string{"demo"}, &buf)
	if err == nil {
		t.Fatal("expected error")
	}
	var ec *exitErr
	if !errors.As(err, &ec) {
		t.Fatalf("expected *exitErr; got %T", err)
	}
	if ec.ExitCode() != int(ExitUnreachable) {
		t.Errorf("expected exit code %d; got %d", ExitUnreachable, ec.ExitCode())
	}
}

func TestCheck_JSONOutput(t *testing.T) {
	withStub(t, &eligpb.CheckResponse{
		Routed:               false,
		DisqualifyingReasons: []string{"reason-x"},
	}, nil)

	var buf bytes.Buffer
	err := runCheck(nil, []string{"--json", "demo"}, &buf)
	if err == nil {
		t.Fatal("expected non-nil error for not-routed")
	}
	out := buf.String()
	if !strings.Contains(out, `"routed": false`) || !strings.Contains(out, `"reason-x"`) {
		t.Errorf("expected JSON with routed=false + reason; got %q", out)
	}
}

func TestCheck_MissingScenario(t *testing.T) {
	withStub(t, &eligpb.CheckResponse{Routed: true}, nil)
	var buf bytes.Buffer
	err := runCheck(nil, []string{}, &buf)
	if err == nil {
		t.Fatal("expected error when scenario is missing")
	}
}
