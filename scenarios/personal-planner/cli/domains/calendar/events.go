package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
)

type calendarEvent struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Subject          string `json:"subject"`
	Notes            string `json:"notes"`
	Availability     string `json:"availability"`
	Timezone         string `json:"timezone"`
	AllDay           bool   `json:"all_day"`
	StartDate        string `json:"start_date"`
	EndDateExclusive string `json:"end_date_exclusive"`
	StartAt          string `json:"start_at"`
	EndAt            string `json:"end_at"`
	Provider         string `json:"provider"`
	ProviderCalendar string `json:"provider_calendar_id"`
	ProviderEvent    string `json:"provider_event_id"`
	Occurrence       string `json:"occurrence_id"`
	Revision         int64  `json:"revision"`
}

const testModeEnv = "PERSONAL_PLANNER_TEST_MODE"

func (h *handlers) listEventsCall(c cliapp.OperationContext) (*v.ListEventsResponse, error) {
	response, err := h.eventClient.ListEvents(context.Background(), connect.NewRequest(&v.ListEventsRequest{StartLocalDate: c.Flag("start-date"), EndLocalDate: c.Flag("end-date")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list calendar events", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listEventsReport(_ cliapp.OperationContext, response *v.ListEventsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.Events))
	for _, event := range response.Events {
		when := event.StartDate + "–" + event.EndDateExclusive + " (end exclusive)"
		if !event.AllDay {
			when = event.StartAt + " – " + event.EndAt + " " + event.Timezone
		}
		rows = append(rows, fmt.Sprintf("%s | %s | %s | %s | %s | rev %d | source %s/%s/%s/%s", event.Id, event.Title, event.Subject, when, event.Availability, event.Revision, event.Provider, event.ProviderCalendarId, event.ProviderEventId, event.OccurrenceId))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Calendar events: %d", len(rows))}, ResultsHeading: "Events", Results: rows}
}

func (h *handlers) getEventCall(c cliapp.OperationContext) (*v.GetEventResponse, error) {
	response, err := h.eventClient.GetEvent(context.Background(), connect.NewRequest(&v.GetEventRequest{EventId: c.Flag("event-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get calendar event", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) eventStatusCall(c cliapp.OperationContext) (calendarEvent, error) {
	var event calendarEvent
	err := h.eventRequest(context.Background(), http.MethodGet, "/api/v1/calendar/event-commands/"+url.PathEscape(c.Flag("idempotency-key")), nil, &event)
	return event, err
}

func (h *handlers) eventStatusReport(_ cliapp.OperationContext, event calendarEvent) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Event %s: %s (revision %d)", event.ID, event.Title, event.Revision)}, Changes: []string{fmt.Sprintf("subject=%s availability=%s timezone=%s", event.Subject, event.Availability, event.Timezone)}}
}

func (h *handlers) eventReport(_ cliapp.OperationContext, response *v.GetEventResponse) cliapp.ListReport {
	event := response.Event
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Event %s at revision %d", event.Id, event.Revision)}, ResultsHeading: "Event", Results: []string{fmt.Sprintf("%s | %s | %s | %s", event.Title, event.Subject, event.Availability, event.Timezone)}}
}

func (h *handlers) createEventCall(c cliapp.OperationContext) (*v.CreateEventResponse, error) {
	event, err := eventFromFlags(c)
	if err != nil {
		return nil, err
	}
	response, err := h.eventClient.CreateEvent(context.Background(), connect.NewRequest(&v.CreateEventRequest{Event: event, IdempotencyKey: c.Flag("idempotency-key")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create calendar event", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) updateEventCall(c cliapp.OperationContext) (*v.UpdateEventResponse, error) {
	event, err := eventFromFlags(c)
	if err != nil {
		return nil, err
	}
	revision, err := strconv.ParseInt(c.Flag("revision"), 10, 64)
	if err != nil || revision < 1 {
		return nil, fmt.Errorf("revision must be a positive integer")
	}
	event.Id = c.Flag("event-id")
	response, err := h.eventClient.UpdateEvent(context.Background(), connect.NewRequest(&v.UpdateEventRequest{Event: event, ExpectedRevision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("update calendar event", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) createEventReport(_ cliapp.OperationContext, response *v.CreateEventResponse) cliapp.MutationReport {
	event := response.Event
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved event %s at revision %d.", event.Id, event.Revision)}, Changes: []string{fmt.Sprintf("%s — %s", event.Title, event.Subject)}, NextCommand: []string{"`calendar get-event --event-id " + event.Id + "` — read the durable event"}}
}

func (h *handlers) updateEventReport(_ cliapp.OperationContext, response *v.UpdateEventResponse) cliapp.MutationReport {
	event := response.Event
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Saved event %s at revision %d.", event.Id, event.Revision)}, Changes: []string{fmt.Sprintf("%s — %s", event.Title, event.Subject)}, NextCommand: []string{"`calendar get-event --event-id " + event.Id + "` — read the durable event"}}
}

func eventFromFlags(c cliapp.OperationContext) (*v.CalendarEvent, error) {
	allDay, err := strconv.ParseBool(c.Flag("all-day"))
	if err != nil {
		return nil, fmt.Errorf("all-day must be true or false")
	}
	return &v.CalendarEvent{Title: c.Flag("title"), Subject: c.Flag("subject"), Notes: c.Flag("notes"), Availability: c.Flag("availability"), Timezone: c.Flag("timezone"), AllDay: allDay,
		StartDate: c.Flag("start-date"), EndDateExclusive: c.Flag("end-date-exclusive"), StartAt: c.Flag("start-at"), EndAt: c.Flag("end-at")}, nil
}

func (h *handlers) eventRequest(ctx context.Context, method, path string, body any, out any) error {
	if strings.TrimSpace(h.apiBase) == "" || h.httpClient == nil {
		return fmt.Errorf("Personal Planner API client is not configured")
	}
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode calendar event request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(h.apiBase, "/")+path, requestBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if os.Getenv(testModeEnv) == "1" {
		req.Header.Set("X-Vrooli-Test-Mode", "1")
	}
	response, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call Personal Planner calendar event API: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("calendar event API returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(out); err != nil {
		return fmt.Errorf("decode calendar event API response: %w", err)
	}
	return nil
}
