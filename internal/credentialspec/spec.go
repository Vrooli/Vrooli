// Package credentialspec holds the single credential declaration shape shared
// by resource manifests and scenario service manifests.
//
// It sits below both of them for the same reason hostreqspec does: resource
// manifests import internal/scenario, so a type both need cannot live in
// either one. Before this package existed only resources could declare a
// credential, which left every scenario-owned secret — tunnel-manager's
// Cloudflare token, for one — with no declaration at all. Undeclared meant
// invisible to `credentials list`, to `credentials doctor`, and, worst of the
// three, to `recovery export --all`: a credential nothing declares is a
// credential no backup captures.
package credentialspec

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultField is the field name used when a descriptor omits one. A single
// unnamed value is the common case; naming it keeps the store key total.
const DefaultField = "value"

var tierNamePattern = regexp.MustCompile(`^tier-[1-5]-[a-z0-9-]+$`)
var metadataRefPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)
var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateEnvironmentName keeps runtime injection on the conventional process
// environment namespace. Rejecting malformed names at the shared contract
// boundary prevents callers from treating an assignment, path, or whitespace-
// containing string as a child-process environment key.
func ValidateEnvironmentName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || !environmentNamePattern.MatchString(name) {
		return fmt.Errorf("credential environment name %q must match [A-Za-z_][A-Za-z0-9_]*", name)
	}
	return nil
}

// Descriptor declares one credential without binding it to Vault, an
// environment variable, or a local file. Values are always held by the
// credential authority.
type Descriptor struct {
	Version string `json:"version,omitempty"`
	// LogicalID and Field are the durable, backend-neutral name. They are the
	// only part of a descriptor the store ever sees.
	LogicalID string `json:"logical_id"`
	Field     string `json:"field,omitempty"`

	// Env is the process-scoped injection name, and it is optional on purpose.
	//
	// Declare it only when the consumer is a process Vrooli does not author —
	// a database container, a third-party CLI — which can receive a value no
	// other way. Vrooli-authored code resolves through
	// packages/credential-authority-go instead, which keeps the value out of
	// the process environment, where it would be readable at
	// /proc/<pid>/environ and inherited by every subprocess the consumer
	// spawns.
	//
	// A descriptor with no Env still participates fully in status reporting,
	// diagnosis, and recovery. It simply is not injected anywhere.
	Env string `json:"env,omitempty"`

	Required    bool   `json:"required,omitempty"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	// Placeholder is safe, non-secret sample text shown in operator input
	// controls. It communicates the expected shape without ever representing a
	// usable credential value.
	Placeholder string `json:"placeholder,omitempty"`
	ObtainURL   string `json:"obtain_url,omitempty"`
	// Provisioning identifies who supplies the value. The default is operator.
	//
	// A derived value is written by the declaring component once derived_from is
	// available. A generated value is minted by the declaring component out of
	// nothing — a signing secret has no external source and no place an operator
	// could go to obtain it, so asking for one is asking a question with no
	// answer. Neither kind should be requested from the operator, and neither
	// should block configuration while it is absent, because the operator cannot
	// supply it.
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
	// Tiers limits a descriptor to named deployment tiers. Empty means all
	// tiers, preserving the ordinary runtime credential declaration.
	Tiers []string `json:"tiers,omitempty"`

	// Provider-neutral setup metadata. These fields point at registered owner
	// actions and typed settings; they never contain a secret value or a
	// provider-shaped storage path.
	Owner                string         `json:"owner,omitempty"`
	Kind                 string         `json:"kind,omitempty"`
	Provider             string         `json:"provider,omitempty"`
	AppliesWhen          *Applicability `json:"applies_when,omitempty"`
	RequirementGroup     string         `json:"requirement_group,omitempty"`
	ConsumerRefs         []string       `json:"consumer_refs,omitempty"`
	CompanionSettings    []string       `json:"companion_settings,omitempty"`
	CompanionCredentials []string       `json:"companion_credentials,omitempty"`
	AcquisitionRef       string         `json:"acquisition_ref,omitempty"`
	VerificationRef      string         `json:"verification_ref,omitempty"`
	RecoveryRef          string         `json:"recovery_ref,omitempty"`
	HelpRef              string         `json:"help_ref,omitempty"`
	EvidencePolicy       string         `json:"evidence_policy,omitempty"`
	ProviderVersion      string         `json:"provider_version,omitempty"`
}

// Consumer declares an owner-observed runtime use of a credential reference.
// It may describe a literal authority call, an environment injection, or a
// dynamic/generated/delegated path that cannot be recovered safely from a
// source-text scan. SourceRef is relative to the declaring owner when it is
// stored in a manifest; inventory projections expand it to an absolute path.
type Consumer struct {
	LogicalID      string   `json:"logical_id,omitempty"`
	AddressPattern string   `json:"address_pattern,omitempty"`
	Field          string   `json:"field,omitempty"`
	Kind           string   `json:"kind"`
	Consumer       string   `json:"consumer"`
	SourceRef      string   `json:"source_ref"`
	Required       bool     `json:"required,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Tiers          []string `json:"tiers,omitempty"`
}

