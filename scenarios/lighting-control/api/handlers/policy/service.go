package policy

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
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
)

const defaultReconcileInterval = 30 * time.Second

type Service struct {
	store    internalpolicy.Store
	sources  sourceReader
	control  controlWriter
	now      func() time.Time
	opMu     sync.Mutex
	lastAt   time.Time
	lastView *policyv1.EffectivePolicy
}

type sourceReader interface {
	Read(context.Context) internalpolicy.Inputs
}

type controlWriter interface {
	ListControllers(context.Context, ...bool) (*controlv1.BackendStatus, []*controlv1.Controller, error)
	ApplyPolicyState(context.Context, *controlv1.ZoneState) (*controlv1.ZoneApplyResult, error)
	Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error)
	PolicyRevision(context.Context) (int64, error)
	SelectPolicy(context.Context, internalcontrol.PolicySelection) (bool, error)
}

func NewService(store internalpolicy.Store, source sourceReader, control controlWriter) *Service {
	if writer, ok := control.(*internalcontrol.Service); ok {
		writer.BindBrightnessPolicy(store)
	}
	return &Service{store: store, sources: source, control: control, now: func() time.Time { return time.Now().UTC() }}
}

// Get returns a read-only presentation preview. Only explicit mutations and
// the background cadence commit engine transitions or control ownership.
func (s *Service) Get(ctx context.Context) (*policyv1.PolicyConfig, *policyv1.EffectivePolicy, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	config, state, err := internalpolicy.LoadForRead(ctx, s.store)
	if err != nil {
		return nil, nil, err
	}
	if cachedPolicyFresh(s.lastView, s.lastAt, s.now(), config.SourceMaxAgeSeconds) {
		return config, cloneEffective(s.lastView), nil
	}
	inputs := s.sources.Read(ctx)
	_, view, err := evaluatePolicy(config, state, inputs)
	if err != nil {
		return nil, nil, err
	}
	s.lastAt, s.lastView = inputs.Now, cloneEffective(view)
	return config, view, nil
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
	save := func() error { return s.store.Save(ctx, config, state) }
	if writer, ok := s.control.(interface {
		UpdateBrightnessPolicy(context.Context, func() error) error
	}); ok && current.BrightnessCapPercent != config.BrightnessCapPercent {
		err = writer.UpdateBrightnessPolicy(ctx, save)
	} else {
		err = save()
	}
	if err != nil {
		s.invalidate()
		return nil, err
	}
	s.invalidate()
	// A maximum-cap edit takes effect now, including fresh automatic output.
	// Reconciliation never replays saved automatic effects from stale evidence.
	if current.BrightnessCapPercent != config.BrightnessCapPercent && s.sources != nil {
		if _, _, err := s.evaluateLocked(ctx); err != nil {
			return nil, fmt.Errorf("maximum brightness saved; fresh policy apply pending: %w", err)
		}
	}
	// Ordinary preset definition edits remain drafts until Evaluate or selection.
	return config, nil
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
	if s.control != nil {
		revision, err := s.control.PolicyRevision(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("prepare manual selection: %w", err)
		}
		state.ManualControlRevision = &revision
	}
	state.ManualSelectionID = rand.Text()
	state.ManualSelectionPending = true
	if err := s.store.Save(ctx, config, state); err != nil {
		return nil, nil, err
	}
	s.invalidate()
	if _, err := s.finishSelection(ctx, config, state); err != nil {
		return nil, nil, err
	}
	effective, results, err := s.evaluateLocked(ctx)
	return effective, results, err
}

// finishSelection retries only an explicitly saved operator selection.
func (s *Service) finishSelection(ctx context.Context, config *policyv1.PolicyConfig, state internalpolicy.State) (internalpolicy.State, error) {
	if !state.ManualSelectionPending {
		return state, nil
	}
	if state.ManualPreset < policyv1.PresetKind_PRESET_KIND_OFF || state.ManualPreset > policyv1.PresetKind_PRESET_KIND_STATIC {
		return state, errors.New("saved pending selection has no valid preset; control state preserved")
	}
	// Legacy pending requests gain a stable identity before touching control.
	if state.ManualSelectionID == "" {
		state.ManualSelectionID = rand.Text()
		if err := s.store.Save(ctx, config, state); err != nil {
			return state, fmt.Errorf("manual selection identity checkpoint pending: %w", err)
		}
	}
	if !state.ManualExpiry.IsZero() && !s.now().Before(state.ManualExpiry) {
		state = (internalpolicy.Engine{Config: config}).ClearManual(state)
	} else {
		if s.control != nil {
			var expires *timestamppb.Timestamp
			if !state.ManualExpiry.IsZero() {
				expires = timestamppb.New(state.ManualExpiry)
			}
			selected, err := s.control.SelectPolicy(ctx, internalcontrol.PolicySelection{ID: state.ManualSelectionID, BaseRevision: state.ManualControlRevision, RequestedAt: state.ManualRequestedAt, Off: state.ManualPreset == policyv1.PresetKind_PRESET_KIND_OFF, ExpiresAt: expires})
			if err != nil {
				return state, fmt.Errorf("manual selection saved; control selection pending: %w", err)
			}
			if !selected {
				state = (internalpolicy.Engine{Config: config}).ClearManual(state)
			}
		}
		state.ManualSelectionPending = false
	}
	if err := s.store.Save(ctx, config, state); err != nil {
		return state, fmt.Errorf("control selection completed; policy checkpoint pending: %w", err)
	}
	return state, nil
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
	effective, results, err := s.evaluateLocked(ctx)
	return wasActive, effective, results, err
}

