package settings

type PolicyControls struct {
	Execution ExecutionControls
	Retry     RetryControls
	Review    ReviewControls
	Budgets   AgentBudgetControls
}
type (
	ExecutionControls struct{ DefaultMode string }
	RetryControls     struct {
		AutoFixup        bool
		MaxFixupAttempts int
	}
)

type ReviewControls struct {
	AgentEnabled          bool
	CodeQualityMinScore   float64
	TestMinPassRate       float64
	MaxBlockingViolations int
	MaxWarnings           int
	RequireScreenshots    bool
	RequireTests          bool
}
type AgentBudgetControls struct {
	MaxTurns       int
	TimeoutSeconds int
}
type PolicyControlsProvider interface {
	LoadPolicyControls() (PolicyControls, error)
}

func DefaultPolicyControls() PolicyControls { return ProjectPolicyControls(DefaultSettings()) }

// ProjectPolicyControls derives the typed policy-controls projection
// from a (normalized) Settings value. This is the SINGLE mapping from
// persisted user settings to the controls orchestration consumers read; the
// legacy execution.Policy / execution.ReviewThresholds adapters are derived
// from this projection so the two surfaces can never disagree.
//
// The mapping is field-for-field lossless: identical settings produce
// identical controls, so re-plumbing a consumer onto the projection is a
// behavior no-op (asserted by TestProjectionEquivalence).
func ProjectPolicyControls(s Settings) PolicyControls {
	return PolicyControls{
		Execution: ExecutionControls{
			DefaultMode: s.DefaultMode,
		},
		Retry: RetryControls{
			AutoFixup:        s.AutoFixup,
			MaxFixupAttempts: s.MaxFixupAttempts,
		},
		Review: ReviewControls{
			AgentEnabled:          s.ReviewAgentEnabled,
			CodeQualityMinScore:   s.ReviewCodeQualityMinScore,
			TestMinPassRate:       s.ReviewTestMinPassRate,
			MaxBlockingViolations: s.ReviewMaxBlockingViolations,
			MaxWarnings:           s.ReviewMaxWarnings,
			RequireScreenshots:    s.ReviewRequireScreenshots,
			RequireTests:          s.ReviewRequireTests,
		},
		Budgets: AgentBudgetControls{
			MaxTurns:       s.AgentMaxTurns,
			TimeoutSeconds: s.AgentTimeoutSeconds,
		},
	}
}

// policyControlsAdapter bridges Store to the policy-controls provider.
// Load happens on every call so operator updates apply without restart.
type policyControlsAdapter struct {
	store *Store
}

// NewPolicyControlsAdapter creates a PolicyControlsProvider backed by the
// given store. A nil store resolves the default scenario settings path on
// every load (matching the legacy per-call settings.NewStore("") pattern, so
// tests that repoint SCENARIO_ROOT keep working).
func NewPolicyControlsAdapter(store *Store) PolicyControlsProvider {
	return &policyControlsAdapter{store: store}
}

func (a *policyControlsAdapter) LoadPolicyControls() (PolicyControls, error) {
	store := a.store
	if store == nil {
		store = NewStore("")
	}
	s, err := store.Load()
	if err != nil {
		return PolicyControls{}, err
	}
	return ProjectPolicyControls(s), nil
}

// Settings-field classification roles for the public projection surface.
// Values mirror the proto SettingsFieldRole enum in api/settings.proto.
const (
	FieldRoleUserPreference = "user_preference"
	FieldRolePolicyControl  = "policy_control"
	FieldRoleGovernance     = "governance"
	FieldRoleDormant        = "dormant"
)

// FieldClassification records where one persisted settings field lands in the
// declarative-operations model: a retained user preference, a policy control
// (with its destination path inside PolicyControls), system governance, or a
// dormant field with no runtime reader.
type FieldClassification struct {
	// Field is the persisted JSON field name in Settings.
	Field string
	// Role is one of the FieldRole* constants.
	Role string
	// Control is the destination path inside PolicyControls (JSON path
	// segments, e.g. "retry.max_fixup_attempts"). Empty unless Role is
	// policy_control or dormant-with-destination.
	Control string
	// Note is a short human-readable explanation.
	Note string
}

// PolicyFieldClassifications is the authoritative classification of every
// orchestration-flavored settings field (plus the dormant one). Phase 8 uses
// this table (mirrored in docs/operations/migration/LEGACY-MAPPING.md §9)
// when migrating persisted settings into system-default bindings and
// transition-policy revisions. Pure-UI preferences (theme, debounce, toast,
// delete confirmations) and system governance (lanes, queue depth, circuit
// breaker, cost caps, fix-before-feature, auto-filer) are intentionally not
// enumerated field-by-field; they stay in settings unchanged.
func PolicyFieldClassifications() []FieldClassification {
	return []FieldClassification{
		{Field: "default_mode", Role: FieldRolePolicyControl, Control: "execution.default_mode", Note: "Default execution posture when a queue request omits mode."},
		{Field: "auto_fixup", Role: FieldRolePolicyControl, Control: "retry.auto_fixup", Note: "Automatic fixup re-run after a failed review."},
		{Field: "max_fixup_attempts", Role: FieldRolePolicyControl, Control: "retry.max_fixup_attempts", Note: "Retained user preference: retry limit consumed via policy controls."},
		{Field: "review_agent_enabled", Role: FieldRolePolicyControl, Control: "review.agent_enabled", Note: "Gates autonomous review-round (evidence) spawning."},
		{Field: "agent_max_turns", Role: FieldRolePolicyControl, Control: "budgets.max_turns", Note: "Per-run turn budget; cost-cap estimation input."},
		{Field: "agent_timeout_seconds", Role: FieldRoleDormant, Control: "budgets.timeout_seconds", Note: "No runtime reader today (spawn timeouts come from the agent profile); persisted field retained."},
		{Field: "review_code_quality_min_score", Role: FieldRolePolicyControl, Control: "review.code_quality_min_score", Note: "Retained user preference: review threshold consumed via policy controls."},
		{Field: "review_test_min_pass_rate", Role: FieldRolePolicyControl, Control: "review.test_min_pass_rate", Note: "Retained user preference: review threshold consumed via policy controls."},
		{Field: "review_max_blocking_violations", Role: FieldRolePolicyControl, Control: "review.max_blocking_violations", Note: "Retained user preference: review threshold consumed via policy controls."},
		{Field: "review_max_warnings", Role: FieldRolePolicyControl, Control: "review.max_warnings", Note: "Retained user preference: review threshold consumed via policy controls."},
		{Field: "review_require_screenshots", Role: FieldRolePolicyControl, Control: "review.require_screenshots", Note: "Retained user preference: review threshold consumed via policy controls."},
		{Field: "review_require_tests", Role: FieldRolePolicyControl, Control: "review.require_tests", Note: "Retained user preference: review threshold consumed via policy controls."},
	}
}
