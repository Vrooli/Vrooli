package codecs

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"sort"
	"time"
)

// CodexLiveQuotaWindow is a provider-reported window, not a token estimate.
// A pool distinguishes independent provider limit buckets.
type CodexLiveQuotaWindow struct {
	Pool          string     `json:"pool"`
	WindowMinutes int64      `json:"windowMinutes"`
	UsedPercent   float64    `json:"usedPercent"`
	ResetsAt      *time.Time `json:"resetsAt,omitempty"`
}

// ReadLiveQuota reads this process's Codex login in a short-lived app-server.
// The optional provider account ID is intentionally never returned or logged.
// A caller must bound ctx and must not interpret this as a launch binding.
func (c *Codex) ReadLiveQuota(ctx context.Context) ([]CodexLiveQuotaWindow, error) {
	if c.BinaryPath() == "" {
		return nil, errors.New("codex is unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.BinaryPath(), "app-server")
	cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	// Never capture provider stderr: it may contain credentials or account data.
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	defer func() { _ = stdin.Close(); cancel(); _ = cmd.Wait() }()
	session := newCodexAppServerSession(stdin, stdout)
	defer session.close()
	if _, err := session.call(ctx, codexMethodInitialize, map[string]any{"clientInfo": map[string]any{"name": "agent-manager", "version": "1"}, "capabilities": map[string]any{}}); err != nil {
		return nil, quotaReadFailure(ctx, "initialize")
	}
	if err := session.notify(codexMethodInitialized); err != nil {
		return nil, quotaReadFailure(ctx, "initialize")
	}
	raw, err := session.call(ctx, "account/rateLimits/read", nil)
	if err != nil {
		return nil, quotaReadFailure(ctx, "read")
	}
	return parseCodexLiveQuota(raw)
}

func quotaReadFailure(ctx context.Context, step string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Provider messages can contain account or credential details.
	return errors.New("codex quota " + step + " failed")
}

type codexLiveWindow struct {
	UsedPercent   *float64 `json:"usedPercent"`
	WindowMinutes *int64   `json:"windowDurationMins"`
	ResetsAt      *int64   `json:"resetsAt"`
}
type codexLiveBucket struct {
	Primary   *codexLiveWindow `json:"primary"`
	Secondary *codexLiveWindow `json:"secondary"`
}

func parseCodexLiveQuota(raw json.RawMessage) ([]CodexLiveQuotaWindow, error) {
	var response struct {
		RateLimits codexLiveBucket            `json:"rateLimits"`
		Buckets    map[string]codexLiveBucket `json:"rateLimitsByLimitId"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, errors.New("invalid provider response")
	}
	if len(response.Buckets) == 0 {
		response.Buckets = map[string]codexLiveBucket{"codex": response.RateLimits}
	}
	keys := make([]string, 0, len(response.Buckets))
	for k := range response.Buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var result []CodexLiveQuotaWindow
	for _, key := range keys {
		if key == "" || len(key) > 128 {
			continue
		}
		bucket := response.Buckets[key]
		for _, w := range []*codexLiveWindow{bucket.Primary, bucket.Secondary} {
			if w == nil || w.UsedPercent == nil || w.WindowMinutes == nil || *w.WindowMinutes <= 0 || math.IsNaN(*w.UsedPercent) || *w.UsedPercent < 0 || *w.UsedPercent > 100 {
				continue
			}
			item := CodexLiveQuotaWindow{Pool: key, WindowMinutes: *w.WindowMinutes, UsedPercent: *w.UsedPercent}
			if w.ResetsAt != nil && *w.ResetsAt > 0 {
				reset := time.Unix(*w.ResetsAt, 0).UTC()
				item.ResetsAt = &reset
			}
			result = append(result, item)
		}
	}
	return result, nil
}
