package machines

import (
	"encoding/json"
	"sort"
)

// Profile is a versioned built-in policy template. It may suggest setup and
// scopes but cannot grant Registry authorization.
type Profile struct {
	ID                   string
	Version              string
	SetupEnvironment     string
	SuggestedScopes      []string
	RequiredCapabilities []string
}

type PolicySnapshot struct {
	MachineID            string
	ProfileID            string
	ProfileVersion       string
	SetupEnvironment     string
	SuggestedScopes      []string
	RequiredCapabilities []string
	JSON                 string
}

var builtInProfiles = map[string]Profile{
	"managed-connection": {ID: "managed-connection", Version: "v1", SetupEnvironment: "minimal", SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{"ssh.management"}},
	"presence":           {ID: "presence", Version: "v1", SetupEnvironment: "minimal", SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{"agent.presence"}},
	"deployment-target":  {ID: "deployment-target", Version: "v1", SetupEnvironment: "production", SuggestedScopes: []string{"presence.read", "provision.execute"}, RequiredCapabilities: []string{"agent.presence", "provision"}},
	"production-runtime": {ID: "production-runtime", Version: "v1", SetupEnvironment: "production", SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{"agent.presence", "runtime"}},
	"development-runner": {ID: "development-runner", Version: "v1", SetupEnvironment: "development", SuggestedScopes: []string{"presence.read", "runs.execute"}, RequiredCapabilities: []string{"agent.presence", "runner"}},
	// custom is intentionally a constrained built-in base plus explicit
	// overrides; it is not profile CRUD or an authorization escape hatch.
	"custom": {ID: "custom", Version: "v1", SetupEnvironment: "development", SuggestedScopes: []string{"presence.read"}, RequiredCapabilities: []string{"agent.presence"}},
}

func ResolveProfile(machineID, id, version string, overrides map[string]string) (PolicySnapshot, error) {
	profile, ok := builtInProfiles[id]
	if !ok {
		return PolicySnapshot{}, ErrInvalid{"profile_id", "unknown built-in profile"}
	}
	if version != "" && version != profile.Version {
		return PolicySnapshot{}, ErrInvalid{"profile_version", "unsupported"}
	}
	if env := overrides["setup_environment"]; env != "" {
		switch env {
		case "development", "production", "minimal":
		default:
			return PolicySnapshot{}, ErrInvalid{"overrides.setup_environment", "must be development, production, or minimal"}
		}
		profile.SetupEnvironment = env
	}
	// Map values hold slices; copy before sorting so one resolution cannot mutate
	// the registry and make a later snapshot depend on call order.
	profile.SuggestedScopes = append([]string(nil), profile.SuggestedScopes...)
	profile.RequiredCapabilities = append([]string(nil), profile.RequiredCapabilities...)
	sort.Strings(profile.SuggestedScopes)
	sort.Strings(profile.RequiredCapabilities)
	payload := struct {
		ProfileID, ProfileVersion, SetupEnvironment string
		SuggestedScopes, RequiredCapabilities       []string
	}{profile.ID, profile.Version, profile.SetupEnvironment, profile.SuggestedScopes, profile.RequiredCapabilities}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return PolicySnapshot{}, err
	}
	return PolicySnapshot{MachineID: machineID, ProfileID: profile.ID, ProfileVersion: profile.Version, SetupEnvironment: profile.SetupEnvironment, SuggestedScopes: profile.SuggestedScopes, RequiredCapabilities: profile.RequiredCapabilities, JSON: string(encoded)}, nil
}
