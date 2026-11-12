package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"connectrpc.com/connect"
)

type agentManagerSourceVerifier struct {
	httpClient connect.HTTPClient
	baseURL    string
}

type agentManagerEventPage struct {
	Events []struct {
		ID         string          `json:"id"`
		RunID      string          `json:"runId"`
		RunIDSnake string          `json:"run_id"`
		Sequence   json.RawMessage `json:"sequence"`
		EventType  string          `json:"event_type"`
		Message    *struct {
			Role                string `json:"role"`
			Source              string `json:"source"`
			Content             string `json:"content"`
			ConversationID      string `json:"conversation_id"`
			EvidenceOnly        bool   `json:"evidence_only"`
			ConversationIDCamel string `json:"conversationId"`
		} `json:"message,omitempty"`
	} `json:"events"`
}

func newAgentManagerSourceVerifier(httpClient connect.HTTPClient, rawURL string) (*agentManagerSourceVerifier, error) {
	baseURL, err := localAgentManagerURL(rawURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		return nil, fmt.Errorf("Agent Manager authenticated HTTP client is not configured")
	}
	return &agentManagerSourceVerifier{httpClient: httpClient, baseURL: baseURL}, nil
}

// verify fetches the exact source event from Agent Manager and checks that the
// replay prompt is its current user message. UUID-shaped caller assertions are
// insufficient to dispatch any inference.
func (v *agentManagerSourceVerifier) verify(ctx context.Context, c ReplayCase) error {
	if v == nil || v.httpClient == nil || v.baseURL == "" {
		return fmt.Errorf("Agent Manager source client is not configured")
	}
	if c.SourceSequence < 0 {
		return fmt.Errorf("source_sequence must be nonnegative")
	}
	query := url.Values{}
	query.Set("after_sequence", strconv.FormatInt(c.SourceSequence-1, 10))
	query.Set("limit", "1")
	query.Set("event_types", "message")
	endpoint := v.baseURL + "/api/v1/runs/" + url.PathEscape(c.SourceRunID) + "/events?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build Agent Manager source request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	response, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("read Agent Manager source event: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Agent Manager source event returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read Agent Manager event response: %w", err)
	}
	var page agentManagerEventPage
	if err := json.Unmarshal(body, &page); err != nil {
		return fmt.Errorf("decode Agent Manager event response: %w", err)
	}
	if len(page.Events) != 1 {
		return fmt.Errorf("Agent Manager source event is unavailable")
	}
	event := page.Events[0]
	runID := event.RunID
	if runID == "" {
		runID = event.RunIDSnake
	}
	sequence, err := agentManagerSequence(event.Sequence)
	if err != nil {
		return err
	}
	if event.ID != c.SourceEventID || runID != c.SourceRunID || sequence != c.SourceSequence {
		return fmt.Errorf("Agent Manager event identity mismatch (returned_run_id=%q event_id_match=%t sequence_match=%t)", runID, event.ID == c.SourceEventID, sequence == c.SourceSequence)
	}
	message := event.Message
	if event.EventType != "RUN_EVENT_TYPE_MESSAGE" || message == nil || message.Role != "user" || message.Source != "operator" || message.EvidenceOnly || message.Content != c.InputText {
		return fmt.Errorf("replay input is not the exact Agent Manager user message")
	}
	conversationID := message.ConversationID
	if conversationID == "" {
		conversationID = message.ConversationIDCamel
	}
	if c.ConversationID != "" && conversationID != c.ConversationID {
		return fmt.Errorf("Agent Manager conversation identity does not match")
	}
	return nil
}

func agentManagerSequence(raw json.RawMessage) (int64, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" {
		// Agent Manager uses proto3 JSON with EmitUnpopulated disabled. The
		// default int64 sequence zero is therefore omitted; identity matching
		// below accepts it only for a replay case whose expected sequence is 0.
		return 0, nil
	}
	if strings.HasPrefix(value, "\"") {
		var decoded string
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return 0, fmt.Errorf("decode Agent Manager event sequence: %w", err)
		}
		value = decoded
	}
	sequence, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("decode Agent Manager event sequence: %w", err)
	}
	return sequence, nil
}

func localAgentManagerURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", fmt.Errorf("--agent-manager-url must be an absolute local URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("--agent-manager-url must use http or https")
	}
	host := strings.Trim(parsed.Hostname(), "[]")
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return "", fmt.Errorf("--agent-manager-url must point to a loopback host; caller prompts stay local")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}
