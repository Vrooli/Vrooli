package env

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/credentialspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const (
	credentialsParameterC = 2
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
//
// env is empty for a descriptor that names no environment variable. That is a
// declaration Vrooli-authored code resolves for itself through
// packages/credential-authority-go, so there is nothing to inject — but it is
// still declared, and therefore still diagnosed and still captured by a
// recovery bundle.
type resolvedDescriptor struct {
	descriptor credentialspec.Descriptor
	identity   credentialauthority.Identity
	env        string
	field      string
}

// validateDeclaration performs every check whose failure the operator cannot
// fix at runtime. These are the only conditions under which credential
// resolution is allowed to return an error. owner names the manifest for the
// message and is a resource or scenario name.
func validateDeclaration(owner string, declaration credentialspec.Declaration) ([]resolvedDescriptor, error) {
	if err := declaration.Validate(owner); err != nil {
		return nil, err
	}
	descriptors := declaration.All()
	resolved := make([]resolvedDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
		if err != nil {
			return nil, fmt.Errorf("%s credential %s: %w", owner, describeDescriptor(descriptor), err)
		}
		resolved = append(resolved, resolvedDescriptor{
			descriptor: descriptor,
			identity:   identity,
			env:        strings.TrimSpace(descriptor.Env),
			field:      descriptor.ResolvedField(),
		})
	}
	return resolved, nil
}

// describeDescriptor names a descriptor in an error. An injected one is best
// known by its variable; one resolved directly has only its field.
func describeDescriptor(descriptor credentialspec.Descriptor) string {
	if env := strings.TrimSpace(descriptor.Env); env != "" {
		return env
	}
	return descriptor.ResolvedField()
}

// validateCredentialDescriptors is the resource-manifest entry point.
func validateCredentialDescriptors(resourceManifest manifestpkg.ResourceManifest) ([]resolvedDescriptor, error) {
	return validateDeclaration("resource "+resourceManifest.Name, resourceManifest.Credentials)
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
		// A descriptor with no env names no injection target. Its consumer is
		// Vrooli-authored code that resolves the value itself, so the value is
		// never materialized here — but whether it is configured is still a
		// gap worth reporting, and Status answers that without reading it.
		if descriptor.env == "" {
			status := authority.Status(descriptor.identity, descriptor.field)
			if status.ProviderState != credentialauthority.ProviderAvailable {
				resolution.Provider = status.ProviderState
				reason := providerErrorFor(status)
				resolution.Missing = append(resolution.Missing, allMissing(resourceManifest.Name, descriptors[index:], reason)...)
				return resolution, nil
			}
			if !status.Configured {
				resolution.Missing = append(resolution.Missing,
					missingCredential(resourceManifest.Name, descriptor, credentialauthority.ErrUnconfigured))
			}
			continue
		}
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
	descriptors, err := validateCredentialDescriptors(resourceManifest)
	if err != nil {
		return CredentialResolution{}, err
	}
	return resolveGaps(resourceManifest.Name, descriptors), nil
}

// ResolveScenarioCredentialGaps reports the same for a scenario's own
// declaration. A scenario declares a credential when its own code resolves the
// value through packages/credential-authority-go rather than reading an
// injected variable; without this entry point such a credential would be
// undiagnosable and, more seriously, invisible to `recovery export --all`.
func ResolveScenarioCredentialGaps(scenarioName string, declaration credentialspec.Declaration) (CredentialResolution, error) {
	descriptors, err := validateDeclaration("scenario "+scenarioName, declaration)
	if err != nil {
		return CredentialResolution{}, err
	}
	return resolveGaps(scenarioName, descriptors), nil
}

// resolveGaps is the shared presence check. Keeping one body is what stops a
// scenario-declared credential and a resource-declared one from developing
// different ideas about what "configured" means.
func resolveGaps(owner string, descriptors []resolvedDescriptor) CredentialResolution {
	resolution := CredentialResolution{Provider: credentialauthority.ProviderAvailable}
	if len(descriptors) == 0 {
		return resolution
	}

	authority, authorityErr := credentialauthority.DefaultAuthority()
	if authorityErr != nil {
		resolution.Provider = credentialauthority.ProviderStateFor(authorityErr)
		resolution.Missing = allMissing(owner, descriptors, authorityErr)
		return resolution
	}

	for index, descriptor := range descriptors {
		status := authority.Status(descriptor.identity, descriptor.field)
		if status.ProviderState != credentialauthority.ProviderAvailable {
			resolution.Provider = status.ProviderState
			reason := providerErrorFor(status)
			resolution.Missing = append(resolution.Missing, allMissing(owner, descriptors[index:], reason)...)
			return resolution
		}
		if status.Configured {
			continue
		}
		resolution.Missing = append(resolution.Missing,
			missingCredential(owner, descriptor, credentialauthority.ErrUnconfigured))
	}
	return resolution
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
		credentialauthority.ProviderAbsent:      credentialsParameterC,
	}
	if severity[next] > severity[current] {
		return next
	}
	return current
}