func (s *Service) Evaluate(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	return s.evaluateAndApply(ctx)
}

func (s *Service) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = defaultReconcileInterval
	}
	s.reconcileTick(ctx)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileTick(ctx)
		}
	}
}

func (s *Service) reconcileTick(ctx context.Context) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err == nil {
		_, err = s.finishSelection(ctx, config, state)
	}
	if err != nil {
		log.Printf("lighting-control selection reconciliation pending: %v", err)
		return
	}
	s.reconcileControl(ctx)
	if _, _, err := s.evaluateLocked(ctx); err != nil {
		log.Printf("lighting-control policy reconciliation failed: %v", err)
	}
}

// reconcileControl runs at startup and on the bounded policy cadence so a
// durable manual Off or saved zone state is retried after backend/device return.
func (s *Service) reconcileControl(ctx context.Context) {
	if s.control == nil {
		return
	}
	if _, err := s.control.Reconcile(ctx); err != nil && !errors.Is(err, internalcontrol.ErrBackendUnavailable) {
		log.Printf("lighting-control desired-state reconcile failed: %v", err)
	}
}

func (s *Service) Config(ctx context.Context) (*policyv1.PolicyConfig, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	config, _, err := internalpolicy.LoadForRead(ctx, s.store)
	return config, err
}

func (s *Service) evaluateAndApply(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.evaluateLocked(ctx)
}

// evaluateLocked is the authoritative writer, called only by explicit
// mutations or the background reconciler while holding opMu.
func (s *Service) evaluateLocked(ctx context.Context) (*policyv1.EffectivePolicy, []*policyv1.PolicyZoneResult, error) {
	config, state, err := internalpolicy.LoadOrInitialize(ctx, s.store)
	if err != nil {
		return nil, nil, err
	}
	state, err = s.finishSelection(ctx, config, state)
	if err != nil {
		return nil, nil, err
	}
	inputs := s.sources.Read(ctx)
	state, effective, err := evaluatePolicy(config, state, inputs)
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
	s.lastAt, s.lastView = inputs.Now, cloneEffective(effective)
	return effective, results, nil
}

// evaluatePolicy shares the single engine between previews and committed ticks.
func evaluatePolicy(config *policyv1.PolicyConfig, state internalpolicy.State, inputs internalpolicy.Inputs) (internalpolicy.State, *policyv1.EffectivePolicy, error) {
	state, effective, err := (internalpolicy.Engine{Config: config}).Evaluate(state, inputs)
	if err != nil {
		return state, nil, err
	}
	if state.ManualSelectionPending {
		effective.Reason = "manual selection saved; control selection pending; " + effective.Reason
	}
	return state, effective, nil
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
			result, err := s.control.ApplyPolicyState(ctx, desired)
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
	if color == nil {
		color = config.StaleFallbackColor
		brightness = config.StaleFallbackBrightnessPercent
		fallback = true
	}
	brightness = min(brightness, config.BrightnessCapPercent)
	if color == nil {
		color = &controlv1.Color{}
	}
	if effect := strings.TrimSpace(effective.EffectId); effect != "" && !fallback {
		for _, supported := range zone.SupportedEffects {
			if strings.EqualFold(strings.TrimSpace(supported), effect) {
				desired.DesiredMode = &controlv1.ZoneState_Effect{Effect: &controlv1.EffectZoneState{EffectId: effect, Color: color, BrightnessPercent: brightness}}
				return desired, fallback
			}
		}
		// The policy's requested effect is not advertised by this exact zone.
		// Keep the color and brightness while reporting static fallback.
		fallback = true
	}
	desired.DesiredMode = &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: color, BrightnessPercent: brightness}}
	return desired, fallback
}

func (s *Service) invalidate() {
	s.lastAt = time.Time{}
	s.lastView = nil
}

func cachedPolicyFresh(view *policyv1.EffectivePolicy, at, now time.Time, maxAge uint32) bool {
	if view == nil || now.Before(at) || now.Sub(at) >= 10*time.Second {
		return false
	}
	if expiry := view.RequestExpiresAt; expiry != nil && !now.Before(expiry.AsTime()) {
		return false
	}
	for _, source := range view.Evidence {
		if source.State != policyv1.EvidenceState_EVIDENCE_STATE_FRESH {
			continue
		}
		stamp := source.SourceTimestamp
		if stamp == nil {
			stamp = source.ObservedAt
		}
		if stamp == nil || now.Sub(stamp.AsTime()) > time.Duration(maxAge)*time.Second {
			return false
		}
	}
	return true
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
