// Package operatorcapability defines the provider-neutral contract shared by
// setup, onboarding, and control-plane owners.
//
// A capability describes operator work; it does not describe an implementation
// package. Providers own policy and mutations, while consumers render the
// descriptor and carry typed inputs through the lifecycle. Secret inputs are
// intentionally represented only by an in-memory InputSet and cannot be
// marshaled back into an action result or evidence receipt.
package operatorcapability

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const ContractVersion = "operator-capability/v1"

const (
	maxDescriptorTextBytes = 4096
	maxInputBytes          = 64 * 1024
	maxOptions             = 128
	maxInputs              = 128
	maxCandidates          = 256
	maxMetadataEntries     = 64
	maxActionTextBytes     = 256
)

// ManifestReference is the optional declaration embedded in a scenario,
// resource, or tool manifest. It names a provider without embedding provider
// policy in the manifest. Empty declarations remain valid for old manifests.
type ManifestReference struct {
	Version      string `json:"version,omitempty"`
	CapabilityID string `json:"capability_id"`
	ProviderID   string `json:"provider_id"`
}

func (r ManifestReference) Validate() error {
	if strings.TrimSpace(r.CapabilityID) == "" || strings.TrimSpace(r.ProviderID) == "" {
		return errors.New("capability_id and provider_id are required")
	}
	if r.Version != "" && r.Version != ContractVersion {
		return fmt.Errorf("manifest capability %q uses unsupported contract version %q", r.CapabilityID, r.Version)
	}
	return nil
}

type InputKind string

const (
	KindSecret       InputKind = "secret"
	KindChoice       InputKind = "choice"
	KindConfirm      InputKind = "confirm"
	KindPath         InputKind = "path"
	KindEnum         InputKind = "enum"
	KindBoolean      InputKind = "boolean"
	KindDuration     InputKind = "duration"
	KindConfirmation InputKind = "confirmation"
)

type State string

const (
	StateDiscovered       State = "discovered"
	StateNeedsInput       State = "needs_operator_input"
	StateReadyToPreview   State = "ready_to_preview"
	StateApplying         State = "applying"
	StateVerifying        State = "verifying"
	StateReady            State = "ready"
	StateRetryableFailure State = "retryable_failure"
	StateDegraded         State = "degraded"
	StateUnsupported      State = "unsupported"
)

// Disposition describes how an operator-facing control participates in setup.
// It is deliberately separate from the observed State: a configurable control
// can be temporarily degraded, and a protected control can be ready without
// becoming operator-editable.
type Disposition string

const (
	DispositionConfigurable Disposition = "configurable"
	DispositionRequired     Disposition = "required"
	DispositionProtected    Disposition = "protected"
	DispositionDeferred     Disposition = "deferred"
	DispositionUnsupported  Disposition = "unsupported"
)

type Sensitivity string

const (
	SensitivityPublic   Sensitivity = "public"
	SensitivityOperator Sensitivity = "operator"
	SensitivitySecret   Sensitivity = "secret"
)

// Applicability is descriptive metadata. Providers remain responsible for
// evaluating it against their target facts; consumers only render it.
type Applicability struct {
	Platforms    []string `json:"platforms,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Targets      []string `json:"targets,omitempty"`
}

// PermissionProvenance is the explanation returned by the owner that enforces
// the permission. It must describe the effective grant, not a UI assumption.
type PermissionProvenance struct {
	Requester       string `json:"requester"`
	Scope           string `json:"scope"`
	GrantSource     string `json:"grant_source"`
	RevocationLimit string `json:"revocation_limit"`
}

// Lifecycle declares which provider-owned operations are available. A false
// value means the owner has no operation at that stage; it is not permission
// for a consumer to synthesize one.
type Lifecycle struct {
	Preview  bool   `json:"preview"`
	Apply    bool   `json:"apply"`
	Verify   bool   `json:"verify"`
	Revoke   bool   `json:"revoke"`
	Recover  bool   `json:"recover"`
	Recovery string `json:"recovery,omitempty"`
}

type Candidate struct {
	ID                    string            `json:"id"`
	Kind                  string            `json:"kind"`
	Label                 string            `json:"label"`
	Location              string            `json:"location,omitempty"`
	StableIdentity        string            `json:"stable_identity,omitempty"`
	DeviceIdentity        string            `json:"device_identity,omitempty"`
	Writable              bool              `json:"writable"`
	PhysicallyIndependent string            `json:"physical_independence,omitempty"`
	Status                string            `json:"status"`
	Risk                  string            `json:"risk,omitempty"`
	Remediation           string            `json:"remediation,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
}

