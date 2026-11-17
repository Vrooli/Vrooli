package lifecycle

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AuthModeSourceManifest names where a managed scenario's authentication mode
// comes from: the manifest's authentication.default_mode. The caller's
// environment is never a source. The lifecycle records the source in each
// process environment (VROOLI_AUTH_MODE_SOURCE) so status can tell a process
// started under this rule from one started by an older control plane.
const (
	AuthModeSourceManifest = "manifest-default"

	authModeEnv       = "VROOLI_AUTH_MODE"
	authModeSourceEnv = "VROOLI_AUTH_MODE_SOURCE"
	authProvidersEnv  = "VROOLI_AUTH_PROVIDERS"
)

// AuthModePersonalLocal is the mode in which a scenario trusts its machine:
// every caller that reaches it over loopback is the owner.
const AuthModePersonalLocal = "personal_local"

// PersonalLocalRisk names what personal_local trusts. Local agents call
// scenarios over loopback, and a tunnel connector delivers remote requests
// over loopback, so neither is distinguished from the operator.
const PersonalLocalRisk = "every loopback caller, including local agents and requests a tunnel connector delivers, is treated as the machine owner"

// CheckPersonalLocal refuses personal_local where the manifest itself says
// the scenario's callers must be verified remotely, and otherwise returns the
// risk a start in that mode accepts. A start in any other mode returns
// neither.
//
// The refusals are declarations that contradict loopback trust: a
// cloudflare_access profile names Cloudflare Access as its only verifier, and
// a declared public hostname says the scenario is served to remote callers.
// A tunnel route alone is not a refusal: tunnel-manager puts every route
// behind an Access application, and the scenarios that run personal_local
// today are reached that way. The start warning names the residual risk.
func CheckPersonalLocal(profile *scenario.AuthenticationProfile, mode string) (risk string, err error) {
	if profile == nil || strings.TrimSpace(mode) != AuthModePersonalLocal {
		return "", nil
	}
	if strings.EqualFold(strings.TrimSpace(profile.Profile), "cloudflare_access") {
		return "", fmt.Errorf("authentication profile cloudflare_access verifies every caller through Cloudflare Access, so default_mode %s is refused: %s", AuthModePersonalLocal, PersonalLocalRisk)
	}
	if hostname := strings.TrimSpace(profile.Hostname); hostname != "" {
		return "", fmt.Errorf("authentication declares public hostname %q, so default_mode %s is refused: %s", hostname, AuthModePersonalLocal, PersonalLocalRisk)
	}
	return PersonalLocalRisk, nil
}

// AuthModeResolution is the authentication mode a managed start applies and
// where it came from.
type AuthModeResolution struct {
	Mode   string `json:"mode,omitempty"`
	Source string `json:"source"`
}

// ResolveAuthMode selects the mode for a scenario's authentication profile:
// the declared default_mode. Changing a scenario's mode is a change to its
// declared configuration, never to the shell that starts it.
func ResolveAuthMode(profile *scenario.AuthenticationProfile) AuthModeResolution {
	if profile == nil {
		return AuthModeResolution{}
	}
	return AuthModeResolution{Mode: strings.TrimSpace(profile.DefaultMode), Source: AuthModeSourceManifest}
}

// ProcessAuthentication is the authentication setting a running scenario
// process actually carries, read from its environment.
type ProcessAuthentication struct {
	Step      string `json:"step,omitempty"`
	PID       int    `json:"pid"`
	Mode      string `json:"mode,omitempty"`
	Source    string `json:"source,omitempty"`
	Providers string `json:"providers,omitempty"`
	Error     string `json:"error,omitempty"`
}

// AuthenticationStatus describes a scenario's authentication configuration:
// the mode its next start applies (with its source) and the mode each running
// process carries. Mismatch is set when a running process differs from the
// configured mode, which means it was started from a different configuration
// (for example by a control plane that still honoured caller environment).
type AuthenticationStatus struct {
	Profile        string                  `json:"profile"`
	SupportedModes []string                `json:"supported_modes,omitempty"`
	Configured     AuthModeResolution      `json:"configured"`
	Processes      []ProcessAuthentication `json:"processes,omitempty"`
	Mismatch       bool                    `json:"mismatch,omitempty"`
	// Risk names what the configured mode trusts when it is personal_local.
	Risk string `json:"risk,omitempty"`
	// Refused is why the next start will refuse the configured mode.
	Refused string `json:"refused,omitempty"`
}

// DescribeAuthentication reports a scenario's configured and observed
// authentication mode. It returns nil for a scenario without an
// authentication profile.
func DescribeAuthentication(item scenario.Scenario, records []process.Record) *AuthenticationStatus {
	profile := item.Manifest.Authentication
	if profile == nil {
		return nil
	}
	status := &AuthenticationStatus{
		Profile:        strings.TrimSpace(profile.Profile),
		SupportedModes: slices.Clone(profile.SupportedModes),
		Configured:     ResolveAuthMode(profile),
	}
	if risk, err := CheckPersonalLocal(profile, status.Configured.Mode); err != nil {
		status.Refused = err.Error()
	} else {
		status.Risk = risk
	}
	for _, record := range records {
		if record.PID <= 0 {
			continue
		}
		observed := observeProcessAuthentication(record.PID)
		observed.Step = record.Step
		status.Processes = append(status.Processes, observed)
		if observed.Error == "" && observed.Mode != status.Configured.Mode {
			status.Mismatch = true
		}
	}
	return status
}

// readProcessEnviron is replaced in tests.
var readProcessEnviron = func(pid int) ([]byte, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("process environment is not observable on %s", runtime.GOOS)
	}
	return os.ReadFile("/proc/" + strconv.Itoa(pid) + "/environ")
}

func observeProcessAuthentication(pid int) ProcessAuthentication {
	observed := ProcessAuthentication{PID: pid}
	data, err := readProcessEnviron(pid)
	if err != nil {
		observed.Error = err.Error()
		return observed
	}
	for _, entry := range strings.Split(string(data), "\x00") {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		switch key {
		case authModeEnv:
			observed.Mode = value
		case authModeSourceEnv:
			observed.Source = value
		case authProvidersEnv:
			observed.Providers = value
		}
	}
	return observed
}
