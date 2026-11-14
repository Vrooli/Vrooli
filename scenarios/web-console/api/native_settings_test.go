package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"web-console/backends/codex"
	"web-console/internal/sessionstore"
)

func TestNativeCodexLaunchReplaysExplicitControls(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
	raw := `{"agent":"codex","launchMode":"codex_app_server","controlsVersion":1,"model":"gpt-6-luna","reasoningEffort":"high","contextLimitTokens":12345,"approvalPolicy":"never","sandbox":"workspace-write","networkAccessDisabled":true}`
	d, err := sessionstore.ParseLaunchDescriptor(raw)
	if err != nil {
		t.Fatal(err)
	}
	args := strings.Join(codexLaunchArgs(d), " ")
	for _, want := range []string{`model="gpt-6-luna"`, `model_reasoning_effort="high"`, `model_auto_compact_token_limit=12345`, `approval_policy="never"`, `sandbox_mode="workspace-write"`, `sandbox_workspace_write.network_access=false`, `sandbox_workspace_write.writable_roots=[]`} {
		if !strings.Contains(args, want) {
			t.Fatalf("missing %q: %s", want, args)
		}
	}
	if got := strings.Join(codexCreateArgs(d), " "); got != args {
		t.Fatalf("creation changed admitted settings: %s vs %s", got, args)
	}
}

func TestNativeCodexLaunchEscapesConfigValues(t *testing.T) {
	d := sessionstore.LaunchDescriptor{Model: "custom\\path\"\nmodel"}
	args := strings.Join(codexLaunchArgs(d), " ")
	if strings.Contains(args, "\n") || !strings.Contains(args, `model="custom\\path\"\nmodel"`) {
		t.Fatalf("config value not escaped: %q", args)
	}
}

func TestNativeCodexLaunchAdmittedZeroContextResistsModeDefaults(t *testing.T) {
	usageModeFixture(t)
	writeCodexSaveMode(t)
	for _, model := range []string{"gpt-6.1-terra", ""} {
		d := sessionstore.LaunchDescriptor{ControlsVersion: 1, Agent: "codex", Model: model, ApprovalPolicy: "never", Sandbox: "danger-full-access"}
		if got := codexCreateDescriptor(d); got.Model != model {
			t.Fatalf("mode replaced admitted model %q with %q", model, got.Model)
		}
		args := strings.Join(codexCreateArgs(d), " ")
		if strings.Contains(args, "auto_compact") || strings.Contains(args, "luna") {
			t.Fatalf("mode changed admitted settings: %s", args)
		}
	}
}

func TestNativeCodexRecoveryReplaysAdmittedSettings(t *testing.T) {
	usageModeFixture(t)
	writeCodexSaveMode(t)
	descriptor := `{"agent":"codex","launchMode":"codex_app_server","controlsVersion":1,"model":"gpt-6.1-terra","reasoningEffort":"low","contextLimitTokens":32100,"approvalPolicy":"never","sandbox":"workspace-write","networkAccessDisabled":true}`
	var got []string
	r := &managedCodexRegistry{
		store:   recoverStore{rows: []sessionstore.Metadata{{ID: "original", Status: sessionstore.StatusLive, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "thread-original", LaunchDescriptorJSON: descriptor}}},
		entries: map[string]managedCodexEntry{},
		resume: func(_ context.Context, _, _, _, _, thread, _ string, _ []string, args []string) (*codex.ManagedOwner, error) {
			if thread != "thread-original" {
				t.Fatalf("replaced thread identity: %s", thread)
			}
			got = append([]string(nil), args...)
			return nil, errors.New("fixture ends at resume")
		},
	}
	if err := r.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, " ")
	for _, want := range []string{`model="gpt-6.1-terra"`, `model_reasoning_effort="low"`, `model_auto_compact_token_limit=32100`, `sandbox_mode="workspace-write"`, `sandbox_workspace_write.network_access=false`, `sandbox_workspace_write.writable_roots=[]`} {
		if !strings.Contains(joined, want) {
			t.Fatalf("recovery omitted %s: %s", want, joined)
		}
	}
	if strings.Contains(joined, "luna") || strings.Contains(joined, "180000") {
		t.Fatalf("recovery applied current mode: %s", joined)
	}
}

