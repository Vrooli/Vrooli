package metricshttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	feedbackhttp "landing-page-business-suite-api/handlers/feedback"
	metrics "landing-page-business-suite-api/internal/metrics"
)

func asFeedbackDependencies(dependencies Dependencies) feedbackhttp.Dependencies {
	return feedbackhttp.Dependencies{
		DecodeJSON:     dependencies.DecodeJSON,
		PathInt:        dependencies.PathInt,
		WriteErrorType: dependencies.WriteErrorType,
		WriteJSON:      dependencies.WriteJSON,
		LogError:       dependencies.LogError,
	}
}

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

type feedbackFake struct {
	requests []metrics.FeedbackRequest
}

func (f *feedbackFake) Create(context.Context, *metrics.CreateFeedbackInput) (*metrics.FeedbackRequest, error) {
	return nil, nil
}

func (f *feedbackFake) List(context.Context, string) ([]metrics.FeedbackRequest, error) {
	return f.requests, nil
}

func (f *feedbackFake) GetByID(_ context.Context, id int) (*metrics.FeedbackRequest, error) {
	for _, request := range f.requests {
		if request.ID == id {
			return &request, nil
		}
	}
	return nil, errors.New("not found")
}

func (f *feedbackFake) UpdateStatus(context.Context, int, string) (*metrics.FeedbackRequest, error) {
	return nil, nil
}
func (f *feedbackFake) Delete(context.Context, int) error                { return nil }
func (f *feedbackFake) DeleteBulk(context.Context, []int) (int64, error) { return 0, nil }

type feedbackCreateFake struct{ input metrics.CreateFeedbackInput }

func (f *feedbackCreateFake) Create(_ context.Context, input *metrics.CreateFeedbackInput) (*metrics.FeedbackRequest, error) {
	f.input = *input
	return &metrics.FeedbackRequest{ID: 7, Type: input.Type, Email: input.Email}, nil
}

func (f *feedbackCreateFake) List(context.Context, string) ([]metrics.FeedbackRequest, error) {
	return nil, nil
}

func (f *feedbackCreateFake) GetByID(context.Context, int) (*metrics.FeedbackRequest, error) {
	return nil, nil
}

func (f *feedbackCreateFake) UpdateStatus(context.Context, int, string) (*metrics.FeedbackRequest, error) {
	return nil, nil
}
func (f *feedbackCreateFake) Delete(context.Context, int) error                { return nil }
func (f *feedbackCreateFake) DeleteBulk(context.Context, []int) (int64, error) { return 0, nil }

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

func TestListFeedbackSerializesAnEmptyArrayInsteadOfNull(t *testing.T) {
	deps := Dependencies{
		WriteJSON: func(w http.ResponseWriter, value interface{}) error { return json.NewEncoder(w).Encode(value) },
		LogError:  func(string, map[string]interface{}) {},
	}
	response := httptest.NewRecorder()

	feedbackhttp.List(asFeedbackDependencies(deps), &feedbackFake{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/admin/feedback", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := strings.TrimSpace(response.Body.String()); body != "[]" {
		t.Fatalf("body = %s, want []", body)
	}
}

func TestCreateFeedbackDefaultsTheTypeAndNotifiesAfterPersistence(t *testing.T) {
	service := &feedbackCreateFake{}
	notifier := &feedbackNotifierFake{}
	deps := Dependencies{
		DecodeJSON: func(_ http.ResponseWriter, r *http.Request, target interface{}) bool {
			return json.NewDecoder(r.Body).Decode(target) == nil
		},
		WriteJSON: func(w http.ResponseWriter, value interface{}) error { return json.NewEncoder(w).Encode(value) },
		LogError:  func(string, map[string]interface{}) {},
	}
	response := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/feedback", strings.NewReader(`{"type":"unexpected","email":"customer@example.com","subject":"Question","message":"Hello"}`))

	feedbackhttp.Create(asFeedbackDependencies(deps), service, notifier).ServeHTTP(response, req)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if service.input.Type != "general" {
		t.Fatalf("type = %q, want general", service.input.Type)
	}
	if len(notifier.notifications) != 1 || notifier.notifications[0].ID != 7 {
		t.Fatalf("notifications = %+v", notifier.notifications)
	}
}
