// Package agentmanager provides a higher-level integration seam for agent-manager.
//
// This service hides HTTP/proto details from handlers and owns profile setup,
// tagging, and spawn orchestration for Swarm Manager.
//
// DOC: docs/concepts/ARCHITECTURE.md#design-principles
// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/INTENT.md#what-not-to-modify-here
package agentmanager

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/scenario"
)

// freshConversationID mints a new ConversationID for a swarm-manager-spawned
// run. Per Decision D7 of the auditability contract, spawn surfaces SHOULD
// populate ConversationID explicitly rather than rely on agent-manager's
// fallback. Each top-level spawn from swarm-manager (research, backlog,
// initiative) is conceptually a fresh conversation.
func freshConversationID() *string {
	id := uuid.NewString()
	return &id
}

// Service defines the seam handlers depend on.
type Service interface {
	IsEnabled() bool
	IsAvailable(ctx context.Context) bool
	ResolveURL(ctx context.Context) (string, error)
	GetProfileID() string

	GetRunState(ctx context.Context, runID string) (RunState, error)
	GetRunDiff(ctx context.Context, runID string) (RunDiff, error)
	ApproveRun(ctx context.Context, runID, actor, commitMsg string) error
	StopRun(ctx context.Context, runID string) error
	ContinueRun(ctx context.Context, runID string, message string) error
}

// AgentService implements the Service interface.
type AgentService struct {
	client       *HTTPClient
	profileName  string
	profileKey   string
	requiredKeys []string
	profileID    string
	profileIDs   map[string]string
	mu           sync.RWMutex
	enabled      bool
}

// AgentServiceConfig contains configuration for the agent service.
type AgentServiceConfig struct {
	ProfileName  string
	ProfileKey   string
	RequiredKeys []string
	Timeout      time.Duration
	Enabled      bool
}

// NewAgentService creates a new agent service.
func NewAgentService(cfg AgentServiceConfig) *AgentService {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	client := NewHTTPClientWithTimeout(cfg.Timeout)
	return &AgentService{
		client:       client,
		profileName:  strings.TrimSpace(cfg.ProfileName),
		profileKey:   strings.TrimSpace(cfg.ProfileKey),
		requiredKeys: normalizeProfileKeys(cfg.RequiredKeys),
		profileIDs:   make(map[string]string),
		enabled:      cfg.Enabled,
	}
}

func normalizeProfileKeys(keys []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func (s *AgentService) requiredProfileKeys() []string {
	return normalizeProfileKeys(append([]string{s.profileKey}, s.requiredKeys...))
}

func (s *AgentService) validateRequiredProfiles(profileIDs map[string]string) error {
	const prefix = "swarm-manager/"
	for _, key := range s.requiredProfileKeys() {
		if !strings.HasPrefix(key, prefix) {
			return fmt.Errorf("required profile %q is not owned by scenario %q", key, "swarm-manager")
		}
		if strings.TrimSpace(profileIDs[key]) == "" {
			return fmt.Errorf("required profile %q was not returned", key)
		}
	}
	return nil
}

// IsEnabled returns whether agent-manager integration is enabled.
func (s *AgentService) IsEnabled() bool {
	return s.enabled
}

// IsAvailable checks if agent-manager is reachable.
func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if !s.enabled {
		return false
	}
	ok, err := s.client.Health(ctx)
	return err == nil && ok
}

// ResolveURL returns the current agent-manager base URL.
func (s *AgentService) ResolveURL(ctx context.Context) (string, error) {
	if !s.enabled {
		return "", fmt.Errorf("agent-manager not enabled")
	}
	return s.client.ResolveURL(ctx)
}

// Initialize resolves the swarm-manager profile IDs once at startup. Agent
// Manager owns declaration registration through its own startup sweep, so there
// is no per-start reconcile: this one-time call reconciles idempotently only to
// resolve the stable profile-key -> id mapping the run surface reports.
func (s *AgentService) Initialize(ctx context.Context) error {
	if !s.enabled {
		return nil
	}

	resp, err := s.client.ReconcileScenarioProfiles(ctx, scenario.Name())
	if err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}

	s.mu.Lock()
	found := false
	profileIDs := make(map[string]string, len(resp.Results))
	for _, item := range resp.Results {
		profileIDs[item.ProfileKey] = item.ProfileId
		if item.ProfileKey == s.profileKey {
			s.profileID = item.ProfileId
			found = true
		}
	}
	s.profileIDs = profileIDs
	s.mu.Unlock()
	if resp.Failed > 0 {
		return fmt.Errorf("reconcile scenario profiles: %d profile source(s) failed validation", resp.Failed)
	}
	if err := s.validateRequiredProfiles(profileIDs); err != nil {
		return fmt.Errorf("reconcile scenario profiles: %w", err)
	}
	if !found {
		return fmt.Errorf("profile %q was not returned", s.profileKey)
	}

	slog.Info("reconciled agent profiles", "scenario", resp.Scenario, "created", resp.Created, "updated", resp.Updated, "unchanged", resp.Unchanged, "failed", resp.Failed)

	return nil
}