type InputDescriptor struct {
	ID          string      `json:"id"`
	Kind        InputKind   `json:"kind"`
	Label       string      `json:"label"`
	Description string      `json:"description,omitempty"`
	Required    bool        `json:"required"`
	Declinable  bool        `json:"declinable,omitempty"`
	Options     []string    `json:"options,omitempty"`
	Default     string      `json:"default,omitempty"`
	Candidates  []Candidate `json:"candidates,omitempty"`
	Validation  string      `json:"validation,omitempty"`
	Constraints Constraints `json:"constraints,omitempty"`
	// Credential and owner metadata let a generic setup surface render an
	// input without hard-coding a provider or secret field in the UI.
	CredentialLogicalID string   `json:"credential_logical_id,omitempty"`
	CredentialField     string   `json:"credential_field,omitempty"`
	Provider            string   `json:"provider,omitempty"`
	RequirementGroup    string   `json:"requirement_group,omitempty"`
	ConsumerRefs        []string `json:"consumer_refs,omitempty"`
	CompanionSettings   []string `json:"companion_settings,omitempty"`
	AcquisitionRef      string   `json:"acquisition_ref,omitempty"`
	VerificationRef     string   `json:"verification_ref,omitempty"`
	RecoveryRef         string   `json:"recovery_ref,omitempty"`
	HelpRef             string   `json:"help_ref,omitempty"`
	EvidencePolicy      string   `json:"evidence_policy,omitempty"`
}

func (i InputDescriptor) Secret() bool { return i.Kind == KindSecret }

type Constraints struct {
	MinLength   int    `json:"min_length,omitempty"`
	MaxLength   int    `json:"max_length,omitempty"`
	MinDuration string `json:"min_duration,omitempty"`
	MaxDuration string `json:"max_duration,omitempty"`
}

type Policy struct {
	RequiresConfirmation bool     `json:"requires_confirmation"`
	Idempotent           bool     `json:"idempotent"`
	Retryable            bool     `json:"retryable"`
	ProtectedRoots       []string `json:"protected_roots,omitempty"`
	Remediation          string   `json:"remediation,omitempty"`
}

type EvidenceContract struct {
	Kinds          []string `json:"kinds,omitempty"`
	RequiredFields []string `json:"required_fields,omitempty"`
	SecretFree     bool     `json:"secret_free"`
	Freshness      string   `json:"freshness,omitempty"`
}

type Descriptor struct {
	Version           string               `json:"version"`
	ID                string               `json:"id"`
	Owner             string               `json:"owner"`
	Scope             string               `json:"scope"`
	Purpose           string               `json:"purpose"`
	Sensitivity       Sensitivity          `json:"sensitivity"`
	Applicability     Applicability        `json:"applicability,omitempty"`
	Disposition       Disposition          `json:"disposition"`
	DispositionReason string               `json:"disposition_reason,omitempty"`
	Provenance        PermissionProvenance `json:"provenance"`
	Lifecycle         Lifecycle            `json:"lifecycle"`
	ReferenceURL      string               `json:"reference_url,omitempty"`
	Title             string               `json:"title"`
	Description       string               `json:"description,omitempty"`
	Risk              string               `json:"risk,omitempty"`
	Inputs            []InputDescriptor    `json:"inputs,omitempty"`
	Prerequisites     []string             `json:"prerequisites,omitempty"`
	Policy            Policy               `json:"policy"`
	Evidence          EvidenceContract     `json:"evidence"`
	Remediation       string               `json:"remediation,omitempty"`
}

