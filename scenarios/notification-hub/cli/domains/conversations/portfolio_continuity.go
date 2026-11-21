package conversations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

type portfolioContinuityResult struct {
	AskID      string `json:"ask_id"`
	Source     string `json:"source"`
	Recipient  string `json:"recipient"`
	SubjectKey string `json:"subject_key"`
	Attested   string `json:"attested"`
}

func (s *service) portfolioContinuity(ctx cliapp.RunContext) error {
	askID := strings.TrimSpace(ctx.Positional("id"))
	if askID == "" {
		return fmt.Errorf("give the pending ask id")
	}
	result, err := requestPortfolioContinuity(s.httpClient, s.apiBase, askID)
	if err != nil {
		return err
	}
	if ctx.JSON() {
		encoder := json.NewEncoder(ctx.Stdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{
		fmt.Sprintf("Portfolio steward can now continue pending ask %s for its stored recipient.", result.AskID),
	}})
}

func requestPortfolioContinuity(client interface {
	Do(*http.Request) (*http.Response, error)
}, apiBase, askID string) (portfolioContinuityResult, error) {
	if client == nil || strings.TrimSpace(apiBase) == "" {
		return portfolioContinuityResult{}, fmt.Errorf("Notification Hub API client is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(apiBase, "/") + "/api/v1/conversations/asks/" + url.PathEscape(askID) + "/portfolio-continuity"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return portfolioContinuityResult{}, fmt.Errorf("build portfolio continuity request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return portfolioContinuityResult{}, fmt.Errorf("request portfolio continuity: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return portfolioContinuityResult{}, fmt.Errorf("Notification Hub refused portfolio continuity with status %d", resp.StatusCode)
	}
	var result portfolioContinuityResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result); err != nil {
		return portfolioContinuityResult{}, fmt.Errorf("read portfolio continuity result: %w", err)
	}
	if result.AskID != askID || result.Source != "portfolio-steward" || result.Recipient == "" || result.SubjectKey == "" || result.Attested != "true" {
		return portfolioContinuityResult{}, fmt.Errorf("Notification Hub returned an incomplete portfolio continuity result")
	}
	return result, nil
}
