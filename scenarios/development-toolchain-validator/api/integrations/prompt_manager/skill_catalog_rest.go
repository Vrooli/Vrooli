// Package prompt_manager adapts DTV's outbound view of prompt-manager.
package prompt_manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"development-toolchain-validator/internal/httpc"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

const promptManagerScenario = "prompt-manager"

// SkillCatalogRESTAdapter implements skill_catalog.SkillCatalogSource through
// prompt-manager's generated Connect client. Its historical name is retained
// to avoid breaking downstream constructor call sites.
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

// Fetch resolves prompt-manager's base URL via discovery, invokes SyncSkills,
// and translates the response into Skill values.
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
	client := skillsconnect.NewSkillsServiceClient(a.doer, strings.TrimRight(base, "/"))
	resp, err := client.SyncSkills(ctx, connect.NewRequest(&skillsv1.SyncSkillsRequest{}))
	if err != nil {
		return nil, skillcatalog.ErrSyncFailed{Reason: "sync skills", Wrapped: err}
	}
	out := make([]skillcatalog.Skill, 0, len(resp.Msg.GetSkills()))
	for _, s := range resp.Msg.GetSkills() {
		id := strings.TrimSpace(s.GetId())
		if id == "" {
			continue
		}
		out = append(out, skillcatalog.Skill{
			ID:          id,
			Version:     versionFromUpdatedAt(s.GetUpdatedAt()),
			ContentHash: contentHash(s.GetContent()),
		})
	}
	return out, nil
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

func isRetriable(err error) bool {
	var ce *connect.Error
	if errors.As(err, &ce) && (ce.Code() == connect.CodeUnavailable || ce.Code() == connect.CodeInternal || ce.Code() == connect.CodeDeadlineExceeded) {
		return true
	}
	// Bare network errors from the doer are also retriable. Heuristic:
	// any error wrapping io / context-deadline / "connection refused".
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "i/o timeout")
}
