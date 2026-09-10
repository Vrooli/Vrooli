package development

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

var (
	effectIdentityPattern  = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+$`)
	effectParameterPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	ErrEffectNotAllowed    = errors.New("development effect is not approved")
	ErrEffectUnsupported   = errors.New("development effect is unsupported")
	ErrEffectAmbiguous     = errors.New("development effect is ambiguous")
)

// EffectSpec is the owner contract for one effect identity. The strings in a
// proposal are only transport syntax; a spec names the boundary that must
// enforce the effect and the parameters that make its scope unambiguous.
type EffectSpec struct {
	Identity           string
	Owner              string
	RequiredParameters []string
	Validate           func(map[string]string) error
}

var effectSpecs = map[string]EffectSpec{
	"billing.reserve":   {Identity: "billing.reserve", Owner: "lpbs", RequiredParameters: []string{"account", "mode"}, Validate: validateAccountMode},
	"credentials.use":   {Identity: "credentials.use", Owner: "credential-owner", RequiredParameters: []string{"account", "name"}, Validate: validateAccountName},
	"filesystem.write":  {Identity: "filesystem.write", Owner: "workspace-sandbox", RequiredParameters: []string{"paths"}, Validate: validatePathParameter("paths")},
	"network.external":  {Identity: "network.external", Owner: "runner-policy", RequiredParameters: []string{"hosts"}, Validate: validateHostParameter},
	"network.internal":  {Identity: "network.internal", Owner: "scenario-owner", RequiredParameters: []string{"scope"}, Validate: validateScopeParameter},
	"process.test":      {Identity: "process.test", Owner: "agent-manager", RequiredParameters: []string{"scope"}, Validate: validateScopeParameter},
	"publication.write": {Identity: "publication.write", Owner: "publication-owner", RequiredParameters: []string{"target"}, Validate: validateTargetParameter},
}

// SupportedEffectSpecs returns a stable copy for review UIs and readiness
// reports. Callers cannot mutate the registry through the returned values.
func SupportedEffectSpecs() []EffectSpec {
	keys := make([]string, 0, len(effectSpecs))
	for key := range effectSpecs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]EffectSpec, 0, len(keys))
	for _, key := range keys {
		spec := effectSpecs[key]
		spec.RequiredParameters = append([]string(nil), spec.RequiredParameters...)
		result = append(result, spec)
	}
	return result
}

// Effect is the machine-readable side-effect capability carried by an owner
// request. Parameters are part of the approved identity and cannot be widened
// by the goal prompt or by a retry.
type Effect struct {
	Identity   string
	Parameters map[string]string
}

// ParseEffect accepts the transport-compatible form identity[key=value,...].
// Keeping the encoding in the existing repeated-string field avoids a second
// API shape while making the approved vocabulary structurally checkable.
func ParseEffect(raw string) (Effect, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Effect{}, fmt.Errorf("effect identity is required: %w", ErrInvalid)
	}
	effect := Effect{Parameters: map[string]string{}}
	identity := raw
	if open := strings.IndexByte(raw, '['); open >= 0 {
		if !strings.HasSuffix(raw, "]") || strings.Contains(raw[open+1:len(raw)-1], "[") {
			return Effect{}, fmt.Errorf("effect parameters are malformed: %w", ErrInvalid)
		}
		identity = strings.TrimSpace(raw[:open])
		body := strings.TrimSpace(raw[open+1 : len(raw)-1])
		if body != "" {
			for _, pair := range strings.Split(body, ",") {
				parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
				key := strings.TrimSpace(parts[0])
				if len(parts) != 2 || !effectParameterPattern.MatchString(key) || strings.TrimSpace(parts[1]) == "" || strings.ContainsAny(parts[1], "[]\n\r") {
					return Effect{}, fmt.Errorf("effect parameter %q is malformed: %w", pair, ErrInvalid)
				}
				if _, exists := effect.Parameters[key]; exists {
					return Effect{}, fmt.Errorf("effect parameter %q is repeated: %w", key, ErrEffectAmbiguous)
				}
				effect.Parameters[key] = strings.TrimSpace(parts[1])
			}
		}
	}
	if !effectIdentityPattern.MatchString(identity) {
		return Effect{}, fmt.Errorf("effect identity %q is not machine-readable: %w", identity, ErrInvalid)
	}
	effect.Identity = identity
	return effect, nil
}

func (e Effect) String() string {
	if len(e.Parameters) == 0 {
		return e.Identity
	}
	keys := make([]string, 0, len(e.Parameters))
	for key := range e.Parameters {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+e.Parameters[key])
	}
	return e.Identity + "[" + strings.Join(values, ",") + "]"
}

func validateAllowedEffects(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		effect, err := ParseEffect(raw)
		if err != nil {
			return fmt.Errorf("allowed effect %q is not machine-readable: %w", raw, err)
		}
		canonical, err := validateRegisteredEffect(effect)
		if err != nil {
			return fmt.Errorf("allowed effect %q is not supported: %w", raw, err)
		}
		if _, exists := seen[canonical]; exists {
			return fmt.Errorf("allowed effect %q is repeated: %w", raw, ErrEffectAmbiguous)
		}
		seen[canonical] = struct{}{}
	}
	return nil
}

func validateRequestedEffects(requested []Effect, allowed []string) error {
	if err := validateAllowedEffects(allowed); err != nil {
		return fmt.Errorf("approved effect set is invalid: %w", err)
	}
	approved := make(map[string]struct{}, len(allowed))
	for _, raw := range allowed {
		effect, err := ParseEffect(raw)
		if err != nil {
			return fmt.Errorf("approved effect set is invalid: %w", err)
		}
		canonical, err := validateRegisteredEffect(effect)
		if err != nil {
			return err
		}
		approved[canonical] = struct{}{}
	}
	seen := make(map[string]struct{}, len(requested))
	for _, effect := range requested {
		canonical, err := validateRegisteredEffect(effect)
		if err != nil {
			// A request outside the reviewed set is denied even when its name is
			// also unknown. Preserve both facts for callers and audit output.
			if parsed, parseErr := ParseEffect(effect.String()); parseErr == nil {
				if _, approvedHere := approved[parsed.String()]; !approvedHere {
					return fmt.Errorf("effect %q is outside the approved effect set and unsupported: %w: %w: %w", parsed.String(), err, ErrEffectNotAllowed, ErrDenied)
				}
			}
			return err
		}
		if _, exists := seen[canonical]; exists {
			return fmt.Errorf("effect %q is repeated: %w", canonical, ErrEffectAmbiguous)
		}
		seen[canonical] = struct{}{}
		if _, ok := approved[canonical]; !ok {
			return fmt.Errorf("effect %q is outside the approved effect set: %w: %w", canonical, ErrEffectNotAllowed, ErrDenied)
		}
	}
	return nil
}

func validateRegisteredEffect(effect Effect) (string, error) {
	parsed, err := ParseEffect(effect.String())
	if err != nil {
		return "", err
	}
	spec, ok := effectSpecs[parsed.Identity]
	if !ok {
		return "", fmt.Errorf("effect identity %q has no owning enforcement semantics: %w", parsed.Identity, ErrEffectUnsupported)
	}
	for _, required := range spec.RequiredParameters {
		if strings.TrimSpace(parsed.Parameters[required]) == "" {
			return "", fmt.Errorf("effect %q requires parameter %q: %w", parsed.Identity, required, ErrInvalid)
		}
	}
	required := make(map[string]struct{}, len(spec.RequiredParameters))
	for _, key := range spec.RequiredParameters {
		required[key] = struct{}{}
	}
	for key := range parsed.Parameters {
		if _, ok := required[key]; !ok {
			return "", fmt.Errorf("effect %q parameter %q is unsupported: %w", parsed.Identity, key, ErrEffectAmbiguous)
		}
	}
	if err := spec.Validate(parsed.Parameters); err != nil {
		return "", fmt.Errorf("effect %q: %w", parsed.Identity, err)
	}
	return parsed.String(), nil
}

func validatePathParameter(name string) func(map[string]string) error {
	return func(parameters map[string]string) error {
		value := strings.TrimSpace(parameters[name])
		if !fs.ValidPath(value) || strings.HasPrefix(value, "/") || strings.Contains(value, "..") || strings.Contains(value, "\\") {
			return fmt.Errorf("%s must be a repository-relative path pattern", name)
		}
		return nil
	}
}

func validateScopeParameter(parameters map[string]string) error {
	value := strings.TrimSpace(parameters["scope"])
	if slugPattern.MatchString(value) {
		return nil
	}
	if strings.HasPrefix(value, "scenarios/") && slugPattern.MatchString(strings.TrimPrefix(value, "scenarios/")) {
		return nil
	}
	return errors.New("scope must be a canonical scenario slug or scenarios/<slug>")
}

func validateTargetParameter(parameters map[string]string) error {
	value := strings.TrimSpace(parameters["target"])
	if !effectParameterPattern.MatchString(value) {
		return errors.New("target must be a canonical owner target")
	}
	return nil
}

func validateAccountMode(parameters map[string]string) error {
	if err := validateTargetParameter(map[string]string{"target": parameters["account"]}); err != nil {
		return fmt.Errorf("account: %w", err)
	}
	mode := strings.TrimSpace(parameters["mode"])
	if mode != "simulation" && mode != "live" {
		return errors.New("mode must be simulation or live")
	}
	return nil
}

func validateAccountName(parameters map[string]string) error {
	if err := validateTargetParameter(map[string]string{"target": parameters["account"]}); err != nil {
		return fmt.Errorf("account: %w", err)
	}
	if !effectParameterPattern.MatchString(strings.TrimSpace(parameters["name"])) {
		return errors.New("name must be a canonical credential name")
	}
	return nil
}

func validateHostParameter(parameters map[string]string) error {
	for _, host := range strings.Split(parameters["hosts"], ";") {
		host = strings.TrimSpace(host)
		if host == "" || strings.ContainsAny(host, "\\/\n\r") {
			return errors.New("hosts must be a semicolon-separated host allowlist")
		}
	}
	return nil
}
