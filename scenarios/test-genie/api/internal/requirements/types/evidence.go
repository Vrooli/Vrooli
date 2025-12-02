package types

import "time"

// EvidenceRecord represents test execution evidence from a single source.
type EvidenceRecord struct {
	RequirementID   string
	ValidationRef   string
	Status          LiveStatus
	Phase           string
	Evidence        string
	UpdatedAt       time.Time
	DurationSeconds float64
	SourcePath      string
	Metadata        map[string]any
}

// EvidenceMap indexes evidence by requirement ID.
type EvidenceMap map[string][]EvidenceRecord

// Get retrieves all evidence for a requirement ID.
func (m EvidenceMap) Get(id string) []EvidenceRecord {
	if m == nil {
		return nil
	}
	return m[id]
}

// Add appends an evidence record for a requirement.
func (m EvidenceMap) Add(record EvidenceRecord) {
	if m == nil {
		return
	}
	m[record.RequirementID] = append(m[record.RequirementID], record)
}

// Merge combines another evidence map into this one.
func (m EvidenceMap) Merge(other EvidenceMap) {
	if m == nil || other == nil {
		return
	}
	for id, records := range other {
		m[id] = append(m[id], records...)
	}
}

// RequirementIDs returns all requirement IDs with evidence.
func (m EvidenceMap) RequirementIDs() []string {
	if m == nil {
		return nil
	}
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	return ids
}

// Count returns the total number of evidence records.
func (m EvidenceMap) Count() int {
	if m == nil {
		return 0
	}
	count := 0
	for _, records := range m {
		count += len(records)
	}
	return count
}

// ManualValidation represents a manual validation entry from log.jsonl.
type ManualValidation struct {
	RequirementID string    `json:"requirement_id"`
	Status        string    `json:"status"`
	ValidatedAt   time.Time `json:"validated_at"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	ValidatedBy   string    `json:"validated_by,omitempty"`
	ArtifactPath  string    `json:"artifact_path,omitempty"`
	Notes         string    `json:"notes,omitempty"`
}

// IsExpired returns true if the manual validation has expired.
func (m *ManualValidation) IsExpired() bool {
	if m == nil || m.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(m.ExpiresAt)
}

// IsValid returns true if the manual validation is current and not expired.
func (m *ManualValidation) IsValid() bool {
	return m != nil && !m.IsExpired()
}

// ToLiveStatus converts the manual validation status to LiveStatus.
func (m *ManualValidation) ToLiveStatus() LiveStatus {
	if m == nil || m.IsExpired() {
		return LiveNotRun
	}
	return NormalizeLiveStatus(m.Status)
}

// PhaseResult represents aggregated test results for a single phase.
type PhaseResult struct {
	Phase           string         `json:"phase"`
	Status          string         `json:"status"`
	ExecutedAt      time.Time      `json:"executed_at,omitempty"`
	DurationSeconds float64        `json:"duration_seconds,omitempty"`
	TestCount       int            `json:"test_count,omitempty"`
	PassCount       int            `json:"pass_count,omitempty"`
	FailCount       int            `json:"fail_count,omitempty"`
	SkipCount       int            `json:"skip_count,omitempty"`
	CoveragePercent float64        `json:"coverage_percent,omitempty"`
	RequirementIDs  []string       `json:"requirement_ids,omitempty"`
	Errors          []string       `json:"errors,omitempty"`
	LogPath         string         `json:"log_path,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// ToLiveStatus converts the phase result status to LiveStatus.
func (p *PhaseResult) ToLiveStatus() LiveStatus {
	if p == nil {
		return LiveNotRun
	}
	return NormalizeLiveStatus(p.Status)
}

// HasFailures returns true if the phase had any failures.
func (p *PhaseResult) HasFailures() bool {
	return p != nil && p.FailCount > 0
}

// VitestResult represents parsed vitest coverage data.
type VitestResult struct {
	FilePath      string         `json:"file_path"`
	TestNames     []string       `json:"test_names,omitempty"`
	RequirementID string         `json:"requirement_id,omitempty"`
	Status        string         `json:"status,omitempty"`
	Duration      float64        `json:"duration,omitempty"`
	CoveredLines  int            `json:"covered_lines,omitempty"`
	TotalLines    int            `json:"total_lines,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// ToLiveStatus converts the vitest result status to LiveStatus.
func (v *VitestResult) ToLiveStatus() LiveStatus {
	if v == nil {
		return LiveNotRun
	}
	return NormalizeLiveStatus(v.Status)
}

// CoveragePercent calculates line coverage percentage.
func (v *VitestResult) CoveragePercent() float64 {
	if v == nil || v.TotalLines == 0 {
		return 0
	}
	return float64(v.CoveredLines) / float64(v.TotalLines) * 100
}

// EvidenceBundle contains all loaded evidence from various sources.
type EvidenceBundle struct {
	PhaseResults      EvidenceMap
	VitestEvidence    map[string][]VitestResult // keyed by requirement ID
	ManualValidations *ManualManifest
	LoadedAt          time.Time
}

// NewEvidenceBundle creates an empty evidence bundle.
func NewEvidenceBundle() *EvidenceBundle {
	return &EvidenceBundle{
		PhaseResults:      make(EvidenceMap),
		VitestEvidence:    make(map[string][]VitestResult),
		ManualValidations: nil,
		LoadedAt:          time.Now(),
	}
}

// IsEmpty returns true if no evidence was loaded.
func (b *EvidenceBundle) IsEmpty() bool {
	if b == nil {
		return true
	}
	return len(b.PhaseResults) == 0 &&
		len(b.VitestEvidence) == 0 &&
		(b.ManualValidations == nil || len(b.ManualValidations.Entries) == 0)
}

// ManualManifest contains parsed manual validations.
type ManualManifest struct {
	ManifestPath  string
	Entries       []ManualValidation
	ByRequirement map[string]ManualValidation
}

// NewManualManifest creates a new manual manifest.
func NewManualManifest(path string) *ManualManifest {
	return &ManualManifest{
		ManifestPath:  path,
		Entries:       nil,
		ByRequirement: make(map[string]ManualValidation),
	}
}

// Add adds a manual validation and indexes it.
func (m *ManualManifest) Add(v ManualValidation) {
	if m == nil {
		return
	}
	m.Entries = append(m.Entries, v)
	// Later entries override earlier ones for the same requirement
	m.ByRequirement[v.RequirementID] = v
}

// Get retrieves the most recent manual validation for a requirement.
func (m *ManualManifest) Get(requirementID string) (ManualValidation, bool) {
	if m == nil || m.ByRequirement == nil {
		return ManualValidation{}, false
	}
	v, ok := m.ByRequirement[requirementID]
	return v, ok
}

// Count returns the number of manual validation entries.
func (m *ManualManifest) Count() int {
	if m == nil {
		return 0
	}
	return len(m.Entries)
}
