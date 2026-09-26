package integrations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleAdapterListsCalendarsAndNormalizesEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("authorization=%q", got)
		}
		if r.URL.Path == "/users/me/calendarList" {
			if r.URL.Query().Get("pageToken") == "next" {
				_, _ = w.Write([]byte(`{"items":[{"id":"work","summary":"Work","timeZone":"America/New_York"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"home","summary":"Home","timeZone":"America/New_York"}],"nextPageToken":"next"}`))
			return
		}
		if r.URL.Path == "/calendars/work/events" {
			if r.URL.Query().Get("syncToken") != "" {
				t.Errorf("initial fixture unexpectedly used sync token")
			}
			_, _ = w.Write([]byte(`{"items":[{"id":"event-1","status":"confirmed","summary":"Deep work","start":{"dateTime":"2026-09-19T09:30:00-04:00"},"end":{"dateTime":"2026-09-19T10:45:00-04:00"}},{"id":"all-day","status":"confirmed","summary":"Holiday","start":{"date":"2026-09-20"},"end":{"date":"2026-09-21"}},{"id":"free","status":"confirmed","summary":"Optional","transparency":"transparent","start":{"dateTime":"2026-09-19T11:00:00Z"},"end":{"dateTime":"2026-09-19T11:30:00Z"}}],"nextSyncToken":"cursor-1"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter := NewGoogleAdapter(server.Client(), "secret-token")
	adapter.baseURL = server.URL
	calendars, err := adapter.ListCalendars(context.Background())
	if err != nil || len(calendars) != 2 {
		t.Fatalf("calendars=%#v err=%v", calendars, err)
	}
	page, err := adapter.ListEvents(context.Background(), "work", "")
	if err != nil || page.NextSyncToken != "cursor-1" || len(page.Events) != 3 {
		t.Fatalf("page=%#v err=%v", page, err)
	}
	if page.Events[0].LocalDate != "2026-09-19" || page.Events[0].StartMinutes != 570 || page.Events[0].DurationMinutes != 75 || !page.Events[0].Busy {
		t.Fatalf("event=%#v", page.Events[0])
	}
	if page.Events[2].Busy {
		t.Fatalf("transparent event should not consume capacity: %#v", page.Events[2])
	}
}

func TestGoogleAdapterTreatsGoneAsFullReset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, `{"error":"expired"}`, http.StatusGone) }))
	defer server.Close()
	adapter := NewGoogleAdapter(server.Client(), "token")
	adapter.baseURL = server.URL
	page, err := adapter.ListEvents(context.Background(), "work", "expired")
	if !strings.Contains(err.Error(), "full resync") || !page.ResetRequired {
		t.Fatalf("page=%#v err=%v", page, err)
	}
}
