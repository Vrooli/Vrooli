package opencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// V2Client talks to the OpenCode 2 HTTP API. It projects the V2 session/message
// shapes onto the same wire model V1 uses, so the watcher and normalizer stay
// version-independent.
type V2Client struct {
	BaseURL  string
	Password string
	HTTP     *http.Client
}

func NewV2HTTPClient(baseURL, password string) *V2Client {
	return &V2Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Password: password,
		HTTP:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *V2Client) request(ctx context.Context, method, path string, body, out any) error {
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	setBasicAuth(req, c.Password)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("opencode %s: status %d", path, resp.StatusCode)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *V2Client) ListSessions(ctx context.Context, directory string) ([]Session, error) {
	var sessions []Session
	cursor := ""
	for {
		query := url.Values{"limit": []string{"200"}}
		query.Set("order", "asc")
		if directory != "" {
			query.Set("directory", directory)
		}
		if cursor != "" {
			query.Del("order")
			query.Set("cursor", cursor)
		}
		var page struct {
			Data []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Location struct {
					Directory string `json:"directory"`
				} `json:"location"`
				Time struct {
					Created int64 `json:"created"`
					Updated int64 `json:"updated"`
				} `json:"time"`
				Revert *struct {
					MessageID string `json:"messageID"`
					PartID    string `json:"partID"`
				} `json:"revert"`
			} `json:"data"`
			Cursor struct {
				Next string `json:"next"`
			} `json:"cursor"`
		}
		if err := c.request(ctx, http.MethodGet, "/api/session?"+query.Encode(), nil, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Data {
			session := Session{ID: item.ID, Directory: item.Location.Directory, Title: item.Title}
			session.Time.Created = item.Time.Created
			session.Time.Updated = item.Time.Updated
			if item.Revert != nil {
				session.Revert = &struct {
					MessageID string `json:"messageID"`
					PartID    string `json:"partID"`
				}{MessageID: item.Revert.MessageID, PartID: item.Revert.PartID}
			}
			sessions = append(sessions, session)
		}
		if page.Cursor.Next == "" {
			return sessions, nil
		}
		cursor = page.Cursor.Next
	}
}

func (c *V2Client) SessionMessages(ctx context.Context, sessionID string) ([]MessageWithParts, error) {
	var messages []MessageWithParts
	cursor := ""
	for {
		query := url.Values{"limit": []string{"200"}, "order": []string{"asc"}}
		if cursor != "" {
			query.Del("order")
			query.Set("cursor", cursor)
		}
		var page struct {
			Data []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
				Text string `json:"text"`
				Time struct {
					Created   int64 `json:"created"`
					Completed int64 `json:"completed"`
				} `json:"time"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"data"`
			Cursor struct {
				Next string `json:"next"`
			} `json:"cursor"`
		}
		path := "/api/session/" + url.PathEscape(sessionID) + "/message?" + query.Encode()
		if err := c.request(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Data {
			message := MessageWithParts{}
			message.Info.ID = item.ID
			message.Info.SessionID = sessionID
			message.Info.Time.Created = item.Time.Created
			message.Info.Time.Completed = item.Time.Completed
			switch item.Type {
			case "user":
				message.Info.Role = "user"
				message.Parts = append(message.Parts, Part{ID: item.ID + "_text", MessageID: item.ID, Type: "text", Text: item.Text})
			case "assistant":
				message.Info.Role = "assistant"
				for index, content := range item.Content {
					if content.Type != "text" || strings.TrimSpace(content.Text) == "" {
						continue
					}
					id := fmt.Sprintf("%s_%d", item.ID, index)
					message.Parts = append(message.Parts, Part{ID: id, MessageID: item.ID, Type: "text", Text: content.Text})
				}
			default:
				continue
			}
			messages = append(messages, message)
		}
		if page.Cursor.Next == "" {
			return messages, nil
		}
		cursor = page.Cursor.Next
	}
}

func (c *V2Client) SessionStatus(ctx context.Context) (map[string]SessionStatus, error) {
	var page struct {
		Data map[string]struct {
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := c.request(ctx, http.MethodGet, "/api/session/active", nil, &page); err != nil {
		return nil, err
	}
	status := make(map[string]SessionStatus, len(page.Data))
	for sessionID, item := range page.Data {
		if item.Type == "running" {
			status[sessionID] = SessionStatus{Type: "busy"}
		}
	}
	return status, nil
}

func (c *V2Client) AbortSession(ctx context.Context, sessionID string) error {
	path := "/api/session/" + url.PathEscape(sessionID) + "/interrupt?resume=false"
	return c.request(ctx, http.MethodPost, path, nil, nil)
}

func (c *V2Client) RevertMessage(ctx context.Context, sessionID, messageID, _ string) error {
	path := "/api/session/" + url.PathEscape(sessionID) + "/revert/stage"
	return c.request(ctx, http.MethodPost, path, map[string]any{"messageID": messageID, "files": false}, nil)
}

func (c *V2Client) ReplyPermission(ctx context.Context, sessionID, requestID, reply string) error {
	path := "/api/session/" + url.PathEscape(sessionID) + "/permission/" + url.PathEscape(requestID) + "/reply"
	return c.request(ctx, http.MethodPost, path, map[string]string{"decision": reply}, nil)
}

func (c *V2Client) Events(ctx context.Context, onEvent func(Event)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/event", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	setBasicAuth(req, c.Password)
	streamClient := &http.Client{Timeout: 0}
	resp, err := streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("opencode /api/event: status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		var item v2Event
		if err := json.Unmarshal([]byte(payload), &item); err != nil {
			continue
		}
		onEvent(mapV2Event(item))
	}
	return scanner.Err()
}

type v2Event struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

func mapV2Event(item v2Event) Event {
	properties := item.Data
	if properties == nil {
		properties = map[string]any{}
	}
	eventType := item.Type
	switch item.Type {
	case "permission.asked":
		if action, _ := properties["action"].(string); action != "" {
			properties["permission"] = action
		}
		if resources, ok := properties["resources"]; ok {
			properties["patterns"] = resources
		}
	case "session.execution.started":
		eventType = "session.status"
		properties["status"] = map[string]any{"type": "busy"}
	case "session.execution.succeeded", "session.execution.failed", "session.execution.interrupted":
		eventType = "session.status"
		properties["status"] = map[string]any{"type": "idle"}
	}
	return Event{Type: eventType, Properties: properties}
}

// NewClient returns the API client matching the server's major version.
func NewClient(server Server) Client {
	if server.Major >= 2 {
		return NewV2HTTPClient(server.URL, server.Password)
	}
	return NewHTTPClient(server.URL)
}
