package peerrecord

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

// This file owns the derived on-disk peer identity. Its source participates
// in the persistent-state contract because old readers recompute these paths.

func peerDir(home string) string {
	return filepath.Join(home, repocontractmeta.ProjectConfigDir, "peers")
}

func Path(home, name string) string {
	if node, scenario, variant, err := cliutil.SplitAddress(name); err == nil && node == "" {
		name = cliutil.InstanceSlug(scenario, variant)
	}
	return filepath.Join(peerDir(home), name+".json")
}

func targetAddress(address string) (slug, scenario, variant string, err error) {
	node, scenario, variant, err := cliutil.SplitAddress(address)
	if err != nil {
		return "", "", "", err
	}
	if node != "" {
		return "", "", "", fmt.Errorf("peer record target must be local")
	}
	if cliutil.BareScenarioName(scenario) != scenario {
		return "", "", "", fmt.Errorf("peer record target has an invalid scenario name")
	}
	if strings.ContainsAny(scenario, `/\\`) || strings.ContainsAny(variant, `/\\`) {
		return "", "", "", fmt.Errorf("peer record target contains a path separator")
	}
	// Peer record identity is the supplied address/record identity, not the
	// caller's routing preference. ResolveVariant would let ambient shadow
	// configuration or a CLI override redirect a publisher's live record.
	if variant == "" {
		variant = cliutil.DefaultVariant
	}
	slug = cliutil.InstanceSlug(scenario, variant)
	_, variant = cliutil.SplitInstance(slug)
	if variant == "" {
		variant = cliutil.DefaultVariant
	}
	return slug, scenario, variant, nil
}

func recordTarget(scenario, instance string) (slug, normalizedScenario, normalizedInstance string, err error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" || cliutil.BareScenarioName(scenario) != scenario {
		return "", "", "", fmt.Errorf("record scenario must be a bare scenario name")
	}
	instance = strings.TrimSpace(instance)
	if strings.ContainsAny(scenario, `/\\`) || strings.ContainsAny(instance, `/\\`) {
		return "", "", "", fmt.Errorf("record identity contains a path separator")
	}
	address := scenario
	if instance != "" {
		address += "@" + instance
	}
	return targetAddress(address)
}