// MigrationDiagnostic is an actionable, non-fatal report for declarations
// that still use the original logical-id/field contract. Diagnostics keep old
// manifests readable while giving owners a bounded path to richer setup
// semantics; they never expose or inspect a credential value.
type MigrationDiagnostic struct {
	Address  string `json:"address"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// MigrationDiagnostics reports missing provider-neutral metadata without
// making it a runtime validation error. Existing descriptors intentionally
// remain valid and continue to resolve through the same authority address.
func (c Declaration) MigrationDiagnostics() []MigrationDiagnostic {
	result := make([]MigrationDiagnostic, 0)
	for _, descriptor := range c.Descriptors {
		address := strings.TrimSpace(descriptor.LogicalID) + ":" + descriptor.ResolvedField()
		add := func(code, severity, message string) {
			result = append(result, MigrationDiagnostic{Address: address, Code: code, Severity: severity, Message: message})
		}
		if strings.TrimSpace(descriptor.Version) == "" {
			add("descriptor_version_missing", "info", "add version credential-descriptor/v1 when this owner next updates the declaration")
		}
		if strings.TrimSpace(descriptor.Owner) == "" {
			add("owner_missing", "warning", "declare the owner that supplies setup, verification, and recovery actions")
		}
		if strings.TrimSpace(descriptor.Kind) == "" {
			add("kind_missing", "info", "declare whether this input is secret, public, generated, delegated, external, or hardware-backed")
		}
		if descriptor.OperatorSupplied() && strings.TrimSpace(descriptor.ObtainURL) == "" {
			add("acquisition_guide_missing", "warning", "provide an owner-approved acquisition guide or registered acquisition action")
		}
		if strings.TrimSpace(descriptor.AcquisitionRef) == "" {
			add("acquisition_action_missing", "info", "register the owner action that acquires or imports this input")
		}
		if strings.TrimSpace(descriptor.VerificationRef) == "" {
			add("verification_action_missing", "info", "register the owner action that verifies format, identity, and effective permission")
		}
		if strings.TrimSpace(descriptor.RecoveryRef) == "" {
			add("recovery_action_missing", "info", "register the owner recovery or reissue action")
		}
	}
	return result
}

// The provisioning kinds. Compare against these rather than string literals so
// a new kind is a compile-time question rather than a grep.
const (
	// ProvisioningOperator is the default: a person supplies the value.
	ProvisioningOperator = "operator"
	// ProvisioningDerived: the declaring component writes it once the credential
	// named by DerivedFrom is available.
	ProvisioningDerived = "derived"
	// ProvisioningGenerated: the declaring component mints it, typically on
	// first start. There is nowhere for an operator to obtain such a value.
	ProvisioningGenerated = "generated"

	// DescriptorVersion is the first provider-neutral metadata contract. It is
	// optional on declarations so existing manifests can migrate without
	// changing their stable authority address.
	DescriptorVersion = "credential-descriptor/v1"

	KindSecret    = "secret"
	KindPublic    = "public"
	KindGenerated = "generated"
	KindDelegated = "delegated"
	KindExternal  = "external"
	KindHardware  = "hardware"
)

// Applicability describes the operator context in which a descriptor is
// required. It is intentionally a small data language: owners register the
// actions that interpret these facts, while consumers only use the result to
// explain why an input is or is not in scope.
type Applicability struct {
	All []ApplicabilityRule `json:"all,omitempty"`
	Any []ApplicabilityRule `json:"any,omitempty"`
	Not *ApplicabilityRule  `json:"not,omitempty"`
}

type ApplicabilityRule struct {
	Eq  *ApplicabilityMatch `json:"eq,omitempty"`
	Neq *ApplicabilityMatch `json:"neq,omitempty"`
}

type ApplicabilityMatch struct {
	Context     string `json:"context,omitempty"`
	Setting     string `json:"setting,omitempty"`
	Operation   string `json:"operation,omitempty"`
	Role        string `json:"role,omitempty"`
	Target      string `json:"target,omitempty"`
	Environment string `json:"environment,omitempty"`
	Capability  string `json:"capability,omitempty"`
	Provider    string `json:"provider,omitempty"`
	Value       string `json:"value"`
}

// EffectiveKind gives consumers a migration-safe classification for legacy
// descriptors that predate the explicit kind field.
func (d Descriptor) EffectiveKind() string {
	if kind := strings.TrimSpace(d.Kind); kind != "" {
		return kind
	}
	switch strings.TrimSpace(d.Provisioning) {
	case ProvisioningGenerated:
		return KindGenerated
	case ProvisioningDerived:
		return KindDelegated
	default:
		return KindSecret
	}
}

// OperatorSupplied reports whether a person has to provide this value. It is
// the single predicate every surface should use to decide whether to prompt for
// a credential, and whether its absence is the operator's problem.
func (d Descriptor) OperatorSupplied() bool {
	switch strings.TrimSpace(d.Provisioning) {
	case ProvisioningDerived, ProvisioningGenerated:
		return false
	default:
		return true
	}
}

// ResolvedField returns the field this descriptor addresses, applying the
// default. Callers must not re-implement the fallback: a descriptor that
// resolved to "value" in one place and "" in another would read and write
// two different store keys for one declaration.
func (d Descriptor) ResolvedField() string {
	if field := strings.TrimSpace(d.Field); field != "" {
		return field
	}
	return DefaultField
}

// Injectable reports whether this descriptor names a process environment
// variable, and is therefore destined for a process Vrooli does not author.
func (d Descriptor) Injectable() bool { return strings.TrimSpace(d.Env) != "" }

// Declaration is the credentials block on a manifest.
type Declaration struct {
	// Descriptors is the sole credential declaration. The control plane
	// addresses values by LogicalID, never by a provider-shaped path.
	Descriptors []Descriptor `json:"descriptors,omitempty"`
	// Consumers are explicit owner registrations for runtime references that
	// cannot be attributed from a manifest descriptor or a literal source call.
	// They are metadata only and never carry a credential value.
	Consumers []Consumer `json:"consumers,omitempty"`
}

// All returns the canonical descriptors. It intentionally does not synthesize
// descriptors from legacy fields: doing so would make legacy declarations a
// permanent runtime contract.
func (c Declaration) All() []Descriptor {
	return append([]Descriptor(nil), c.Descriptors...)
}

// Injectable returns only the descriptors bound to an environment variable,
// which is the subset an environment resolver has any business with.
func (c Declaration) Injectable() []Descriptor {
	out := make([]Descriptor, 0, len(c.Descriptors))
	for _, descriptor := range c.Descriptors {
		if descriptor.Injectable() {
			out = append(out, descriptor)
		}
	}
	return out
}

// Validate reports the declaration defects an operator cannot fix at runtime,
// so they surface when a manifest is read rather than when a scenario starts.
// Host conditions and unset values are deliberately not checked here: those
// are reported as data, never as a manifest error.
func (c Declaration) Validate(owner string) error {
	byEnv := map[string]string{}
	byKey := map[string]string{}
	for _, descriptor := range c.Descriptors {
		identity := strings.TrimSpace(descriptor.LogicalID)
		if identity == "" {
			return fmt.Errorf("%s declares a credential with no logical_id", owner)
		}
		field := descriptor.ResolvedField()
		if strings.ContainsAny(field, "/\\") {
			return fmt.Errorf("%s credential %s field %q cannot contain a path separator", owner, identity, field)
		}
		if version := strings.TrimSpace(descriptor.Version); version != "" && version != DescriptorVersion {
			return fmt.Errorf("%s credential %s:%s uses unsupported descriptor version %q", owner, identity, field, version)
		}
		if kind := strings.TrimSpace(descriptor.Kind); kind != "" {
			switch kind {
			case KindSecret, KindPublic, KindGenerated, KindDelegated, KindExternal, KindHardware:
			default:
				return fmt.Errorf("%s credential %s:%s has invalid kind %q", owner, identity, field, descriptor.Kind)
			}
		}
		for name, value := range map[string]string{
			"owner": descriptor.Owner, "provider": descriptor.Provider,
			"requirement_group": descriptor.RequirementGroup, "acquisition_ref": descriptor.AcquisitionRef,
			"verification_ref": descriptor.VerificationRef, "recovery_ref": descriptor.RecoveryRef,
			"help_ref": descriptor.HelpRef, "evidence_policy": descriptor.EvidencePolicy,
			"provider_version": descriptor.ProviderVersion,
		} {
			if err := validateMetadataValue(owner, identity, field, name, value, false); err != nil {
				return err
			}
		}
		for name, values := range map[string][]string{
			"consumer_refs": descriptor.ConsumerRefs, "companion_settings": descriptor.CompanionSettings,
			"companion_credentials": descriptor.CompanionCredentials,
		} {
			seen := map[string]struct{}{}
			for _, value := range values {
				if err := validateMetadataValue(owner, identity, field, name, value, true); err != nil {
					return err
				}
				if _, duplicate := seen[value]; duplicate {
					return fmt.Errorf("%s credential %s:%s declares %s %q twice", owner, identity, field, name, value)
				}
				seen[value] = struct{}{}
			}
		}
		if descriptor.AppliesWhen != nil {
			if err := descriptor.AppliesWhen.Validate(owner, identity, field); err != nil {
				return err
			}
		}
		provisioning := strings.TrimSpace(descriptor.Provisioning)
		if provisioning != "" && provisioning != ProvisioningOperator && provisioning != ProvisioningDerived && provisioning != ProvisioningGenerated {
			return fmt.Errorf("%s credential %s:%s has invalid provisioning %q", owner, identity, field, provisioning)
		}
		if provisioning == ProvisioningDerived && strings.TrimSpace(descriptor.DerivedFrom) == "" {
			return fmt.Errorf("%s credential %s:%s derived provisioning requires derived_from", owner, identity, field)
		}
		// A generated value has no source credential by definition. Naming one
		// would describe a derivation that does not exist.
		if provisioning == ProvisioningGenerated && strings.TrimSpace(descriptor.DerivedFrom) != "" {
			return fmt.Errorf("%s credential %s:%s generated provisioning cannot declare derived_from", owner, identity, field)
		}
		seenTiers := map[string]struct{}{}
		for _, tier := range descriptor.Tiers {
			tier = strings.TrimSpace(tier)
			if !tierNamePattern.MatchString(tier) {
				return fmt.Errorf("%s credential %s:%s has invalid tier %q", owner, identity, field, tier)
			}
			if _, duplicate := seenTiers[tier]; duplicate {
				return fmt.Errorf("%s credential %s:%s declares tier %q twice", owner, identity, field, tier)
			}
			seenTiers[tier] = struct{}{}
		}

		// Two descriptors addressing one store key is a declaration that
		// cannot mean two things at once, whether or not either is injected.
		key := identity + ":" + field
		if prior, duplicate := byKey[key]; duplicate {
			return fmt.Errorf("%s declares %s twice, as %s and %s", owner, key, prior, describe(descriptor))
		}
		byKey[key] = describe(descriptor)

		if !descriptor.Injectable() {
			continue
		}
		env := strings.TrimSpace(descriptor.Env)
		if err := ValidateEnvironmentName(env); err != nil {
			return fmt.Errorf("%s credential %s:%s: %w", owner, identity, field, err)
		}
		if prior, duplicate := byEnv[env]; duplicate {
			return fmt.Errorf("%s declares credential env %s twice, for %s and %s", owner, env, prior, identity)
		}
		byEnv[env] = identity
	}
	seenConsumers := map[string]struct{}{}
	for _, consumer := range c.Consumers {
		identity := strings.TrimSpace(consumer.LogicalID)
		pattern := strings.TrimSpace(consumer.AddressPattern)
		if identity == "" && pattern == "" {
			return fmt.Errorf("%s declares a consumer with no logical_id or address_pattern", owner)
		}
		if identity != "" && pattern != "" {
			return fmt.Errorf("%s consumer %s cannot declare both logical_id and address_pattern", owner, identity)
		}
		field := consumer.Field
		if strings.TrimSpace(field) == "" {
			field = DefaultField
		}
		if strings.ContainsAny(field, "/\\") {
			return fmt.Errorf("%s consumer %s field %q cannot contain a path separator", owner, identity, field)
		}
		kind := strings.TrimSpace(consumer.Kind)
		switch kind {
		case "authority", "environment", "generated", "provider_default", "dynamic", "delegated", "external", "public", "backup":
		default:
			return fmt.Errorf("%s consumer %s:%s has invalid kind %q", owner, identity, field, consumer.Kind)
		}
		if strings.TrimSpace(consumer.Consumer) == "" {
			return fmt.Errorf("%s consumer %s:%s has no runtime consumer", owner, identity, field)
		}
		if strings.TrimSpace(consumer.SourceRef) == "" {
			return fmt.Errorf("%s consumer %s:%s has no source_ref", owner, identity, field)
		}
		for _, tier := range consumer.Tiers {
			tier = strings.TrimSpace(tier)
			if !tierNamePattern.MatchString(tier) {
				return fmt.Errorf("%s consumer %s:%s has invalid tier %q", owner, identity, field, tier)
			}
		}
		address := identity + ":" + field
		if pattern != "" {
			address = pattern
		}
		key := address + ":" + kind + ":" + strings.TrimSpace(consumer.Consumer) + ":" + strings.TrimSpace(consumer.SourceRef)
		if _, duplicate := seenConsumers[key]; duplicate {
			return fmt.Errorf("%s declares consumer %s twice", owner, key)
		}
		seenConsumers[key] = struct{}{}
	}
	return nil
}

func validateMetadataValue(owner, identity, field, name, value string, reference bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if len(value) > 256 || strings.IndexByte(value, 0) >= 0 || strings.ContainsAny(value, " \t\r\n") {
		return fmt.Errorf("%s credential %s:%s has invalid %s", owner, identity, field, name)
	}
	if reference && !metadataRefPattern.MatchString(value) {
		return fmt.Errorf("%s credential %s:%s has invalid %s reference %q", owner, identity, field, name, value)
	}
	return nil
}

func (a Applicability) Validate(owner, identity, field string) error {
	if len(a.All) > 64 || len(a.Any) > 64 {
		return fmt.Errorf("%s credential %s:%s applicability has too many clauses", owner, identity, field)
	}
	if len(a.All) == 0 && len(a.Any) == 0 && a.Not == nil {
		return fmt.Errorf("%s credential %s:%s applicability is empty", owner, identity, field)
	}
	for _, rule := range append(append([]ApplicabilityRule{}, a.All...), a.Any...) {
		if err := rule.validate(owner, identity, field); err != nil {
			return err
		}
	}
	if a.Not != nil {
		if err := a.Not.validate(owner, identity, field); err != nil {
			return err
		}
	}
	return nil
}

func (r ApplicabilityRule) validate(owner, identity, field string) error {
	if (r.Eq == nil) == (r.Neq == nil) {
		return fmt.Errorf("%s credential %s:%s applicability rule must contain exactly one comparison", owner, identity, field)
	}
	match := r.Eq
	if match == nil {
		match = r.Neq
	}
	return match.validate(owner, identity, field)
}

func (m ApplicabilityMatch) validate(owner, identity, field string) error {
	selectors := 0
	for _, value := range []string{m.Context, m.Setting, m.Operation, m.Role, m.Target, m.Environment, m.Capability, m.Provider} {
		if strings.TrimSpace(value) != "" {
			selectors++
		}
	}
	if selectors != 1 || strings.TrimSpace(m.Value) == "" {
		return fmt.Errorf("%s credential %s:%s applicability comparison needs one subject and a value", owner, identity, field)
	}
	if len(m.Value) > 256 || strings.IndexByte(m.Value, 0) >= 0 {
		return fmt.Errorf("%s credential %s:%s applicability value is invalid", owner, identity, field)
	}
	return nil
}

func describe(descriptor Descriptor) string {
	if env := strings.TrimSpace(descriptor.Env); env != "" {
		return env
	}
	return descriptor.ResolvedField()
}
