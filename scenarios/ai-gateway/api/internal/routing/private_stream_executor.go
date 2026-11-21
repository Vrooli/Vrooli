package routing

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

func (e *HTTPStreamExecutor) execute(ctx context.Context, endpoint SupplierEndpoint, chat map[string]json.RawMessage, receipt PrivateReceipt, multiplier int64, maxOutputBytes int, w http.ResponseWriter) PrivateReceipt {
	resp, err := e.openUpstreamStream(ctx, endpoint, chat)
	if err != nil {
		receipt.State, receipt.Message = "failed", err.Error()
		return receipt
	}
	if resp == nil {
		receipt.State = "unknown"
		receipt.ErrorCode = "upstream_outcome_unknown"
		return receipt
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		receipt.State, receipt.CostKnown, receipt.ErrorCode = "failed", true, "upstream_refused"
		receipt.Message = fmt.Sprintf("supplier HTTP %d", resp.StatusCode)
		return receipt
	}
	return streamUpstreamResponse(resp, receipt, multiplier, maxOutputBytes, w)
}

func (e *HTTPStreamExecutor) openUpstreamStream(ctx context.Context, endpoint SupplierEndpoint, chat map[string]json.RawMessage) (*http.Response, error) {
	chat["stream"] = json.RawMessage(`true`)
	chat["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	for _, name := range []string{"outside_funded", "business_account_id", "use_class", "funded", "metadata"} {
		delete(chat, name)
	}
	body, _ := json.Marshal(chat)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint.URL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if endpoint.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+endpoint.APIKey)
	}
	client := e.Client
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	return resp, nil
}

func streamUpstreamResponse(resp *http.Response, receipt PrivateReceipt, multiplier int64, maxOutputBytes int, w http.ResponseWriter) PrivateReceipt {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Operation-ID", receipt.OperationID)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	delivered, done, outputBytes := true, false, 0
	for scanner.Scan() {
		line := scanner.Text()
		outputBytes += len(line) + 1
		if outputBytes > maxOutputBytes {
			break
		}
		done = recordStreamEvent(line, &receipt, multiplier) || done
		if delivered {
			_, writeErr := io.WriteString(w, line+"\n")
			delivered = writeErr == nil
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
	receipt.State = "completed"
	if scanner.Err() != nil || !done || !receipt.CostKnown {
		receipt.State = "unknown"
		receipt.ErrorCode = "upstream_outcome_unknown"
	}
	return receipt
}

func recordStreamEvent(line string, receipt *PrivateReceipt, multiplier int64) bool {
	if !strings.HasPrefix(line, "data:") {
		return false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "[DONE]" {
		return true
	}
	var event struct {
		Usage *struct {
			PromptTokens     int      `json:"prompt_tokens"`
			CompletionTokens int      `json:"completion_tokens"`
			Cost             *float64 `json:"cost"`
		} `json:"usage"`
	}
	if json.Unmarshal([]byte(payload), &event) != nil || event.Usage == nil {
		return false
	}
	receipt.PromptTokens, receipt.CompletionTokens = event.Usage.PromptTokens, event.Usage.CompletionTokens
	if event.Usage.Cost == nil || *event.Usage.Cost < 0 {
		return false
	}
	receipt.CostMicros = int64(*event.Usage.Cost*1e6 + 0.5)
	if receipt.CostMicros < 0 || receipt.CostMicros > (math.MaxInt64-9999)/multiplier {
		return false
	}
	receipt.Credits = (receipt.CostMicros*multiplier + 9999) / 10000
	receipt.CostKnown = true
	return false
}