func (d Descriptor) Validate() error {
	if d.Version != ContractVersion {
		return fmt.Errorf("capability %q uses unsupported contract version %q", d.ID, d.Version)
	}
	if strings.TrimSpace(d.ID) == "" || strings.TrimSpace(d.Owner) == "" || strings.TrimSpace(d.Title) == "" || strings.TrimSpace(d.Scope) == "" || strings.TrimSpace(d.Purpose) == "" {
		return errors.New("capability version, id, owner, title, scope, and purpose are required")
	}
	if !validSensitivity(d.Sensitivity) {
		return fmt.Errorf("capability %q has unknown sensitivity %q", d.ID, d.Sensitivity)
	}
	if !validDisposition(d.Disposition) {
		return fmt.Errorf("capability %q has unknown disposition %q", d.ID, d.Disposition)
	}
	if (d.Disposition == DispositionDeferred || d.Disposition == DispositionUnsupported) && strings.TrimSpace(d.DispositionReason) == "" {
		return fmt.Errorf("capability %q disposition %q requires a reason", d.ID, d.Disposition)
	}
	if err := validateDescriptorText(d.ID, "disposition_reason", d.DispositionReason); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "owner", d.Owner); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "scope", d.Scope); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "purpose", d.Purpose); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "title", d.Title); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "description", d.Description); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "risk", d.Risk); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "remediation", d.Remediation); err != nil {
		return err
	}
	if err := validateDescriptorText(d.ID, "lifecycle.recovery", d.Lifecycle.Recovery); err != nil {
		return err
	}
	if err := validateProvenance(d); err != nil {
		return err
	}
	if err := validateApplicability(d); err != nil {
		return err
	}
	if err := validateReferenceURL(d); err != nil {
		return err
	}
	if len(d.Inputs) > maxInputs {
		return fmt.Errorf("capability %q declares too many inputs", d.ID)
	}
	if len(d.Prerequisites) > maxOptions {
		return fmt.Errorf("capability %q declares too many prerequisites", d.ID)
	}
	if len(d.Policy.ProtectedRoots) > maxOptions {
		return fmt.Errorf("capability %q declares too many protected roots", d.ID)
	}
	if len(d.Evidence.Kinds) > maxOptions || len(d.Evidence.RequiredFields) > maxOptions {
		return fmt.Errorf("capability %q declares too many evidence fields", d.ID)
	}
	for field, values := range map[string][]string{
		"prerequisite":   d.Prerequisites,
		"protected root": d.Policy.ProtectedRoots,
		"evidence kind":  d.Evidence.Kinds,
		"evidence field": d.Evidence.RequiredFields,
	} {
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("capability %q contains an empty %s", d.ID, field)
			}
			if err := validateDescriptorText(d.ID, field, value); err != nil {
				return err
			}
		}
	}
	if !d.Policy.Idempotent {
		return fmt.Errorf("capability %q must declare idempotency", d.ID)
	}
	if !d.Evidence.SecretFree {
		return fmt.Errorf("capability %q must declare secret-free evidence", d.ID)
	}
	seen := map[string]struct{}{}
	for _, input := range d.Inputs {
		if strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.Label) == "" {
			return fmt.Errorf("capability %q contains an input without id and label", d.ID)
		}
		if err := validateDescriptorText(d.ID, "input id", input.ID); err != nil {
			return err
		}
		if _, ok := seen[input.ID]; ok {
			return fmt.Errorf("capability %q declares input %q more than once", d.ID, input.ID)
		}
		seen[input.ID] = struct{}{}
		if err := validateDescriptorText(d.ID, "input label", input.Label); err != nil {
			return err
		}
		if err := validateDescriptorText(d.ID, "input description", input.Description); err != nil {
			return err
		}
		if err := validateDescriptorText(d.ID, "input validation", input.Validation); err != nil {
			return err
		}
		for field, value := range map[string]string{
			"input.credential_logical_id": input.CredentialLogicalID, "input.credential_field": input.CredentialField,
			"input.provider": input.Provider, "input.requirement_group": input.RequirementGroup,
			"input.acquisition_ref": input.AcquisitionRef, "input.verification_ref": input.VerificationRef,
			"input.recovery_ref": input.RecoveryRef, "input.help_ref": input.HelpRef, "input.evidence_policy": input.EvidencePolicy,
		} {
			if err := validateDescriptorText(d.ID, field, value); err != nil {
				return err
			}
			if strings.TrimSpace(value) != "" && strings.ContainsAny(value, "\r\n") {
				return fmt.Errorf("capability %q %s contains a line break", d.ID, field)
			}
		}
		if strings.ContainsAny(input.CredentialField, "/\\") {
			return fmt.Errorf("capability %q input %q credential_field cannot contain a path separator", d.ID, input.ID)
		}
		if len(input.ConsumerRefs) > maxOptions || len(input.CompanionSettings) > maxOptions {
			return fmt.Errorf("capability %q input %q declares too many metadata references", d.ID, input.ID)
		}
		for name, values := range map[string][]string{"consumer reference": input.ConsumerRefs, "companion setting": input.CompanionSettings} {
			seenReferences := map[string]struct{}{}
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value == "" || strings.ContainsAny(value, "\r\n") {
					return fmt.Errorf("capability %q input %q contains an invalid %s", d.ID, input.ID, name)
				}
				if _, duplicate := seenReferences[value]; duplicate {
					return fmt.Errorf("capability %q input %q repeats %s %q", d.ID, input.ID, name, value)
				}
				seenReferences[value] = struct{}{}
			}
		}
		if input.Declinable && input.Required {
			return fmt.Errorf("capability %q required input %q cannot be declinable", d.ID, input.ID)
		}
		if len(input.Options) > maxOptions {
			return fmt.Errorf("capability %q input %q has too many options", d.ID, input.ID)
		}
		if err := validateDescriptorText(d.ID, "input default", input.Default); err != nil {
			return err
		}
		for _, option := range input.Options {
			if strings.TrimSpace(option) == "" {
				return fmt.Errorf("capability %q input %q contains an empty option", d.ID, input.ID)
			}
			if err := validateDescriptorText(d.ID, "input option", option); err != nil {
				return err
			}
		}
		if len(input.Candidates) > maxCandidates {
			return fmt.Errorf("capability %q input %q has too many candidates", d.ID, input.ID)
		}
		for _, candidate := range input.Candidates {
			if strings.TrimSpace(candidate.ID) == "" && strings.TrimSpace(candidate.Location) == "" {
				return fmt.Errorf("capability %q input %q contains a candidate without id or location", d.ID, input.ID)
			}
			for field, value := range map[string]string{"candidate.id": candidate.ID, "candidate.label": candidate.Label, "candidate.location": candidate.Location, "candidate.identity": candidate.StableIdentity, "candidate.device": candidate.DeviceIdentity, "candidate.physical_independence": candidate.PhysicallyIndependent, "candidate.status": candidate.Status, "candidate.risk": candidate.Risk, "candidate.remediation": candidate.Remediation} {
				if err := validateDescriptorText(d.ID, field, value); err != nil {
					return err
				}
			}
			if len(candidate.Metadata) > maxMetadataEntries {
				return fmt.Errorf("capability %q input %q candidate metadata is too large", d.ID, input.ID)
			}
			for key, value := range candidate.Metadata {
				if forbiddenExecutionKey(key) {
					return fmt.Errorf("capability %q input %q contains forbidden execution metadata %q", d.ID, input.ID, key)
				}
				if err := validateDescriptorText(d.ID, "candidate metadata key", key); err != nil {
					return err
				}
				if err := validateDescriptorText(d.ID, "candidate metadata value", value); err != nil {
					return err
				}
			}
		}
		if err := input.Constraints.Validate(d.ID, input.ID, input.Kind); err != nil {
			return err
		}
		switch input.Kind {
		case KindSecret, KindPath, KindEnum, KindBoolean, KindDuration, KindConfirmation:
		default:
			return fmt.Errorf("capability %q input %q has unsupported kind %q", d.ID, input.ID, input.Kind)
		}
		if input.Kind == KindEnum && len(input.Options) == 0 {
			return fmt.Errorf("capability %q enum input %q has no options", d.ID, input.ID)
		}
		if input.Kind == KindConfirmation && !input.Required {
			return fmt.Errorf("capability %q confirmation input %q must be required", d.ID, input.ID)
		}
	}
	return nil
}

