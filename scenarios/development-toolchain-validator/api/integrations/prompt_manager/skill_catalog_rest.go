// Package prompt_manager adapts DTV's outbound view of prompt-manager.
//
// Today prompt-manager exposes a REST surface for skills (no proto).
// This adapter implements the skill_catalog.SkillCatalogSource seam by
// calling prompt-manager's /api/v1/skills/sync endpoint and translating
// the response into domain Skill values.
//
// When prompt-manager grows a Connect-RPC surface for skills, swap this
// file's body without touching the SkillCatalogSource seam in
// internal/skill_catalog/. The migration is tracked in
// docs/internal/PROBLEMS.md.
package prompt_manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"development-toolchain-validator/internal/httpc"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	"github.com/vrooli/api-core/discovery"
)

const (
	// skillsSyncPath is the prompt-manager REST endpoint that returns
	// the full skill catalog with content. The DTV adapter prefers this
	// over /skills because we need per-skill content to compute the
	// content_hash that manifests pin against.
	skillsSyncPath = "/api/v1/skills/sync"

	// promptManagerScenario is the scenario slug used by discovery.
	promptManagerScenario = "prompt-manager"
)

// SkillCatalogRESTAdapter implements skill_catalog.SkillCatalogSource by
// calling prompt-manager's /api/v1/skills/sync endpoint.
type SkillCatalogRESTAdapter struct {
	resolver *discovery.Resolver
	doer     httpc.Doer
	// maxAttempts bounds the retry loop on transport errors. 3 is the
	// canonical interoperability-steer ceiling.
	maxAttempts int
}

// Options configures SkillCatalogRESTAdapter. Empty values fall back to
// production defaults.
type Options struct {
	Resolver    *discovery.Resolver
	Doer        httpc.Doer
	MaxAttempts int
}

// NewSkillCatalogRESTAdapter constructs the production adapter.
func NewSkillCatalogRESTAdapter(opts Options) *SkillCatalogRESTAdapter {
	a := &SkillCatalogRESTAdapter{
		resolver:    opts.Resolver,
		doer:        opts.Doer,
		maxAttempts: opts.MaxAttempts,
	}
	if a.resolver == nil {
		a.resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	if a.doer == nil {
		a.doer = &http.Client{Timeout: 30 * time.Second}
	}
	if a.maxAttempts <= 0 {
		a.maxAttempts = 3
	}
	return a
}

var _ skillcatalog.SkillCatalogSource = (*SkillCatalogRESTAdapter)(nil)

// Fetch resolves prompt-manager's base URL via discovery, GETs
// /api/v1/skills/sync, and translates the response into Skill values.
// Re-resolves the base URL on transport failure to survive scenario
// restart cycles.
func (a *SkillCatalogRESTAdapter) Fetch(ctx context.Context) ([]skillcatalog.Skill, error) {
	var lastErr error
	for attempt := 0; attempt < a.maxAttempts; attempt++ {
		base, err := a.resolver.ResolveScenarioURLDefault(ctx, promptManagerScenario)
		if err != nil {
			if discovery.IsScenarioNotRunning(err) {
				return nil, skillcatalog.ErrSyncFailed{
					Reason:   "prompt-manager is not running",
					Wrapped:  err,
					NotReady: true,
				}
			}
			lastErr = skillcatalog.ErrSyncFailed{Reason: "discovery", Wrapped: err}
			continue
		}
		skills, err := a.fetchOnce(ctx, base)
		if err == nil {
			return skills, nil
		}
		lastErr = err
		if !isRetriable(err) {
			return nil, err
		}
		// fall through, re-resolve, retry
	}
	if lastErr == nil {
		lastErr = errors.New("exhausted retries")
	}
	return nil, skillcatalog.ErrSyncFailed{Reason: "transport", Wrapped: lastErr}
}

func (a *SkillCatalogRESTAdapter) fetchOnce(ctx context.Context, base string) ([]skillcatalog.Skill, error) {
	u := strings.TrimRight(base, "/") + skillsSyncPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := a.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return nil, transportError{status: resp.StatusCode, snippet: strings.TrimSpace(string(body))}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<14))
		return nil, skillcatalog.ErrSyncFailed{
			Reason:  fmt.Sprintf("upstream returned %d", resp.StatusCode),
			Wrapped: fmt.Errorf("%s", strings.TrimSpace(string(body))),
		}
	}
	var payload syncPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, skillcatalog.ErrSyncFailed{Reason: "decode response", Wrapped: err}
	}
	out := make([]skillcatalog.Skill, 0, len(payload.Skills))
	for _, s := range payload.Skills {
		id := strings.TrimSpace(s.ID)
		if id == "" {
			continue
		}
		out = append(out, skillcatalog.Skill{
			ID:          id,
			Version:     versionFromUpdatedAt(s.UpdatedAt),
			ContentHash: contentHash(s.Content),
		})
	}
	return out, nil
}

// syncPayload mirrors prompt-manager's skills.SyncResponse / Response
// shape. We only extract the fields we need and rely on json.Decoder's
// default unknown-field tolerance for forward compatibility.
type syncPayload struct {
	Skills []syncSkill `json:"skills"`
}

type syncSkill struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updatedAt"`
}

// versionFromUpdatedAt normalizes prompt-manager's RFC3339 updated_at
// string into the version we store. An empty upstream value maps to
// "unknown" so manifests still have something stable to pin against.
func versionFromUpdatedAt(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	return s
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// transportError marks errors that are worth retrying (5xx, network
// flakes). isRetriable consults this type via errors.As.
type transportError struct {
	status  int
	snippet string
}

func (e transportError) Error() string {
	if e.snippet != "" {
		return fmt.Sprintf("upstream %d: %s", e.status, e.snippet)
	}
	return fmt.Sprintf("upstream %d", e.status)
}

func isRetriable(err error) bool {
	var te transportError
	if errors.As(err, &te) {
		return true
	}
	// Bare network errors from the doer are also retriable. Heuristic:
	// any error wrapping io / context-deadline / "connection refused".
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout")
}
