package effortauthority

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// NativeIsolationQualifier is supplied by trusted owner startup, not HTTP or
// configuration booleans. Continuing witnesses recheck actual current evidence.
type NativeIsolationQualifier interface{ Qualify(context.Context) error }

// Installation describes only immutable nonsecret bindings. Provider handles
// remain in the owning process; descriptor validation creates no host state.
type Installation struct {
	Enabled    bool              `json:"enabled"`
	Policy     Policy            `json:"policy"`
	Origin     string            `json:"origin"`
	Builds     map[string]string `json:"builds"`
	ServerPins map[string]string `json:"serverPins"`
}

func (s Installation) Freeze() (Installation, error) {
	raw, e := json.Marshal(s)
	if e != nil {
		return Installation{}, ErrRefused
	}
	var copy Installation
	if json.Unmarshal(raw, &copy) != nil {
		return Installation{}, ErrRefused
	}
	if !copy.Enabled {
		return copy, nil
	}
	u, e := url.Parse(copy.Origin)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || copy.Policy.Validate() != nil || copy.Policy.MaxTurns > 32 || copy.Policy.MaxToolCalls > 100 || copy.Policy.MaxRunSeconds > 3600 || copy.Policy.TotalTurns > 256 || copy.Policy.TotalToolCalls > 800 || copy.Policy.TotalRunSeconds > 28800 || copy.Policy.MaxStarts > 8 || copy.Policy.MaxConcurrent != 1 || copy.Policy.Deadline.Sub(copy.Policy.NotBefore) > 8*time.Hour {
		return Installation{}, ErrRefused
	}
	for _, role := range []string{"issuer", "pm", "am"} {
		if !validDigest(copy.Builds[role]) || !validDigest(copy.ServerPins[role]) {
			return Installation{}, ErrRefused
		}
	}
	return copy, nil
}

// MainInstallationGate runs before database/preflight/listener effects. Default
// mains remain disabled. Enabled configuration requires a concrete owner-built
// installation supplied through that service's typed startup constructor;
// mains do not invent acceptance/custody/unit-owner providers from environment.
func MainInstallationGate(value string, supplied bool) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", "false":
		return nil
	case "1", "true":
		if supplied {
			return nil
		}
		return fmt.Errorf("%w: enabled finite startup requires owner-built installation", ErrRefused)
	default:
		return ErrRefused
	}
}
