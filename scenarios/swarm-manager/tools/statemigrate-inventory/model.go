package main

// Inventory is the deterministic, machine-readable snapshot of every class of
// swarm-manager persisted state relevant to the Phase 8 state migration.
//
// Determinism contract (byte-stability): the JSON encoding of Inventory must be
// byte-identical across two back-to-back runs over unchanged live state. To
// guarantee that, this type carries NO timestamps and NO wall-clock data, every
// slice is sorted before encoding, and every map is encoded by encoding/json
// with sorted keys. The only machine-varying data are absolute root paths, which
// are home-redacted (see redactHome) so the payload is portable and stable.
type Inventory struct {
	SchemaVersion string `json:"schema_version"`
	Roots         Roots  `json:"roots"`

	// Classes is the per-object-class inventory, sorted by Class.
	Classes []ClassInventory `json:"object_classes"`

	PlanRefs            PlanRefSummary   `json:"plan_refs"`
	Ownership           OwnershipSummary `json:"ownership"`
	ReferentialFindings []Finding        `json:"referential_findings"`
	Anomalies           []Anomaly        `json:"anomalies"`
	ExpectedAbsent      []ExpectedAbsent `json:"expected_absent"`
	Totals              Totals           `json:"totals"`
}

// Roots records the resolved (home-redacted) storage roots the scan covered and
// whether each was present on disk. An absent-but-expected root is a
// blocker-to-migration signal, not empty state.
type Roots struct {
	ResolvedFrom string     `json:"resolved_from"`
	Data         RootStatus `json:"data"`
	State        RootStatus `json:"state"`
	Cache        RootStatus `json:"cache"`
	ConfigFile   RootStatus `json:"config_file"`
	// ShadowNamespacesPresent lists sibling *_shadow namespace roots discovered
	// next to the live roots. Their presence means a Baseline-Modes shadow
	// engagement wrote parallel state that a migration must account for.
	ShadowNamespacesPresent []string `json:"shadow_namespaces_present"`
}

// RootStatus is a root path plus whether it exists.
type RootStatus struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// ClassInventory summarizes one object class.
type ClassInventory struct {
	Class string `json:"class"`
	// Kind distinguishes primary documents (each carries a stable identity) from
	// derived/opaque/sub-artifact classes counted only in aggregate.
	Kind        string         `json:"kind"` // primary | artifact | opaque | state
	Count       int            `json:"count"`
	Bytes       int64          `json:"bytes"`
	ByStatus    map[string]int `json:"by_status,omitempty"`
	ByKind      map[string]int `json:"by_kind,omitempty"`
	WithPlanRef int            `json:"with_plan_ref,omitempty"`
	// ContentHash is sha256 over the sorted (relpath, filehash) pairs of every
	// file attributed to this class — stable, and covers sub-artifacts so nothing
	// is silently uncounted.
	ContentHash string `json:"content_hash"`
	// Objects lists primary documents only, sorted by Identity. Empty for
	// artifact/opaque/state aggregate classes.
	Objects []ObjectRecord `json:"objects,omitempty"`
}

// ObjectRecord is one primary persisted document with a stable identity.
type ObjectRecord struct {
	Identity string `json:"identity"`
	RelPath  string `json:"rel_path"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
	Status   string `json:"status,omitempty"`
	// Refs are outbound references (dependencies, membership, plan links) used
	// for referential-integrity checks. Sorted.
	Refs []string `json:"refs,omitempty"`
}

// PlanRefSummary counts plan-manager references and isolates unmanaged usage.
type PlanRefSummary struct {
	Total     int             `json:"total"`
	Managed   int             `json:"managed"`
	Unmanaged int             `json:"unmanaged"`
	Details   []PlanRefDetail `json:"unmanaged_details"`
}

// PlanRefDetail describes one unmanaged plan-ref occurrence (missing plan_id,
// non-plan-manager provider, or unknown role).
type PlanRefDetail struct {
	Owner    string `json:"owner"`
	Provider string `json:"provider"`
	PlanID   string `json:"plan_id"`
	Role     string `json:"role"`
	Reason   string `json:"reason"`
}

// OwnershipSummary captures run-owner index state and ambiguity.
type OwnershipSummary struct {
	GlobalRunOwnerIndexPresent bool             `json:"global_run_owner_index_present"`
	ScopeRunOwnerIndexes       int              `json:"scope_run_owner_indexes"`
	EngagementOwnersPresent    bool             `json:"engagement_owners_present"`
	AmbiguousRunOwners         []AmbiguousOwner `json:"ambiguous_run_owners"`
}

// AmbiguousOwner is a run id that resolves to more than one owner.
type AmbiguousOwner struct {
	RunID  string   `json:"run_id"`
	Owners []string `json:"owners"`
	Source string   `json:"source"`
}

// Finding is a referential-integrity or structural finding (non-fatal, but
// migration-relevant).
type Finding struct {
	Type   string `json:"type"`
	From   string `json:"from"`
	To     string `json:"to,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Anomaly is a piece of state that could not be read or parsed, or that carries
// an invalid value. Anomalies are REPORTED, never dropped.
type Anomaly struct {
	Type    string `json:"type"`
	RelPath string `json:"rel_path"`
	Detail  string `json:"detail"`
}

// ExpectedAbsent records a known state file/root that the code writes but that
// is absent on disk. Explicitly surfaced so "not present" is never confused with
// "not scanned".
type ExpectedAbsent struct {
	RelPath string `json:"rel_path"`
	Root    string `json:"root"`
	Note    string `json:"note"`
}

// Totals are the global reconciliation anchors.
type Totals struct {
	FilesScanned int   `json:"files_scanned"`
	Bytes        int64 `json:"bytes"`
	ObjectCount  int   `json:"object_count"`
	AnomalyCount int   `json:"anomaly_count"`
	FindingCount int   `json:"finding_count"`
	// ContentHash is sha256 over the sorted "root/relpath\x00filehash" lines of
	// every readable file across all roots. This is the master byte-stability and
	// pre/post-migration reconciliation anchor.
	ContentHash string `json:"content_hash"`
}
