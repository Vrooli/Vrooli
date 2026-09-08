package contextcapture

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"connectrpc.com/connect"

	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/contextcapture/contextcapture_v1connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	domain "portal/internal/contextcapture"
	"portal/internal/module"
)

// Runtime binds durable context bytes to the same production metadata lifecycle.
// Routed test requests must never fall through to this production blob owner.
func Runtime(db *database.RoutedDB, clk schedule.Clock, report func()) (module.Module, func(context.Context) error, Service, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return module.Module{}, nil, nil, err
	}
	root, err := resolver.EnsureArtifactDir(storage.Options{ScenarioID: "portal"}, storage.ArtifactRef{Owner: "portal", Domain: "contextcapture", Class: storage.ClassData, Segments: []string{"blobs"}}, 0o700)
	if err != nil {
		return module.Module{}, nil, nil, err
	}
	if err = os.Chmod(root, 0o700); err != nil {
		return module.Module{}, nil, nil, err
	}
	service := domain.NewService(domain.NewSQLiteRepository(db.Primary()), blobstore.NewFilesystemBlobStore(root), clk.Now)
	identity := owneridentity.NewClient(owneridentity.Config{Now: clk.Now})

	var testMu sync.Mutex
	var testPool *sql.DB
	var testHandler http.Handler
	var testService *domain.Service
	selectService := func(ctx context.Context) (*domain.Service, error) {
		if !database.IsTestMode(ctx) {
			return service, nil
		}
		pool, err := db.PoolForContext(ctx)
		if err != nil {
			return nil, err
		}
		testMu.Lock()
		defer testMu.Unlock()
		if testPool != pool {
			if err = database.EnsureSchemas(ctx, pool, database.SchemaProviderFunc(domain.Schema)); err != nil {
				return nil, err
			}
			testService = domain.NewService(domain.NewSQLiteRepository(pool), blobstore.NewMemoryBlobStore(), clk.Now)
			testPool = pool
			testHandler = nil
		}
		return testService, nil
	}
	routed := &routedService{selectService: selectService}
	handler := NewHandler(routed, identity, clk.Now)
	mod := Module(handler)
	mount := mod.Mount
	mod.Mount = func(r *mux.Router) {
		private := r.NewRoute().Subrouter()
		private.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !database.IsTestMode(r.Context()) {
					next.ServeHTTP(w, r)
					return
				}
				pool, err := db.PoolForContext(r.Context())
				if err != nil {
					http.Error(w, "context test lease unavailable", http.StatusServiceUnavailable)
					return
				}
				testMu.Lock()
				if testPool != pool {
					testMu.Unlock()
					http.Error(w, "context test storage unavailable", http.StatusServiceUnavailable)
					return
				}
				if testHandler == nil {
					_, testHandler = rpc.NewContextCaptureServiceHandler(NewHandler(testService, identity, clk.Now), connect.WithReadMaxBytes(45*1024*1024))
				}
				selected := testHandler
				testMu.Unlock()
				if selected == nil {
					http.Error(w, "context test storage unavailable", http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Cache-Control", "no-store")
				selected.ServeHTTP(w, r)
			})
		})
		mount(private)
	}
	cleanup := reaperFunc(func(ctx context.Context, limit int) error {
		pool, err := db.PoolForContext(database.WithTestMode(ctx))
		testMu.Lock()
		if err != nil || pool != testPool {
			testPool = nil
			testHandler = nil
			testService = nil
		}
		activeTest := testService
		testMu.Unlock()
		if activeTest != nil {
			_ = activeTest.Reap(ctx, limit)
		}
		return service.Reap(ctx, limit)
	})
	stop := startCleanup(cleanup, clk, report)
	return mod, stop, routed, nil
}

type routedService struct {
	selectService func(context.Context) (*domain.Service, error)
}

func (s *routedService) selected(ctx context.Context) (*domain.Service, error) {
	return s.selectService(ctx)
}

func (s *routedService) Import(ctx context.Context, owner string, input domain.Import, retention time.Duration) (domain.Document, error) {
	service, err := s.selected(ctx)
	if err != nil {
		return domain.Document{}, err
	}
	return service.Import(ctx, owner, input, retention)
}

func (s *routedService) Read(ctx context.Context, owner, id string) (domain.Document, []byte, error) {
	service, err := s.selected(ctx)
	if err != nil {
		return domain.Document{}, nil, err
	}
	return service.Read(ctx, owner, id)
}

func (s *routedService) ValidateReference(ctx context.Context, owner, id string) error {
	service, err := s.selected(ctx)
	if err != nil {
		return err
	}
	return service.ValidateReference(ctx, owner, id)
}

func (s *routedService) Delete(ctx context.Context, owner, id string) error {
	service, err := s.selected(ctx)
	if err != nil {
		return err
	}
	return service.Delete(ctx, owner, id)
}

func (s *routedService) ReconcileImport(ctx context.Context, owner, requestID string) (domain.ImportStatus, error) {
	service, err := s.selected(ctx)
	if err != nil {
		return domain.ImportStatus{}, err
	}
	return service.ReconcileImport(ctx, owner, requestID)
}

func (s *routedService) CancelImport(ctx context.Context, owner, requestID string) error {
	service, err := s.selected(ctx)
	if err != nil {
		return err
	}
	return service.CancelImport(ctx, owner, requestID)
}

func (s *routedService) Render(ctx context.Context, owner, id string) (domain.Document, []byte, string, error) {
	service, err := s.selected(ctx)
	if err != nil {
		return domain.Document{}, nil, "", err
	}
	return service.Render(ctx, owner, id)
}

type reaperFunc func(context.Context, int) error

func (f reaperFunc) Reap(ctx context.Context, limit int) error { return f(ctx, limit) }

type reaper interface {
	Reap(context.Context, int) error
}

func startCleanup(service reaper, clk schedule.Clock, report func()) func(context.Context) error {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := clk.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			sweep, finish := context.WithTimeout(ctx, 5*time.Second)
			err := service.Reap(sweep, 64)
			finish()
			if err != nil && ctx.Err() == nil && report != nil {
				report()
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C():
			}
		}
	}()
	return func(wait context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-wait.Done():
			return errors.New("context cleanup shutdown incomplete")
		}
	}
}
