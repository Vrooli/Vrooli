package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	conversationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/conversation"
	conversationH "web-console/handlers/conversation"
)

func TestConversationHTTPPathsRetainRoutedPersistence(t *testing.T) {
	primary, leased := setupTestDB(t), setupTestDB(t)
	for _, pool := range []*sql.DB{primary, leased} {
		if err := ensureConversationFTS(context.Background(), pool); err != nil {
			t.Fatal(err)
		}
	}
	var seq int
	var name, dsn string
	if err := leased.QueryRow("PRAGMA database_list").Scan(&seq, &name, &dsn); err != nil {
		t.Fatal(err)
	}
	var primarySeq int
	var primaryName, primaryDSN string
	if err := primary.QueryRow("PRAGMA database_list").Scan(&primarySeq, &primaryName, &primaryDSN); err != nil {
		t.Fatal(err)
	}
	routed, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryDSN})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routed.Close() })
	if err := routed.InstallTestPool(context.Background(), dsn, "conversation-routing", time.Minute); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routed.ClearTestPool("conversation-routing") })
	srv, sessionID := newSessionForUpload(t)
	srv.conversations = NewConversationStoreWithRepository(NewSQLConversationRepository(routed))
	srv.hookAuthToken = "fixture-token"
	ctx := database.WithTestMode(context.Background())
	for _, assistant := range []bool{false, true} {
		var body map[string]any
		if assistant {
			body = map[string]any{"last_assistant_message": "routed needle answer", "web_console_session_id": sessionID}
		} else {
			body = map[string]any{"userPrompt": "routed needle question", "webConsoleSessionId": sessionID}
		}
		raw, _ := json.Marshal(body)
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(raw))).WithContext(ctx)
		request.Header.Set("X-Hook-Token", "fixture-token")
		response := httptest.NewRecorder()
		if assistant {
			srv.handleHookStop(response, request)
		} else {
			srv.handleHookPromptSubmit(response, request)
		}
		if response.Code != 200 {
			t.Fatalf("hook: %d %s", response.Code, response.Body.String())
		}
	}
	for _, pool := range []struct {
		name     string
		expected int
		query    func() (int, error)
	}{
		{"primary", 0, func() (int, error) {
			var count int
			err := primary.QueryRow("SELECT count(*) FROM conversation_events WHERE session_id=?", sessionID).Scan(&count)
			return count, err
		}},
		{"leased", 2, func() (int, error) {
			var count int
			err := leased.QueryRow("SELECT count(*) FROM conversation_events WHERE session_id=?", sessionID).Scan(&count)
			return count, err
		}},
	} {
		count, err := pool.query()
		if err != nil || count != pool.expected {
			t.Errorf("%s event count=%d, %v; want %d", pool.name, count, err, pool.expected)
		}
	}
	// The same ID deliberately exists in both pools. Reads and cursor writes
	// must select by request context rather than accidentally passing via absence.
	primaryRepo := NewSQLConversationRepository(primary)
	if _, err := primaryRepo.AppendEvent(context.Background(), ConversationEvent{ID: "primary-only", SessionID: sessionID, Source: "test", Role: ConversationRoleUser, Text: "primary private material", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	handler := conversationH.NewConnectHandler(conversationH.Deps{Service: newConversationAdapter(srv)})
	got, err := handler.Get(ctx, connect.NewRequest(&conversationv1.GetRequest{SessionId: sessionID, Limit: 10}))
	if err != nil || len(got.Msg.GetEvents()) != 2 || got.Msg.GetTotalCount() != 2 {
		t.Fatalf("routed Get: %v, %+v", err, got)
	}
	for _, event := range got.Msg.Events {
		if strings.Contains(event.Text, "private") {
			t.Fatal("routed read exposed primary conversation")
		}
	}
	found, err := handler.Search(ctx, connect.NewRequest(&conversationv1.SearchRequest{SessionId: sessionID, Query: "private", Limit: 10}))
	if err != nil || len(found.Msg.GetMatches()) != 0 {
		t.Fatalf("routed search exposed primary: %v %+v", err, found)
	}
	ranged, err := handler.GetRange(ctx, connect.NewRequest(&conversationv1.GetRangeRequest{SessionId: sessionID, FromSequence: 1, ToSequence: 2}))
	if err != nil || len(ranged.Msg.GetEvents()) != 2 {
		t.Fatalf("routed range: %v %+v", err, ranged)
	}
	if _, err := handler.UpdateCursor(ctx, connect.NewRequest(&conversationv1.UpdateCursorRequest{SessionId: sessionID, HasLastSeenSequence: true, LastSeenSequence: 2})); err != nil {
		t.Fatal(err)
	}
	var seen int
	if err := primary.QueryRow("SELECT last_seen_sequence FROM conversation_sessions WHERE session_id=?", sessionID).Scan(&seen); err != nil || seen != 0 {
		t.Fatalf("primary cursor changed: %d %v", seen, err)
	}
}
