package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	"web-console/backends/codex"
	sessionsH "web-console/handlers/sessions"
	"web-console/internal/sessionstore"
)

type nativeObservationWriter struct{ strings.Builder }

func (*nativeObservationWriter) Close() error { return nil }

func managedObservationFixture(t *testing.T, response string) (*Server, *codex.ManagedOwner, sessionstore.Metadata, *nativeObservationWriter) {
	t.Helper()
	t.Setenv("WC_SESSION_STATE_ROOT", t.TempDir())
	id, thread := uuid.NewString(), uuid.NewString()
	response = strings.ReplaceAll(response, "THREAD", thread)
	reader, stop := heldCodexResponses(response)
	writer := &nativeObservationWriter{}
	owner, err := codex.NewManagedOwner(codex.NewClient(writer, reader), "exact-generation", thread)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(stop); _ = owner.Close() })
	descriptor, _ := json.Marshal(sessionstore.LaunchDescriptor{Agent: "codex", LaunchMode: "codex_app_server", Environment: map[string]string{"CODEX_HOME": t.TempDir()}})
	meta := sessionstore.Metadata{ID: id, Created: time.Now(), Status: sessionstore.StatusLive, AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeCodexAppServer, ControlMode: sessionstore.ControlModeNativeCapable, NativeOwner: owner.OwnerID(), NativeThreadID: thread, LaunchDescriptorJSON: string(descriptor)}
	store := sessionstore.NewInMemory()
	if err := store.Save(t.Context(), meta); err != nil {
		t.Fatal(err)
	}
	registry := newManagedCodexRegistry(store)
	registry.entries[id] = managedCodexEntry{owner: owner, session: sessionsH.Session{ID: id, NativeOwner: meta.NativeOwner, NativeThreadID: thread, LaunchMode: "codex_app_server", ControlMode: "native_capable"}}
	return &Server{sessionStore: store, managedCodex: registry}, owner, meta, writer
}

func TestManagedNativeStateUsesRetainedStdioOwnerWithoutDaemon(t *testing.T) {
	responses := `{"jsonrpc":"2.0","id":1,"result":{"thread":{"id":"THREAD","status":{"type":"idle"}}}}` + "\n" + `{"jsonrpc":"2.0","id":2,"result":{"goal":null}}` + "\n"
	server, _, meta, writer := managedObservationFixture(t, responses)
	response, err := server.readSessionNativeThreadState(t.Context(), &sessionsv1.ReadNativeThreadStateRequest{SessionId: meta.ID, ThreadIds: []string{meta.NativeThreadID}})
	if err != nil {
		t.Fatal(err)
	}
	row := response.GetThreads()[0]
	if row.GetStatus() != "idle" || !row.GetGoalAvailable() || row.GetHasGoal() {
		t.Fatalf("observed managed state = %+v", row)
	}
	if !strings.Contains(writer.String(), `"method":"thread/read"`) || !strings.Contains(writer.String(), `"method":"thread/goal/get"`) {
		t.Fatalf("wrong owner protocol: %s", writer.String())
	}
}

func TestManagedNativeStateRejectsForeignThreadOrChangedGenerationBeforeRPC(t *testing.T) {
	for _, failure := range []string{"foreign thread", "changed generation", "retiring owner", "unknown transport"} {
		t.Run(failure, func(t *testing.T) {
			server, owner, meta, writer := managedObservationFixture(t, "")
			ids := []string{meta.NativeThreadID}
			switch failure {
			case "foreign thread":
				ids = append(ids, uuid.NewString())
			case "changed generation":
				meta.NativeOwner = "replacement-generation"
				if err := server.sessionStore.UpdateAgentInfo(t.Context(), meta.ID, sessionstore.AgentInfo{NativeOwner: meta.NativeOwner}); err != nil {
					t.Fatal(err)
				}
			case "retiring owner":
				owner.BeginRetirement()
			case "unknown transport":
				meta.LaunchMode = sessionstore.LaunchModeUnknown
				if err := server.sessionStore.UpdateAgentInfo(t.Context(), meta.ID, sessionstore.AgentInfo{LaunchMode: meta.LaunchMode}); err != nil {
					t.Fatal(err)
				}
			}
			_, err := server.readSessionNativeThreadState(t.Context(), &sessionsv1.ReadNativeThreadStateRequest{SessionId: meta.ID, ThreadIds: ids})
			if connect.CodeOf(err) != connect.CodeUnavailable || writer.Len() != 0 {
				t.Fatalf("unbound read reached provider: err=%v input=%s", err, writer.String())
			}
		})
	}
}

