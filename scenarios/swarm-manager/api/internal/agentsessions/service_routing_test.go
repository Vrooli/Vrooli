package agentsessions

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/agentmanager"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// The production store is built by NewRoutedFileStore and owns no root of its
// own; only ForContext supplies one. A Service method that reaches past
// storeFor into s.store therefore resolves paths against the process working
// directory, which surfaces as "agent session not found" on every read and a
// silently empty list on every scan.
//
// Every Service test other than these builds NewFileStore(t.TempDir()), where
// the raw store and the routed view are the same object — so the bypass is
// structurally invisible there. These two tests are the ones that see it:
// the first exercises the real wiring end to end, the second forbids the
// bypass syntactically so a new call site cannot reintroduce it.

// routedTestService builds a Service over a routed store whose leased root is
// deliberately different from the primary root, mirroring production.
func routedTestService(t *testing.T, spawner *fakeSessionSpawner) (*Service, context.Context, string) {
	t.Helper()
	primary, leased := t.TempDir(), t.TempDir()
	paths := func(root string) storage.Paths { return storage.Paths{DataDir: filepath.Join(root, "data")} }
	roots := filerouting.New(paths(primary))
	if err := roots.InstallTestRoots(paths(leased), "session-routing", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots() error = %v", err)
	}
	svc, err := NewService(ServiceConfig{
		Store:       NewRoutedFileStore(roots),
		Spawner:     spawner,
		ProjectRoot: "/repo",
		ProfileKey:  "swarm-manager/default",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc, database.WithTestMode(context.Background()), leased
}

func TestRoutedServiceReadsResolveThroughTheLeasedRoot(t *testing.T) {
	spawner := &fakeSessionSpawner{runState: agentmanager.RunState{Status: "running"}}
	svc, ctx, leased := routedTestService(t, spawner)

	draft, err := svc.Create(ctx, CreateRequest{Kind: KindMetaOrchestration, Title: "Routed reads"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	// Anchor the rest of the test: the session really is on disk under the
	// leased root, so any later "not found" is a routing failure, not a
	// missing fixture.
	if _, err := os.Stat(filepath.Join(leased, "data", "agent-sessions", draft.ID, sessionFileName)); err != nil {
		t.Fatalf("session missing from leased root: %v", err)
	}

	t.Run("Get", func(t *testing.T) {
		// Get loads through storeFor but then hydrates artifacts; a bypass in
		// the hydration step fails the whole read.
		got, err := svc.Get(ctx, draft.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.ID != draft.ID {
			t.Fatalf("Get() id = %q, want %q", got.ID, draft.ID)
		}
	})

	t.Run("List", func(t *testing.T) {
		sessions, err := svc.List(ctx, ListFilters{})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(sessions) != 1 || sessions[0].ID != draft.ID {
			t.Fatalf("List() = %+v, want the one leased session", sessions)
		}
	})

	t.Run("ListWithoutArtifacts", func(t *testing.T) {
		// This one returned an empty slice with a nil error in production:
		// os.ReadDir("") is ErrNotExist, which the store reads as "no sessions".
		sessions, err := svc.ListWithoutArtifacts(ctx, ListFilters{})
		if err != nil {
			t.Fatalf("ListWithoutArtifacts() error = %v", err)
		}
		if len(sessions) != 1 || sessions[0].ID != draft.ID {
			t.Fatalf("ListWithoutArtifacts() = %+v, want the one leased session", sessions)
		}
	})

	t.Run("ListArtifacts", func(t *testing.T) {
		artifacts, err := svc.ListArtifacts(ctx, draft.ID)
		if err != nil {
			t.Fatalf("ListArtifacts() error = %v", err)
		}
		if len(artifacts) != 0 {
			t.Fatalf("ListArtifacts() = %+v, want none", artifacts)
		}
	})

	t.Run("ListEvents", func(t *testing.T) {
		// A draft has no run, so the interesting assertion is that the session
		// resolves at all rather than 404ing before the run check.
		result, err := svc.ListEvents(ctx, ListEventsRequest{SessionID: draft.ID})
		if err != nil {
			t.Fatalf("ListEvents() error = %v", err)
		}
		if len(result.Events) != 0 {
			t.Fatalf("ListEvents() = %+v, want none for a draft", result.Events)
		}
	})

	t.Run("Attachments", func(t *testing.T) {
		attachments, err := svc.UploadAttachments(ctx, draft.ID, []AttachmentUpload{{
			Filename:    "shot.png",
			ContentType: "image/png",
			SizeBytes:   4,
			Reader:      strings.NewReader("data"),
		}})
		if err != nil {
			t.Fatalf("UploadAttachments() error = %v", err)
		}
		if len(attachments) != 1 {
			t.Fatalf("UploadAttachments() = %+v, want one", attachments)
		}
		path, stored, err := svc.AttachmentPath(ctx, draft.ID, attachments[0].ID)
		if err != nil {
			t.Fatalf("AttachmentPath() error = %v", err)
		}
		if stored.ID != attachments[0].ID {
			t.Fatalf("AttachmentPath() attachment = %+v, want %q", stored, attachments[0].ID)
		}
		if !strings.HasPrefix(path, leased) {
			t.Fatalf("AttachmentPath() = %q, want a path under the leased root %q", path, leased)
		}
	})
}

func TestRoutedStoreRejectsUseOutsideARequestScope(t *testing.T) {
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	store := NewRoutedFileStore(roots)

	// Reads must not report "not found" and scans must not report "empty":
	// both read as ordinary, healthy answers and hid a total outage.
	if _, err := store.LoadSession("sess_unrouted"); err == nil || !strings.Contains(err.Error(), "routed request scope") {
		t.Fatalf("LoadSession() error = %v, want an unrouted-store error", err)
	}
	sessions, err := store.ListSessions(ListFilters{})
	if err == nil || len(sessions) != 0 {
		t.Fatalf("ListSessions() = (%v, %v), want an unrouted-store error", sessions, err)
	}
	if err := store.SaveSession(validStoredSession("sess_unrouted")); err == nil {
		t.Fatal("SaveSession() error = nil, want an unrouted-store error")
	}
}

func TestRoutedStoreViewsShareOneLock(t *testing.T) {
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(t.TempDir(), "data")})
	store := NewRoutedFileStore(roots)
	ctx := database.WithTestMode(context.Background())

	first, err := store.ForContext(ctx)
	if err != nil {
		t.Fatalf("ForContext() error = %v", err)
	}
	second, err := store.ForContext(ctx)
	if err != nil {
		t.Fatalf("ForContext() error = %v", err)
	}
	// Per-request views write the same directory, so a per-view mutex would
	// leave concurrent writers unserialized.
	if first.(*FileStore).mu != second.(*FileStore).mu {
		t.Fatal("request-scoped stores hold different mutexes; concurrent writers to one root would not serialize")
	}
}

// TestServiceMethodsNeverBypassStoreFor is the durable half of the fix: it
// fails on any new `s.store.` reference outside storeFor itself, which is what
// silently broke every agent-session read.
func TestServiceMethodsNeverBypassStoreFor(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	fileSet := token.NewFileSet()
	var offenders []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", name, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "storeFor" || !isServiceMethod(fn) {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				selector, ok := node.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "store" {
					return true
				}
				receiver, ok := selector.X.(*ast.Ident)
				if !ok || receiver.Name != "s" {
					return true
				}
				offenders = append(offenders, name+":"+fileSet.Position(selector.Pos()).String()+" in "+fn.Name.Name)
				return true
			})
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("Service methods must reach storage through storeFor(ctx), not s.store; the raw store has no root until the request lease supplies one:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

func isServiceMethod(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Body == nil {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	ident, ok := star.X.(*ast.Ident)
	return ok && ident.Name == "Service"
}
