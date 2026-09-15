package main

import (
	"context"
	"io"
	"strings"
	"testing"

	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

func TestCodexLaunchArgsMakeApprovalPolicyExplicit(t *testing.T) {
	args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex", LaunchMode: "codex_app_server"}), " ")
	if !strings.Contains(args, `approval_policy="never"`) {
		t.Fatalf("launch args = %q", args)
	}
}

func TestCodexLaunchArgsPreserveDescriptorSettings(t *testing.T) {
	args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{
		Agent: "codex", LaunchMode: "codex_app_server", Model: "gpt-5-codex", Profile: "safe", ApprovalPolicy: "on-request", Sandbox: "workspace-write",
	}), " ")
	for _, expected := range []string{`model="gpt-5-codex"`, `profile="safe"`, `approval_policy="on-request"`, `sandbox_mode="workspace-write"`} {
		if !strings.Contains(args, expected) {
			t.Fatalf("launch args %q missing %q", args, expected)
		}
	}
}

func TestCodexConversationAdapterForksThroughNativeTurn(t *testing.T) {
	reader, stop := heldCodexResponses(`{"jsonrpc":"2.0","id":1,"result":{"threadId":"thread-2"}}` + "\n" + `{"jsonrpc":"2.0","id":2,"result":{"thread":{"id":"thread-2","turns":[{"id":"turn-4","status":"completed"}]}}}` + "\n")
	defer close(stop)
	c := codex.NewClient(testWriteCloser{io.Discard}, reader)
	owner, err := codex.NewManagedOwner(c, "web-console:session-1", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	adapter := codexConversationAdapter{owner: owner}
	event := ConversationEvent{NativeProvenance: &NativeProvenance{Provider: "codex", SessionID: "thread-1", TurnID: "turn-4"}}
	p, err := adapter.Preflight(context.Background(), event, ConversationControlPreflight{Capability: conversationCapabilities("codex")})
	if err != nil || p.Decision != ControlSupported || p.Capability.Available != true {
		t.Fatalf("preflight = %+v, err=%v", p, err)
	}
	for _, option := range p.Capability.Options {
		if option.ID == "restore_conversation" && !option.Supported {
			t.Fatal("conversation restore should be supported")
		}
		if option.ID != "restore_conversation" && option.Supported {
			t.Fatalf("unverified option %q was advertised", option.ID)
		}
	}
	thread, err := adapter.Execute(context.Background(), event, true)
	if err != nil || thread != "thread-2" {
		t.Fatalf("execute = %q, err=%v", thread, err)
	}
	if ok, err := adapter.Verify(context.Background(), event, thread); err != nil || !ok {
		t.Fatalf("verify = %v, err=%v", ok, err)
	}
}

func heldCodexResponses(body string) (io.Reader, chan struct{}) {
	reader, writer := io.Pipe()
	stop := make(chan struct{})
	go func() {
		_, _ = io.WriteString(writer, body)
		<-stop
		_ = writer.Close()
	}()
	return reader, stop
}

func TestRegisterManagedCodexOwnerPersistsHandshakeBeforeCapability(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{ID: "session-1", AgentType: sessionstore.AgentCodex}); err != nil {
		t.Fatal(err)
	}
	c := codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader(""))
	owner, err := codex.NewManagedOwner(c, "web-console:session-1", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.registerManagedCodexOwner(context.Background(), "session-1", owner, "0.153.4", "stdio", `{"agent":"codex","launchMode":"codex_app_server"}`); err != nil {
		t.Fatal(err)
	}
	meta, err := srv.sessionStore.Get(context.Background(), "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if meta.LaunchMode != sessionstore.LaunchModeCodexAppServer || meta.ControlMode != sessionstore.ControlModeNativeCapable || meta.NativeThreadID != "thread-1" {
		t.Fatalf("persisted handshake = %+v", meta)
	}
}

func TestCaptureManagedCodexEventUsesNativeIdentityAndSkipsDeltas(t *testing.T) {
	srv := newFakeTestServer()
	srv.managedCodex = newManagedCodexRegistry(nil)
	owner, err := codex.NewManagedOwner(codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader("")), "owner", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	srv.managedCodex.entries["session-1"] = managedCodexEntry{owner: owner}
	srv.captureManagedCodexEvent("session-1", codex.NativeEvent{Kind: "item/agentMessage/delta", ThreadID: "thread-1", TurnID: "turn-1", ItemID: "item-1", Text: "partial", Partial: true})
	srv.captureManagedCodexEvent("session-1", codex.NativeEvent{Kind: "item/completed", ThreadID: "thread-1", TurnID: "turn-1", ItemID: "item-1", BoundaryID: "item-1", Text: "complete"})
	state := srv.conversations.ListSession(context.Background(), "session-1")
	if len(state.Events) != 1 || state.Events[0].Text != "complete" || state.Events[0].NativeProvenance.BoundaryID != "item-1" {
		t.Fatalf("captured events = %+v", state.Events)
	}
}

func TestManagedCodexRegistryCreatesAndPersistsNativeSession(t *testing.T) {
	store := sessionstore.NewInMemory()
	registry := newManagedCodexRegistry(store)
	c := codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader(""))
	owner, err := codex.NewManagedOwner(c, "owner", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	registry.start = func(context.Context, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
		return owner, nil
	}
	s, err := registry.Create(context.Background(), sessionsCreateInputForManagedTest())
	if err != nil {
		t.Fatal(err)
	}
	if s.LaunchMode != "codex_app_server" || s.NativeThreadID != "thread-1" {
		t.Fatalf("created session = %+v", s)
	}
	meta, err := store.Get(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if meta.LaunchMode != sessionstore.LaunchModeCodexAppServer || meta.ControlMode != sessionstore.ControlModeNativeCapable {
		t.Fatalf("stored managed mode = %+v", meta)
	}
}

func TestManagedCodexRegistryPersistsForkBranchMetadata(t *testing.T) {
	store := sessionstore.NewInMemory()
	registry := newManagedCodexRegistry(store)
	if err := store.Save(context.Background(), sessionstore.Metadata{ID: "session-1", AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeCodexAppServer}); err != nil {
		t.Fatal(err)
	}
	if err := registry.recordFork(context.Background(), "session-1", "thread-original", "thread-active", "turn-4"); err != nil {
		t.Fatal(err)
	}
	meta, err := store.Get(context.Background(), "session-1")
	if err != nil || meta.NativeThreadID != "thread-active" || meta.LastVerifiedTurnID != "turn-4" || meta.ForkedFromNativeSession != "thread-original" {
		t.Fatalf("fork metadata = %+v, err=%v", meta, err)
	}
}

func sessionsCreateInputForManagedTest() sessionsH.CreateInput {
	return sessionsH.CreateInput{AgentType: "codex", LaunchMode: "codex_app_server", WorkingDir: "/tmp", Cols: 80, Rows: 24}
}

type testWriteCloser struct{ io.Writer }

func (testWriteCloser) Close() error { return nil }