func TestNativeCodexRecoveryNeverRevivesRetiredOrRecoverableSessions(t *testing.T) {
	for name, row := range map[string]sessionstore.Metadata{
		"dismissed":                  {Status: sessionstore.StatusDismissed},
		"awaiting explicit recovery": {Status: sessionstore.StatusAwaitingRecovery},
		"archived":                   {Status: sessionstore.StatusLive, ArchivedAt: time.Now()},
		"superseded":                 {Status: sessionstore.StatusLive, RecoveredInto: "replacement"},
		"blank raw status":           {},
		"unknown status":             {Status: sessionstore.Status("future")},
	} {
		t.Run(name, func(t *testing.T) {
			row.ID = "retired"
			row.LaunchMode = sessionstore.LaunchModeCodexAppServer
			row.NativeThreadID = "old"
			row.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
			called := false
			r := &managedCodexRegistry{store: recoverStore{rows: []sessionstore.Metadata{row}}, resume: func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
				called = true
				return nil, errors.New("must not resume")
			}}
			if err := r.Recover(context.Background()); err != nil {
				t.Fatal(err)
			}
			if called {
				t.Fatal("automatic recovery restarted an ineligible provider")
			}
		})
	}
}

func TestNativeCodexRecoveryRefusesUnrestorableManagedIdentity(t *testing.T) {
	for name, row := range map[string]sessionstore.Metadata{
		"verified service owner":        {Owner: "agent-manager", LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`},
		"human labelled managed launch": {Owner: "human@example.com", LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server","environment":{"VROOLI_AGENT_MANAGER_STORE_ID":"owner-store"}}`},
	} {
		t.Run(name, func(t *testing.T) {
			row.ID = "managed"
			row.Status = sessionstore.StatusLive
			row.NativeThreadID = "original-thread"
			row.NativeOwner = "stale-generation"
			row.LaunchMode = sessionstore.LaunchModeCodexAppServer
			row.ControlMode = sessionstore.ControlModeNativeCapable
			store := sessionstore.NewInMemory()
			if err := store.Save(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			resumed, registered := false, false
			r := &managedCodexRegistry{store: store, resume: func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
				resumed = true
				return nil, errors.New("must not resume")
			}, register: func(context.Context, string, *codex.ManagedOwner, string, string, string) error {
				registered = true
				return nil
			}}
			if err := r.Recover(context.Background()); err != nil {
				t.Fatal(err)
			}
			if resumed || registered {
				t.Fatal("managed identity resumed without a fresh authorized handoff")
			}
			got, err := store.Get(context.Background(), row.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.NativeOwner != "" || got.ControlMode != sessionstore.ControlModeRecoveryOnly || got.NativeThreadID != row.NativeThreadID || got.LaunchDescriptorJSON != row.LaunchDescriptorJSON {
				t.Fatalf("unsafe recovery downgrade: %+v", got)
			}
		})
	}
}

func TestNativeCodexRecoveryRestoresOrdinaryLiveNativeSession(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
	row := sessionstore.Metadata{ID: "ordinary", Owner: "human@example.com", Status: sessionstore.StatusLive, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "thread-human", LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server","model":"gpt-6-luna"}`}
	store := sessionstore.NewInMemory()
	if err := store.Save(context.Background(), row); err != nil {
		t.Fatal(err)
	}
	client := codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader(""))
	owner, err := codex.NewManagedOwner(client, "new-generation", row.NativeThreadID)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	resumed, registered := false, false
	r := newManagedCodexRegistry(store)
	r.resume = func(_ context.Context, _, _, _, _, thread, _ string, _ []string, _ []string) (*codex.ManagedOwner, error) {
		resumed = true
		if thread != row.NativeThreadID {
			t.Fatalf("wrong thread: %s", thread)
		}
		return owner, nil
	}
	r.register = func(context.Context, string, *codex.ManagedOwner, string, string, string) error {
		registered = true
		return nil
	}
	if err := r.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !resumed || !registered {
		t.Fatal("ordinary live native session was not recovered")
	}
	got, err := r.Get(context.Background(), row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NativeOwner != "new-generation" || got.NativeThreadID != row.NativeThreadID {
		t.Fatalf("recovered session = %+v", got)
	}
}

func TestNativeCodexRecoveryRejectsInvalidDescriptorAndClearsOwner(t *testing.T) {
	store := sessionstore.NewInMemory()
	row := sessionstore.Metadata{ID: "invalid", Status: sessionstore.StatusLive, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "original", NativeOwner: "stale-generation", ControlMode: sessionstore.ControlModeNativeCapable, LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server","controlsVersion":2}`}
	if err := store.Save(context.Background(), row); err != nil {
		t.Fatal(err)
	}
	called := false
	r := &managedCodexRegistry{store: store, resume: func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
		called = true
		return nil, errors.New("must not resume")
	}}
	if err := r.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), row.ID)
	if err != nil {
		t.Fatal(err)
	}
	if called || got.NativeOwner != "" || got.ControlMode != sessionstore.ControlModeRecoveryOnly || got.NativeThreadID != row.NativeThreadID || got.LaunchDescriptorJSON != row.LaunchDescriptorJSON {
		t.Fatalf("invalid descriptor acquired live capability: %+v resumed=%v", got, called)
	}
}

type nativeRecoveryFallibleStore struct {
	sessionstore.Store
	updateErr error
	liveErr   error
}

