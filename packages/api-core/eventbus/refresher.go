package eventbus

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// RefreshConfig controls the optional background snapshot refresher. All
// failures are intentionally contained here; no caller's domain request waits
// for or fails because a refresh is unavailable.
type RefreshConfig struct {
	Interval   time.Duration
	MinBackoff time.Duration
	MaxBackoff time.Duration
	Jitter     func(time.Duration) time.Duration
}

func (c RefreshConfig) normalized() RefreshConfig {
	if c.Interval <= 0 {
		c.Interval = 30 * time.Second
	}
	if c.MinBackoff <= 0 {
		c.MinBackoff = time.Second
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = time.Minute
	}
	if c.Jitter == nil {
		c.Jitter = func(d time.Duration) time.Duration { return d/2 + time.Duration(rand.Int64N(int64(d/2)+1)) }
	}
	return c
}

// StartRefresher starts one background refresh loop and returns immediately.
// It performs an initial best-effort load, then applies exponential backoff on
// failures. Cancelling ctx stops it without mutating the last snapshot.
func StartRefresher(ctx context.Context, client Client, cache *Cache, cfg RefreshConfig) {
	if cache == nil || !client.Enabled() {
		return
	}
	cfg = cfg.normalized()
	// SSE push is an optimization over the same complete snapshot contract.
	// The independent polling loop below remains the bootstrap and recovery
	// mechanism, so a dropped stream can never affect a domain request.
	go watchPolicySnapshots(ctx, client, cache, cfg)
	go func() {
		wait, backoff := time.Duration(0), cfg.MinBackoff
		for {
			if wait > 0 {
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
			rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err := client.Refresh(rctx, cache)
			cancel()
			if err == nil {
				wait, backoff = cfg.Interval, cfg.MinBackoff
				continue
			}
			wait = cfg.Jitter(backoff)
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}()
}

// watchPolicySnapshots applies only complete, versioned snapshot events. It
// intentionally ignores legacy incremental policy events, because applying a
// partial rule update could split traffic and receipt projection decisions.
func watchPolicySnapshots(ctx context.Context, client Client, cache *Cache, cfg RefreshConfig) {
	backoff := cfg.MinBackoff
	for {
		if err := client.consumePolicyStream(ctx, cache); err != nil && ctx.Err() == nil {
			log.Printf("vrooli-events policy push degraded: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
		wait := cfg.Jitter(backoff)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff *= 2
		if backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}
}

func (c Client) consumePolicyStream(ctx context.Context, cache *Cache) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.baseURL(), "/")+"/api/v1/policies/subscribe", nil)
	if err != nil {
		return err
	}
	h := c.HTTPClient
	if h == nil {
		h = http.DefaultClient
	}
	resp, err := h.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("policy stream: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Receipt projections can approach 64KiB, so permit a complete snapshot
	// without silently dropping an otherwise valid policy generation.
	scanner.Buffer(make([]byte, 4*1024), 256*1024)
	eventType, data := "", ""
	apply := func() {
		if eventType != "snapshot" || data == "" {
			return
		}
		var snapshot PolicySnapshot
		if err := json.Unmarshal([]byte(data), &snapshot); err != nil || strings.TrimSpace(snapshot.Version) == "" {
			return
		}
		cache.Replace(snapshot, time.Now())
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			apply()
			eventType, data = "", ""
			continue
		}
		if value, ok := strings.CutPrefix(line, "event: "); ok {
			eventType = value
		} else if value, ok := strings.CutPrefix(line, "data: "); ok {
			data += value
		}
	}
	apply()
	return scanner.Err()
}
