package cleanup

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SafetyTier describes the highest-risk effect a provider may perform.
type SafetyTier string

const (
	SafetyTierSafe          SafetyTier = "safe"
	SafetyTierSafeWithOwner SafetyTier = "safe_with_owner"
	SafetyTierConditional   SafetyTier = "conditional"
	SafetyTierForbidden     SafetyTier = "forbidden"
)

// ApprovalMode is the policy gate required before Apply can run.
type ApprovalMode string

const (
	ApprovalModeNone     ApprovalMode = "none"
	ApprovalModeOwner    ApprovalMode = "owner"
	ApprovalModeOperator ApprovalMode = "operator"
	ApprovalModeDisabled ApprovalMode = "disabled"
)

// ProviderMode describes whether a provider is enabled by default.
type ProviderMode string

const (
	ProviderModeEnabled  ProviderMode = "enabled"
	ProviderModeDisabled ProviderMode = "disabled"
)

type ProviderMetadata struct {
	ID                  string
	Name                string
	Version             string
	OwnerScenario       string
	SafetyTier          SafetyTier
	DefaultMode         ProviderMode
	DefaultApproval     ApprovalMode
	SupportedPlatforms  []string
	RequiredPrivileges  []string
	IrreversibleEffects []string
	TestSubstitute      string
}

func (m ProviderMetadata) Validate() error {
	if m.ID == "" {
		return errors.New("provider id is required")
	}
	if m.Name == "" {
		return fmt.Errorf("provider %q name is required", m.ID)
	}
	if m.Version == "" {
		return fmt.Errorf("provider %q version is required", m.ID)
	}
	if m.OwnerScenario == "" {
		return fmt.Errorf("provider %q owner scenario is required", m.ID)
	}
	if !validSafetyTier(m.SafetyTier) {
		return fmt.Errorf("provider %q has invalid safety tier %q", m.ID, m.SafetyTier)
	}
	if !validProviderMode(m.DefaultMode) {
		return fmt.Errorf("provider %q has invalid default mode %q", m.ID, m.DefaultMode)
	}
	if !validApprovalMode(m.DefaultApproval) {
		return fmt.Errorf("provider %q has invalid approval mode %q", m.ID, m.DefaultApproval)
	}
	if m.SafetyTier == SafetyTierConditional && m.DefaultApproval != ApprovalModeOperator && m.DefaultApproval != ApprovalModeDisabled {
		return fmt.Errorf("conditional provider %q must require operator approval or be disabled", m.ID)
	}
	if m.SafetyTier == SafetyTierForbidden && (m.DefaultMode != ProviderModeDisabled || m.DefaultApproval != ApprovalModeDisabled) {
		return fmt.Errorf("forbidden provider %q must be disabled", m.ID)
	}
	if len(m.SupportedPlatforms) == 0 {
		return fmt.Errorf("provider %q must declare supported platforms", m.ID)
	}
	if len(m.IrreversibleEffects) == 0 {
		return fmt.Errorf("provider %q must declare irreversible effects", m.ID)
	}
	if m.TestSubstitute == "" {
		return fmt.Errorf("provider %q must declare a test substitute", m.ID)
	}
	return nil
}

func validSafetyTier(t SafetyTier) bool {
	switch t {
	case SafetyTierSafe, SafetyTierSafeWithOwner, SafetyTierConditional, SafetyTierForbidden:
		return true
	default:
		return false
	}
}

func validProviderMode(m ProviderMode) bool {
	switch m {
	case ProviderModeEnabled, ProviderModeDisabled:
		return true
	default:
		return false
	}
}

func validApprovalMode(m ApprovalMode) bool {
	switch m {
	case ApprovalModeNone, ApprovalModeOwner, ApprovalModeOperator, ApprovalModeDisabled:
		return true
	default:
		return false
	}
}

type ObservationScope struct {
	RootPaths []string
	Now       time.Time
}

type EstimateRequest struct {
	Scope  ObservationScope
	Policy ProviderPolicy
}

type Estimate struct {
	ProviderID       string
	ProviderVersion  string
	EstimatedBytes   int64
	ItemCount        int
	RequiresApproval bool
	BlockedReason    string
	ObservedAt       time.Time
}

type PreviewRequest struct {
	Scope    ObservationScope
	Policy   ProviderPolicy
	Estimate Estimate
}

type PreviewItem struct {
	ID          string
	Path        string
	Description string
	Bytes       int64
	Action      string
	SafetyTier  SafetyTier
}

type Preview struct {
	ProviderID      string
	ProviderVersion string
	Items           []PreviewItem
	Warnings        []string
	BlockedReason   string
}

type ApplyRequest struct {
	PlanID          string
	PolicyVersion   string
	ProviderVersion string
	ApprovalMode    ApprovalMode
	IdempotencyKey  string
	Preview         Preview
}

type ApplyResult struct {
	ProviderID     string
	Applied        bool
	AlreadyDone    bool
	ReclaimedBytes int64
	SkippedItems   []string
	Warnings       []string
}

type VerifyRequest struct {
	ApplyResult ApplyResult
}

type VerifyResult struct {
	Verified bool
	Message  string
}

type ProviderPolicy struct {
	Enabled      bool
	MinAge       time.Duration
	MaxBytes     int64
	ApprovalMode ApprovalMode
}

// Provider is intentionally preview-first. Apply-only implementations cannot
// satisfy this interface.
type Provider interface {
	Metadata() ProviderMetadata
	Estimate(ctx context.Context, req EstimateRequest) (Estimate, error)
	Preview(ctx context.Context, req PreviewRequest) (Preview, error)
	Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error)
	Verify(ctx context.Context, req VerifyRequest) (VerifyResult, error)
}

func ValidateProvider(p Provider) error {
	if p == nil {
		return errors.New("provider is nil")
	}
	return p.Metadata().Validate()
}
