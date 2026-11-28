package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleCalendarAPI = "https://www.googleapis.com/calendar/v3"

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type GoogleAdapter struct {
	client               HTTPDoer
	baseURL, accessToken string
}

func NewGoogleAdapter(client HTTPDoer, accessToken string) *GoogleAdapter {
	return &GoogleAdapter{client: client, baseURL: googleCalendarAPI, accessToken: strings.TrimSpace(accessToken)}
}

type GoogleCalendar struct{ ID, Summary, TimeZone string }

type ImportedEvent struct {
	RemoteID, LocalDate, Title, Status string
	StartMinutes, DurationMinutes      int
	Busy                               bool
}

type GoogleSyncPage struct {
	Events        []ImportedEvent
	NextSyncToken string
	ResetRequired bool
}

var ErrGoogleSyncResetRequired = errors.New("google calendar sync token is no longer valid; full resync required")

func (a *GoogleAdapter) ListCalendars(ctx context.Context) ([]GoogleCalendar, error) {
	var out []GoogleCalendar
	pageToken := ""
	for {
		query := url.Values{"maxResults": {"250"}, "showDeleted": {"false"}}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var page struct {
			Items         []struct{ ID, Summary, TimeZone string } `json:"items"`
			NextPageToken string                                   `json:"nextPageToken"`
		}
		if err := a.get(ctx, "/users/me/calendarList", query, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			out = append(out, GoogleCalendar{ID: item.ID, Summary: item.Summary, TimeZone: item.TimeZone})
		}
		if page.NextPageToken == "" {
			return out, nil
		}
		pageToken = page.NextPageToken
	}
}

func (a *GoogleAdapter) ListEvents(ctx context.Context, calendarID, syncToken string) (GoogleSyncPage, error) {
	if strings.TrimSpace(calendarID) == "" {
		return GoogleSyncPage{}, fmt.Errorf("calendar id is required")
	}
	pageToken := ""
	result := GoogleSyncPage{}
	for {
		query := url.Values{"maxResults": {"250"}, "singleEvents": {"true"}, "showDeleted": {"true"}}
		if syncToken != "" {
			query.Set("syncToken", syncToken)
		}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		var page struct {
			Items []struct {
				ID, Status, Summary, Transparency string
				Start                             struct{ Date, DateTime string } `json:"start"`
				End                               struct{ Date, DateTime string } `json:"end"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
			NextSyncToken string `json:"nextSyncToken"`
		}
		if err := a.get(ctx, "/calendars/"+url.PathEscape(calendarID)+"/events", query, &page); err != nil {
			if errors.Is(err, ErrGoogleSyncResetRequired) {
				result.ResetRequired = true
			}
			return result, err
		}
		for _, item := range page.Items {
			event, ok := normalizeGoogleEvent(item.ID, item.Status, item.Summary, item.Transparency, item.Start.Date, item.Start.DateTime, item.End.Date, item.End.DateTime)
			if ok {
				result.Events = append(result.Events, event)
			}
		}
		if page.NextPageToken != "" {
			pageToken = page.NextPageToken
			continue
		}
		result.NextSyncToken = page.NextSyncToken
		return result, nil
	}
}

func normalizeGoogleEvent(id, status, summary, transparency, startDate, startDateTime, endDate, endDateTime string) (ImportedEvent, bool) {
	if strings.TrimSpace(id) == "" {
		return ImportedEvent{}, false
	}
	event := ImportedEvent{RemoteID: id, Title: strings.TrimSpace(summary), Status: status, Busy: transparency != "transparent" && status != "cancelled"}
	if startDate != "" {
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return ImportedEvent{}, false
		}
		end, err := time.Parse("2006-01-02", endDate)
		if err != nil || !end.After(start) {
			return ImportedEvent{}, false
		}
		event.LocalDate, event.StartMinutes, event.DurationMinutes = start.Format("2006-01-02"), 0, int(end.Sub(start).Minutes())
		return event, true
	}
	start, err := time.Parse(time.RFC3339, startDateTime)
	if err != nil {
		return ImportedEvent{}, false
	}
	end, err := time.Parse(time.RFC3339, endDateTime)
	if err != nil || !end.After(start) {
		return ImportedEvent{}, false
	}
	event.LocalDate = start.Format("2006-01-02")
	event.StartMinutes = start.Hour()*60 + start.Minute()
	event.DurationMinutes = int(end.Sub(start).Minutes())
	return event, true
}

func (a *GoogleAdapter) get(ctx context.Context, path string, query url.Values, target any) error {
	if a.client == nil {
		return fmt.Errorf("google adapter HTTP client is required")
	}
	endpoint := strings.TrimRight(a.baseURL, "/") + path + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if a.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.accessToken)
	}
	response, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusGone {
		return ErrGoogleSyncResetRequired
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("google calendar returned HTTP %s", response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode google calendar response: %w", err)
	}
	return nil
}
