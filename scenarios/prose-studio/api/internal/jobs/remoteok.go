package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"prose-studio/internal/httpc"
)

const remoteOKFeed = "https://remoteok.com/api"

// RemoteOKProvider reads the provider's public job feed. A caller must pass
// the search query explicitly; no resume or evidence-bank content is sent.
type RemoteOKProvider struct{ doer httpc.Doer }

func NewRemoteOKProvider(doer httpc.Doer) *RemoteOKProvider {
	return &RemoteOKProvider{doer: doer}
}

func (p *RemoteOKProvider) Name() string { return "Remote OK" }

func (p *RemoteOKProvider) Search(ctx context.Context, query string) ([]Opening, error) {
	if p == nil || p.doer == nil {
		return nil, fmt.Errorf("Remote OK provider has no HTTP client")
	}
	u, err := url.Parse(remoteOKFeed)
	if err != nil {
		return nil, err
	}
	tags := searchTags(query)
	if tags != "" {
		q := u.Query()
		q.Set("tags", tags)
		u.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Prose-Studio-Job-Readiness/1.0")
	resp, err := p.doer.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Remote OK returned HTTP %d", resp.StatusCode)
	}
	const maxFeedBytes = 8 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Remote OK feed: %w", err)
	}
	if len(body) > maxFeedBytes {
		return nil, fmt.Errorf("Remote OK feed exceeds %d bytes", maxFeedBytes)
	}
	var feed []remoteOKJob
	if err := json.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("decode Remote OK feed: %w", err)
	}
	openings := make([]Opening, 0, len(feed))
	for _, job := range feed {
		if strings.TrimSpace(job.Position) == "" || strings.TrimSpace(job.Company) == "" {
			continue
		}
		jobURL := strings.TrimSpace(job.URL)
		if jobURL == "" && strings.TrimSpace(job.ID) != "" {
			jobURL = "https://remoteok.com/remote-jobs/" + url.PathEscape(job.ID)
		}
		openings = append(openings, Opening{
			ID: job.ID, Company: strings.TrimSpace(job.Company),
			Title: strings.TrimSpace(job.Position), Location: strings.TrimSpace(job.Location),
			URL: jobURL, Source: "Remote OK", SourceID: job.ID,
			Description: strings.TrimSpace(job.Description), Status: "found",
		})
	}
	return openings, nil
}

type remoteOKJob struct {
	ID          string `json:"id"`
	Company     string `json:"company"`
	Position    string `json:"position"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func searchTags(query string) string {
	parts := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '+' && r != '#'
	})
	seen := make(map[string]struct{}, len(parts))
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 || len(tags) >= 8 {
			continue
		}
		if _, exists := seen[part]; exists {
			continue
		}
		seen[part] = struct{}{}
		tags = append(tags, part)
	}
	return strings.Join(tags, ",")
}
