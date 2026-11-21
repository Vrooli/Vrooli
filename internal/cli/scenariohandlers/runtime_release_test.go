package scenariohandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/lifecycle/retainedruntime"
	"io"
	"reflect"
	"strings"
	"testing"
)

type releaseFake struct {
	scenarioapp.PhaseRunner // Any unexpected lifecycle call panics instead of touching the host.
	calls                   []string
	result                  retainedruntime.Release
	err                     error
	selected                string
}

func (r *releaseFake) CaptureRuntimeRelease(_ context.Context, n, p string) (retainedruntime.Release, error) {
	r.calls = append(r.calls, "capture:"+n+":"+p)
	return r.result, r.err
}
func (r *releaseFake) VerifyRuntimeRelease(_ context.Context, n, id string) (retainedruntime.Release, error) {
	r.calls = append(r.calls, "verify:"+n+":"+id)
	return r.result, r.err
}
func (r *releaseFake) SelectedRuntimeReleaseID(_ context.Context, n string) (string, error) {
	return r.selected, r.err
}

func TestRuntimeReleaseHandlersPreserveMetadataAndFailClosed(t *testing.T) {
	for _, id := range []CommandID{CommandCaptureRuntime, CommandVerifyRuntime} {
		t.Run(string(id), func(t *testing.T) {
			var out bytes.Buffer
			fake := &releaseFake{result: retainedruntime.Release{ID: "exact-id", Scenario: "demo", RuntimeDir: "/retained", BuildIdentity: "build", ServiceManifestSHA256: "manifest-sha256"}}
			deps := rootcli.HandlerDeps[struct{}]{Stdout: func(struct{}) io.Writer { return &out }, Globals: func(struct{}) rootcli.GlobalOptions { return rootcli.GlobalOptions{} }, LifecycleRunner: func(struct{}) (scenarioapp.PhaseRunner, error) { return fake, nil }}
			args := []string{"demo", "--json"}
			want := "capture:demo:"
			if id == CommandVerifyRuntime {
				args = append(args, "exact-id")
				want = "verify:demo:exact-id"
			}
			handler := buildRuntimeReleaseHandlers(deps)[id]
			if err := handler(struct{}{}, args); err != nil {
				t.Fatal(err)
			}
			var got retainedruntime.Release
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, fake.result) || !reflect.DeepEqual(fake.calls, []string{want}) {
				t.Fatalf("result=%#v calls=%v", got, fake.calls)
			}
			out.Reset()
			fake.err = errors.New("verification failed")
			if err := handler(struct{}{}, args); !errors.Is(err, fake.err) {
				t.Fatalf("error=%v", err)
			}
			if out.Len() != 0 {
				t.Fatalf("success output after error: %s", out.String())
			}
			out.Reset()
			fake.err = nil
			textArgs := []string{"demo"}
			if id == CommandVerifyRuntime {
				textArgs = append(textArgs, "exact-id")
			}
			if err := handler(struct{}{}, textArgs); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "exact-id") {
				t.Fatalf("missing exact ID: %s", out.String())
			}
		})
	}
}

func TestRuntimeReleaseHelpDoesNotResolveRunner(t *testing.T) {
	var out bytes.Buffer
	deps := rootcli.HandlerDeps[struct{}]{Stdout: func(struct{}) io.Writer { return &out }, Globals: func(struct{}) rootcli.GlobalOptions { return rootcli.GlobalOptions{} }}
	for _, id := range []CommandID{CommandCaptureRuntime, CommandVerifyRuntime} {
		if err := buildRuntimeReleaseHandlers(deps)[id](struct{}{}, []string{"--help"}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSourceCLIPreparationHonorsRetainedSelection(t *testing.T) {
	for _, tc := range []struct {
		name, requested, selected string
		fail, want                bool
	}{
		{"source", "", "", false, true}, {"durable retained", "", "id", false, false}, {"explicit retained", "id", "", false, false}, {"explicit source", "source", "id", false, true}, {"unreadable selection", "", "", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &releaseFake{selected: tc.selected}
			if tc.fail {
				fake.err = errors.New("selection unreadable")
			}
			called := false
			deps := rootcli.HandlerDeps[struct{}]{LifecycleRunner: func(struct{}) (scenarioapp.PhaseRunner, error) { return fake, nil }, EnsureCLI: func(struct{}, string) error { called = true; return nil }}
			err := ensureSourceScenarioCLIs(deps, struct{}{}, tc.requested, "demo")
			if (err != nil) != tc.fail || called != tc.want {
				t.Fatalf("err=%v called=%v", err, called)
			}
		})
	}
}

func TestRemoteRestartPreservesRuntimeSelection(t *testing.T) {
	args := remoteStartArgs(lifecycle.StartOptions{RetainedReleaseID: "id", ExpectedRetainedReleaseID: "source"}, false, 0)
	if !reflect.DeepEqual(args, []string{"--runtime-release", "id", "--expected-runtime-release", "source"}) {
		t.Fatalf("args=%v", args)
	}
}
