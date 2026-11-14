package telegram

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
	"time"

	"switchboard/internal/channels"
)

type Adapter struct {
	token   string
	baseURL string
	client  *http.Client
	offset  int64
}

func New() *Adapter {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("TELEGRAM_API_BASE_URL")), "/")
	if base == "" {
		base = "https://api.telegram.org"
	}
	return &Adapter{token: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")), baseURL: base, client: http.DefaultClient}
}

func NewWithConfig(token, baseURL string, client *http.Client) *Adapter {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.telegram.org"
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{token: strings.TrimSpace(token), baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: client}
}

func (a *Adapter) ID() string { return "telegram" }

func (a *Adapter) Connect(ctx context.Context, receive func(channels.Envelope) error) error {
	if a.token == "" {
		return fmt.Errorf("telegram bot token is unavailable")
	}
	for {
		updates, err := a.getUpdates(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		for _, update := range updates {
			if update.UpdateID >= a.offset {
				a.offset = update.UpdateID + 1
			}
			if update.Message == nil {
				continue
			}
			e, err := a.envelope(ctx, update)
			if err != nil {
				return err
			}
			if err := receive(e); err != nil {
				return err
			}
		}
		if len(updates) == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
}

func (a *Adapter) envelope(ctx context.Context, update update) (channels.Envelope, error) {
	m := update.Message
	chatID := strconv.FormatInt(m.Chat.ID, 10)
	text := m.Text
	if text == "" {
		text = m.Caption
	}
	e := channels.Envelope{ChannelID: "telegram", RemoteMessageID: strconv.FormatInt(m.ID, 10), ThreadKey: chatID, Group: m.Chat.Type == "group" || m.Chat.Type == "supergroup", SenderAddress: strconv.FormatInt(m.From.ID, 10), AuthorKind: channels.AuthorHuman, Text: text, ReceivedAt: time.Unix(m.Date, 0)}
	if m.ReplyTo != nil {
		e.ReplyToRemoteID = strconv.FormatInt(m.ReplyTo.ID, 10)
	}
	if m.Document != nil {
		fileURL, err := a.fileURL(ctx, m.Document.FileID)
		if err != nil {
			return channels.Envelope{}, err
		}
		e.Media = append(e.Media, channels.Media{Name: m.Document.FileName, MIME: m.Document.MIME, URL: fileURL})
	}
	if len(m.Photo) > 0 {
		photo := m.Photo[len(m.Photo)-1]
		fileURL, err := a.fileURL(ctx, photo.FileID)
		if err != nil {
			return channels.Envelope{}, err
		}
		e.Media = append(e.Media, channels.Media{Name: "telegram-photo.jpg", MIME: "image/jpeg", URL: fileURL})
	}
	return e, nil
}

func (a *Adapter) Send(ctx context.Context, out channels.Outbound) error {
	if a.token == "" {
		return fmt.Errorf("telegram bot token is unavailable")
	}
	if strings.TrimSpace(out.ThreadKey) == "" {
		return fmt.Errorf("telegram destination is required")
	}
	if len(out.Media) > 0 {
		for _, media := range out.Media {
			method, field := "sendDocument", "document"
			if strings.HasPrefix(strings.ToLower(media.MIME), "image/") {
				method, field = "sendPhoto", "photo"
			}
			if err := a.call(ctx, method, map[string]any{"chat_id": out.ThreadKey, field: media.URL, "caption": out.Text}); err != nil {
				return err
			}
		}
		return nil
	}
	payload := map[string]any{"chat_id": out.ThreadKey, "text": out.Text}
	if out.ReplyToRemoteID != "" {
		if id, err := strconv.ParseInt(out.ReplyToRemoteID, 10, 64); err == nil {
			payload["reply_parameters"] = map[string]any{"message_id": id}
		}
	}
	return a.call(ctx, "sendMessage", payload)
}

func (a *Adapter) Probe(ctx context.Context) channels.ProbeResult {
	if a.token == "" {
		return channels.ProbeResult{Reason: "telegram bot token is unavailable"}
	}
	if err := a.call(ctx, "getMe", nil); err != nil {
		return channels.ProbeResult{Reason: err.Error()}
	}
	return channels.ProbeResult{Available: true}
}

type update struct {
	UpdateID int64    `json:"update_id"`
	Message  *message `json:"message"`
}
type message struct {
	ID      int64  `json:"message_id"`
	Date    int64  `json:"date"`
	Text    string `json:"text"`
	Caption string `json:"caption"`
	From    struct {
		ID int64 `json:"id"`
	} `json:"from"`
	Chat struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	Photo []struct {
		FileID string `json:"file_id"`
	} `json:"photo"`
	Document *struct {
		FileID   string `json:"file_id"`
		FileName string `json:"file_name"`
		MIME     string `json:"mime_type"`
	} `json:"document"`
	ReplyTo *struct {
		ID int64 `json:"message_id"`
	} `json:"reply_to_message"`
}

func (a *Adapter) fileURL(ctx context.Context, fileID string) (string, error) {
	var response struct {
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}
	if err := a.request(ctx, "getFile", nil, map[string]any{"file_id": fileID}, &response); err != nil {
		return "", err
	}
	if response.Result.FilePath == "" {
		return "", fmt.Errorf("telegram file %q has no path", fileID)
	}
	return a.baseURL + "/file/bot" + a.token + "/" + response.Result.FilePath, nil
}

func (a *Adapter) getUpdates(ctx context.Context) ([]update, error) {
	query := url.Values{"timeout": {"10"}, "offset": {strconv.FormatInt(a.offset, 10)}}
	var response struct {
		OK     bool     `json:"ok"`
		Result []update `json:"result"`
		Error  string   `json:"description"`
	}
	if err := a.request(ctx, "getUpdates", query, nil, &response); err != nil {
		return nil, err
	}
	return response.Result, nil
}

func (a *Adapter) call(ctx context.Context, method string, payload map[string]any) error {
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"description"`
	}
	return a.request(ctx, method, nil, payload, &response)
}

func (a *Adapter) request(ctx context.Context, method string, query url.Values, payload any, result any) error {
	endpoint := a.baseURL + "/bot" + a.token + "/" + method
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram API returned %s", resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result); err != nil {
		return err
	}
	if !resultOK(result) {
		return fmt.Errorf("telegram API request %s failed", method)
	}
	return nil
}

func resultOK(result any) bool {
	value, ok := result.(*struct {
		OK    bool   `json:"ok"`
		Error string `json:"description"`
	})
	return !ok || value.OK
}
