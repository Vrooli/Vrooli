package scenarioenv

import (
	"context"
	"os"

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

func runtimeStore(ctx context.Context, home string) (peerStore, error) {
	return scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home, ReadOnly: true})
}

func projectRuntimePeer(ctx context.Context, store peerStore, name string) (PeerRecord, error) {
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{
		Scenario: name,
		Variant:  scenarioruntime.DefaultVariant,
		Statuses: []string{scenarioruntime.StatusRunning},
	})
	if err != nil {
		return PeerRecord{}, err
	}
	if len(instances) == 0 {
		return PeerRecord{}, os.ErrNotExist
	}
	instance := instances[0]
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		InstanceID: instance.InstanceID,
		Statuses:   []string{scenarioruntime.ClaimStatusBound},
	})
	if err != nil {
		return PeerRecord{}, err
	}
	ports := make(map[string]int, len(claims))
	for _, claim := range claims {
		ports[claim.PortName] = claim.Port
	}
	ownerPID := 0
	if instance.OwnerPID != nil {
		ownerPID = *instance.OwnerPID
	}
	return PeerRecord{
		SchemaVersion: PeerSchemaVersion,
		Scenario:      instance.Scenario,
		Instance:      instance.Variant,
		Tier:          1,
		OwnerPID:      ownerPID,
		StartedAt:     instance.StartedAt,
		Ports:         ports,
	}, nil
}

// Write atomically publishes a mode-0600 peer record.
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