func forbiddenExecutionKey(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "command", "commands", "argv", "exec", "executable", "shell", "script":
		return true
	default:
		return false
	}
}

func validSensitivity(value Sensitivity) bool {
	return value == SensitivityPublic || value == SensitivityOperator || value == SensitivitySecret
}

func validDisposition(value Disposition) bool {
	return value == DispositionConfigurable || value == DispositionRequired || value == DispositionProtected || value == DispositionDeferred || value == DispositionUnsupported
}

func validateDescriptorText(capabilityID, field, value string) error {
	if len(value) > maxDescriptorTextBytes {
		return fmt.Errorf("capability %q %s exceeds %d bytes", capabilityID, field, maxDescriptorTextBytes)
	}
	if strings.IndexByte(value, 0) >= 0 {
		return fmt.Errorf("capability %q %s contains a NUL byte", capabilityID, field)
	}
	return nil
}

func validateProvenance(d Descriptor) error {
	for field, value := range map[string]string{
		"provenance.requester":        d.Provenance.Requester,
		"provenance.scope":            d.Provenance.Scope,
		"provenance.grant_source":     d.Provenance.GrantSource,
		"provenance.revocation_limit": d.Provenance.RevocationLimit,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("capability %q %s is required", d.ID, field)
		}
		if err := validateDescriptorText(d.ID, field, value); err != nil {
			return err
		}
	}
	return nil
}