func TestManagedNativeUsageReadsAdmittedHomeAndRetainsMissingUsage(t *testing.T) {
	server, _, meta, _ := managedObservationFixture(t, "")
	home := t.TempDir()
	descriptor, _ := json.Marshal(sessionstore.LaunchDescriptor{Agent: "codex", LaunchMode: "codex_app_server", Environment: map[string]string{"CODEX_HOME": home}})
	if err := server.sessionStore.UpdateAgentInfo(t.Context(), meta.ID, sessionstore.AgentInfo{LaunchDescriptorJSON: string(descriptor)}); err != nil {
		t.Fatal(err)
	}
	// No audit DB exists yet: do not invent zero usage or requested model proof.
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	request := &sessionsv1.ReadNativeUsageRequest{SessionId: meta.ID, ThreadIds: []string{meta.NativeThreadID}, Start: start.Format(time.RFC3339), End: start.Add(time.Hour).Format(time.RFC3339)}
	if _, err := server.readSessionNativeUsage(t.Context(), request); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("missing audit became zero usage: %v", err)
	}
	setupManagedUsageAudit(t, home, meta.NativeThreadID)
	response, err := server.readSessionNativeUsage(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetThreads()) != 1 || response.GetThreads()[0].GetObservedModel() != "gpt-6-luna" || response.GetThreads()[0].GetObservedEffort() != "high" || response.GetThreads()[0].GetReport().GetUsage().GetInputTokens() != 10 {
		t.Fatalf("wrong managed audit home/report: %+v", response)
	}
	request.ThreadIds = append(request.ThreadIds, uuid.NewString())
	if _, err := server.readSessionNativeUsage(t.Context(), request); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("foreign audit thread admitted: %v", err)
	}
}

func setupManagedUsageAudit(t *testing.T, home, thread string) {
	t.Helper()
	dir := filepath.Join(home, "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	rollout := filepath.Join(dir, "rollout.jsonl")
	record := `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"` + thread + `","response_id":"r1","usage":{"input_tokens":10,"cached_input_tokens":7,"output_tokens":2}}}` + "\n"
	if err := os.WriteFile(rollout, []byte(record), 0600); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(home, "state_5.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE threads(id TEXT PRIMARY KEY, rollout_path TEXT, model TEXT, model_provider TEXT, reasoning_effort TEXT); INSERT INTO threads VALUES (?, ?, 'gpt-6-luna', 'openai', 'high')", thread, rollout); err != nil {
		t.Fatal(err)
	}
}

func TestManagedNativeStateReadsOnlyActiveAndOriginalLineage(t *testing.T) {
	original := uuid.NewString()
	responses := `{"jsonrpc":"2.0","id":1,"result":{"thread":{"id":"THREAD","status":{"type":"idle"}}}}` + "\n" + `{"jsonrpc":"2.0","id":2,"result":{"goal":null}}` + "\n" + `{"jsonrpc":"2.0","id":3,"result":{"thread":{"id":"` + original + `","status":{"type":"idle"}}}}` + "\n" + `{"jsonrpc":"2.0","id":4,"result":{"goal":null}}` + "\n"
	server, owner, meta, _ := managedObservationFixture(t, responses)
	if err := owner.RestoreLineage(original, ""); err != nil {
		t.Fatal(err)
	}
	if err := server.sessionStore.UpdateAgentInfo(t.Context(), meta.ID, sessionstore.AgentInfo{ForkedFromNativeSession: original}); err != nil {
		t.Fatal(err)
	}
	response, err := server.readSessionNativeThreadState(t.Context(), &sessionsv1.ReadNativeThreadStateRequest{SessionId: meta.ID, ThreadIds: []string{meta.NativeThreadID, original}})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetThreads()) != 2 || response.GetThreads()[1].GetThreadId() != original || response.GetThreads()[1].GetStatus() != "idle" || !response.GetThreads()[1].GetGoalAvailable() {
		t.Fatalf("owned original lineage unavailable: %+v", response)
	}
}

