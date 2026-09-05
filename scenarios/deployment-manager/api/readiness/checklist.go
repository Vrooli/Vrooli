// Package readiness owns the policy and decision state used to prepare a
// release. Evidence producers retain ownership of the measurements themselves.
package readiness

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const ChecklistVersion = 2

//go:embed readiness-policy.json
var builtInPolicyJSON []byte

//go:embed readiness-policy.schema.json
var builtInPolicySchemaJSON []byte

type CleanRequirement string

const (
	Required    CleanRequirement = "required"
	Advisory    CleanRequirement = "advisory"
	Uncheckable CleanRequirement = "human_review"
)

type GlobalImpact string

const (
	FoundationBlocker GlobalImpact = "foundation_blocker"
	SafetyBlocker     GlobalImpact = "safety_blocker"
	CapabilityGap     GlobalImpact = "capability_gap"
	HardeningGap      GlobalImpact = "hardening_gap"
	AdvisoryImpact    GlobalImpact = "advisory"
	UnknownImpact     GlobalImpact = "unknown"
)

type FreshnessPolicy struct {
	Basis         string `json:"basis"`
	MaxAgeSeconds int64  `json:"max_age_seconds,omitempty"`
}

type ProducerRoute struct {
	Binding string `json:"binding"`
}
type HumanReviewRoute struct {
	Kind string `json:"kind"`
}

type GherkinAcceptance struct {
	Given string `json:"given"`
	When  string `json:"when"`
	Then  string `json:"then"`
}

func (a GherkinAcceptance) Sentence() string {
	return fmt.Sprintf("Given %s, when %s, then %s", a.Given, a.When, a.Then)
}

type Remediation struct {
	Skill string `json:"skill"`
	Topic string `json:"topic"`
}

type WaiverPolicy struct {
	Eligible      bool  `json:"eligible"`
	MaxAgeSeconds int64 `json:"max_age_seconds,omitempty"`
}

type Item struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Category           string            `json:"category"`
	Owner              string            `json:"owner"`
	Applicability      string            `json:"applicability"`
	CleanRequirement   CleanRequirement  `json:"requirement"`
	GlobalImpact       GlobalImpact      `json:"global_impact"`
	Freshness          FreshnessPolicy   `json:"freshness"`
	Producer           *ProducerRoute    `json:"producer,omitempty"`
	HumanReview        *HumanReviewRoute `json:"human_review,omitempty"`
	Acceptance         GherkinAcceptance `json:"acceptance"`
	Remediation        Remediation       `json:"remediation"`
	Waiver             WaiverPolicy      `json:"waiver"`
	AcceptanceCriteria string            `json:"-"`
}

type Checklist struct {
	Version int    `json:"version"`
	Items   []Item `json:"items"`
}

var knownOwners = map[string]struct{}{
	"business-health": {}, "content-desk": {}, "deployment-manager": {},
	"git-control-tower": {}, "measures-health": {}, "offer-desk": {},
	"scenario-dependency-analyzer": {}, "scenario-to-desktop": {},
	"secrets-manager": {}, "security-health": {}, "storage-manager": {},
	"swarm-manager": {}, "test-genie": {},
}