func validateApplicability(d Descriptor) error {
	for field, values := range map[string][]string{
		"applicability.platforms":    d.Applicability.Platforms,
		"applicability.environments": d.Applicability.Environments,
		"applicability.targets":      d.Applicability.Targets,
	} {
		if len(values) > maxOptions {
			return fmt.Errorf("capability %q has too many %s", d.ID, field)
		}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("capability %q contains an empty %s entry", d.ID, field)
			}
			if err := validateDescriptorText(d.ID, field, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateReferenceURL(d Descriptor) error {
	if strings.TrimSpace(d.ReferenceURL) == "" {
		return nil
	}
	parsed, err := url.Parse(d.ReferenceURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("capability %q reference_url must be an https URL without credentials or fragments", d.ID)
	}
	return nil
}

func (c Constraints) Validate(capabilityID, inputID string, kind InputKind) error {
	if c.MinLength < 0 || c.MaxLength < 0 || (c.MaxLength > 0 && c.MinLength > c.MaxLength) {
		return fmt.Errorf("capability %q input %q has invalid length constraints", capabilityID, inputID)
	}
	if c.MaxLength > maxDescriptorTextBytes || c.MinLength > maxDescriptorTextBytes {
		return fmt.Errorf("capability %q input %q length constraints are too large", capabilityID, inputID)
	}
	if kind != KindDuration && (c.MinDuration != "" || c.MaxDuration != "") {
		return fmt.Errorf("capability %q input %q duration constraints require a duration input", capabilityID, inputID)
	}
	for name, value := range map[string]string{"min_duration": c.MinDuration, "max_duration": c.MaxDuration} {
		if value == "" {
			continue
		}
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("capability %q input %q has invalid %s", capabilityID, inputID, name)
		}
	}
	return nil
}

type ActionRequest struct {
	CapabilityID   string                     `json:"capability_id"`
	IdempotencyKey string                     `json:"idempotency_key"`
	Confirm        bool                       `json:"confirm"`
	Inputs         map[string]json.RawMessage `json:"inputs,omitempty"`
}

func (r ActionRequest) Validate() error {
	if strings.TrimSpace(r.CapabilityID) == "" {
		return errors.New("capability_id is required")
	}
	if len(r.CapabilityID) > maxActionTextBytes {
		return fmt.Errorf("capability_id exceeds %d bytes", maxActionTextBytes)
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	if len(r.IdempotencyKey) > maxActionTextBytes {
		return fmt.Errorf("idempotency_key exceeds %d bytes", maxActionTextBytes)
	}
	return nil
}

// InputSet is the only form in which typed operator answers reach a provider.
// It is deliberately not JSON serializable. Call Clear as soon as the owner
// has completed its mutation so secret strings do not remain in a long-lived
// request object.
type InputSet struct{ values map[string]validatedInput }

type validatedInput struct {
	kind     InputKind
	text     string
	boolean  bool
	duration time.Duration
}

func (s InputSet) Text(id string) (string, bool) {
	v, ok := s.values[id]
	if !ok || (v.kind != KindSecret && v.kind != KindPath && v.kind != KindEnum) {
		return "", false
	}
	return v.text, true
}

func (s InputSet) Boolean(id string) (bool, bool) {
	v, ok := s.values[id]
	return v.boolean, ok && (v.kind == KindBoolean || v.kind == KindConfirmation)
}

func (s InputSet) Duration(id string) (time.Duration, bool) {
	v, ok := s.values[id]
	return v.duration, ok && v.kind == KindDuration
}

func (s *InputSet) Clear() {
	if s == nil {
		return
	}
	for id, value := range s.values {
		value.text = ""
		s.values[id] = value
	}
	s.values = nil
}

func (d Descriptor) ValidateInputs(raw map[string]json.RawMessage) (InputSet, error) {
	if err := d.Validate(); err != nil {
		return InputSet{}, err
	}
	if len(raw) > maxInputs {
		return InputSet{}, fmt.Errorf("capability %q received too many inputs", d.ID)
	}
	byID := make(map[string]InputDescriptor, len(d.Inputs))
	for _, input := range d.Inputs {
		byID[input.ID] = input
	}
	for id := range raw {
		if _, ok := byID[id]; !ok {
			return InputSet{}, fmt.Errorf("capability %q received unknown input %q", d.ID, id)
		}
		if len(raw[id]) > maxInputBytes {
			return InputSet{}, fmt.Errorf("capability %q input %q exceeds %d bytes", d.ID, id, maxInputBytes)
		}
	}
	values := make(map[string]validatedInput, len(d.Inputs))
	for _, input := range d.Inputs {
		data, present := raw[input.ID]
		if !present || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			if input.Required && strings.TrimSpace(input.Default) == "" {
				return InputSet{}, fmt.Errorf("capability %q requires input %q", d.ID, input.ID)
			}
			if input.Default != "" {
				data = json.RawMessage(strconvQuote(input.Default))
			} else {
				continue
			}
		}
		value, err := parseInput(input, data)
		if err != nil {
			return InputSet{}, fmt.Errorf("capability %q input %q: %w", d.ID, input.ID, err)
		}
		values[input.ID] = value
	}
	return InputSet{values: values}, nil
}

func parseInput(input InputDescriptor, data []byte) (validatedInput, error) {
	value := validatedInput{kind: input.Kind}
	switch input.Kind {
	case KindSecret, KindPath, KindEnum:
		if err := json.Unmarshal(data, &value.text); err != nil || strings.TrimSpace(value.text) == "" {
			return validatedInput{}, errors.New("must be a non-empty string")
		}
		if len(value.text) > maxDescriptorTextBytes {
			return validatedInput{}, fmt.Errorf("must be at most %d bytes", maxDescriptorTextBytes)
		}
		if input.Constraints.MinLength > 0 && len([]rune(value.text)) < input.Constraints.MinLength {
			return validatedInput{}, fmt.Errorf("must be at least %d characters", input.Constraints.MinLength)
		}
		if input.Constraints.MaxLength > 0 && len([]rune(value.text)) > input.Constraints.MaxLength {
			return validatedInput{}, fmt.Errorf("must be at most %d characters", input.Constraints.MaxLength)
		}
		if input.Kind == KindPath && strings.IndexByte(value.text, 0) >= 0 {
			return validatedInput{}, errors.New("path contains a NUL byte")
		}
		if input.Kind == KindEnum && !slices.Contains(input.Options, value.text) {
			return validatedInput{}, fmt.Errorf("%q is not an allowed option", value.text)
		}
	case KindBoolean, KindConfirmation:
		if err := json.Unmarshal(data, &value.boolean); err != nil {
			return validatedInput{}, errors.New("must be a boolean")
		}
		if input.Kind == KindConfirmation && !value.boolean {
			return validatedInput{}, errors.New("confirmation must be true")
		}
	case KindDuration:
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return validatedInput{}, errors.New("must be a duration string")
		}
		parsed, err := time.ParseDuration(text)
		if err != nil || parsed <= 0 {
			return validatedInput{}, errors.New("must be a positive duration")
		}
		if input.Constraints.MinDuration != "" {
			minimum, _ := time.ParseDuration(input.Constraints.MinDuration)
			if parsed < minimum {
				return validatedInput{}, fmt.Errorf("must be at least %s", input.Constraints.MinDuration)
			}
		}
		if input.Constraints.MaxDuration != "" {
			maximum, _ := time.ParseDuration(input.Constraints.MaxDuration)
			if parsed > maximum {
				return validatedInput{}, fmt.Errorf("must be at most %s", input.Constraints.MaxDuration)
			}
		}
		value.text, value.duration = text, parsed
	default:
		return validatedInput{}, fmt.Errorf("unsupported input kind %q", input.Kind)
	}
	return value, nil
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

type EvidenceReference struct {
	SchemaVersion    string                 `json:"schema_version,omitempty"`
	CapabilityID     string                 `json:"capability_id,omitempty"`
	Kind             string                 `json:"kind"`
	ArtifactIdentity string                 `json:"artifact_identity"`
	CredentialRef    *CredentialEvidenceRef `json:"credential_ref,omitempty"`
	TargetID         string                 `json:"target_id,omitempty"`
	Environment      string                 `json:"environment,omitempty"`
	AccountIdentity  string                 `json:"account_identity,omitempty"`
	Operation        string                 `json:"operation,omitempty"`
	Status           string                 `json:"status,omitempty"`
	SourceGeneration string                 `json:"source_generation,omitempty"`
	Checksum         string                 `json:"checksum,omitempty"`
	Coverage         []string               `json:"coverage,omitempty"`
	ObservedAt       time.Time              `json:"observed_at"`
	ExpiresAt        time.Time              `json:"expires_at,omitempty"`
	ArtifactRefs     []string               `json:"artifact_refs,omitempty"`
	Limitations      []string               `json:"limitations,omitempty"`
	NextAction       string                 `json:"next_action,omitempty"`
	EffectClass      string                 `json:"effect_class,omitempty"`
	EffectsUsed      int                    `json:"effects_used,omitempty"`
	CleanupCompleted bool                   `json:"cleanup_completed,omitempty"`
	Verified         bool                   `json:"verified"`
	Remediation      string                 `json:"remediation,omitempty"`
}

// CredentialEvidenceRef identifies an authority version without carrying a
// value or a reversible hash of one. Providers issue the opaque version.
type CredentialEvidenceRef struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Version   string `json:"version"`
}

