package control

import (
	"context"
	"reflect"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"github.com/google/uuid"
)

// ActuationCorrelationWindow links a transport event to the operator command
// that just caused it without misattributing a later physical-remote change.
const ActuationCorrelationWindow = 5 * time.Second

type actuationCause struct {
	id        string
	createdAt time.Time
}

type scopedStateSink struct {
	service   *Service
	deviceID  string
	transport string
}

func (s scopedStateSink) Publish(event strategy.StateChangeEvent) {
	event.DeviceID = s.deviceID
	if strings.TrimSpace(event.Transport) == "" {
		event.Transport = s.transport
	}
	s.service.EmitStateChange(event)
}

// observerJob identifies one reservation so a cancelled, stale worker cannot
// remove or replace an observer started for a later incarnation of a device.
type observerJob struct {
	cancel   context.CancelFunc
	stopping bool
	done     chan struct{}
}

func (s *Service) startObserverLocked(record devicedomain.Record) {
	if s.observerAdmissionClosed || s.observerCancels[record.ID] != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &observerJob{cancel: cancel, done: make(chan struct{})}
	s.observerCancels[record.ID] = job
	go s.runDeviceObserver(ctx, record, job)
}

func (s *Service) runDeviceObserver(ctx context.Context, record devicedomain.Record, job *observerJob) {
	transport := record.StrategyID
	defer func() {
		if job.done != nil {
			defer close(job.done)
		}
		if recover() != nil && ctx.Err() == nil {
			s.EmitStateChange(strategy.StateChangeEvent{DeviceID: record.ID, Transport: transport, Attribute: "transport_health", NewValue: "unreachable", StateClass: strategy.EventBearing})
		}
		job.cancel()
		s.mu.Lock()
		if s.observerCancels[record.ID] == job {
			delete(s.observerCancels, record.ID)
		}
		s.mu.Unlock()
	}()
	type candidate struct{ strategyID, endpoint string }
	candidates := make([]candidate, 0, len(record.Transports)+1)
	seen := make(map[candidate]bool)
	add := func(id, endpoint string) {
		item := candidate{strings.TrimSpace(id), strings.TrimSpace(endpoint)}
		if item.strategyID != "" && !seen[item] {
			seen[item] = true
			candidates = append(candidates, item)
		}
	}
	for _, profile := range record.Transports {
		id := profile.StrategyID
		if strings.TrimSpace(id) == "" {
			id = profile.Name
		}
		add(id, profile.Endpoint)
	}
	add(record.StrategyID, record.Endpoint)
	for _, selected := range candidates {
		transport = selected.strategyID
		if ctx.Err() != nil {
			return
		}
		base, ok := s.registry.Get(selected.strategyID)
		if !ok {
			continue
		}
		if scoped, ok := base.(strategy.DeviceScoped); ok && record.Serial != "" {
			base = scoped.ForDevice(record.Serial)
		}
		if scoped, ok := base.(interface {
			ForEndpoint(string) strategy.Strategy
		}); ok && selected.endpoint != "" {
			base = scoped.ForEndpoint(selected.endpoint)
		}
		observer, hasObserver := base.(strategy.StateObserver)
		reader, hasReader := base.(strategy.StateReader)
		if !hasObserver && !hasReader {
			continue
		}
		probeCtx, stopProbe := context.WithTimeout(ctx, 8*time.Second)
		declaration, err := base.Describe(probeCtx)
		stopProbe()
		if err != nil || ctx.Err() != nil {
			continue
		}
		mode, interval := declaration.StateObservation.Mode, declaration.StateObservation.Interval
		if mode == "" {
			mode = declaration.ObservationMode
		}
		if interval <= 0 {
			interval = declaration.ObservationInterval
		}
		if !hasObserver {
			if mode != "poll" {
				continue
			}
			if interval <= 0 {
				interval = time.Second
			}
			s.runPollObserver(ctx, record.ID, selected.strategyID, reader, interval)
			return
		}
		backoff := time.Second
		for {
			err := observer.ObserveState(ctx, scopedStateSink{service: s, deviceID: record.ID, transport: selected.strategyID})
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				s.EmitStateChange(strategy.StateChangeEvent{DeviceID: record.ID, Transport: selected.strategyID, Attribute: "transport_health", OldValue: "available", NewValue: "unreachable", StateClass: strategy.EventBearing, CausationID: uuid.NewString()})
			} else {
				s.EmitStateChange(strategy.StateChangeEvent{DeviceID: record.ID, Transport: selected.strategyID, Attribute: "transport_health", OldValue: "unreachable", NewValue: "available", StateClass: strategy.EventBearing, CausationID: uuid.NewString()})
			}
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}
			if backoff < 30*time.Second {
				backoff *= 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
		}
	}
}

func (s *Service) runPollObserver(ctx context.Context, deviceID, transport string, reader strategy.StateReader, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var previous map[string]strategy.PropertyValue
	for {
		state, err := reader.ReadState(ctx)
		if err != nil {
			s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: transport, Attribute: "transport_health", NewValue: "unreachable", StateClass: strategy.EventBearing})
		} else {
			for name, value := range state.Properties {
				old, existed := previous[name]
				if !existed || !reflect.DeepEqual(old.Value, value.Value) {
					var oldValue any
					if existed {
						oldValue = old.Value
					}
					s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: transport, Attribute: name, OldValue: oldValue, NewValue: value.Value, StateClass: strategy.StateBearing})
				}
			}
			previous = state.Properties
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SubscribeStateChanges exposes the local fast-path seam for a future rule
// engine. It intentionally has no vrooli-events integration.
func (s *Service) SubscribeStateChanges(buffer int) strategy.StateSubscription {
	s.mu.Lock()
	if s.stateEvents == nil {
		s.stateEvents = strategy.NewEventBus()
	}
	bus := s.stateEvents
	s.mu.Unlock()
	return bus.Subscribe(buffer)
}

func (s *Service) EmitStateChange(event strategy.StateChangeEvent) {
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	s.mu.Lock()
	if strings.TrimSpace(event.CausationID) == "" {
		if cause, ok := s.actuationCauses[event.DeviceID]; ok && time.Since(cause.createdAt) <= ActuationCorrelationWindow {
			event.CausationID = cause.id
		} else {
			event.CausationID = uuid.NewString()
		}
	}
	if s.stateEvents == nil {
		s.stateEvents = strategy.NewEventBus()
	}
	bus := s.stateEvents
	s.mu.Unlock()
	bus.Publish(event)
}