func (s nativeRecoveryFallibleStore) UpdateAgentInfo(ctx context.Context, id string, info sessionstore.AgentInfo) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Store.UpdateAgentInfo(ctx, id, info)
}
func (s nativeRecoveryFallibleStore) MarkLive(ctx context.Context, id string) error {
	if s.liveErr != nil {
		return s.liveErr
	}
	return s.Store.MarkLive(ctx, id)
}

type nativeRecoveryCloseWriter struct{ closed bool }

func (*nativeRecoveryCloseWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *nativeRecoveryCloseWriter) Close() error              { w.closed = true; return nil }

func TestNativeCodexRecoverySurfacesFailedInvalidation(t *testing.T) {
	sentinel := errors.New("fixture cannot persist invalidation")
	for _, failure := range []string{"descriptor", "resume", "lineage", "registration", "mark live", "identity"} {
		t.Run(failure, func(t *testing.T) {
			store := sessionstore.NewInMemory()
			row := sessionstore.Metadata{ID: "failed", Status: sessionstore.StatusLive, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "original", NativeOwner: "stale-generation", ControlMode: sessionstore.ControlModeNativeCapable, LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`}
			if failure == "descriptor" {
				row.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server","controlsVersion":2}`
			}
			if failure == "identity" {
				row.Owner = "agent-manager"
			}
			if err := store.Save(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			writer := &nativeRecoveryCloseWriter{}
			owner, err := codex.NewManagedOwner(codex.NewClient(writer, strings.NewReader("")), "candidate", row.NativeThreadID)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			resumed, registered := false, false
			wrapper := nativeRecoveryFallibleStore{Store: store, updateErr: sentinel}
			if failure == "mark live" {
				wrapper.liveErr = errors.New("fixture cannot mark live")
			}
			r := newManagedCodexRegistry(wrapper)
			r.resume = func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
				resumed = true
				if failure == "resume" {
					return nil, errors.New("fixture resume failed")
				}
				if failure == "lineage" {
					return nil, nil
				}
				return owner, nil
			}
			r.register = func(context.Context, string, *codex.ManagedOwner, string, string, string) error {
				registered = true
				if failure == "registration" {
					return errors.New("fixture registration failed")
				}
				return nil
			}
			if err := r.Recover(context.Background()); !errors.Is(err, sentinel) {
				t.Fatalf("lost persistence failure: %v", err)
			}
			if (failure == "descriptor" || failure == "identity") && (resumed || registered) {
				t.Fatal("refused descriptor/identity reached provider")
			}
			if _, err := r.Get(context.Background(), row.ID); err == nil {
				t.Fatal("failed recovery installed live registry entry")
			}
			if (failure == "registration" || failure == "mark live") && !writer.closed {
				t.Fatal("failed candidate owner was not closed")
			}
		})
	}
}

func TestNativeCodexRecoveryInvalidatesClosedUnregisteredCandidates(t *testing.T) {
	for _, failure := range []string{"registration", "mark live"} {
		t.Run(failure, func(t *testing.T) {
			store := sessionstore.NewInMemory()
			row := sessionstore.Metadata{ID: "candidate", Status: sessionstore.StatusLive, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "original", NativeOwner: "stale-generation", ControlMode: sessionstore.ControlModeNativeCapable, LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`}
			if err := store.Save(context.Background(), row); err != nil {
				t.Fatal(err)
			}
			writer := &nativeRecoveryCloseWriter{}
			owner, err := codex.NewManagedOwner(codex.NewClient(writer, strings.NewReader("")), "candidate", row.NativeThreadID)
			if err != nil {
				t.Fatal(err)
			}
			defer owner.Close()
			wrapper := nativeRecoveryFallibleStore{Store: store}
			if failure == "mark live" {
				wrapper.liveErr = errors.New("fixture live transition failed")
			}
			r := newManagedCodexRegistry(wrapper)
			r.resume = func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
				return owner, nil
			}
			registered := false
			r.register = func(context.Context, string, *codex.ManagedOwner, string, string, string) error {
				registered = true
				if failure == "registration" {
					return errors.New("fixture registration failed")
				}
				return nil
			}
			if err := r.Recover(context.Background()); err != nil {
				t.Fatal(err)
			}
			if failure == "mark live" && registered {
				t.Fatal("storage failure published candidate owner")
			}
			if !writer.closed {
				t.Fatal("failed candidate owner remained open")
			}
			if _, err := r.Get(context.Background(), row.ID); err == nil {
				t.Fatal("failed candidate entered live registry")
			}
			got, err := store.Get(context.Background(), row.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.ControlMode != sessionstore.ControlModeRecoveryOnly || got.NativeOwner != "" || got.NativeThreadID != row.NativeThreadID || got.LaunchDescriptorJSON != row.LaunchDescriptorJSON {
				t.Fatalf("candidate failure not safely retained: %+v", got)
			}
		})
	}
}
