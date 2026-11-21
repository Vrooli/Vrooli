package machines

import (
	"encoding/json"
	"sort"
	"strings"

	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

// Profile is a versioned built-in policy template. It may suggest setup and
// scopes but cannot grant Registry authorization.
type Profile struct {
	ID                   string
	Version              string
	Preset               string
	Scenarios            []string
	OptionalResources    []string
	SuggestedScopes      []string
	RequiredCapabilities []string
}

type PolicySnapshot struct {
	MachineID            string
	ProfileID            string
	ProfileVersion       string
	Preset               string
	Scenarios            []string
	OptionalResources    []string
	SuggestedScopes      []string
	RequiredCapabilities []string
	SelectionJSON        string
	JSON                 string
}

var builtInProfiles = map[string]Profile{
	"managed-connection": {ID: "managed-connection", Version: "v1", Preset: "managed-connection", Scenarios: []string{"vrooli-bridge"}, SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{privilegedops.CapabilitySSHManagement}},
	"presence":           {ID: "presence", Version: "v1", Preset: "presence", Scenarios: []string{"vrooli-bridge"}, SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{privilegedops.CapabilityAgentPresence}},
	"deployment-target":  {ID: "deployment-target", Version: "v1", Preset: "deployment-target", Scenarios: []string{"deployment-manager", "vrooli-bridge"}, SuggestedScopes: []string{"presence.read", "provision.execute"}, RequiredCapabilities: []string{privilegedops.CapabilityAgentPresence, privilegedops.CapabilityProvisioning}},
	"production-runtime": {ID: "production-runtime", Version: "v1", Preset: "production-runtime", Scenarios: []string{"system-monitor", "vrooli-bridge"}, SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{privilegedops.CapabilityAgentPresence, privilegedops.CapabilityRuntime}},
	"development-runner": {ID: "development-runner", Version: "v1", Preset: "development-runner", Scenarios: []string{"test-genie", "vrooli-bridge"}, SuggestedScopes: []string{"presence.read", "runs.execute"}, RequiredCapabilities: []string{privilegedops.CapabilityAgentPresence, "runner"}},
	// custom is intentionally a constrained built-in base plus explicit
	// overrides; it is not profile CRUD or an authorization escape hatch.
	"custom": {ID: "custom", Version: "v1", Preset: "custom", Scenarios: []string{"vrooli-bridge"}, SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{privilegedops.CapabilityAgentPresence}},
}

func ResolveProfile(machineID, id, version string, overrides map[string]string) (PolicySnapshot, error) {
	profile, ok := builtInProfiles[id]
	if !ok {
		return PolicySnapshot{}, ErrInvalid{"profile_id", "unknown built-in profile"}
	}
	if version != "" && version != profile.Version {
		return PolicySnapshot{}, ErrInvalid{"profile_version", "unsupported"}
	}
	if preset := strings.TrimSpace(overrides["preset"]); preset != "" {
		if _, ok := builtInProfiles[preset]; !ok {
			return PolicySnapshot{}, ErrInvalid{"overrides.preset", "must name a built-in setup preset"}
		}
		profile.Preset = preset
	}
	if value, ok := overrides["scenarios"]; ok {
		profile.Scenarios = splitProfileList(value)
	}
	if value, ok := overrides["optional_resources"]; ok {
		profile.OptionalResources = splitProfileList(value)
	}
	// Map values hold slices; copy before sorting so one resolution cannot mutate
	// the registry and make a later snapshot depend on call order.
	profile.SuggestedScopes = append([]string(nil), profile.SuggestedScopes...)
	profile.Scenarios = append([]string(nil), profile.Scenarios...)
	profile.OptionalResources = append([]string(nil), profile.OptionalResources...)
	profile.RequiredCapabilities = append([]string(nil), profile.RequiredCapabilities...)
	sort.Strings(profile.Scenarios)
	sort.Strings(profile.OptionalResources)
	sort.Strings(profile.SuggestedScopes)
	sort.Strings(profile.RequiredCapabilities)
	payload := struct {
		ProfileID, ProfileVersion, Preset     string
		Scenarios, OptionalResources          []string
		SuggestedScopes, RequiredCapabilities []string
	}{profile.ID, profile.Version, profile.Preset, profile.Scenarios, profile.OptionalResources, profile.SuggestedScopes, profile.RequiredCapabilities}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return PolicySnapshot{}, err
	}
	selection, err := json.Marshal(&setupv1.Selection{SchemaVersion: "v1", Target: machineID, Scenarios: profile.Scenarios, OptionalResources: profile.OptionalResources, Apply: true})
	if err != nil {
		return PolicySnapshot{}, err
	}
	return PolicySnapshot{MachineID: machineID, ProfileID: profile.ID, ProfileVersion: profile.Version, Preset: profile.Preset, Scenarios: profile.Scenarios, OptionalResources: profile.OptionalResources, SuggestedScopes: profile.SuggestedScopes, RequiredCapabilities: profile.RequiredCapabilities, SelectionJSON: string(selection), JSON: string(encoded)}, nil
}

func splitProfileList(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}
