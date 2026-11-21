package cloudtarget

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type observationRunner struct {
	name string
	args []string
	out  string
}

type oversizedObservationRunner struct{}

func (oversizedObservationRunner) LookPath(string) (string, error) { return "/fixed", nil }
func (oversizedObservationRunner) Run(context.Context, string, ...string) ([]byte, error) {
	return []byte(strings.Repeat("x", MaxObservationOutputBytes+1)), nil
}

func (r *observationRunner) LookPath(string) (string, error) { return "/fixed", nil }
func (r *observationRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.name, r.args, r.out = name, append([]string(nil), args...), "ok\n"
	return []byte(r.out), nil
}

func TestObserveUsesOwnerAllowlistedCommand(t *testing.T) {
	runner := &observationRunner{}
	result, err := Observe(context.Background(), ObservationRequest{Kind: "file", Args: []string{"--", "/etc/os-release"}}, runner)
	if err != nil {
		t.Fatal(err)
	}
	if runner.name != "cat" || strings.Join(runner.args, " ") != "-- /etc/os-release" || result.Stdout != "ok\n" {
		t.Fatalf("owner observation = runner=%s %v result=%+v", runner.name, runner.args, result)
	}
}

func TestObserveRejectsTraversalAndSymlinkEscape(t *testing.T) {
	runner := &observationRunner{}
	for _, args := range [][]string{{"--", "/etc/../root/.ssh/authorized_keys"}, {"--", "/dev/null"}} {
		if _, err := Observe(context.Background(), ObservationRequest{Kind: "file", Args: args}, runner); err == nil {
			t.Fatalf("unsafe path accepted: %v", args)
		}
	}
	root := t.TempDir()
	if err := os.Symlink("/dev", filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(context.Background(), ObservationRequest{Kind: "file", Args: []string{"--", filepath.Join(root, "escape", "null")}}, runner); err == nil {
		t.Fatal("symlink escape accepted")
	}
	if runner.name != "" {
		t.Fatalf("rejected observation reached runner: %s %v", runner.name, runner.args)
	}
}

func TestObserveRejectsMutationShapedArguments(t *testing.T) {
	if _, err := Observe(context.Background(), ObservationRequest{Kind: "process_match", Args: []string{"-x", "caddy;rm"}}, &observationRunner{}); err == nil {
		t.Fatal("mutation-shaped process observation accepted")
	}
	if _, err := Observe(context.Background(), ObservationRequest{Kind: "file_head", Args: []string{"-c", "999999999", "--", "/etc/os-release"}}, &observationRunner{}); err == nil {
		t.Fatal("unbounded file observation accepted")
	}
}

func TestObserveBoundsOwnerOutput(t *testing.T) {
	result, err := Observe(context.Background(), ObservationRequest{Kind: "file", Args: []string{"--", "/etc/os-release"}}, oversizedObservationRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Stdout) != MaxObservationOutputBytes || result.ExitCode != 1 || result.Stderr == "" {
		t.Fatalf("unbounded observation result: len=%d exit=%d stderr=%q", len(result.Stdout), result.ExitCode, result.Stderr)
	}
}