func TestManagedNativeUsageRetainsArchivedAuditWithoutRuntime(t *testing.T) {
	server, owner, meta, writer := managedObservationFixture(t, "")
	descriptor, err := sessionstore.ParseLaunchDescriptor(meta.LaunchDescriptorJSON)
	if err != nil {
		t.Fatal(err)
	}
	setupManagedUsageAudit(t, descriptor.Environment["CODEX_HOME"], meta.NativeThreadID)
	// The actual EOF/unavailable transition clears live-generation authority;
	// durable audit evidence must remain readable after that fence and archive.
	server.managedCodex.markUnavailable(meta.ID, owner)
	retired, err := server.sessionStore.Get(t.Context(), meta.ID)
	if err != nil || retired.NativeOwner != "" || retired.ControlMode != sessionstore.ControlModeRecoveryOnly {
		t.Fatalf("owner retirement did not fence live authority: metadata=%+v err=%v", retired, err)
	}
	if err := server.sessionStore.MarkArchived(t.Context(), meta.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	server.managedCodex = nil
	response, err := server.readSessionNativeUsage(t.Context(), &sessionsv1.ReadNativeUsageRequest{SessionId: meta.ID, ThreadIds: []string{meta.NativeThreadID}, Start: "2026-01-01T00:00:00Z", End: "2026-01-01T01:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetThreads()[0].GetObservedModel() != "gpt-6-luna" || writer.Len() != 0 {
		t.Fatalf("archived audit changed owner or inferred intent: %+v input=%s", response, writer.String())
	}
}

func TestManagedNativeUsageRejectsUnknownAuditHome(t *testing.T) {
	for _, home := range []string{"", "relative/home"} {
		t.Run(home, func(t *testing.T) {
			server, _, meta, writer := managedObservationFixture(t, "")
			raw, _ := json.Marshal(sessionstore.LaunchDescriptor{Agent: "codex", LaunchMode: "codex_app_server", Environment: map[string]string{"CODEX_HOME": home}})
			if err := server.sessionStore.UpdateAgentInfo(t.Context(), meta.ID, sessionstore.AgentInfo{LaunchDescriptorJSON: string(raw)}); err != nil {
				t.Fatal(err)
			}
			_, err := server.readSessionNativeUsage(t.Context(), &sessionsv1.ReadNativeUsageRequest{SessionId: meta.ID, ThreadIds: []string{meta.NativeThreadID}, Start: "2026-01-01T00:00:00Z", End: "2026-01-01T01:00:00Z"})
			if connect.CodeOf(err) != connect.CodeUnavailable || writer.Len() != 0 {
				t.Fatalf("unknown home used ambient owner: err=%v input=%s", err, writer.String())
			}
		})
	}
}

type nativeProvenanceCounter struct {
	sessionstore.Store
	calls int
}

func (s *nativeProvenanceCounter) SetProvenance(ctx context.Context, id string, origin sessionstore.Origin, owner, label string) error {
	s.calls++
	return s.Store.SetProvenance(ctx, id, origin, owner, label)
}
func TestManagedNativeCreatePersistsProvenanceOnce(t *testing.T) {
	store := &nativeProvenanceCounter{Store: sessionstore.NewInMemory()}
	registry := newManagedCodexRegistry(store)
	owner, err := codex.NewManagedOwner(codex.NewClient(testWriteCloser{io.Discard}, strings.NewReader("")), "generation", uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = owner.Close() })
	registry.start = func(context.Context, string, string, string, string, string, []string, []string) (*codex.ManagedOwner, error) {
		return owner, nil
	}
	input := sessionsCreateInputForManagedTest()
	input.Origin = "programmatic"
	input.Owner = "agent-manager"
	input.DisplayLabel = "owned fixture"
	created, err := registry.Create(t.Context(), input)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := store.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if store.calls != 1 || meta.Owner != input.Owner || string(meta.Origin) != input.Origin || meta.DisplayLabel != input.DisplayLabel {
		t.Fatalf("provenance writes=%d metadata=%+v", store.calls, meta)
	}
}
