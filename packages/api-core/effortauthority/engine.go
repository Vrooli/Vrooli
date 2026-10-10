package effortauthority

import (
	"context"
	"encoding/json"
)

// Engine exposes semantic finite transactions, never arbitrary ledger mutation.
// Remote adapters hold neither the issuer database nor a client private key.
type Engine interface {
	CheckBinding(context.Context, Binding) (Policy, error)
	CheckProof(context.Context, Proof) (Binding, error)
	Prepare(context.Context, Proof, *Proof) (Reservation, bool, error)
	Reserve(context.Context, Binding, Intent, *Proof) (Reservation, bool, error)
	Accept(context.Context, Binding, string, string) error
	Settle(context.Context, Binding, string, string) error
	ObserveIngress(context.Context, Proof) (Reservation, bool, error)
	ReadReservation(context.Context, Binding, string) (Reservation, error)
}

func (a *Authority) ObserveIngress(ctx context.Context, proof Proof) (Reservation, bool, error) {
	b, err := a.CheckProof(ctx, proof)
	if err != nil {
		return Reservation{}, false, err
	}
	rec, err := a.live(ctx, b.PolicyID)
	if err != nil {
		return Reservation{}, false, err
	}
	digest, exists := rec.Ingress[proof.Intent.IdempotencyKey]
	if !exists {
		return Reservation{}, false, nil
	}
	slot, exists := rec.Reservations[proof.Intent.IdempotencyKey]
	if digest != Digest(proof.Intent) || !exists || slot.Preparation == nil {
		return Reservation{}, false, ErrRefused
	}
	return slot, true, nil
}
func (a *Authority) ReadReservation(ctx context.Context, b Binding, key string) (Reservation, error) {
	if _, err := a.CheckBinding(ctx, b); err != nil {
		return Reservation{}, err
	}
	rec, err := a.live(ctx, b.PolicyID)
	if err != nil {
		return Reservation{}, err
	}
	slot, exists := rec.Reservations[key]
	if !exists {
		return Reservation{}, ErrRefused
	}
	return slot, nil
}

// SerialEngine is an optional narrowly typed native-owner carrier. Public run
// callers cannot use a Binding or run label as authentication. Protected peers
// must have explicit exact-policy operation grants; no enrollment is implied.
type SerialEngine interface {
	PrepareSerialHandoff(context.Context, Binding, string, Intent) (SerialHandoff, bool, error)
	ReserveSerialEpisode(context.Context, Binding, string) (Reservation, bool, error)
	ReadSerialHandoff(context.Context, Binding, string) (SerialHandoff, error)
}

// SerialPayloadEngine binds bounded owner-derived task projection data to the
// exact next native intent. It exports no signer or new credential route.
type SerialPayloadEngine interface {
	SerialEngine
	PrepareSerialHandoffWithPayload(context.Context, Binding, string, Intent, json.RawMessage) (SerialHandoff, bool, error)
}
