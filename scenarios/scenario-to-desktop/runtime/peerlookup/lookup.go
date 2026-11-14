// Package peerlookup owns desktop peer precedence and version constraints.
package peerlookup

import (
	"errors"
	"fmt"
	"github.com/vrooli/vrooli/internal/peerrecord"
	runtimemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"net/url"
	"slices"
	"strings"
)

func Local(manifest *runtimemanifest.Manifest, scenario, portKey string) (string, string, error) {
	scenario = strings.TrimSpace(scenario)
	if !ValidScenarioName(scenario) {
		return "", "", errors.New("invalid scenario name")
	}
	serviceType := map[string]string{"API_PORT": "api-binary", "UI_PORT": "ui-bundle"}[portKey]
	if serviceType == "" {
		return "", "", fmt.Errorf("unsupported port name: %s", portKey)
	}
	matches := make([]runtimemanifest.Service, 0, 1)
	for _, service := range manifest.Services {
		if service.ComponentSlice != "" {
			continue
		}
		if service.Type != serviceType || service.Ports == nil {
			continue
		}
		if !serviceBelongsToScenario(manifest, service.ID, scenario) {
			continue
		}
		matches = append(matches, service)
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("no service for scenario %s and type %s", scenario, serviceType)
	}
	if scenario != manifest.App.Scenario {
		if len(matches) != 1 {
			return "", "", fmt.Errorf("expected exactly one local service for scenario %s and type %s, found %d", scenario, serviceType, len(matches))
		}
		versionRange, constrained := peerVersionRange(manifest, scenario)
		if constrained {
			version, hasEmbeddedVersion := peerEmbeddedVersion(manifest, scenario)
			if matches[0].ComponentID != "" {
				// Verified installed owner metadata supplies this runtime-only
				// version; a discover edge has no embedded release version.
				version = matches[0].ComponentVersion
				hasEmbeddedVersion = version != ""
			}
			if !hasEmbeddedVersion || !runtimemanifest.VersionSatisfiesRange(version, versionRange) {
				return "", "", fmt.Errorf("local peer %s version %q does not satisfy declared range %q", scenario, version, versionRange)
			}
		}
	}
	service := matches[0]
	portName := "http"
	if len(service.Ports.Requested) > 0 && service.Ports.Requested[0].Name != "" {
		portName = service.Ports.Requested[0].Name
	}
	return service.ID, portName, nil
}

func serviceBelongsToScenario(bundle *runtimemanifest.Manifest, serviceID, scenario string) bool {
	if bundle != nil && bundle.App.Scenario == scenario {
		return !strings.Contains(serviceID, "--")
	}
	return strings.HasPrefix(serviceID, scenario+"--")
}

func ValidScenarioName(value string) bool {
	if value == "" || strings.ContainsAny(value, `/\\`) || strings.Contains(value, "@") || strings.Contains(value, "://") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "" && parsed.Host == "" && parsed.Path == value
}

// External chooses a compatible full peer before an embedded candidate.
func External(bundle *runtimemanifest.Manifest, home, scenario, portName string) (int, error) {
	if !ValidScenarioName(scenario) {
		return 0, errors.New("invalid scenario name")
	}
	if portName != "API_PORT" && portName != "UI_PORT" {
		return 0, fmt.Errorf("unsupported port name: %s", portName)
	}
	if bundle != nil && scenario == bundle.App.Scenario {
		return 0, nil
	}
	record, err := peerrecord.Read(home, scenario)
	if err != nil {
		return 0, err
	}
	versionRange, constrained := peerVersionRange(bundle, scenario)
	if constrained && !runtimemanifest.VersionSatisfiesRange(record.Version, versionRange) {
		return 0, errors.New("peer version does not satisfy the declared range")
	}
	key := strings.TrimSuffix(strings.ToLower(portName), "_port")
	if record.Provides != nil && !slices.Contains(record.Provides, key) {
		return 0, errors.New("peer does not provide requested surface")
	}
	return record.Ports[key], nil
}
func HasEmbedded(bundle *runtimemanifest.Manifest, scenario string) bool {
	if bundle == nil {
		return false
	}
	for _, service := range bundle.Services {
		if strings.HasPrefix(service.ID, scenario+"--") {
			return true
		}
	}
	return false
}

func peerVersionRange(bundle *runtimemanifest.Manifest, scenario string) (string, bool) {
	if bundle == nil {
		return "", false
	}
	for _, peers := range [][]runtimemanifest.Peer{bundle.ProgramBindingPeers, bundle.Peers} {
		for _, peer := range peers {
			if peer.Scenario == scenario {
				return strings.TrimSpace(peer.VersionRange), strings.TrimSpace(peer.VersionRange) != ""
			}
		}
	}
	return "", false
}

func peerEmbeddedVersion(bundle *runtimemanifest.Manifest, scenario string) (string, bool) {
	if bundle == nil {
		return "", false
	}
	for _, peer := range bundle.Peers {
		if peer.Scenario == scenario {
			return strings.TrimSpace(peer.EmbeddedVersion), strings.TrimSpace(peer.EmbeddedVersion) != ""
		}
	}
	return "", false
}

// FeatureExternal resolves only an explicitly advertised full-peer capability.
// Missing or ineligible peers return errors; generic port lookup is never a fallback.
func FeatureExternal(bundle *runtimemanifest.Manifest, home, owner, feature, versionRange string) (int, error) {
	_, port, err := FeaturePeer(bundle, home, owner, feature, versionRange)
	return port, err
}

// FeaturePeer returns one validated snapshot so a health probe and stand-down
// can retain the exact provider identity they admitted. Callers must revalidate
// after those effects; a port alone does not identify a provider generation.
func FeaturePeer(bundle *runtimemanifest.Manifest, home, owner, feature, versionRange string) (peerrecord.Record, int, error) {
	if !ValidScenarioName(owner) || strings.TrimSpace(feature) == "" {
		return peerrecord.Record{}, 0, errors.New("invalid feature owner or ID")
	}
	if bundle != nil && owner == bundle.App.Scenario {
		return peerrecord.Record{}, 0, errors.New("feature peer is the current runtime")
	}
	record, err := peerrecord.Read(home, owner)
	if err != nil {
		return peerrecord.Record{}, 0, err
	}
	declared, ok := record.Features[feature]
	if !ok {
		return peerrecord.Record{}, 0, errors.New("peer does not declare requested feature")
	}
	if !runtimemanifest.VersionSatisfiesRange(declared.OwnerVersion, versionRange) || !runtimemanifest.VersionSatisfiesRange(declared.OwnerVersion, FeatureVersionRange(bundle, owner)) {
		return peerrecord.Record{}, 0, errors.New("peer feature owner version does not satisfy requested range")
	}
	port := record.Ports[declared.Port]
	if declared.Port == "" || port <= 0 || port > 65535 {
		return peerrecord.Record{}, 0, errors.New("peer feature has no valid published port")
	}
	return record, port, nil
}

// FeatureVersionRange returns the current consumer edge constraint, with program bindings taking precedence.
func FeatureVersionRange(bundle *runtimemanifest.Manifest, owner string) string {
	constraint, _ := peerVersionRange(bundle, owner)
	return constraint
}
