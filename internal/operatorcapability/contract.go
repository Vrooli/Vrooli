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
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

const ContractVersion = "operator-capability/v1"

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
	Options     []string    `json:"options,omitempty"`
	Default     string      `json:"default,omitempty"`
	Candidates  []Candidate `json:"candidates,omitempty"`
	Validation  string      `json:"validation,omitempty"`
}

func (i InputDescriptor) Secret() bool { return i.Kind == KindSecret }

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
	Version       string            `json:"version"`
	ID            string            `json:"id"`
	Owner         string            `json:"owner"`
	Title         string            `json:"title"`
	Description   string            `json:"description,omitempty"`
	Risk          string            `json:"risk,omitempty"`
	Inputs        []InputDescriptor `json:"inputs,omitempty"`
	Prerequisites []string          `json:"prerequisites,omitempty"`
	Policy        Policy            `json:"policy"`
	Evidence      EvidenceContract  `json:"evidence"`
	Remediation   string            `json:"remediation,omitempty"`
}

func (d Descriptor) Validate() error {
	if d.Version != ContractVersion {
		return fmt.Errorf("capability %q uses unsupported contract version %q", d.ID, d.Version)
	}
	if strings.TrimSpace(d.ID) == "" || strings.TrimSpace(d.Owner) == "" || strings.TrimSpace(d.Title) == "" {
		return errors.New("capability version, id, owner, and title are required")
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
		if _, ok := seen[input.ID]; ok {
			return fmt.Errorf("capability %q declares input %q more than once", d.ID, input.ID)
		}
		seen[input.ID] = struct{}{}
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
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
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
	byID := make(map[string]InputDescriptor, len(d.Inputs))
	for _, input := range d.Inputs {
		byID[input.ID] = input
	}
	for id := range raw {
		if _, ok := byID[id]; !ok {
			return InputSet{}, fmt.Errorf("capability %q received unknown input %q", d.ID, id)
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
	Kind             string    `json:"kind"`
	ArtifactIdentity string    `json:"artifact_identity"`
	SourceGeneration string    `json:"source_generation,omitempty"`
	Checksum         string    `json:"checksum,omitempty"`
	Coverage         []string  `json:"coverage,omitempty"`
	ObservedAt       time.Time `json:"observed_at"`
	Verified         bool      `json:"verified"`
	Remediation      string    `json:"remediation,omitempty"`
}

func (e EvidenceReference) Validate() error {
	if strings.TrimSpace(e.Kind) == "" || strings.TrimSpace(e.ArtifactIdentity) == "" {
		return errors.New("evidence kind and artifact_identity are required")
	}
	if strings.Contains(strings.ToLower(e.ArtifactIdentity), "passphrase") || strings.Contains(strings.ToLower(e.ArtifactIdentity), "secret") {
		return errors.New("evidence identity cannot contain secret material")
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
