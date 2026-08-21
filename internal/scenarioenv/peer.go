package scenarioenv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/peerrecord"
	"github.com/vrooli/vrooli/internal/scenario"
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

// Resolve adds declared peer bindings to values. Existing keys are an error,
// which gives resource exports and peer bindings one collision namespace.
func Resolve(ctx context.Context, home string, manifest scenario.ServiceManifest, values map[string]string) (map[string]string, error) {
	return resolve(ctx, home, manifest, values, runtimeStore)
}

func resolve(ctx context.Context, home string, manifest scenario.ServiceManifest, values map[string]string, open StoreFactory) (map[string]string, error) {
	resolved := make(map[string]string)
	peers := make([]string, 0, len(manifest.Dependencies.Scenarios))
	for name, dependency := range manifest.Dependencies.Scenarios {
		if len(dependency.Bindings) > 0 {
			peers = append(peers, name)
		}
	}
	slices.Sort(peers)

	var store peerStore
	defer func() {
		if store != nil {
			_ = store.Close()
		}
	}()

	for _, peer := range peers {
		dependency := manifest.Dependencies.Scenarios[peer]
		record, err := Read(home, peer)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("read peer %s: %w", peer, err)
		}
		if errors.Is(err, os.ErrNotExist) {
			if store == nil {
				store, err = open(ctx, home)
				if err != nil {
					return nil, fmt.Errorf("open scenario runtime registry: %w", err)
				}
			}
			record, err = projectRuntimePeer(ctx, store, peer)
		}

		for _, binding := range dependency.Bindings {
			if _, exists := values[binding.EnvVar]; exists {
				return nil, fmt.Errorf("peer env collision for %s: dependency %s conflicts with an existing scenario or resource value", binding.EnvVar, peer)
			}
			if _, exists := resolved[binding.EnvVar]; exists {
				return nil, fmt.Errorf("peer env collision for %s: more than one peer binding claims the variable", binding.EnvVar)
			}

			port, available := record.Ports[binding.Port]
			if err != nil || !available || port <= 0 {
				if binding.WhenUnavailable == "omit" {
					continue
				}
				return nil, fmt.Errorf("peer binding %s from scenario %s is unavailable", binding.EnvVar, peer)
			}
			value, err := formatBinding(binding.Form, port)
			if err != nil {
				return nil, fmt.Errorf("peer binding %s from scenario %s: %w", binding.EnvVar, peer, err)
			}
			resolved[binding.EnvVar] = value
		}
	}
	return resolved, nil
}

func formatBinding(form string, port int) (string, error) {
	switch strings.TrimSpace(form) {
	case "http_base_url":
		return fmt.Sprintf("http://127.0.0.1:%d", port), nil
	case "ws_base_url":
		return fmt.Sprintf("ws://127.0.0.1:%d", port), nil
	case "host_port":
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	case "port_number":
		return strconv.Itoa(port), nil
	default:
		return "", fmt.Errorf("unsupported form %q", form)
	}
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