func (e EvidenceReference) Validate() error {
	if strings.TrimSpace(e.Kind) == "" || strings.TrimSpace(e.ArtifactIdentity) == "" {
		return errors.New("evidence kind and artifact_identity are required")
	}
	if strings.Contains(strings.ToLower(e.ArtifactIdentity), "passphrase") || strings.Contains(strings.ToLower(e.ArtifactIdentity), "secret") {
		return errors.New("evidence identity cannot contain secret material")
	}
	if e.SchemaVersion != "" && e.SchemaVersion != EvidenceSchemaVersion {
		return fmt.Errorf("unsupported evidence schema version %q", e.SchemaVersion)
	}
	if e.CredentialRef != nil {
		if strings.TrimSpace(e.CredentialRef.LogicalID) == "" || strings.TrimSpace(e.CredentialRef.Field) == "" || strings.TrimSpace(e.CredentialRef.Version) == "" {
			return errors.New("credential evidence reference requires logical_id, field, and opaque version")
		}
	}
	if e.ExpiresAt.IsZero() == false && !e.ObservedAt.IsZero() && e.ExpiresAt.Before(e.ObservedAt) {
		return errors.New("evidence expiry cannot precede observation")
	}
	if e.EffectsUsed < 0 {
		return errors.New("evidence effects_used cannot be negative")
	}
	if e.EffectClass != "" && e.EffectClass != string(EffectReadOnly) && e.EffectClass != string(EffectBoundedWrite) {
		return fmt.Errorf("unsupported evidence effect class %q", e.EffectClass)
	}
	for _, value := range append(append([]string{}, e.ArtifactRefs...), e.Limitations...) {
		if strings.Contains(strings.ToLower(value), "secret") || strings.Contains(strings.ToLower(value), "passphrase") {
			return errors.New("evidence references cannot contain secret material")
		}
	}
	return nil
}

