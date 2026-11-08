package handlers

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite"
)

func TestWebSocketRunRevisionMatchesDurableDetailAndList(t *testing.T) {
	db, err := sqlx.Open("sqlite", filepath.Join(t.TempDir(), "runs.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	store := database.NewDB(db, logger)
	if err := store.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	repos := database.NewRepositories(store, logger)
	task := &domain.Task{ID: uuid.New(), Title: "revision", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusPending, RunMode: domain.RunModeSandboxed, Phase: domain.RunPhaseQueued}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	// A committed revision with sub-millisecond precision is followed by a
	// stale callback, just as an earlier writer can publish after a later one.
	revision := time.Date(2026, 11, 6, 20, 0, 0, 11, time.UTC)
	if _, err := db.ExecContext(t.Context(), "UPDATE runs SET status=?, updated_at=? WHERE id=?", domain.RunStatusComplete, revision.Format(time.RFC3339Nano), run.ID); err != nil {
		t.Fatal(err)
	}
	stored, err := repos.Runs.Get(t.Context(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	listed, err := repos.Runs.List(t.Context(), repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 10}})
	if err != nil || len(listed) != 1 {
		t.Fatalf("list=%v err=%v", listed, err)
	}
	detail, list := protoconv.RunToProto(stored), protoconv.RunToProto(listed[0])
	if !stored.UpdatedAt.Equal(revision) || !proto.Equal(detail.UpdatedAt, list.UpdatedAt) {
		t.Fatalf("durable revision lost precision or list/detail parity: stored=%s expected=%s detail=%v list=%v", stored.UpdatedAt.Format(time.RFC3339Nano), revision.Format(time.RFC3339Nano), detail.UpdatedAt, list.UpdatedAt)
	}

	hub := NewWebSocketHub()
	hub.SetRunStatusReader(repos.Runs.Get)
	t.Cleanup(hub.Close)
	go hub.Run()
	server := httptest.NewServer((&Handler{}).HandleWebSocket(hub))
	t.Cleanup(server.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	read := func() *domainpb.AgentManagerWsMessage {
		t.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, b, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		var message domainpb.AgentManagerWsMessage
		if err := protoconv.UnmarshalJSON(b, &message); err != nil {
			t.Fatal(err)
		}
		return &message
	}
	read() // welcome
	if err := conn.WriteJSON(map[string]string{"type": "ping"}); err != nil {
		t.Fatal(err)
	}
	read() // registration barrier
	run.Status, run.UpdatedAt = domain.RunStatusRunning, revision.Add(time.Hour)
	hub.BroadcastRunStatus(t.Context(), run)
	status := read().GetRunStatus()
	if status == nil || status.Status != detail.Status || !proto.Equal(status.UpdatedAt, detail.UpdatedAt) {
		t.Fatalf("stale callback replaced persisted status/revision: %v", status)
	}
}

func TestWebSocketFailedStatusReadCannotPublishAnUncommittedRevision(t *testing.T) {
	hub := NewWebSocketHub()
	t.Cleanup(hub.Close)
	primary := &WebSocketClient{send: make(chan []byte, 1)}
	other := &WebSocketClient{send: make(chan []byte, 1), partition: coredb.ContextPartition(coredb.WithTestMode(t.Context()))}
	hub.clients[primary], hub.clients[other] = true, true
	hub.SetRunStatusReader(func(ctx context.Context, _ uuid.UUID) (*domain.Run, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 2*time.Second {
			t.Error("status read has no bounded deadline")
		}
		return nil, errors.New("unavailable")
	})
	hub.BroadcastRunStatus(t.Context(), &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete, UpdatedAt: time.Now()})
	if len(hub.broadcast) != 0 {
		t.Fatal("uncommitted callback published after durable read failure")
	}
	if _, open := <-primary.send; open {
		t.Fatal("affected subscriber did not reconcile")
	}
	select {
	case <-other.send:
		t.Fatal("foreign routing partition disconnected")
	default:
	}
	hub.Close()
	hub.SetRunStatusReader(func(context.Context, uuid.UUID) (*domain.Run, error) {
		t.Fatal("closed hub read storage")
		return nil, nil
	})
	hub.BroadcastRunStatus(t.Context(), &domain.Run{ID: uuid.New()})
}
