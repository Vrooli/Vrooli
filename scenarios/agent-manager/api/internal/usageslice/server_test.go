package usageslice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This exercises the resource's feed through the real Agent Manager reader and
// handler, including a restart that rebuilds the view from the same durable feed.
func TestUsageSliceServesRealOwnerFeedAndRebuilds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	now := time.Now().UTC()
	sample := fmt.Sprintf(`{"observed_at":%q,"session_id":"isolated","five_hour":{"used_percentage":23,"resets_at":%d}}`+"\n", now.Format(time.RFC3339Nano), now.Add(time.Hour).Unix())
	if err := os.WriteFile(path, []byte(sample), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		server := httptest.NewServer(New(path))
		resp, err := http.Get(server.URL + "/api/v1/pricing/live-quota?agent=claude-code")
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Match, Source string
			Windows       []struct {
				UsedPercent   float64
				WindowMinutes int64
			}
		}
		err = json.NewDecoder(resp.Body).Decode(&got)
		resp.Body.Close()
		server.Close()
		if err != nil || resp.StatusCode != 200 || got.Match != "likely" || got.Source != "Claude Code statusline" || len(got.Windows) != 1 || got.Windows[0].UsedPercent != 23 || got.Windows[0].WindowMinutes != 300 {
			t.Fatalf("restart %d real feed response: %+v, %v", i, got, err)
		}
		t.Logf("restart=%d source=%s match=%s used_percent=%v window_minutes=%d", i, got.Source, got.Match, got.Windows[0].UsedPercent, got.Windows[0].WindowMinutes)
	}
}

func TestUsageSliceDoesNotExposeFullManager(t *testing.T) {
	server := httptest.NewServer(New(filepath.Join(t.TempDir(), "missing")))
	defer server.Close()
	for path, want := range map[string]int{"/health": 200, "/api/v1/runs": 404, "/api/v1/pricing/models": 404, "/api/v1/pricing/live-quota?agent=claude-code": 200} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("%s: %d want %d", path, resp.StatusCode, want)
		}
	}
}

func TestUsageSliceRefreshesFeedAndRejectsConflictingAccounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	server := httptest.NewServer(New(path))
	defer server.Close()
	now := time.Now().UTC()
	appendSample := func(session string, percent float64, reset time.Time) {
		t.Helper()
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = fmt.Fprintf(file, `{"observed_at":%q,"session_id":%q,"five_hour":{"used_percentage":%v,"resets_at":%d}}`+"\n", now.Format(time.RFC3339Nano), session, percent, reset.Unix())
		_ = file.Close()
		if err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Second)
	}
	read := func() (string, float64) {
		t.Helper()
		resp, err := http.Get(server.URL + "/api/v1/pricing/live-quota?agent=claude-code")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var got struct {
			Match   string
			Windows []struct{ UsedPercent float64 }
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if len(got.Windows) == 0 {
			return got.Match, -1
		}
		return got.Match, got.Windows[0].UsedPercent
	}
	reset := now.Add(time.Hour)
	appendSample("first", 23, reset)
	if match, percent := read(); match != "likely" || percent != 23 {
		t.Fatalf("first: %s %v", match, percent)
	}
	appendSample("first", 24, reset)
	if match, percent := read(); match != "likely" || percent != 24 {
		t.Fatalf("refresh: %s %v", match, percent)
	}
	appendSample("second", 24, reset.Add(20*time.Minute))
	if match, _ := read(); match != "unavailable" {
		t.Fatalf("conflicting account became %s", match)
	}
}
