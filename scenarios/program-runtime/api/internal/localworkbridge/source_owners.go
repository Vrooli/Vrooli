package localworkbridge

import (
	"context"
	"encoding/json"
)

// CanonicalEventReader is the finite structural seam implemented by the Events
// owner's bridge.SelectedReader. Its callbacks execute under the real shared
// authority/session boundary. Neither a worker nor HTTP input supplies them.
// Implementations own canonical storage, bounded decoding and buffer ownership.
type CanonicalEventReader interface {
	ReadSelected(context.Context, string, func(context.Context) error, func(context.Context, string, string, string) error) ([]byte, error)
}

type CollectorOwner interface {
	AdmitCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
	ObserveCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
	WaitCollector(context.Context, *SourcePermit) (SourceBrokerProgram, error)
}

// FiniteSourceOwners composes canonical finite owner adapters. It is not a
// transport, does not discover services, and is never registered by main.
type FiniteSourceOwners struct {
	enabled   bool
	events    CanonicalEventReader
	collector CollectorOwner
}

func NewFiniteSourceOwners(enabled bool, events CanonicalEventReader, collector CollectorOwner) (*FiniteSourceOwners, error) {
	o := &FiniteSourceOwners{enabled: enabled}
	if !enabled {
		return o, nil
	}
	if events == nil || collector == nil {
		return nil, ErrSourceAuthority
	}
	o.events, o.collector = events, collector
	return o, nil
}
func (o *FiniteSourceOwners) ReadEvent(ctx context.Context, p *SourcePermit) (json.RawMessage, error) {
	if o == nil || !o.enabled {
		return nil, ErrDisabled
	}
	if ctx == nil || p == nil || p.call.Operation != "local-work.event.read" || p.Recheck(ctx) != nil {
		return nil, ErrSourceAuthority
	}
	raw, err := o.events.ReadSelected(ctx, p.call.EventID, p.Recheck, func(ctx context.Context, id, typ, source string) error {
		return p.CheckEvent(ctx, SourceEventIdentity{ID: id, Type: typ, Scenario: source})
	})
	if err != nil || len(raw) > sourceBrokerResponseLimit || p.Recheck(ctx) != nil {
		return nil, ErrSourceAuthority
	}
	// Validate before exposing any owner bytes, even when called outside Execute.
	if _, err = brokerEventResult(ctx, p, raw); err != nil {
		return nil, ErrSourceAuthority
	}
	return append(json.RawMessage(nil), raw...), nil
}
func (o *FiniteSourceOwners) collectorPermit(ctx context.Context, p *SourcePermit, operation string) error {
	if o == nil || !o.enabled {
		return ErrDisabled
	}
	if ctx == nil || p == nil || p.call.Operation != operation || p.Recheck(ctx) != nil {
		return ErrSourceAuthority
	}
	return nil
}
func (o *FiniteSourceOwners) AdmitCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	if err := o.collectorPermit(ctx, p, "local-work.collector.admit"); err != nil {
		return SourceBrokerProgram{}, err
	}
	return o.collector.AdmitCollector(ctx, p)
}
func (o *FiniteSourceOwners) ObserveCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	if err := o.collectorPermit(ctx, p, "local-work.collector.observe"); err != nil {
		return SourceBrokerProgram{}, err
	}
	return o.collector.ObserveCollector(ctx, p)
}
func (o *FiniteSourceOwners) WaitCollector(ctx context.Context, p *SourcePermit) (SourceBrokerProgram, error) {
	if err := o.collectorPermit(ctx, p, "local-work.collector.wait"); err != nil {
		return SourceBrokerProgram{}, err
	}
	return o.collector.WaitCollector(ctx, p)
}

var _ SourceBrokerOwners = (*FiniteSourceOwners)(nil)
