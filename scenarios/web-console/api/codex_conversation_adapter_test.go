package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

func TestCodexLaunchArgsMakeApprovalPolicyExplicit(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
	args := strings.Join(codexLaunchArgs(sessionstore.LaunchDescriptor{Agent: "codex", LaunchMode: "codex_app_server"}), " ")
	if !strings.Contains(args, `approval_policy="never"`) {
		t.Fatalf("launch args = %q", args)
	}
}

func TestCodexLaunchArgsPreserveDescriptorSettings(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
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
	srv.captureManagedCodexEvent("session-1", codex.NativeEvent{Kind: "item/completed", ThreadID: "thread-1", TurnID: "turn-2", MessageID: "voice-input", BoundaryID: "voice-input", Role: "user", Text: "Run tests", VoiceHandoff: true})
	state = srv.conversations.ListSession(context.Background(), "session-1")
	if len(state.Events) != 2 || state.Events[1].Source != codexVoiceHandoffSource || state.Events[1].Role != ConversationRoleUser {
		t.Fatalf("voice handoff receipt was not persisted: %+v", state.Events)
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
	var launchEnv []string
	registry.start = func(_ context.Context, _, _, _, _, _ string, env, _ []string) (*codex.ManagedOwner, error) {
		launchEnv = append([]string(nil), env...)
		return owner, nil
	}
	input := sessionsCreateInputForManagedTest()
	input.RequestedSessionID = "c06dce6e-c663-4898-bf98-d66f13013197"
	input.RequestFingerprint = "request-fingerprint-digest"
	input.AgentRunIdentityToken = "signed-child-token"
	input.Origin = "programmatic"
	input.Owner = "agent-manager"
	input.DisplayLabel = "interactive goal"
	input.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server"}`
	if !sessionsH.StageSessionRunIdentity(input.RequestedSessionID, input.AgentRunIdentityToken) {
		t.Fatal("could not stage test child identity")
	}
	s, err := registry.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != input.RequestedSessionID || s.CreateFingerprint != input.RequestFingerprint || s.LaunchMode != "codex_app_server" || s.NativeThreadID != "thread-1" {
		t.Fatalf("created session = %+v", s)
	}
	identityCount := 0
	for _, value := range launchEnv {
		if value == "VROOLI_AGENT_IDENTITY_TOKEN=signed-child-token" {
			identityCount++
		}
	}
	if identityCount != 1 {
		t.Fatalf("native owner launch env omitted reserved child identity: %v", launchEnv)
	}
	meta, err := store.Get(context.Background(), s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if meta.LaunchMode != sessionstore.LaunchModeCodexAppServer || meta.ControlMode != sessionstore.ControlModeNativeCapable || meta.CreateFingerprint != input.RequestFingerprint || strings.Contains(meta.LaunchDescriptorJSON, input.AgentRunIdentityToken) || meta.Owner != input.Owner || meta.DisplayLabel != input.DisplayLabel || meta.Origin != sessionstore.OriginProgrammatic {
		t.Fatalf("stored managed mode = %+v", meta)
	}
}

func TestManagedCodexRegistryRejectsIdentityCredentialInDescriptor(t *testing.T) {
	store := sessionstore.NewInMemory()
	registry := newManagedCodexRegistry(store)
	starts := 0
	registry.start = func(context.Context, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
		starts++
		return nil, nil
	}
	in := sessionsCreateInputForManagedTest()
	in.LaunchDescriptorJSON = `{"agent":"codex","launchMode":"codex_app_server","environment":{"VROOLI_AGENT_IDENTITY_TOKEN":"descriptor-injected"}}`
	if _, err := registry.Create(context.Background(), in); err == nil {
		t.Fatal("managed Codex accepted an identity credential in its durable descriptor")
	}
	rows, err := store.List(context.Background())
	if err != nil || starts != 0 || len(rows) != 0 {
		t.Fatalf("invalid descriptor side effects: starts=%d rows=%d err=%v", starts, len(rows), err)
	}
}

func TestManagedCodexRegistryRejectsStaleReceiverGenerationWithoutNativeInput(t *testing.T) {
	var wire strings.Builder
	owner, err := codex.NewManagedOwner(codex.NewClient(testWriteCloser{&wire}, strings.NewReader("")), "owner-generation-2", "thread-1")
	if err != nil {
		t.Fatal(err)
	}
	registry := newManagedCodexRegistry(nil)
	registry.entries["session-1"] = managedCodexEntry{
		session: sessionsH.Session{ID: "session-1", LaunchMode: "codex_app_server", ControlMode: "native_capable", NativeOwner: "owner-generation-2", NativeThreadID: "thread-1"},
		owner:   owner,
	}
	if _, _, err := registry.DeliverPrompt(context.Background(), "session-1", "owner-generation-1", "thread-1", "", "stale request"); err != codex.ErrGenerationChanged {
		t.Fatalf("stale receiver generation error = %v", err)
	}
	if wire.Len() != 0 {
		t.Fatalf("stale receiver generation reached native transport: %s", wire.String())
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

func TestManagedCodexRecoveryKeepsOrphanAwaitingExplicitRecovery(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	original := sessionstore.Metadata{ID: "session-1", AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeCodexAppServer, NativeThreadID: "thread-1", LaunchDescriptorJSON: `{"agent":"codex","launchMode":"codex_app_server"}`, CreateFingerprint: "persisted-create-digest", Origin: sessionstore.OriginProgrammatic, Owner: "agent-manager", DisplayLabel: "interactive goal"}
	if err := store.Save(ctx, original); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkOrphaned(ctx, original.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	registry := newManagedCodexRegistry(store)
	resumed := false
	registry.resume = func(context.Context, string, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
		resumed = true
		return nil, errors.New("must not auto-resume orphan")
	}
	if err := registry.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	meta, err := store.Get(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed || meta.Status != sessionstore.StatusAwaitingRecovery {
		t.Fatalf("orphan was automatically revived: resumed=%v status=%s", resumed, meta.Status)
	}
	if _, err := registry.Get(ctx, original.ID); err == nil {
		t.Fatal("orphan registered a live native owner")
	}
	if meta.CreateFingerprint != original.CreateFingerprint || meta.Owner != original.Owner || meta.DisplayLabel != original.DisplayLabel || meta.NativeThreadID != original.NativeThreadID || meta.LaunchDescriptorJSON != original.LaunchDescriptorJSON {
		t.Fatalf("orphan recovery evidence lost: %+v", meta)
	}
}
