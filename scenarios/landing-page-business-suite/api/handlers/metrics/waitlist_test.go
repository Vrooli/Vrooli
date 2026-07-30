package metricshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	metrics "landing-page-business-suite-api/internal/metrics"
)

type waitlistFake struct{ created metrics.WaitlistEmail }

func (f *waitlistFake) Create(_ context.Context, email, source string) (*metrics.WaitlistEmail, error) {
	f.created = metrics.WaitlistEmail{ID: 1, Email: email, Source: source, CreatedAt: time.Now()}
	return &f.created, nil
}
func (f *waitlistFake) List(context.Context) ([]metrics.WaitlistEmail, error) { return nil, nil }
func (f *waitlistFake) Delete(context.Context, int64) error                   { return nil }
func (f *waitlistFake) Count(context.Context) (int64, error)                  { return 0, nil }

func TestCreateWaitlistNormalizesTheTransportInput(t *testing.T) {
	fake := &waitlistFake{}
	deps := Dependencies{
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, dst interface{}) bool {
			return json.NewDecoder(r.Body).Decode(dst) == nil
		},
		ValidateEmail: func(_ http.ResponseWriter, email string) (string, bool) {
			return strings.ToLower(strings.TrimSpace(email)), true
		},
		WriteSuccess: func(w http.ResponseWriter, _ string) { w.WriteHeader(http.StatusCreated) },
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/waitlist", strings.NewReader(`{"email":" USER@example.com "}`))
	response := httptest.NewRecorder()

	deps.CreateWaitlist(fake).ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if fake.created.Email != "user@example.com" || fake.created.Source != "coming_soon" {
		t.Fatalf("created = %+v", fake.created)
	}
}

type trackerFake struct {
	event metrics.Event
	err   error
}

type feedbackNotifierFake struct{ notifications []*metrics.FeedbackRequest }

func (f *feedbackNotifierFake) Notify(feedback *metrics.FeedbackRequest) {
	f.notifications = append(f.notifications, feedback)
}

func (f *trackerFake) TrackEvent(event metrics.Event) error {
	f.event = event
	return f.err
}

func TestTrackUsesTheDomainContractAndPreservesCreatedResponse(t *testing.T) {
	fake := &trackerFake{}
	deps := Dependencies{
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, dst interface{}) bool {
			return json.NewDecoder(r.Body).Decode(dst) == nil
		},
		WriteJSON: func(w http.ResponseWriter, value interface{}) error {
			return json.NewEncoder(w).Encode(value)
		},
		LogError: func(string, map[string]interface{}) {},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader(`{"event_type":"page_view","variant_slug":"control","session_id":"session-1"}`))
	response := httptest.NewRecorder()

	deps.Track(fake).ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if fake.event.EventType != "page_view" || fake.event.VariantSlug != "control" {
		t.Fatalf("tracked event = %+v", fake.event)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
}

func TestTrackMapsDomainValidationErrorsToThePublicValidationResponse(t *testing.T) {
	fake := &trackerFake{err: &metrics.ValidationError{Field: "event_type", Reason: "event_type is required"}}
	var gotStatus int
	var gotMessage, gotType string
	deps := Dependencies{
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, dst interface{}) bool {
			return json.NewDecoder(r.Body).Decode(dst) == nil
		},
		WriteErrorType: func(_ http.ResponseWriter, status int, message, errorType string) {
			gotStatus, gotMessage, gotType = status, message, errorType
		},
		LogError: func(string, map[string]interface{}) {},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/track", strings.NewReader(`{}`))

	deps.Track(fake).ServeHTTP(httptest.NewRecorder(), req)

	if gotStatus != http.StatusBadRequest || gotMessage != "event_type is required" || gotType != "validation" {
		t.Fatalf("error response = (%d, %q, %q)", gotStatus, gotMessage, gotType)
	}
}