func (i Item) Validate() error {
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.Title) == "" {
		return fmt.Errorf("checklist item requires id and title")
	}
	if strings.TrimSpace(i.Category) == "" || strings.TrimSpace(i.Owner) == "" || strings.TrimSpace(i.Applicability) == "" {
		return fmt.Errorf("checklist item %q requires category, owner, and applicability", i.ID)
	}
	if _, ok := knownOwners[i.Owner]; !ok {
		return fmt.Errorf("checklist item %q has unknown owner %q", i.ID, i.Owner)
	}
	switch i.CleanRequirement {
	case Required, Advisory, Uncheckable:
	default:
		return fmt.Errorf("checklist item %q has invalid requirement %q", i.ID, i.CleanRequirement)
	}
	switch i.GlobalImpact {
	case FoundationBlocker, SafetyBlocker, CapabilityGap, HardeningGap, AdvisoryImpact, UnknownImpact:
	default:
		return fmt.Errorf("checklist item %q has invalid global_impact %q", i.ID, i.GlobalImpact)
	}
	if i.Freshness.Basis != "candidate_identity" && i.Freshness.Basis != "max_age" {
		return fmt.Errorf("checklist item %q has invalid freshness basis %q", i.ID, i.Freshness.Basis)
	}
	if i.Freshness.Basis == "max_age" && i.Freshness.MaxAgeSeconds <= 0 {
		return fmt.Errorf("checklist item %q max_age freshness requires max_age_seconds", i.ID)
	}
	if (i.Producer == nil) == (i.HumanReview == nil) {
		return fmt.Errorf("checklist item %q requires exactly one producer or human_review route", i.ID)
	}
	if i.Producer != nil && (!strings.Contains(i.Producer.Binding, ".") || strings.ContainsAny(i.Producer.Binding, " \t\n")) {
		return fmt.Errorf("checklist item %q has invalid producer binding %q", i.ID, i.Producer.Binding)
	}
	if i.HumanReview != nil && strings.TrimSpace(i.HumanReview.Kind) == "" {
		return fmt.Errorf("checklist item %q has empty human_review kind", i.ID)
	}
	structuredAcceptance := strings.TrimSpace(i.Acceptance.Given) != "" && strings.TrimSpace(i.Acceptance.When) != "" && strings.TrimSpace(i.Acceptance.Then) != ""
	if !structuredAcceptance && strings.TrimSpace(i.AcceptanceCriteria) == "" {
		return fmt.Errorf("checklist item %q requires Given/When/Then acceptance", i.ID)
	}
	if strings.TrimSpace(i.Remediation.Skill) == "" || strings.TrimSpace(i.Remediation.Topic) == "" {
		return fmt.Errorf("checklist item %q requires remediation skill and topic", i.ID)
	}
	return nil
}

func (c Checklist) Validate() error {
	if c.Version != ChecklistVersion {
		return fmt.Errorf("unsupported checklist version %d", c.Version)
	}
	if len(c.Items) == 0 {
		return fmt.Errorf("readiness checklist is empty")
	}
	seen := make(map[string]struct{}, len(c.Items))
	for _, item := range c.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("duplicate checklist item %q", item.ID)
		}
		seen[item.ID] = struct{}{}
	}
	return nil
}

func decodePolicy(data []byte) (Checklist, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("readiness-policy.schema.json", bytes.NewReader(builtInPolicySchemaJSON)); err != nil {
		return Checklist{}, fmt.Errorf("load readiness policy schema: %w", err)
	}
	schema, err := compiler.Compile("readiness-policy.schema.json")
	if err != nil {
		return Checklist{}, fmt.Errorf("compile readiness policy schema: %w", err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return Checklist{}, fmt.Errorf("decode readiness policy document: %w", err)
	}
	if err := schema.Validate(document); err != nil {
		return Checklist{}, fmt.Errorf("validate readiness policy schema: %w", err)
	}
	var checklist Checklist
	if err := json.Unmarshal(data, &checklist); err != nil {
		return Checklist{}, fmt.Errorf("decode readiness policy: %w", err)
	}
	if err := checklist.Validate(); err != nil {
		return Checklist{}, err
	}
	return checklist, nil
}

func Load(path string) (Checklist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Checklist{}, fmt.Errorf("read readiness policy: %w", err)
	}
	return decodePolicy(data)
}

func mustBuiltInPolicy() Checklist {
	policy, err := decodePolicy(builtInPolicyJSON)
	if err != nil {
		panic(fmt.Sprintf("invalid built-in readiness policy: %v", err))
	}
	return policy
}

var builtInPolicy = mustBuiltInPolicy()

func DefaultChecklist() Checklist {
	policy := builtInPolicy
	policy.Items = append([]Item(nil), builtInPolicy.Items...)
	return policy
}

func CheckProjection(data []byte) error {
	candidate, err := decodePolicy(data)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(candidate, builtInPolicy) {
		return fmt.Errorf("readiness policy projection differs from built-in policy version %d", ChecklistVersion)
	}
	return nil
}

func BuiltInPolicyJSON() []byte { return append([]byte(nil), builtInPolicyJSON...) }
