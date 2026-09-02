// Package readiness defines the deployment-time checklist used to assemble a
// scenario release verdict. It deliberately has no Test Genie phase or
// scenario-specific implementation dependency.
package readiness

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const ChecklistVersion = 1

type CleanRequirement string

const (
	Required    CleanRequirement = "required"
	Advisory    CleanRequirement = "advisory"
	Uncheckable CleanRequirement = "uncheckable"
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

type Item struct {
	ID                 string           `json:"id"`
	Title              string           `json:"title"`
	Category           string           `json:"category"`
	CleanRequirement   CleanRequirement `json:"clean_requirement"`
	GlobalImpact       GlobalImpact     `json:"global_impact"`
	AcceptanceCriteria string           `json:"acceptance_criteria"`
}

type Checklist struct {
	Version int    `json:"version"`
	Items   []Item `json:"items"`
}

func (i Item) Validate() error {
	if strings.TrimSpace(i.ID) == "" || strings.TrimSpace(i.Title) == "" {
		return fmt.Errorf("checklist item requires id and title")
	}
	if strings.TrimSpace(i.Category) == "" || strings.TrimSpace(i.AcceptanceCriteria) == "" {
		return fmt.Errorf("checklist item %q requires category and acceptance criteria", i.ID)
	}
	switch i.CleanRequirement {
	case Required, Advisory, Uncheckable:
	default:
		return fmt.Errorf("checklist item %q has invalid clean_requirement %q", i.ID, i.CleanRequirement)
	}
	switch i.GlobalImpact {
	case FoundationBlocker, SafetyBlocker, CapabilityGap, HardeningGap, AdvisoryImpact, UnknownImpact:
	default:
		return fmt.Errorf("checklist item %q has invalid global_impact %q", i.ID, i.GlobalImpact)
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

func Load(path string) (Checklist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Checklist{}, fmt.Errorf("read readiness checklist: %w", err)
	}
	var checklist Checklist
	if err := json.Unmarshal(data, &checklist); err != nil {
		return Checklist{}, fmt.Errorf("decode readiness checklist: %w", err)
	}
	if err := checklist.Validate(); err != nil {
		return Checklist{}, err
	}
	return checklist, nil
}

func DefaultChecklist() Checklist {
	return Checklist{Version: ChecklistVersion, Items: []Item{
		{ID: "storefront-registered", Title: "Storefront application is registered", Category: "mechanical", CleanRequirement: Required, GlobalImpact: FoundationBlocker, AcceptanceCriteria: "The release has a registered storefront application and reachable listing."},
		{ID: "declared-meters-covered", Title: "Declared meters have limits and enforcement", Category: "mechanical", CleanRequirement: Required, GlobalImpact: SafetyBlocker, AcceptanceCriteria: "Every declared meter has a tier limit and an exercised enforcement path."},
		{ID: "bundle-price-present", Title: "A current bundle price exists", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "The bundle has at least one enabled price row for this release."},
		{ID: "update-policy-set", Title: "Update policy is declared", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "The scenario declares how updates are delivered and supported."},
		{ID: "platform-assets-set", Title: "Per-platform assets are set", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "Every declared target platform has its required release assets."},
		{ID: "brand-assignment-exists", Title: "A brand assignment exists", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "The scenario has a current brand-manager assignment or an explicit advisory finding."},
		{ID: "marketing-assets-available", Title: "Launch assets are answerable", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "The launch-assets report names available channels and open artifact slots for the scenario."},
		{ID: "go-to-market-authentic", Title: "Go-to-market and monetization documents have substance", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "The declared commercial documents contain scenario-specific content rather than generator scaffold."},
		{ID: "ramp-evidence-complete", Title: "Ramp evidence exists for every target", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "Each release target has current, attributable ramp evidence."},
		{ID: "suite-state-known", Title: "Suite state is known", Category: "mechanical", CleanRequirement: Required, GlobalImpact: SafetyBlocker, AcceptanceCriteria: "All required readiness signal producers return known, attributable states."},
		{ID: "declared-features-reachable", Title: "Declared features are reachable", Category: "correspondence", CleanRequirement: Required, GlobalImpact: SafetyBlocker, AcceptanceCriteria: "Every feature declared for sale is reachable in the built scenario."},
		{ID: "enforcement-paths-gate", Title: "Enforcement paths actually gate", Category: "correspondence", CleanRequirement: Required, GlobalImpact: SafetyBlocker, AcceptanceCriteria: "Each declared enforcement path refuses the corresponding simulated commercial condition."},
		{ID: "storefront-copy-current", Title: "Storefront copy describes this version", Category: "correspondence", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "Storefront claims correspond to the current release behavior."},
		{ID: "audience-matches-build", Title: "Declared audience matches the build", Category: "correspondence", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "The declared audience is supported by the built experience."},
		{ID: "branding-coherent", Title: "Branding is applied and coherent", Category: "correspondence", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "The released experience applies the declared brand consistently."},
		{ID: "requirements-cover-sale", Title: "Requirements cover what is sold", Category: "correspondence", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "Requirements and evidence cover every load-bearing commercial promise."},
		{ID: "subscription-sign-in", Title: "Subscription sign-in works end to end", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "A new customer can sign in and receive the shared paid session."},
		{ID: "paying-customer-onboarding", Title: "Onboarding is usable by a paying customer", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "A new paying customer can complete onboarding and reach the paid surface."},
		{ID: "non-subscriber-gate-honest", Title: "The gate is honest to a non-subscriber", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "A refused user sees a clear reason and a reachable upgrade destination."},
		{ID: "payment-stop-survivable", Title: "Stopping payment is survivable", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "Cancellation or payment failure leaves the customer with an honest degraded experience."},
		{ID: "paying-customer-contact", Title: "A paying customer has a contact path", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "A paying customer can reach the declared support or contact channel."},
	}}
}
