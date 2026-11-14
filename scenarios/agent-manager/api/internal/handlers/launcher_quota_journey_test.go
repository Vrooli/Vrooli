//go:build journey

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/pricing"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestLauncherQuotaOwner serves the real quota reader, repository and HTTP
// handler for Web Console's cross-scenario browser journeys. The fixture
// control accepts statusline samples; it never replaces a product response.
func TestLauncherQuotaOwner(t *testing.T) {
	home := os.Getenv("LAUNCHER_QUOTA_HOME")
	if home == "" {
		t.Fatal("LAUNCHER_QUOTA_HOME is required")
	}
	db, err := sqlx.Open("sqlite", "file:"+filepath.Join(home, "quota.db")+"?_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	log := logrus.New()
	store := database.NewDB(db, log)
	if err := store.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	repository := database.NewPricingRepository(store, log)
	quota := repository.(pricing.QuotaObservationRepository)
	reader := &pricing.ClaudeStatuslineReader{Path: filepath.Join(home, "statusline.ndjson"), Store: quota}
	handler := NewPricingHandlerWithQuota(nil, nil, nil, quota)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/pricing/live-quota", handler.LiveQuota)
	mux.HandleFunc("/__journey/sample", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("reset") == "1" {
			if _, err := db.Exec("DELETE FROM quota_observations"); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
		var sample map[string]any
		if err := json.NewDecoder(r.Body).Decode(&sample); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sample["observed_at"] = time.Now().UTC().Format(time.RFC3339Nano)
		sample["session_id"] = "isolated-launcher-journey"
		data, _ := json.Marshal(sample)
		file, err := os.OpenFile(reader.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = file.Write(append(data, '\n'))
		_ = file.Close()
		if err == nil {
			_, err = reader.Ingest(r.Context())
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	done := make(chan struct{})
	var stop sync.Once
	mux.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { stop.Do(func() { close(done) }); w.WriteHeader(204) })
	server := httptest.NewServer(mux)
	defer server.Close()
	fmt.Println("LAUNCHER_QUOTA_URL=" + server.URL)
	select {
	case <-done:
	case <-t.Context().Done():
		t.Fatal("quota owner did not receive its shutdown")
	}
}
