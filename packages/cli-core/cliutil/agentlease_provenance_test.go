package cliutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

type bindingLeaseRecorder struct {
	recordingLeaseRecorder
	ids                     []string
	evidence                map[string][]EditorLeaseContainment
	stops                   []string
	beginErr, errorEvidence error
	beginErrors             []error
	beginCalls              int
	legacyStopAt            []int
}

func (r *bindingLeaseRecorder) BeginBinding(ctx context.Context, record EditorLeaseRecord) (string, error) {
	r.beginCalls++
	if r.beginCalls <= len(r.beginErrors) && r.beginErrors[r.beginCalls-1] != nil {
		return "", r.beginErrors[r.beginCalls-1]
	}
	if r.beginErr != nil {
		return "", r.beginErr
	}
	id := fmt.Sprintf("attempt-%d", len(r.ids)+1)
	r.ids = append(r.ids, id)
	if r.evidence == nil {
		r.evidence = make(map[string][]EditorLeaseContainment)
	}
	return id, r.Create(ctx, record)
}
func (r *bindingLeaseRecorder) Stop(ctx context.Context, sessionID, reason string) error {
	r.legacyStopAt = append(r.legacyStopAt, len(r.ids))
	return r.recordingLeaseRecorder.Stop(ctx, sessionID, reason)
}

func (r *bindingLeaseRecorder) RecordContainment(_ context.Context, _ string, id string, e EditorLeaseContainment) error {
	r.evidence[id] = append(r.evidence[id], e)
	return r.errorEvidence
}
func (r *bindingLeaseRecorder) HeartbeatBinding(context.Context, string, string) error { return nil }
func (r *bindingLeaseRecorder) StopBinding(_ context.Context, _ string, id, reason string) error {
	r.stops = append(r.stops, id)
	return nil
}

type provenanceContainer struct {
	mode       string
	runs       int
	beforeWait func()
	selfBefore func()
}

