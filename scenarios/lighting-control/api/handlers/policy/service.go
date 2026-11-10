package policy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	internalcontrol "lighting-control/internal/control"
	internalpolicy "lighting-control/internal/policy"
	"lighting-control/internal/policy/sources"
)

const defaultReconcileInterval = 30 * time.Second

type Service struct {
	store    internalpolicy.Store
	sources  *sources.Sources
	control  *internalcontrol.Service
	now      func() time.Time
	opMu     sync.Mutex
	mu       sync.Mutex
	lastAt   time.Time
	lastView *policyv1.EffectivePolicy
}

func NewService(store internalpolicy.Store, source *sources.Sources, control *internalcontrol.Service) *Service {
	return &Service{store: store, sources: source, control: control, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) Get(ctx context.Context) (*policyv1.PolicyConfig, *policyv1.EffectivePolicy, error) {
	config, _, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	if s.lastView != nil && s.now().Sub(s.lastAt) < 10*time.Second {
		view := cloneEffective(s.lastView)
		s.mu.Unlock()
		return config, view, nil
	}
	s.mu.Unlock()
	view, err := s.evaluate(ctx, false)
	return config, view, err
}

func (s *Service) SetConfig(ctx context.Context, config *policyv1.PolicyConfig) (*policyv1.PolicyConfig, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if config == nil {
		return nil, errors.New("policy config is required")
	}
	if err := (internalpolicy.Engine{Config: config}).Validate(); err != nil {
		return nil, err
	}
	current, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, err
	}
	if current.DefaultPreset != config.DefaultPreset {
		state.ActivePreset = policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
		state.Candidate = policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
		state.CandidateAt = time.Time{}
	}
	if err := s.store.Save(ctx, config, state); err != nil {
		return nil, err
	}
	s.invalidate()
	_, err = s.evaluateLocked(ctx, true)
	return config, err
}

func (s *Service) SetManual(ctx context.Context, kind policyv1.PresetKind, expiry time.Time) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, nil, err
	}
	now := s.now().UTC()
	engine := internalpolicy.Engine{Config: config}
	state, err = engine.SetManual(state, kind, expiry, now)
	if err != nil {
		return nil, nil, err
	}
	if err := s.store.Save(ctx, config, state); err != nil {
		return nil, nil, err
	}
	s.invalidate()
	effective, results, err := s.evaluateAndApplyLocked(ctx)
	return effective, results, err
}

func (s *Service) ClearManual(ctx context.Context) (bool, *policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return false, nil, nil, err
	}
	engine := internalpolicy.Engine{Config: config}
	wasActive := state.ManualPreset != policyv1.PresetKind_PRESET_KIND_UNSPECIFIED
	state = engine.ClearManual(state)
	if err := s.store.Save(ctx, config, state); err != nil {
		return false, nil, nil, err
	}
	s.invalidate()
	effective, results, err := s.evaluateAndApplyLocked(ctx)
	return wasActive, effective, results, err
}

func (s *Service) Evaluate(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	return s.evaluateAndApply(ctx)
}

func (s *Service) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = defaultReconcileInterval
	}
	_, _, _ = s.evaluateAndApply(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, _ = s.evaluateAndApply(ctx)
		}
	}
}

func (s *Service) evaluate(ctx context.Context, apply bool) (*policyv1.EffectivePolicy, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.evaluateLocked(ctx, apply)
}

func (s *Service) evaluateLocked(ctx context.Context, apply bool) (*policyv1.EffectivePolicy, error) {
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, err
	}
	inputs := s.sources.Read(ctx)
	state, effective, err := (internalpolicy.Engine{Config: config}).Evaluate(state, inputs)
	if err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, config, state); err != nil {
		return nil, err
	}
	if apply {
		if _, err := s.apply(ctx, config, effective); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	s.lastAt = inputs.Now
	s.lastView = cloneEffective(effective)
	s.mu.Unlock()
	return effective, nil
}

func (s *Service) Config(ctx context.Context) (*policyv1.PolicyConfig, error) {
	config, _, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	return config, err
}

func (s *Service) evaluateAndApply(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.evaluateAndApplyLocked(ctx)
}

func (s *Service) evaluateAndApplyLocked(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, nil, err
	}
	inputs := s.sources.Read(ctx)
	state, effective, err := (internalpolicy.Engine{Config: config}).Evaluate(state, inputs)
	if err != nil {
		return nil, nil, err
	}
	if err := s.store.Save(ctx, config, state); err != nil {
		return nil, nil, err
	}
	results, err := s.apply(ctx, config, effective)
	if err != nil {
		return nil, nil, err
	}
	s.mu.Lock()
	s.lastAt = inputs.Now
	s.lastView = cloneEffective(effective)
	s.mu.Unlock()
	return effective, results, nil
}

func (s *Service) apply(ctx context.Context, config *policyv1.PolicyConfig, effective *policyv1.EffectivePolicy) ([]*policyv1.PolicyZoneResult, error) {
	if s.control == nil {
		return nil, nil
	}
	_, controllers, err := s.control.ListControllers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list controllers for policy apply: %w", err)
	}
	results := make([]*policyv1.PolicyZoneResult, 0)
	for _, controller := range controllers {
		if controller == nil || !controller.Enrolled {
			continue
		}
		for _, zone := range controller.Zones {
			if zone == nil {
				continue
			}
			desired, staticFallback := desiredForZone(controller.ControllerId, zone, config, effective)
			result, err := s.control.SetZoneState(ctx, desired)
			if err != nil {
				return results, fmt.Errorf("apply policy to %s/%s: %w", controller.ControllerId, zone.ZoneId, err)
			}
			results = append(results, &policyv1.PolicyZoneResult{ControllerId: controller.ControllerId, ZoneId: zone.ZoneId, StaticFallback: staticFallback, Result: result})
		}
	}
	return results, nil
}

func desiredForZone(controllerID string, zone *controlv1.ZoneCapability, config *policyv1.PolicyConfig, effective *policyv1.EffectivePolicy) (*controlv1.ZoneState, bool) {
	desired := &controlv1.ZoneState{ControllerId: controllerID, ZoneId: zone.ZoneId}
	if effective.Preset == policyv1.PresetKind_PRESET_KIND_OFF {
		desired.DesiredMode = &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}
		return desired, false
	}
	color := effective.Color
	brightness := effective.BrightnessPercent
	fallback := effective.StaticFallback
	if fallback || color == nil {
		color = config.StaleFallbackColor
		brightness = config.StaleFallbackBrightnessPercent
		fallback = true
	}
	if color == nil {
		color = &controlv1.Color{}
	}
	if effective.EffectId != "" {
		if supported(zone.SupportedEffects, effective.EffectId) {
			desired.DesiredMode = &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: effective.EffectId, Color: color, BrightnessPercent: brightness}}
			return desired, fallback
		}
		fallback = true
	}
	desired.DesiredMode = &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: color, BrightnessPercent: brightness}}
	return desired, fallback
}

func supported(values []string, requested string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(requested)) {
			return true
		}
	}
	return false
}

func (s *Service) invalidate() {
	s.mu.Lock()
	s.lastAt = time.Time{}
	s.lastView = nil
	s.mu.Unlock()
}

func cloneEffective(v *policyv1.EffectivePolicy) *policyv1.EffectivePolicy {
	if v == nil {
		return nil
	}
	return proto.Clone(v).(*policyv1.EffectivePolicy)
}

func timeFromProto(ts *timestamppb.Timestamp) (time.Time, error) {
	if ts == nil {
		return time.Time{}, nil
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, err
	}
	return ts.AsTime().UTC(), nil
}
