package prompt_manager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"development-toolchain-validator/internal/httpc"
	vrun "development-toolchain-validator/internal/validation_run"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	skillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills"
	skillsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/skills/skills_v1connect"
)

// SkillContentRESTAdapter implements validation_run.SkillContentSource by
// calling prompt-manager's generated Connect client. It returns the skill's
// markdown body, which the validation_run worker injects as the sandboxed
// agent's prompt so the agent executes the actual skill. Its historical name
// is retained to avoid breaking downstream constructor call sites.
type SkillContentRESTAdapter struct {
	resolver    *discovery.Resolver
	doer        httpc.Doer
	maxAttempts int
}

// NewSkillContentRESTAdapter constructs the production adapter. Empty
// option fields fall back to the same defaults as the catalog adapter.
func NewSkillContentRESTAdapter(opts Options) *SkillContentRESTAdapter {
	a := &SkillContentRESTAdapter{
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

var _ vrun.SkillContentSource = (*SkillContentRESTAdapter)(nil)

// SkillContent resolves prompt-manager's base URL via discovery and GETs
// the skill by id, returning its markdown content. Re-resolves the base
// URL on transport failure to survive scenario restart cycles.
func (a *SkillContentRESTAdapter) SkillContent(ctx context.Context, skillID string) (string, error) {
	id := strings.TrimSpace(skillID)
	if id == "" {
		return "", fmt.Errorf("skill id is empty")
	}
	var lastErr error
	for attempt := 0; attempt < a.maxAttempts; attempt++ {
		base, err := a.resolver.ResolveScenarioURLDefault(ctx, promptManagerScenario)
		if err != nil {
			if discovery.IsScenarioNotRunning(err) {
				return "", fmt.Errorf("prompt-manager is not running: %w", err)
			}
			lastErr = err
			continue
		}
		content, err := a.fetchOnce(ctx, base, id)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !isRetriable(err) {
			return "", err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("exhausted retries")
	}
	return "", fmt.Errorf("fetch skill %q content: %w", id, lastErr)
}

func (a *SkillContentRESTAdapter) fetchOnce(ctx context.Context, base, id string) (string, error) {
	client := skillsconnect.NewSkillsServiceClient(a.doer, strings.TrimRight(base, "/"))
	resp, err := client.GetSkill(ctx, connect.NewRequest(&skillsv1.GetSkillRequest{Id: id}))
	if err != nil {
		return "", fmt.Errorf("get skill: %w", err)
	}
	if resp.Msg.GetSkill() == nil {
		return "", fmt.Errorf("skill %q returned no record", id)
	}
	content := strings.TrimSpace(resp.Msg.GetSkill().GetContent())
	if content == "" {
		return "", fmt.Errorf("skill %q returned empty content", id)
	}
	return resp.Msg.GetSkill().GetContent(), nil
}
