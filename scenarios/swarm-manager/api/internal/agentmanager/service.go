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
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/scenario"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
	createCallerVerifier owneridentity.Validator
	client               *HTTPClient
	profileName          string
	profileKey           string
	requiredKeys         []string
	profileID            string
	profileIDs           map[string]string
	mu                   sync.RWMutex
	enabled              bool
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
		createCallerVerifier: owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})}),
		client:               client,
		profileName:          strings.TrimSpace(cfg.ProfileName),
		profileKey:           strings.TrimSpace(cfg.ProfileKey),
		requiredKeys:         normalizeProfileKeys(cfg.RequiredKeys),
		profileIDs:           make(map[string]string),
		enabled:              cfg.Enabled,
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

// Initialize resolves existing, scenario-owned profile IDs from a bounded read.
// Agent Manager owns declaration registration; Swarm must not reconcile or
// create profiles as a side effect of starting its own service.
func (s *AgentService) Initialize(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	const profilePageSize = 128
	const maxProfilePages = 8
	var profiles []*domainpb.AgentProfile
	coverageComplete := false
	for page := 0; page < maxProfilePages; page++ {
		resp, err := s.client.ListProfiles(ctx, profilePageSize, page*profilePageSize)
		if err != nil {
			return fmt.Errorf("read existing agent-manager profiles: %w", err)
		}
		if resp == nil {
			return fmt.Errorf("read existing agent-manager profiles: empty response")
		}
		profiles = append(profiles, resp.Profiles...)
		if len(resp.Profiles) < profilePageSize {
			coverageComplete = true
			break
		}
	}

	owner := scenario.Name()
	prefix := owner + "/"
	profileIDs := make(map[string]string)
	for _, profile := range profiles {
		if profile == nil || profile.GetOwnerScenario() != owner {
			continue
		}
		key := strings.TrimSpace(profile.GetProfileKey())
		id := strings.TrimSpace(profile.GetId())
		if key == "" || id == "" || !strings.HasPrefix(key, prefix) {
			return fmt.Errorf("read existing agent-manager profiles: owner %q returned an invalid profile identity", owner)
		}
		if prior, exists := profileIDs[key]; exists && prior != id {
			return fmt.Errorf("read existing agent-manager profiles: duplicate owner profile key %q", key)
		}
		profileIDs[key] = id
		if profile.GetLocalOverride() || (profile.GetSourcePath() != "" && profile.GetSourceHash() != profile.GetLastAppliedHash()) {
			slog.Warn("agent-manager profile has declaration drift metadata; declaration owner must reconcile it", "profile_key", key)
		}
	}
	if err := s.validateRequiredProfiles(profileIDs); err != nil {
		coverage := "bounded read did not observe every required profile"
		if coverageComplete {
			coverage = "bounded read completed but required registration was not observed"
		}
		return fmt.Errorf("resolve existing agent-manager profiles: %w; %s; check the Agent Manager declaration owner; no profile was created or updated", err, coverage)
	}
	if !coverageComplete {
		slog.Warn("agent-manager profile read reached its page bound; only observed owner keys are enabled", "scenario", owner, "pages", maxProfilePages)
	}

	s.mu.Lock()
	s.profileIDs = profileIDs
	s.profileID = profileIDs[s.profileKey]
	s.mu.Unlock()
	slog.Info("resolved existing scenario-owned agent-manager profiles", "scenario", owner, "profiles", len(profileIDs))
	return nil
}

// PrepareCreateRunCaller verifies manual goal admission before queue effects.
func (s *AgentService) PrepareCreateRunCaller(ctx context.Context) (context.Context, error) {
	verifier := s.createCallerVerifier
	if verifier == nil {
		verifier = owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})})
	}
	return owneridentity.AuthorizeBoundCreateRunCaller(ctx, verifier, time.Now())
}
