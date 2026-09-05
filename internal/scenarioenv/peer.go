package scenarioenv

import (
	"context"

	"github.com/vrooli/vrooli/internal/peerrecord"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const PeerSchemaVersion = peerrecord.SchemaVersion

// PeerRecord is the cross-tier discovery projection. It contains only runtime
// coordinates; credentials remain owned by their existing authorities.
type PeerRecord = peerrecord.Record

type peerStore interface {
	ListInstances(context.Context, scenarioruntime.InstanceFilter) ([]scenarioruntime.Instance, error)
	ListPortClaims(context.Context, scenarioruntime.PortClaimFilter) ([]scenarioruntime.PortClaim, error)
	Close() error
}

type StoreFactory func(context.Context, string) (peerStore, error)

// Write atomically publishes a mode-tuning.PermSecret peer record.
func Write(home string, record PeerRecord) error {
	return peerrecord.Write(home, record)
}

// Read rejects stale records whose declared owner is no longer alive.
func Read(home, name string) (PeerRecord, error) {
	return peerrecord.Read(home, name)
}

func Remove(home, name string) error {
	return peerrecord.Remove(home, name)
}
