package profiles

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

func TestSQLLPBSReleaseConfigRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLLPBSReleaseConfigRepository(db)
	ctx := context.Background()
	query := regexp.QuoteMeta("SELECT profile_id, lpbs_domain, lpbs_remote_profile, lpbs_app_key")
	now := time.Now()
	mock.ExpectQuery(query).WithArgs("p1").WillReturnRows(sqlmock.NewRows([]string{"profile_id", "lpbs_domain", "lpbs_remote_profile", "lpbs_app_key", "default_channel", "update_url", "created_at", "updated_at"}).AddRow("p1", "https://lpbs", "demo", "app", "stable", "https://updates", now, now))
	cfg, err := repo.Get(ctx, "p1")
	if err != nil || cfg == nil || cfg.LPBSAppKey != "app" {
		t.Fatalf("get = %#v, %v", cfg, err)
	}
	mock.ExpectQuery(query).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if cfg, err := repo.Get(ctx, "missing"); err != nil || cfg != nil {
		t.Fatalf("missing get = %#v, %v", cfg, err)
	}
	cfg = &LPBSReleaseConfig{ProfileID: "p1", LPBSDomain: "https://lpbs", LPBSRemoteProfile: "demo", LPBSAppKey: "app", UpdateURL: "https://updates"}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_lpbs_release_config")).WithArgs("p1", "https://lpbs", "demo", "app", "stable", "https://updates").WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Upsert(ctx, cfg); err != nil || cfg.DefaultChannel != "stable" {
		t.Fatalf("upsert = %#v, %v", cfg, err)
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM profile_lpbs_release_config")).WithArgs("p1").WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Delete(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type lpbsProfileRepo struct {
	profile *Profile
	err     error
}

func (r lpbsProfileRepo) List(context.Context) ([]Profile, error)          { return nil, nil }
func (r lpbsProfileRepo) Get(context.Context, string) (*Profile, error)    { return r.profile, r.err }
func (r lpbsProfileRepo) Create(context.Context, *Profile) (string, error) { return "", nil }
func (r lpbsProfileRepo) Update(context.Context, string, map[string]interface{}) (*Profile, error) {
	return nil, nil
}
func (r lpbsProfileRepo) Delete(context.Context, string) (bool, error)           { return false, nil }
func (r lpbsProfileRepo) GetVersions(context.Context, string) ([]Version, error) { return nil, nil }
func (r lpbsProfileRepo) GetScenarioAndTier(context.Context, string) (string, int, error) {
	return "", 0, nil
}
func (r lpbsProfileRepo) AddSwap(context.Context, string, Swap) error      { return nil }
func (r lpbsProfileRepo) GetSwaps(context.Context, string) ([]Swap, error) { return nil, nil }

type lpbsConfigFakeRepo struct {
	cfg *LPBSReleaseConfig
	err error
}

func (r *lpbsConfigFakeRepo) Get(context.Context, string) (*LPBSReleaseConfig, error) {
	return r.cfg, r.err
}

func (r *lpbsConfigFakeRepo) Upsert(_ context.Context, cfg *LPBSReleaseConfig) error {
	r.cfg = cfg
	return r.err
}
func (r *lpbsConfigFakeRepo) Delete(context.Context, string) error { return r.err }

func lpbsRequest(method, path, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	return mux.SetURLVars(req, map[string]string{"id": "p1"})
}

func TestLPBSConfigHandlerValidatesProfileAndPersists(t *testing.T) {
	log := func(string, map[string]interface{}) {}
	configRepo := &lpbsConfigFakeRepo{}
	h := NewLPBSConfigHandler(lpbsProfileRepo{profile: &Profile{ID: "p1"}}, configRepo, log)
	rec := httptest.NewRecorder()
	h.Get(rec, lpbsRequest(http.MethodGet, "/", ""))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "stable") {
		t.Fatalf("empty config = %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	h.Upsert(rec, lpbsRequest(http.MethodPut, "/", `{"lpbs_app_key":"app"}`))
	if rec.Code != http.StatusOK || configRepo.cfg.ProfileID != "p1" {
		t.Fatalf("upsert = %d %#v", rec.Code, configRepo.cfg)
	}
	rec = httptest.NewRecorder()
	h.Delete(rec, lpbsRequest(http.MethodDelete, "/", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("delete = %d", rec.Code)
	}
	for name, profileRepo := range map[string]lpbsProfileRepo{
		"missing profile":      {},
		"profile lookup error": {err: errors.New("database")},
	} {
		t.Run(name, func(t *testing.T) {
			h := NewLPBSConfigHandler(profileRepo, &lpbsConfigFakeRepo{}, log)
			rec := httptest.NewRecorder()
			h.Upsert(rec, lpbsRequest(http.MethodPut, "/", `{}`))
			want := http.StatusNotFound
			if name == "profile lookup error" {
				want = http.StatusInternalServerError
			}
			if rec.Code != want {
				t.Fatalf("status = %d, want %d", rec.Code, want)
			}
		})
	}
}

func TestLPBSConfigHandlerReportsRepositoryFailures(t *testing.T) {
	log := func(string, map[string]interface{}) {}
	getErr := &lpbsConfigFakeRepo{err: errors.New("get failed")}
	h := NewLPBSConfigHandler(lpbsProfileRepo{profile: &Profile{ID: "p1"}}, getErr, log)
	rec := httptest.NewRecorder()
	h.Get(rec, lpbsRequest(http.MethodGet, "/", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("get error = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.Upsert(rec, lpbsRequest(http.MethodPut, "/", "{"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON = %d", rec.Code)
	}
	deleteErr := &lpbsConfigFakeRepo{err: errors.New("delete failed")}
	h = NewLPBSConfigHandler(lpbsProfileRepo{profile: &Profile{ID: "p1"}}, deleteErr, log)
	rec = httptest.NewRecorder()
	h.Delete(rec, lpbsRequest(http.MethodDelete, "/", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("delete error = %d", rec.Code)
	}
}
