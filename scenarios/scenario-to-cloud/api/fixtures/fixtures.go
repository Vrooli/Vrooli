// Package fixtures loads the certification fixture catalog under
// <scenario>/fixtures: leased workload fixtures with deterministic seed data
// and an expected-state oracle, plus coexistence journeys.
//
// The package is the single reader of fixture.json; it validates ownership
// (lease scope + target ownership) and recomputes every seed and oracle
// checksum from the seed bytes so a fixture cannot drift from its declared
// expected state.
package fixtures

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Ownership declares which lease every fixture write belongs to and who
// cleans up. Both fields are required for a fixture to be admitted.
type Ownership struct {
	LeaseScope          string `json:"lease_scope"`
	TargetOwnership     string `json:"target_ownership"`
	CleanupOwner        string `json:"cleanup_owner"`
	CleanupPolicy       string `json:"cleanup_policy"`
	NeighborInvariant   string `json:"neighbor_invariant"`
	WriteScopeInvariant string `json:"write_scope_invariant"`
}

// LeaseScopePrefix is the required prefix for every fixture lease scope.
const LeaseScopePrefix = "fixture:"

// TargetOwnershipOwnedEphemeral is the only target ownership a fixture may
// declare: fixtures never adopt operator-owned targets.
const TargetOwnershipOwnedEphemeral = "owned_ephemeral"

// Validate checks that ownership is fully declared.
func (o Ownership) Validate() error {
	if !strings.HasPrefix(o.LeaseScope, LeaseScopePrefix) || len(o.LeaseScope) == len(LeaseScopePrefix) {
		return fmt.Errorf("ownership.lease_scope must start with %q and name the fixture, got %q", LeaseScopePrefix, o.LeaseScope)
	}
	if o.TargetOwnership != TargetOwnershipOwnedEphemeral {
		return fmt.Errorf("ownership.target_ownership must be %q, got %q", TargetOwnershipOwnedEphemeral, o.TargetOwnership)
	}
	if strings.TrimSpace(o.CleanupOwner) == "" || strings.TrimSpace(o.CleanupPolicy) == "" {
		return fmt.Errorf("ownership.cleanup_owner and cleanup_policy are required")
	}
	return nil
}

// Lifecycle records who returns setup/teardown actions. At phase 2 no owner
// runtime exists, so Status is "declared_not_executable".
type Lifecycle struct {
	Status        string `json:"status"`
	Note          string `json:"note"`
	SetupOwner    string `json:"setup_owner"`
	TeardownOwner string `json:"teardown_owner"`
}

// Listener is one declared network listener.
type Listener struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Role       string `json:"role"`
	Public     bool   `json:"public"`
	PathPrefix string `json:"path_prefix"`
}

// PersistentData is one declared durable data binding.
type PersistentData struct {
	Binding                 string `json:"binding"`
	Kind                    string `json:"kind"`
	Resource                string `json:"resource,omitempty"`
	OwnerScenario           string `json:"owner_scenario,omitempty"`
	Database                string `json:"database,omitempty"`
	Mount                   string `json:"mount,omitempty"`
	QuotaMiB                int    `json:"quota_mib,omitempty"`
	RecoveryPointSecondsMax *int   `json:"recovery_point_seconds_max"`
	ProtectedWriteStream    string `json:"protected_write_stream,omitempty"`
	Regenerable             bool   `json:"regenerable,omitempty"`
}

// Declaration is the declaration-derived shape of the workload.
type Declaration struct {
	PrimaryScenario  string                     `json:"primary_scenario"`
	Resources        []string                   `json:"resources"`
	Scenarios        []string                   `json:"scenarios"`
	Transitive       map[string]json.RawMessage `json:"transitive"`
	Listeners        []Listener                 `json:"listeners"`
	PersistentData   []PersistentData           `json:"persistent_data"`
	ExpectedCapacity map[string]any             `json:"expected_capacity"`
}

// Seed describes the deterministic seed bytes.
type Seed struct {
	Version  string   `json:"version"`
	Files    []string `json:"files"`
	Records  int      `json:"records"`
	Checksum string   `json:"checksum"`
}

// Oracle is the declared expected state derived from the seed.
type Oracle struct {
	Algorithm     string        `json:"algorithm"`
	Description   string        `json:"description"`
	ExpectedState ExpectedState `json:"expected_state"`
	Checksum      string        `json:"checksum"`
}

// Workload is one workloads/<id>/fixture.json document.
type Workload struct {
	SchemaVersion  int         `json:"schema_version"`
	ID             string      `json:"id"`
	Kind           string      `json:"kind"`
	Shape          string      `json:"shape"`
	Classification string      `json:"classification"`
	Description    string      `json:"description"`
	PolicyRef      string      `json:"policy_ref"`
	Declaration    Declaration `json:"declaration"`
	Seed           Seed        `json:"seed"`
	Oracle         Oracle      `json:"oracle"`
	Ownership      Ownership   `json:"ownership"`
	Lifecycle      Lifecycle   `json:"lifecycle"`

	// Dir is the fixture directory the document was loaded from.
	Dir string `json:"-"`
}

// Participant is one deployment or pre-existing workload in a coexistence journey.
type Participant struct {
	Role        string  `json:"role"`
	Deployment  *string `json:"deployment"`
	Workload    string  `json:"workload"`
	Environment string  `json:"environment"`
}

