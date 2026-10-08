package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	d "personal-planner/internal/calendar"

	"github.com/gorilla/mux"
)

type httpEventRepository struct {
	events    map[string]d.Event
	eventKeys map[string]string
}

func (r *httpEventRepository) ListEvents(_ context.Context, start, end string) ([]d.Event, error) {
	var out []d.Event
	for _, event := range r.events {
		if event.AllDay && event.StartDate <= end && event.EndDateExclusive > start {
			out = append(out, event)
		}
	}
	return out, nil
}
func (r *httpEventRepository) GetEvent(_ context.Context, id string) (d.Event, error) {
	event, ok := r.events[id]
	if !ok {
		return d.Event{}, d.ErrEventNotFound{ID: id}
	}
	return event, nil
}
func (r *httpEventRepository) GetEventByIdempotencyKey(_ context.Context, key string) (d.Event, error) {
	return r.GetEvent(context.Background(), r.eventKeys[key])
}
func (r *httpEventRepository) CreateEvent(_ context.Context, input d.CreateEventInput) (d.Event, error) {
	event := input.Event
	event.ID, event.Revision = "event-http-1", 1
	event.CreatedAt, event.UpdatedAt = time.Unix(1, 0), time.Unix(1, 0)
	r.events[event.ID] = event
	if r.eventKeys == nil {
		r.eventKeys = map[string]string{}
	}
	r.eventKeys[input.IdempotencyKey] = event.ID
	return event, nil
}
func (r *httpEventRepository) UpdateEvent(_ context.Context, input d.UpdateEventInput) (d.Event, error) {
	current, ok := r.events[input.ID]
	if !ok {
		return d.Event{}, d.ErrEventNotFound{ID: input.ID}
	}
	if current.Revision != input.ExpectedRevision {
		return d.Event{}, d.ErrEventRevisionConflict{Expected: input.ExpectedRevision, Current: current.Revision}
	}
	event := input.Event
	event.ID, event.Revision, event.CreatedAt, event.UpdatedAt = current.ID, current.Revision+1, current.CreatedAt, current.UpdatedAt.Add(time.Minute)
	r.events[event.ID] = event
	return event, nil
}

func TestCalendarEventHTTPCreateEditReopenAndList(t *testing.T) {
	repository := &httpEventRepository{events: map[string]d.Event{}}
	router := mux.NewRouter()
	mountEventRoutes(router, d.NewEventService(repository))
	createBody := `{"title":"Synthetic trip","subject":"owner","availability":"busy","timezone":"America/New_York","all_day":true,"start_date":"2026-10-04","end_date_exclusive":"2026-10-07","idempotency_key":"synthetic-http-1"}`
	createdResponse := httptest.NewRecorder()
	router.ServeHTTP(createdResponse, httptest.NewRequest(http.MethodPost, "/api/v1/calendar/events", strings.NewReader(createBody)))
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var created d.Event
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	statusResponse := httptest.NewRecorder()
	router.ServeHTTP(statusResponse, httptest.NewRequest(http.MethodGet, "/api/v1/calendar/event-commands/synthetic-http-1", nil))
	var statusEvent d.Event
	if statusResponse.Code != http.StatusOK || json.Unmarshal(statusResponse.Body.Bytes(), &statusEvent) != nil || statusEvent.ID != created.ID {
		t.Fatalf("idempotency readback status=%d event=%#v body=%s", statusResponse.Code, statusEvent, statusResponse.Body.String())
	}
	editBody := `{"title":"Synthetic trip edited","subject":"owner","availability":"busy","timezone":"America/New_York","all_day":true,"start_date":"2026-10-04","end_date_exclusive":"2026-10-07","expected_revision":1}`
	editResponse := httptest.NewRecorder()
	router.ServeHTTP(editResponse, httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/"+created.ID, strings.NewReader(editBody)))
	if editResponse.Code != http.StatusOK {
		t.Fatalf("edit status=%d body=%s", editResponse.Code, editResponse.Body.String())
	}
	readResponse := httptest.NewRecorder()
	router.ServeHTTP(readResponse, httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/"+created.ID, nil))
	var reopened d.Event
	if readResponse.Code != http.StatusOK || json.Unmarshal(readResponse.Body.Bytes(), &reopened) != nil || reopened.ID != created.ID || reopened.Revision != 2 || reopened.Title != "Synthetic trip edited" {
		t.Fatalf("reopen status=%d event=%#v body=%s", readResponse.Code, reopened, readResponse.Body.String())
	}
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events?start_local_date=2026-10-05&end_local_date=2026-10-05", nil))
	var listed eventListResponse
	if listResponse.Code != http.StatusOK || json.Unmarshal(listResponse.Body.Bytes(), &listed) != nil || len(listed.Events) != 1 || listed.Events[0].ID != created.ID {
		t.Fatalf("list status=%d events=%#v body=%s", listResponse.Code, listed.Events, listResponse.Body.String())
	}
}

func TestCalendarEventHTTPRejectsStaleRevision(t *testing.T) {
	repository := &httpEventRepository{events: map[string]d.Event{"event-1": {
		ID: "event-1", Title: "Original", Subject: "owner", Availability: "busy", Timezone: "America/New_York",
		AllDay: true, StartDate: "2026-10-04", EndDateExclusive: "2026-10-05", Revision: 3,
	}}}
	router := mux.NewRouter()
	mountEventRoutes(router, d.NewEventService(repository))
	body := `{"title":"stale edit","subject":"owner","availability":"busy","timezone":"America/New_York","all_day":true,"start_date":"2026-10-04","end_date_exclusive":"2026-10-05","expected_revision":2}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/calendar/events/event-1", strings.NewReader(body)))
	if response.Code != http.StatusConflict {
		t.Fatalf("stale edit status=%d body=%s", response.Code, response.Body.String())
	}
}
