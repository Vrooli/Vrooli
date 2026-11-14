package brief

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	agentbrief "github.com/vrooli/agentbrief-go"
	"github.com/vrooli/api-core/schedule"
	"portal/internal/integrations/registry"
)

type Service struct {
	repo       Repository
	hub        agentbrief.HubClient
	registry   *registry.Service
	clock      schedule.Clock
	thresholds agentbrief.GateThresholds
}
type Config struct {
	Repository Repository
	Hub        agentbrief.HubClient
	Registry   *registry.Service
	Clock      schedule.Clock
	Thresholds agentbrief.GateThresholds
}

func NewService(cfg Config) *Service {
	if cfg.Clock == nil {
		cfg.Clock = schedule.System()
	}
	if cfg.Thresholds.MinPromptRunes == 0 {
		cfg.Thresholds = agentbrief.DefaultThresholds
	}
	return &Service{repo: cfg.Repository, hub: cfg.Hub, registry: cfg.Registry, clock: cfg.Clock, thresholds: cfg.Thresholds}
}

func (s *Service) Build(ctx context.Context, input BuildInput) (Record, error) {
	mode := agentbrief.ModePassive
	if s.registry != nil {
		status, err := s.registry.Status(ctx)
		if err != nil {
			mode = agentbrief.ModeOff
		} else {
			mode = modeFromRegistry(status.ActiveMode)
		}
	}
	result := agentbrief.BuildBrief(ctx, agentbrief.BuildRequest{Prompt: input.Prompt, Consumer: input.Consumer, Mode: mode, BudgetMS: input.BudgetMS}, s.hub, s.thresholds, s.clock)
	record := Record{ID: newID(), Consumer: result.Consumer, Verdict: result.Verdict, Reason: result.Reason, ChatID: input.ChatID, MessageID: input.MessageID, Harness: input.Harness, SessionRef: input.SessionRef, PromptDigest: result.PromptDigest, EffectiveQuery: result.EffectiveQuery, Rendered: result.Rendered, MaxTrustClass: result.MaxTrustClass, Degraded: result.Degraded, LatencyMS: result.LatencyMS, CreatedAt: s.clock.Now().UTC(), Items: result.Items, QueriedProviders: result.QueriedProviders}
	if s.repo != nil {
		if err := s.repo.Save(ctx, record); err != nil {
			return Record{}, err
		}
	}
	return record, nil
}
func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id))
}
func (s *Service) List(ctx context.Context, input ListInput) ([]Record, error) {
	return s.repo.List(ctx, input)
}
func (s *Service) RecordUse(ctx context.Context, input UseInput) (bool, error) {
	return s.repo.RecordUse(ctx, input, s.clock.Now().UTC())
}
func (s *Service) Stats(ctx context.Context, input StatsInput) ([]StatsRow, error) {
	return s.repo.Stats(ctx, input, s.clock.Now().UTC())
}
func (s *Service) Retain(ctx context.Context) error {
	return s.repo.DeleteBefore(ctx, s.clock.Now().UTC().Add(-30*24*time.Hour))
}
func modeFromRegistry(mode interface{ String() string }) agentbrief.BehaviorMode {
	if mode.String() == "BEHAVIOR_MODE_OFF" {
		return agentbrief.ModeOff
	}
	if mode.String() == "BEHAVIOR_MODE_FULL" {
		return agentbrief.ModeFull
	}
	return agentbrief.ModePassive
}
func newID() string { return uuid.NewString() }