// JourneyStep is one ordered step with its invariant.
type JourneyStep struct {
	Step      int    `json:"step"`
	Action    string `json:"action"`
	Invariant string `json:"invariant"`
}

// Coexistence is one coexistence/<id>.json document.
type Coexistence struct {
	SchemaVersion      int               `json:"schema_version"`
	ID                 string            `json:"id"`
	Kind               string            `json:"kind"`
	Classification     string            `json:"classification"`
	Description        string            `json:"description"`
	PolicyRef          string            `json:"policy_ref"`
	Participants       []Participant     `json:"participants"`
	SharedDependencies []json.RawMessage `json:"shared_dependencies"`
	Journey            []JourneyStep     `json:"journey"`
	MatrixCases        []string          `json:"matrix_cases"`
	Ownership          Ownership         `json:"ownership"`
	Lifecycle          Lifecycle         `json:"lifecycle"`
}

// Catalog is the loaded fixture catalog.
type Catalog struct {
	Root        string
	Workloads   map[string]*Workload
	Coexistence map[string]*Coexistence
}

// Load reads the catalog rooted at root (the scenario's fixtures directory)
// and validates ownership and structure. Checksums are verified separately by
// VerifyOracle so callers can distinguish structural from content drift.
func Load(root string) (*Catalog, error) {
	cat := &Catalog{Root: root, Workloads: map[string]*Workload{}, Coexistence: map[string]*Coexistence{}}
	dirs, err := filepath.Glob(filepath.Join(root, "workloads", "*", "fixture.json"))
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("fixtures: no workloads under %s", root)
	}
	sort.Strings(dirs)
	for _, path := range dirs {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var w Workload
		if err := json.Unmarshal(data, &w); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		w.Dir = filepath.Dir(path)
		if err := w.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if filepath.Base(w.Dir) != w.ID {
			return nil, fmt.Errorf("%s: id %q must match directory name", path, w.ID)
		}
		cat.Workloads[w.ID] = &w
	}
	coex, err := filepath.Glob(filepath.Join(root, "coexistence", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(coex)
	for _, path := range coex {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var c Coexistence
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if err := c.Validate(cat); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		cat.Coexistence[c.ID] = &c
	}
	return cat, nil
}

// Validate checks structural invariants of a workload fixture.
func (w *Workload) Validate() error {
	if w.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version %d", w.SchemaVersion)
	}
	if w.ID == "" || w.Kind != "workload" || w.Shape == "" {
		return fmt.Errorf("id, kind=workload and shape are required")
	}
	switch w.Classification {
	case "certified", "compatibility_only":
	default:
		return fmt.Errorf("classification must be certified or compatibility_only, got %q", w.Classification)
	}
	if err := w.Ownership.Validate(); err != nil {
		return err
	}
	if w.Lifecycle.Status == "" {
		return fmt.Errorf("lifecycle.status is required")
	}
	if len(w.Declaration.Listeners) == 0 {
		return fmt.Errorf("declaration.listeners must declare at least one listener")
	}
	if len(w.Seed.Files) == 0 || w.Seed.Version == "" || !strings.HasPrefix(w.Seed.Checksum, "sha256:") {
		return fmt.Errorf("seed must declare version, files and a sha256 checksum")
	}
	if w.Oracle.Algorithm != OracleAlgorithm || !strings.HasPrefix(w.Oracle.Checksum, "sha256:") {
		return fmt.Errorf("oracle must use %s and declare a sha256 checksum", OracleAlgorithm)
	}
	for _, pd := range w.Declaration.PersistentData {
		if pd.Binding == "" || pd.Kind == "" {
			return fmt.Errorf("persistent_data entries need binding and kind")
		}
		if pd.RecoveryPointSecondsMax == nil && !pd.Regenerable {
			return fmt.Errorf("persistent_data %s must declare recovery_point_seconds_max or regenerable", pd.Binding)
		}
	}
	return nil
}

// Validate checks a coexistence journey and that referenced workloads exist.
func (c *Coexistence) Validate(cat *Catalog) error {
	if c.SchemaVersion != 1 || c.ID == "" || c.Kind != "coexistence" {
		return fmt.Errorf("schema_version=1, id and kind=coexistence are required")
	}
	if err := c.Ownership.Validate(); err != nil {
		return err
	}
	if len(c.Participants) < 2 {
		return fmt.Errorf("a coexistence journey needs at least two participants")
	}
	subjects := 0
	for _, p := range c.Participants {
		if p.Role == "subject" {
			subjects++
		}
		if p.Deployment != nil {
			if _, ok := cat.Workloads[p.Workload]; !ok {
				return fmt.Errorf("participant %s references unknown workload %q", p.Role, p.Workload)
			}
		}
	}
	if subjects != 1 {
		return fmt.Errorf("exactly one subject participant is required, got %d", subjects)
	}
	if len(c.Journey) == 0 {
		return fmt.Errorf("journey must have steps")
	}
	for i, s := range c.Journey {
		if s.Step != i+1 || s.Invariant == "" {
			return fmt.Errorf("journey step %d must be numbered %d and carry an invariant", s.Step, i+1)
		}
	}
	if len(c.MatrixCases) == 0 {
		return fmt.Errorf("matrix_cases must name the certification cases the journey serves")
	}
	return nil
}
