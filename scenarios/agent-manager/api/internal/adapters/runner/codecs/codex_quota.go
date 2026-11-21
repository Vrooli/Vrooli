package codecs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// CodexLiveQuotaWindow is a provider-reported window, not a token estimate.
// A pool distinguishes independent provider limit buckets.
type CodexLiveQuotaWindow struct {
	Pool          string     `json:"pool"`
	WindowMinutes int64      `json:"windowMinutes"`
	UsedPercent   float64    `json:"usedPercent"`
	ResetsAt      *time.Time `json:"resetsAt,omitempty"`
	// ObservedAt is set when the window's reading is older than the response.
	ObservedAt *time.Time `json:"observedAt,omitempty"`
}

// ReadLiveQuota reads this process's Codex login in a short-lived app-server.
// The optional provider account ID is intentionally never returned or logged.
// A caller must bound ctx and must not interpret this as a launch binding.
func (c *Codex) ReadLiveQuota(ctx context.Context) ([]CodexLiveQuotaWindow, error) {
	windows, _, err := c.readQuota(ctx, os.Environ())
	return windows, err
}

// ReadLaunchAccountScope reads the account identity through Codex's owner API
// using the exact run-scoped CODEX_HOME supplied by the launcher. It returns
// only a stable opaque digest; raw account identifiers never cross this API.
func (c *Codex) ReadLaunchAccountScope(ctx context.Context, codexHome string) (string, error) {
	if strings.TrimSpace(codexHome) == "" {
		return "", errors.New("run-scoped CODEX_HOME is required")
	}
	env := replaceEnv(os.Environ(), "CODEX_HOME", codexHome)
	_, accountID, err := c.readQuota(ctx, env)
	if err != nil {
		return "", err
	}
	if accountID == "" {
		return "", errors.New("Codex account identity unavailable")
	}
	digest := sha256.Sum256([]byte("agent-manager:codex-account:v1:" + accountID))
	return "codex:" + hex.EncodeToString(digest[:]), nil
}

func (c *Codex) readQuota(ctx context.Context, env []string) ([]CodexLiveQuotaWindow, string, error) {
	if c.BinaryPath() == "" {
		return nil, "", errors.New("codex is unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.BinaryPath(), "app-server")
	cmd.Env = env
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, "", err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, "", err
	}
	// Never capture provider stderr: it may contain credentials or account data.
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, "", err
	}
	defer func() { _ = stdin.Close(); cancel(); _ = cmd.Wait() }()
	session := newCodexAppServerSession(stdin, stdout)
	defer session.close()
	if _, err := session.call(ctx, codexMethodInitialize, map[string]any{"clientInfo": map[string]any{"name": "agent-manager", "version": "1"}, "capabilities": map[string]any{}}); err != nil {
		return nil, "", quotaReadFailure(ctx, "initialize")
	}
	if err := session.notify(codexMethodInitialized); err != nil {
		return nil, "", quotaReadFailure(ctx, "initialize")
	}
	raw, err := session.call(ctx, "account/rateLimits/read", nil)
	if err != nil {
		return nil, "", quotaReadFailure(ctx, "read")
	}
	windows, accountID, err := parseCodexLiveQuotaResponse(raw)
	return windows, accountID, err
}

func replaceEnv(env []string, key, value string) []string {
	result := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, key+"=") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, key+"="+value)
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
	windows, _, err := parseCodexLiveQuotaResponse(raw)
	return windows, err
}

func parseCodexLiveQuotaResponse(raw json.RawMessage) ([]CodexLiveQuotaWindow, string, error) {
	var response struct {
		AccountID  string                     `json:"accountId"`
		RateLimits codexLiveBucket            `json:"rateLimits"`
		Buckets    map[string]codexLiveBucket `json:"rateLimitsByLimitId"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, "", errors.New("invalid provider response")
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
	return result, strings.TrimSpace(response.AccountID), nil
}
