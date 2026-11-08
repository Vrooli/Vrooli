package devrouting_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
)

func TestHTTPRequestKeepsOriginalSQLAndFileLeaseGeneration(t *testing.T) {
	writeMode(t, "development")
	t.Setenv(apihttp.TestModeForceEnableEnv, "")
	db := openRouted(t)
	if _, err := db.ExecContext(t.Context(), "CREATE TABLE effects(value TEXT)"); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		_, err := pool.ExecContext(ctx, "CREATE TABLE effects(value TEXT)")
		return err
	})
	root := t.TempDir()
	roots := filerouting.New(storage.Paths{ConfigDir: filepath.Join(root, "config"), DataDir: filepath.Join(root, "data"), CacheDir: filepath.Join(root, "cache"), LogsDir: filepath.Join(root, "logs"), StateDir: filepath.Join(root, "state")})
	svc := devrouting.NewService(db, roots)
	install := func() {
		t.Helper()
		_, err := svc.InstallTestPool(t.Context(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "lease.db"), LeaseId: "http-lease", EmptyConfig: true}))
		if err != nil {
			t.Fatal(err)
		}
	}
	install()
	var held context.Context
	h := svc.PinnedTestModeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		held = context.WithoutCancel(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := func() int {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "/fixture", nil)
		r.Header.Set(apihttp.TestModeHeader, "1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if got := request(); got != http.StatusNoContent || held == nil {
		t.Fatalf("initial request did not reach leased handler: status=%d", got)
	}
	old := held
	oldKey := database.ContextPartition(old)
	if _, err := svc.ClearTestPool(t.Context(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "http-lease"})); err != nil {
		t.Fatal(err)
	}
	install()
	if _, err := db.ExecContext(old, "INSERT INTO effects VALUES ('stale')"); err == nil {
		t.Error("old HTTP request selected replacement SQL lease")
	}
	if _, err := roots.Pick(old, storage.ClassState); err == nil {
		t.Error("old HTTP request selected replacement file lease")
	}
	if got := request(); got != http.StatusNoContent || database.ContextPartition(held) == oldKey {
		t.Errorf("new HTTP request retained old delivery partition: status=%d", got)
	}
	if _, err := svc.ClearTestPool(t.Context(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "http-lease"})); err != nil {
		t.Fatal(err)
	}
	if got := request(); got != http.StatusServiceUnavailable {
		t.Errorf("missing lease reached ordinary handler: status=%d", got)
	}
	var count int
	if err := db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM effects").Scan(&count); err != nil || count != 0 {
		t.Errorf("primary storage changed: count=%d err=%v", count, err)
	}
}

func TestPinnedMiddlewarePreservesDevelopmentModeAndExactHeaderGuards(t *testing.T) {
	for _, tc := range []struct {
		name, mode, force, header string
		status                    int
	}{
		{"unmarked development", "development", "", "", http.StatusNoContent},
		{"noncanonical header", "development", "", "true", http.StatusNoContent},
		{"marked without paired lease", "development", "", "1", http.StatusServiceUnavailable},
		{"production remains ordinary", "production", "", "1", http.StatusNoContent},
		{"explicit CI mode still requires lease", "production", "1", "1", http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			writeMode(t, tc.mode)
			t.Setenv(apihttp.TestModeForceEnableEnv, tc.force)
			entered := false
			h := devrouting.NewService(nil).PinnedTestModeMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				entered = true
				if database.IsTestMode(r.Context()) {
					t.Error("ordinary handler received an unpinned test context")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			r := httptest.NewRequest(http.MethodGet, "/fixture", nil)
			r.Header.Set(apihttp.TestModeHeader, tc.header)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tc.status || entered != (tc.status == http.StatusNoContent) {
				t.Fatalf("guard mismatch: status=%d entered=%t", w.Code, entered)
			}
		})
	}
}
