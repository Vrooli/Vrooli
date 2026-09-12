package authcrypto

import (
	"fmt"

	"github.com/vrooli/api-core/storage"
)

// ScenarioID is the storage namespace owner for key material.
const scenarioID = "scenario-authenticator"

// ResolveKeyDir returns the absolute directory the RS256 keypair persists to,
// resolved through the storage seam (CORRECTION §8: replaces the old relative
// CWD path so the keypair survives wherever the lifecycle launches the binary,
// and lands beside the shadow namespace under a Baseline Modes engagement).
func ResolveKeyDir() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	ns, err := storage.ScenarioNamespace(scenarioID)
	if err != nil {
		return "", fmt.Errorf("resolve %s storage namespace: %w", scenarioID, err)
	}
	dir, err := resolver.Path(storage.Options{ScenarioID: ns}, storage.ClassData, "keys")
	if err != nil {
		return "", fmt.Errorf("resolve key directory: %w", err)
	}
	return dir, nil
}
