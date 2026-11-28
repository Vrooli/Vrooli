package sessioncontain

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	platform "github.com/vrooli/platform-go"
)

func TestNativeContainmentEvidenceRequiresBeforeExecAndExactIncarnation(t *testing.T) {
	original := observeContainedProcess
	t.Cleanup(func() { observeContainedProcess = original })
	process := cliutil.ProcessBinding{PID: 100, UID: 1000, Boot: "boot-a", Start: "123", Cgroup: "0::/fixture/scope"}
	for _, tc := range []struct {
		name   string
		before bool
		fail   bool
		move   bool
		reuse  bool
		scope  string
		want   bool
	}{
		{name: "native success", before: true, scope: "cgroup:/fixture/scope", want: true},
		{name: "post-start root move", scope: "cgroup:/fixture/scope"},
		{name: "vanished inferred scope", before: true, fail: true, scope: "cgroup:/fixture/scope"},
		{name: "wrong scope", before: true, scope: "cgroup:/other"},
		{name: "pid reuse", before: true, reuse: true, scope: "cgroup:/fixture/scope"},
		{name: "concurrent move", before: true, move: true, scope: "cgroup:/fixture/scope"},
		{name: "process group not cgroup", before: true, scope: "pgid:100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			observeContainedProcess = func(int) (cliutil.ProcessBinding, error) {
				reads++
				p := process
				if tc.fail {
					return p, errors.New("gone")
				}
				if reads > 1 && tc.reuse {
					p.Start = "456"
				}
				if reads > 1 && tc.move {
					p.Cgroup = "0::/other"
				}
				return p, nil
			}
			got := nativeContainmentEvidence(tc.scope, platform.MethodSystemdRun, 100, tc.before)
			if got.BeforeExec != tc.want {
				t.Fatalf("evidence=%+v", got)
			}
			if tc.want && got.Process != process {
				t.Fatalf("lost incarnation: %+v", got)
			}
			if !tc.want && (got.ProofFailure == "" || got.Process.PID != 0) {
				t.Fatalf("promoted incomplete evidence: %+v", got)
			}
		})
	}
}

func TestContainmentCallbackRunsOnlyAfterSuccessfulStartBeforeWait(t *testing.T) {
	oldNew, oldObserve := newContainedCommand, observeContainedProcess
	t.Cleanup(func() { newContainedCommand, observeContainedProcess = oldNew, oldObserve })
	observeContainedProcess = func(pid int) (cliutil.ProcessBinding, error) {
		return cliutil.ProcessBinding{PID: pid, UID: 1000, Boot: "boot-a", Start: "123", Cgroup: "0::/fixture/scope"}, nil
	}
	for _, mode := range []string{"success with child error", "construction failure", "start failure", "cancelled before start", "post-start placement"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var command *exec.Cmd
			newContainedCommand = func(platform.ContainedSpec) (*platform.Contained, error) {
				if mode == "construction failure" {
					return nil, errors.New("fixture construction failure")
				}
				command = exec.Command("/bin/sh", "-c", "exit 23")
				if mode == "start failure" {
					command = exec.Command("/nonexistent/fixture")
				}
				if mode == "cancelled before start" {
					cancel()
				}
				method := platform.MethodSystemdRun
				if mode == "post-start placement" {
					method = platform.MethodCgroupWrite
				}
				return &platform.Contained{Cmd: command, Scope: platform.ScopeRef{Kind: platform.ScopeKindCgroup, Path: "/fixture/scope"}, Method: method}, nil
			}
			calls := 0
			_, err := (Container{}).Run(ctx, "fixture", cliutil.SessionContainment{}, cliutil.SessionProcess{Path: "ignored", OnContained: func(s cliutil.ContainedSession) {
				calls++
				if command.Process == nil || command.ProcessState != nil {
					t.Fatal("callback was not between native start and wait")
				}
				if s.BeforeExec != (mode == "success with child error") {
					t.Fatalf("proof=%+v", s)
				}
			}})
			want := 0
			if mode == "success with child error" || mode == "post-start placement" {
				want = 1
				if cliutil.ChildExitCode(err) != 23 {
					t.Fatalf("exit changed: %v", err)
				}
			}
			if calls != want {
				t.Fatalf("callbacks=%d want=%d error=%v", calls, want, err)
			}
		})
	}
}