func (c *provenanceContainer) ContainSelf(scope string, _ SessionContainment) (ContainedSession, error) {
	if c.selfBefore != nil {
		c.selfBefore()
	}
	if c.mode == "exec failure" {
		return ContainedSession{}, errors.New("self failed")
	}
	return ContainedSession{Scope: "cgroup:/fixture/" + scope, Method: "transient-unit", BeforeExec: true, Process: ProcessBinding{PID: 100, Boot: "boot", Start: "10"}}, nil
}
func (c *provenanceContainer) Run(_ context.Context, scope string, _ SessionContainment, p SessionProcess) (ContainedSession, error) {
	c.runs++
	session := ContainedSession{Scope: "cgroup:/fixture/" + scope, Method: "systemd-run", BeforeExec: true, Process: ProcessBinding{PID: 200, Boot: "boot", Start: "20"}}
	if c.mode == "uncontained" {
		return ContainedSession{}, &UncontainedError{Err: errors.New("native unavailable")}
	}
	if c.mode == "post-start failure" {
		return session, errors.New("placement failed after spawn")
	}
	if c.mode != "old container" && p.OnContained != nil {
		p.OnContained(session)
	}
	if c.beforeWait != nil {
		c.beforeWait()
	}
	if c.mode == "child exit" {
		return session, errors.New("child exit")
	}
	return session, nil
}
func setupBindingLaunch(t *testing.T, r *bindingLeaseRecorder, c *provenanceContainer) AgentLaunchRequest {
	t.Helper()
	isolateLauncherModelPolicy(t)
	t.Setenv(AgentSessionEnv, "")
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	withLeaseRecorder(t, r)
	old, oldLimits := DefaultSessionContainer, agentContainmentFn
	DefaultSessionContainer = c
	agentContainmentFn = testContainment
	t.Cleanup(func() { DefaultSessionContainer, agentContainmentFn = old, oldLimits })
	return AgentLaunchRequest{Agent: "codex", WorkingDir: t.TempDir(), Args: []string{"-c", "exit 0"}, LookPath: func(string) (string, error) { return "/bin/sh", nil }, Stdout: io.Discard, Stderr: io.Discard, APIBase: "http://localhost", HTTPClient: &http.Client{Transport: credentialRedirectTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("isolated offline attribution") })}}
}
func TestLauncherRecordsNativeProofBeforeWaitWithoutPromotingReturnedScopes(t *testing.T) {
	for _, mode := range []string{"success", "child exit", "old container", "post-start failure", "uncontained"} {
		t.Run(mode, func(t *testing.T) {
			r := &bindingLeaseRecorder{}
			c := &provenanceContainer{mode: mode}
			req := setupBindingLaunch(t, r, c)
			if mode == "success" || mode == "child exit" {
				c.beforeWait = func() {
					if len(r.evidence["attempt-1"]) != 1 || !r.evidence["attempt-1"][0].Session.BeforeExec {
						t.Fatal("proof not persisted before wait")
					}
				}
			}
			result, err := LaunchCodingAgentResult(context.Background(), req)
			if !result.LeaseRecorded || c.runs != 1 || len(r.ids) != 1 || len(r.stops) != 1 {
				t.Fatalf("lifecycle result=%+v recorder=%+v", result, r)
			}
			evidence := r.evidence["attempt-1"]
			if len(evidence) != 1 {
				t.Fatalf("evidence=%+v", evidence)
			}
			if mode == "success" || mode == "child exit" {
				if !evidence[0].Session.BeforeExec {
					t.Fatal("native proof lost")
				}
			} else if evidence[0].Session.BeforeExec || evidence[0].Failure == "" {
				t.Fatalf("incomplete promoted: %+v", evidence)
			}
			if (mode == "child exit" || mode == "post-start failure") != (err != nil) {
				t.Fatalf("launch semantics changed: %v", err)
			}
		})
	}
}
func TestLauncherExecFallbackHasDistinctBindingAndFencedStop(t *testing.T) {
	if !execReplaceSupported {
		t.Skip("exec replacement requires Unix")
	}
	for _, mode := range []string{"success", "exec failure"} {
		t.Run(mode, func(t *testing.T) {
			r := &bindingLeaseRecorder{}
			c := &provenanceContainer{mode: mode}
			req := setupBindingLaunch(t, r, c)
			req.Stdout = nil
			req.Stderr = nil
			old := execAgentReplace
			t.Cleanup(func() { execAgentReplace = old })
			calls := 0
			execAgentReplace = func(string, []string, []string, string) error {
				calls++
				if len(r.evidence["attempt-1"]) != 1 {
					t.Fatal("exec crossed before containment outcome was recorded")
				}
				return errors.New("isolated exec failure")
			}
			c.beforeWait = func() {
				if len(r.ids) != 2 || len(r.evidence["attempt-2"]) != 1 {
					t.Fatal("spawn borrowed the exec binding")
				}
			}
			_, err := LaunchCodingAgentResult(context.Background(), req)
			if err != nil || calls != 1 || c.runs != 1 || len(r.ids) != 2 {
				t.Fatalf("fallback err=%v calls=%d recorder=%+v", err, calls, r)
			}
			if len(r.stops) != 2 || r.stops[0] != "attempt-1" || r.stops[1] != "attempt-2" {
				t.Fatalf("unfenced stop: %v", r.stops)
			}
		})
	}
}
func TestBindingRecorderFailureNeverChangesChildExecution(t *testing.T) {
	for _, mode := range []string{"begin fails", "write fails", "unsupported identity", "custom runner"} {
		t.Run(mode, func(t *testing.T) {
			r := &bindingLeaseRecorder{}
			c := &provenanceContainer{}
			req := setupBindingLaunch(t, r, c)
			if mode == "begin fails" {
				r.beginErr = errors.New("fixture store unavailable")
			}
			if mode == "write fails" {
				r.errorEvidence = errors.New("fixture write failed")
			}
			if mode == "unsupported identity" {
				r.beginErr = ErrEditorLeaseBindingUnavailable
			}
			ran := false
			if mode == "custom runner" {
				req.RunChild = func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
					ran = true
					return nil
				}
			}
			result, err := LaunchCodingAgentResult(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "custom runner" {
				if !ran || !r.evidence["attempt-1"][0].Unavailable {
					t.Fatal("custom runner promoted")
				}
			} else if c.runs != 1 {
				t.Fatal("child was blocked")
			}
			if mode == "begin fails" && result.LeaseRecorded {
				t.Fatal("failed binding claimed recorded")
			}
			if mode == "unsupported identity" && (!result.LeaseRecorded || len(r.created) != 1 || len(r.evidence) != 0) {
				t.Fatalf("legacy visibility lost or promoted: %+v", r)
			}
		})
	}
}

func TestExecFallbackStopsUnfencedLegacyAttemptBeforeReplacement(t *testing.T) {
	if !execReplaceSupported {
		t.Skip("exec replacement requires Unix")
	}
	for _, secondUnavailable := range []bool{false, true} {
		t.Run(fmt.Sprint(secondUnavailable), func(t *testing.T) {
			r := &bindingLeaseRecorder{beginErrors: []error{ErrEditorLeaseBindingUnavailable, nil}}
			if secondUnavailable {
				r.beginErrors[1] = ErrEditorLeaseBindingUnavailable
			}
			c := &provenanceContainer{}
			req := setupBindingLaunch(t, r, c)
			req.Stdout = nil
			req.Stderr = nil
			old := execAgentReplace
			t.Cleanup(func() { execAgentReplace = old })
			execAgentReplace = func(string, []string, []string, string) error { return errors.New("fixture exec failure") }
			c.beforeWait = func() {
				if len(r.legacyStopAt) != 1 || r.legacyStopAt[0] != 0 {
					t.Fatalf("legacy stop ran after replacement: %+v", r)
				}
				if !secondUnavailable && len(r.evidence["attempt-1"]) != 1 {
					t.Fatal("replacement did not record its native proof")
				}
			}
			if _, err := LaunchCodingAgentResult(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			if r.beginCalls != 2 || c.runs != 1 {
				t.Fatalf("fallback changed: %+v", r)
			}
		})
	}
}
