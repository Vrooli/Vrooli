package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPHeavyJobSourceReadsOwnerSnapshot(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != imageToolsHeavyJobTelemetryPath {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = fmt.Fprintf(w, `{"observed_at":%q,"complete":true,"jobs":[{"owner_id":"image-tools","job_id":"job-9","operation":"text_to_image","state":"running","started_at":%q,"waiter_count":0,"heavy":true}]}`,
			now.Format(time.RFC3339), now.Add(-20*time.Minute).Format(time.RFC3339))
	}))
	defer server.Close()
	source, err := NewHTTPHeavyJobSource(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := source.GetHeavyJobs(context.Background())
	if err != nil || !snapshot.Complete || len(snapshot.Jobs) != 1 || snapshot.Jobs[0].JobID != "job-9" || snapshot.Jobs[0].WaiterCount != 0 || !snapshot.Jobs[0].StartedAt.Equal(now.Add(-20*time.Minute)) {
		t.Fatalf("snapshot = %+v, err=%v", snapshot, err)
	}
}

func TestHTTPHeavyJobSourceRejectsInvalidURLAndOwnerFailure(t *testing.T) {
	if _, err := NewHTTPHeavyJobSource("file:///tmp/telemetry", nil); err == nil {
		t.Fatal("file URL accepted")
	}
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	source, err := NewHTTPHeavyJobSource(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.GetHeavyJobs(context.Background()); err == nil {
		t.Fatal("non-200 owner response accepted")
	}
}
