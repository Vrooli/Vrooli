package env

import (
	"errors"
	"fmt"
	"strings"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

// CredentialGapReason names why a declared credential is not usable right now.
// The three reasons map one-to-one onto the credential-authority taxonomy and
// each one carries a different operator action.
type CredentialGapReason string

const (
	// GapUnconfigured: the store works and holds no value. Provision it.
	GapUnconfigured CredentialGapReason = "unconfigured"
	// GapProviderUnavailable: the store exists but is unreachable. Repair the
	// session.
	GapProviderUnavailable CredentialGapReason = "provider_unavailable"
	// GapProviderAbsent: this host has no credential backend at all.
	GapProviderAbsent CredentialGapReason = "provider_absent"
)

// MissingCredential is one declared credential that did not resolve. It is
// data, not a failure: a scenario starts with gaps, and the resources that
// needed the value report unhealthy until it is provisioned.
type MissingCredential struct {
	Resource  string `json:"resource"`
	Env       string `json:"env"`
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Label     string `json:"label,omitempty"`
	// Required mirrors the manifest declaration so a consumer can rank gaps.
	// It no longer decides whether the start survives — every gap is survivable.
	Required bool                `json:"required"`
	Reason   CredentialGapReason `json:"reason"`
	// Detail explains a host condition. It never contains a credential value.
	Detail string `json:"detail,omitempty"`
	// Remediation is the exact operator action that closes this gap.
	Remediation string `json:"remediation"`
}

// CredentialResolution is what credential resolution returns instead of an
// error. Giving partial success its own vocabulary is the whole point: with
// only (values, error) available, an unreachable keyring had no way to express
// itself except by unwinding the scenario start.
type CredentialResolution struct {
	// Values holds only what actually resolved. A credential that did not
	// resolve is absent from the map rather than present and empty, so a
	// consumer checking presence never sees a configured-but-blank credential.
	Values   map[string]string                 `json:"values,omitempty"`
	Missing  []MissingCredential               `json:"missing,omitempty"`
	Provider credentialauthority.ProviderState `json:"provider_state"`
}

// resolvedDescriptor is a manifest descriptor after the checks that can only
// be fixed by editing the manifest have already passed.
type resolvedDescriptor struct {
	descriptor manifestpkg.CredentialDescriptor
	identity   credentialauthority.Identity
	env        string
	field      string
}

// validateCredentialDescriptors performs every check whose failure the
// operator cannot fix at runtime. These are the only conditions under which
// credential resolution is allowed to return an error.
func validateCredentialDescriptors(resourceManifest manifestpkg.ResourceManifest) ([]resolvedDescriptor, error) {
	descriptors := resourceManifest.Credentials.All()
	resolved := make([]resolvedDescriptor, 0, len(descriptors))
	declared := map[string]string{}
	for _, descriptor := range descriptors {
		envName := strings.TrimSpace(descriptor.Env)
		if envName == "" {
			return nil, fmt.Errorf("resource %s declares a credential with an empty env name", resourceManifest.Name)
		}
		if prior, duplicate := declared[envName]; duplicate {
			return nil, fmt.Errorf(
				"resource %s declares credential env %s twice, for %s and %s",
				resourceManifest.Name, envName, prior, strings.TrimSpace(descriptor.LogicalID))
		}
		identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
		if err != nil {
			return nil, fmt.Errorf("resource %s credential %s: %w", resourceManifest.Name, envName, err)
		}
		declared[envName] = strings.TrimSpace(descriptor.LogicalID)

		field := strings.TrimSpace(descriptor.Field)
		if field == "" {
			field = "value"
		}
		resolved = append(resolved, resolvedDescriptor{
			descriptor: descriptor,
			identity:   identity,
			env:        envName,
			field:      field,
		})
	}
	return resolved, nil
}

// ResolveCredentialValues resolves a resource's declared credentials into a
// report. It returns an error only for a manifest defect — an unparsable
// logical_id, an empty env name, or two descriptors fighting over one env
// name. Host conditions and unset values are reported as data so that no
// credential state can prevent a scenario from starting.
func ResolveCredentialValues(resourceManifest manifestpkg.ResourceManifest) (CredentialResolution, error) {
	resolution := CredentialResolution{
		Values:   map[string]string{},
		Provider: credentialauthority.ProviderAvailable,
	}
	descriptors, err := validateCredentialDescriptors(resourceManifest)
	if err != nil {
		return CredentialResolution{}, err
	}
	if len(descriptors) == 0 {
		return resolution, nil
	}

	authority, authorityErr := credentialauthority.DefaultAuthority()
	if authorityErr != nil {
		resolution.Provider = credentialauthority.ProviderStateFor(authorityErr)
		resolution.Missing = allMissing(resourceManifest.Name, descriptors, authorityErr)
		return resolution, nil
	}

	for index, descriptor := range descriptors {
		injectErr := authority.Inject(descriptor.identity, descriptor.field, descriptor.env, resolution.Values)
		if injectErr == nil {
			continue
		}
		state := credentialauthority.ProviderStateFor(injectErr)
		if state != credentialauthority.ProviderAvailable {
			// The provider itself is down, so every remaining descriptor has
			// the same answer. Short-circuiting also keeps a six-descriptor
			// resource from spawning six doomed backend calls per start.
			resolution.Provider = state
			resolution.Missing = append(resolution.Missing, allMissing(resourceManifest.Name, descriptors[index:], injectErr)...)
			return resolution, nil
		}
		resolution.Missing = append(resolution.Missing, missingCredential(resourceManifest.Name, descriptor, injectErr))
	}
	return resolution, nil
}

// ResolveCredentialGaps reports which declared credentials are unusable
// without materializing any value. A presence check must not pay the cost — or
// carry the exposure — of a full secret read.
func ResolveCredentialGaps(resourceManifest manifestpkg.ResourceManifest) (CredentialResolution, error) {
	resolution := CredentialResolution{Provider: credentialauthority.ProviderAvailable}
	descriptors, err := validateCredentialDescriptors(resourceManifest)
	if err != nil {
		return CredentialResolution{}, err
	}
	if len(descriptors) == 0 {
		return resolution, nil
	}

	authority, authorityErr := credentialauthority.DefaultAuthority()
	if authorityErr != nil {
		resolution.Provider = credentialauthority.ProviderStateFor(authorityErr)
		resolution.Missing = allMissing(resourceManifest.Name, descriptors, authorityErr)
		return resolution, nil
	}

	for index, descriptor := range descriptors {
		status := authority.Status(descriptor.identity, descriptor.field)
		if status.ProviderState != credentialauthority.ProviderAvailable {
			resolution.Provider = status.ProviderState
			reason := providerErrorFor(status)
			resolution.Missing = append(resolution.Missing, allMissing(resourceManifest.Name, descriptors[index:], reason)...)
			return resolution, nil
		}
		if status.Configured {
			continue
		}
		resolution.Missing = append(resolution.Missing,
			missingCredential(resourceManifest.Name, descriptor, credentialauthority.ErrUnconfigured))
	}
	return resolution, nil
}

// providerErrorFor rebuilds the sentinel a Status implies so gap construction
// has a single code path for both the value and presence entry points.
func providerErrorFor(status credentialauthority.Status) error {
	detail := strings.TrimSpace(status.ProviderDetail)
	if detail == "" {
		detail = string(status.ProviderState)
	}
	if status.ProviderState == credentialauthority.ProviderAbsent {
		return fmt.Errorf("%w: %s", credentialauthority.ErrProviderAbsent, detail)
	}
	return fmt.Errorf("%w: %s", credentialauthority.ErrProviderUnavailable, detail)
}

func allMissing(resourceName string, descriptors []resolvedDescriptor, reason error) []MissingCredential {
	missing := make([]MissingCredential, 0, len(descriptors))
	for _, descriptor := range descriptors {
		missing = append(missing, missingCredential(resourceName, descriptor, reason))
	}
	return missing
}

func missingCredential(resourceName string, descriptor resolvedDescriptor, reason error) MissingCredential {
	gap := MissingCredential{
		Resource:  resourceName,
		Env:       descriptor.env,
		LogicalID: string(descriptor.identity),
		Field:     descriptor.field,
		Label:     strings.TrimSpace(descriptor.descriptor.Label),
		Required:  descriptor.descriptor.Required,
		Reason:    gapReason(reason),
	}
	if gap.Reason != GapUnconfigured {
		gap.Detail = reason.Error()
	}
	gap.Remediation = remediation(gap, descriptor.descriptor.ObtainURL)
	return gap
}

func gapReason(err error) CredentialGapReason {
	switch {
	case errors.Is(err, credentialauthority.ErrProviderAbsent):
		return GapProviderAbsent
	case errors.Is(err, credentialauthority.ErrProviderUnavailable):
		return GapProviderUnavailable
	default:
		return GapUnconfigured
	}
}

// remediation names the exact command or host fix that closes a gap. An
// operator should never have to translate an error into an action.
func remediation(gap MissingCredential, obtainURL string) string {
	switch gap.Reason {
	case GapProviderUnavailable:
		return "the credential store is unreachable; run `vrooli credentials doctor` for the host diagnosis"
	case GapProviderAbsent:
		return "this host has no credential backend; run `vrooli credentials doctor` to see what to install"
	default:
		instruction := fmt.Sprintf(
			"provision it: `vrooli credentials provision --identity %s --field %s` (value is read from stdin)",
			gap.LogicalID, gap.Field)
		if url := strings.TrimSpace(obtainURL); url != "" {
			instruction += "; obtain a value at " + url
		}
		return instruction
	}
}

// mergeProviderState keeps the most severe condition seen across resources. An
// aggregate that reported "available" because one uncredentialed resource
// resolved cleanly would hide the outage that matters.
func mergeProviderState(current, next credentialauthority.ProviderState) credentialauthority.ProviderState {
	severity := map[credentialauthority.ProviderState]int{
		credentialauthority.ProviderAvailable:   0,
		credentialauthority.ProviderUnavailable: 1,
		credentialauthority.ProviderAbsent:      2,
	}
	if severity[next] > severity[current] {
		return next
	}
	return current
}