type Mutation struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Reversible bool   `json:"reversible"`
}

type Preview struct {
	CapabilityID string      `json:"capability_id"`
	PlanID       string      `json:"plan_id"`
	State        State       `json:"state"`
	Mutations    []Mutation  `json:"mutations,omitempty"`
	Candidates   []Candidate `json:"candidates,omitempty"`
	Remediation  string      `json:"remediation,omitempty"`
	ExpiresAt    time.Time   `json:"expires_at,omitempty"`
}

type Result struct {
	CapabilityID string              `json:"capability_id"`
	State        State               `json:"state"`
	Outcome      string              `json:"outcome"`
	Retryable    bool                `json:"retryable"`
	ErrorCode    string              `json:"error_code,omitempty"`
	Remediation  string              `json:"remediation,omitempty"`
	Evidence     []EvidenceReference `json:"evidence,omitempty"`
	Mutations    []Mutation          `json:"mutations,omitempty"`
	CompletedAt  time.Time           `json:"completed_at,omitempty"`
}

type Status struct {
	Descriptor    Descriptor          `json:"descriptor"`
	State         State               `json:"state"`
	Candidates    []Candidate         `json:"candidates,omitempty"`
	MissingInputs []string            `json:"missing_inputs,omitempty"`
	Evidence      []EvidenceReference `json:"evidence,omitempty"`
	Remediation   string              `json:"remediation,omitempty"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type Provider interface {
	Descriptor() Descriptor
	Discover(context.Context) (Status, error)
	Preview(context.Context, InputSet) (Preview, error)
	Apply(context.Context, InputSet) (Result, error)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry(providers ...Provider) (*Registry, error) {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		if err := r.Register(provider); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(provider Provider) error {
	if provider == nil {
		return errors.New("capability provider is nil")
	}
	descriptor := provider.Descriptor()
	if err := descriptor.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[descriptor.ID]; exists {
		return fmt.Errorf("capability provider %q is already registered", descriptor.ID)
	}
	r.providers[descriptor.ID] = provider
	return nil
}

func (r *Registry) Provider(id string) (Provider, bool) {
	r.mu.RLock()
	provider, ok := r.providers[id]
	r.mu.RUnlock()
	return provider, ok
}

func (r *Registry) Discover(ctx context.Context) ([]Status, error) {
	r.mu.RLock()
	providers := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}
	r.mu.RUnlock()
	sort.Slice(providers, func(i, j int) bool { return providers[i].Descriptor().ID < providers[j].Descriptor().ID })
	statuses := make([]Status, 0, len(providers))
	for _, provider := range providers {
		status, err := provider.Discover(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover capability %q: %w", provider.Descriptor().ID, err)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (r *Registry) Preview(ctx context.Context, request ActionRequest) (Preview, error) {
	if err := request.Validate(); err != nil {
		return Preview{}, err
	}
	provider, ok := r.Provider(request.CapabilityID)
	if !ok {
		return Preview{}, fmt.Errorf("capability %q is not registered", request.CapabilityID)
	}
	inputs, err := provider.Descriptor().ValidateInputs(request.Inputs)
	if err != nil {
		return Preview{}, err
	}
	defer inputs.Clear()
	return provider.Preview(ctx, inputs)
}

func (r *Registry) Apply(ctx context.Context, request ActionRequest) (Result, error) {
	if err := request.Validate(); err != nil {
		return Result{}, err
	}
	provider, ok := r.Provider(request.CapabilityID)
	if !ok {
		return Result{}, fmt.Errorf("capability %q is not registered", request.CapabilityID)
	}
	descriptor := provider.Descriptor()
	if descriptor.Policy.RequiresConfirmation && !request.Confirm {
		return Result{CapabilityID: descriptor.ID, State: StateReadyToPreview, Outcome: "confirmation_required", ErrorCode: "confirmation_required", Retryable: true, Remediation: "review the preview and confirm the exact planned mutations"}, nil
	}
	inputs, err := descriptor.ValidateInputs(request.Inputs)
	if err != nil {
		return Result{CapabilityID: descriptor.ID, State: StateNeedsInput, Outcome: "invalid_input", ErrorCode: "invalid_input", Retryable: true, Remediation: err.Error()}, err
	}
	defer inputs.Clear()
	result, err := provider.Apply(ctx, inputs)
	if err != nil {
		return result, err
	}
	for _, evidence := range result.Evidence {
		if evidenceErr := evidence.Validate(); evidenceErr != nil {
			return Result{CapabilityID: descriptor.ID, State: StateDegraded, Outcome: "invalid_evidence", ErrorCode: "invalid_evidence", Retryable: true, Remediation: evidenceErr.Error()}, evidenceErr
		}
	}
	return result, nil
}

func StableIdempotencyKey(capabilityID string, inputs map[string]json.RawMessage) string {
	h := sha256.New()
	_, _ = h.Write([]byte(capabilityID))
	keys := make([]string, 0, len(inputs))
	for key := range inputs {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		_, _ = h.Write([]byte{'\n'})
		_, _ = h.Write([]byte(key))
		_, _ = h.Write([]byte{'='})
		_, _ = h.Write(bytes.TrimSpace(inputs[key]))
	}
	return hex.EncodeToString(h.Sum(nil))
}
