package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"switchboard/internal/channels"
)

type Adapter struct {
	appToken string
	botToken string
	baseURL  string
	client   *http.Client
}

func New() *Adapter {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("SLACK_API_BASE_URL")), "/")
	if base == "" {
		base = "https://slack.com/api"
	}
	return NewWithConfig(os.Getenv("SLACK_APP_TOKEN"), os.Getenv("SLACK_BOT_TOKEN"), base, http.DefaultClient)
}

func NewWithConfig(appToken, botToken, baseURL string, client *http.Client) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{appToken: strings.TrimSpace(appToken), botToken: strings.TrimSpace(botToken), baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: client}
}

func (a *Adapter) ID() string { return "slack" }

func (a *Adapter) Connect(ctx context.Context, receive func(channels.Envelope) error) error {
	if a.appToken == "" || a.botToken == "" {
		return fmt.Errorf("slack app and bot tokens are unavailable")
	}
	var opened struct {
		OK  bool   `json:"ok"`
		URL string `json:"url"`
	}
	if err := a.api(ctx, "apps.connections.open", nil, a.appToken, &opened); err != nil {
		return err
	}
	if !opened.OK || opened.URL == "" {
		return fmt.Errorf("slack Socket Mode URL is unavailable")
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, opened.URL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		var envelope struct {
			EnvelopeID string         `json:"envelope_id"`
			Type       string         `json:"type"`
			Payload    map[string]any `json:"payload"`
		}
		if err := conn.ReadJSON(&envelope); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if envelope.EnvelopeID != "" {
			_ = conn.WriteJSON(map[string]string{"envelope_id": envelope.EnvelopeID})
		}
		if envelope.Type != "events_api" {
			continue
		}
		event, _ := envelope.Payload["event"].(map[string]any)
		e, ok := normalize(event)
		if ok {
			if err := receive(e); err != nil {
				return err
			}
		}
	}
}

func normalize(event map[string]any) (channels.Envelope, bool) {
	if event == nil || event["type"] != "message" || event["subtype"] == "message_changed" {
		return channels.Envelope{}, false
	}
	channel, _ := event["channel"].(string)
	ts, _ := event["ts"].(string)
	user, _ := event["user"].(string)
	text, _ := event["text"].(string)
	thread, _ := event["thread_ts"].(string)
	if channel == "" || ts == "" {
		return channels.Envelope{}, false
	}
	if thread == "" {
		thread = ts
	}
	author := channels.AuthorHuman
	if _, bot := event["bot_id"]; bot {
		author = channels.AuthorAgent
	}
	return channels.Envelope{ChannelID: "slack", RemoteMessageID: ts, ThreadKey: channel, SenderAddress: user, AuthorKind: author, Text: text, ReplyToRemoteID: thread, ReceivedAt: time.Now().UTC()}, true
}

func (a *Adapter) Send(ctx context.Context, out channels.Outbound) error {
	if a.botToken == "" {
		return fmt.Errorf("slack bot token is unavailable")
	}
	if len(out.Media) > 0 {
		for _, media := range out.Media {
			if strings.TrimSpace(media.URL) == "" || !strings.HasPrefix(media.URL, "http") {
				return fmt.Errorf("slack media %q requires an https URL", media.Name)
			}
			var response struct {
				OK  bool   `json:"ok"`
				Err string `json:"error"`
			}
			if err := a.api(ctx, "files.remote.add", map[string]any{"external_id": media.Name, "external_url": media.URL, "title": media.Name}, a.botToken, &response); err != nil {
				return err
			}
			if !response.OK {
				return fmt.Errorf("slack media upload: %s", response.Err)
			}
			if err := a.api(ctx, "files.remote.share", map[string]any{"external_id": media.Name, "channel_id": out.ThreadKey}, a.botToken, &response); err != nil {
				return err
			}
			if !response.OK {
				return fmt.Errorf("slack media share: %s", response.Err)
			}
		}
		payload := map[string]any{"channel": out.ThreadKey, "text": out.Text}
		if out.ReplyToRemoteID != "" {
			payload["thread_ts"] = out.ReplyToRemoteID
		}
		var response struct {
			OK  bool   `json:"ok"`
			Err string `json:"error"`
		}
		if err := a.api(ctx, "chat.postMessage", payload, a.botToken, &response); err != nil {
			return err
		}
		if !response.OK {
			return fmt.Errorf("slack API: %s", response.Err)
		}
		return nil
	}
	payload := map[string]any{"channel": out.ThreadKey, "text": out.Text}
	if out.ReplyToRemoteID != "" {
		payload["thread_ts"] = out.ReplyToRemoteID
	}
	var response struct {
		OK  bool   `json:"ok"`
		Err string `json:"error"`
	}
	if err := a.api(ctx, "chat.postMessage", payload, a.botToken, &response); err != nil {
		return err
	}
	if !response.OK {
		return fmt.Errorf("slack API: %s", response.Err)
	}
	return nil
}

func (a *Adapter) Probe(ctx context.Context) channels.ProbeResult {
	if a.appToken == "" || a.botToken == "" {
		return channels.ProbeResult{Reason: "slack app and bot tokens are unavailable"}
	}
	var response struct {
		OK bool `json:"ok"`
	}
	if err := a.api(ctx, "auth.test", nil, a.botToken, &response); err != nil || !response.OK {
		if err != nil {
			return channels.ProbeResult{Reason: err.Error()}
		}
		return channels.ProbeResult{Reason: "slack authentication failed"}
	}
	return channels.ProbeResult{Available: true}
}

func (a *Adapter) api(ctx context.Context, method string, payload any, token string, result any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/"+method, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack API returned %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result)
}

var _ channels.Adapter = (*Adapter)(nil)
