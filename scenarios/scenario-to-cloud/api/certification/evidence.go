package certification

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Verdict is the outcome recorded on an evidence receipt.
type Verdict string

// Verdict vocabulary. Only VerdictPassed can satisfy a cell.
const (
	VerdictPassed      Verdict = "passed"
	VerdictFailed      Verdict = "failed"
	VerdictSkipped     Verdict = "skipped"
	VerdictUnavailable Verdict = "unavailable"
	VerdictUnsupported Verdict = "unsupported"
)

// Candidate identifies the exact release under certification.
type Candidate struct {
	ReleaseDigest       string `json:"release_digest"`
	ConfigurationDigest string `json:"configuration_digest"`
	ClosureDigest       string `json:"closure_digest"`
}

// Target identifies the machine a receipt was observed on.
type Target struct {
	Ground       TargetGround `json:"ground,omitempty"`
	MachineID    string       `json:"machine_id"`
	NodeID       string       `json:"node_id,omitempty"`
	Generation   uint64       `json:"generation,omitempty"`
	Transport    string       `json:"transport,omitempty"`
	Architecture string       `json:"architecture"`
	OS           string       `json:"os"`
}

// Receipt is one evidence package for one (case, lane, target-ground) cell.
// Schema v1 receipts remain readable as historical evidence but have no
// target-ground applicability; v2 receipts carry an explicit target identity.
type Receipt struct {
	SchemaVersion             int                        `json:"schema_version"`
	CaseID                    string                     `json:"case_id"`
	DriverRef                 string                     `json:"driver_ref,omitempty"`
	Verdict                   Verdict                    `json:"verdict"`
	Lane                      Lane                       `json:"lane"`
	RequirementRefs           []string                   `json:"requirement_refs"`
	Candidate                 Candidate                  `json:"candidate"`
	Target                    Target                     `json:"target"`
	OperationRefs             []string                   `json:"operation_refs"`
	ValidationReceiptRef      string                     `json:"validation_receipt_ref"`
	ArtifactRefs              []string                   `json:"artifact_refs"`
	Assertions                []string                   `json:"assertions"`
	TargetReceiptProof        *TargetReceiptProof        `json:"target_receipt_proof,omitempty"`
	TestGenieAssertionProof   *TestGenieAssertionProof   `json:"test_genie_assertion_proof,omitempty"`
	BridgeAssertionOwnerProof *BridgeAssertionOwnerProof `json:"bridge_assertion_owner_proof,omitempty"`
	BASAssertionOwnerProof    *BASAssertionOwnerProof    `json:"bas_assertion_owner_proof,omitempty"`
	ObservedAt                time.Time                  `json:"observed_at"`
	Limitations               []string                   `json:"limitations"`

	// Ref is the receipt's own identity (file basename without extension when
	// loaded from a directory). It is reported back in readiness cells.
	Ref string `json:"-"`

	// ownerEvidenceVerified is an in-process capability set only after the
	// certification package has authenticated every external owner artifact.
	// It is never persisted and cannot be supplied in a JSON receipt.
	ownerEvidenceVerified bool
}

// Validate checks a receipt against the vocabularies of m.
func (r Receipt) Validate(m *Matrix) error {
	if r.SchemaVersion != 1 && r.SchemaVersion != 2 {
		return fmt.Errorf("receipt %s: unsupported schema_version %d", r.Ref, r.SchemaVersion)
	}
	caseRow, err := r.validateVocabulary(m)
	if err != nil {
		return err
	}
	if err := r.validateTargetGround(m); err != nil {
		return err
	}
	if r.SchemaVersion == 2 {
		if err := r.validateSchemaV2(m, caseRow); err != nil {
			return err
		}
		if r.Verdict == VerdictPassed {
			if err := r.validatePassedV2(); err != nil {
				return err
			}
		}
	}
	if r.ObservedAt.IsZero() {
		return fmt.Errorf("receipt %s: observed_at is required", r.Ref)
	}
	return nil
}

func (r Receipt) validateVocabulary(m *Matrix) (Case, error) {
	if r.CaseID == "" {
		return Case{}, fmt.Errorf("receipt %s: empty case_id", r.Ref)
	}
	caseRow, ok := m.Case(r.CaseID)
	if !ok {
		return Case{}, fmt.Errorf("receipt %s: unknown case_id %q", r.Ref, r.CaseID)
	}
	if !containsLane(caseRow.Lanes, r.Lane) && r.SchemaVersion != 1 {
		return Case{}, fmt.Errorf("receipt %s: case %s does not require lane %s", r.Ref, r.CaseID, r.Lane)
	}
	switch r.Verdict {
	case VerdictPassed, VerdictFailed, VerdictSkipped, VerdictUnavailable, VerdictUnsupported:
	default:
		return Case{}, fmt.Errorf("receipt %s: unknown verdict %q", r.Ref, r.Verdict)
	}
	if !containsLane(m.LaneVocabulary, r.Lane) {
		return Case{}, fmt.Errorf("receipt %s: unknown lane %q", r.Ref, r.Lane)
	}
	return caseRow, nil
}

func (r Receipt) validateTargetGround(m *Matrix) error {
	if r.SchemaVersion == 1 && (r.Target.Ground != "" || r.Target.Generation != 0) {
		return fmt.Errorf("receipt %s: schema v1 cannot claim a target-ground identity", r.Ref)
	}
	if r.Target.Ground == "" {
		return nil
	}
	if !containsGround(m.TargetGroundVocabulary, r.Target.Ground) {
		return fmt.Errorf("receipt %s: unknown target ground %q", r.Ref, r.Target.Ground)
	}
	if !containsGround(m.TargetGroundsFor(r.CaseID, r.Lane), r.Target.Ground) {
		return fmt.Errorf("receipt %s: target ground %s is not applicable to %s/%s", r.Ref, r.Target.Ground, r.CaseID, r.Lane)
	}
	return nil
}

func (r Receipt) validateSchemaV2(m *Matrix, caseRow Case) error {
	if err := r.validateV2Target(m); err != nil {
		return err
	}
	return r.validateV2Contract(caseRow)
}

func (r Receipt) validateV2Target(m *Matrix) error {
	if r.Target.Ground == "" {
		return fmt.Errorf("receipt %s: schema v2 requires target.ground", r.Ref)
	}
	if err := r.validateV2Driver(m); err != nil {
		return err
	}
	if strings.TrimSpace(r.Target.MachineID) == "" || r.Target.Generation == 0 || strings.TrimSpace(r.Target.Architecture) == "" || strings.TrimSpace(r.Target.OS) == "" {
		return fmt.Errorf("receipt %s: schema v2 requires a machine-bound target identity, generation, architecture and OS", r.Ref)
	}
	if strings.TrimSpace(r.ValidationReceiptRef) == "" {
		return fmt.Errorf("receipt %s: schema v2 requires validation_receipt_ref", r.Ref)
	}
	if !strings.HasPrefix(r.ValidationReceiptRef, "test-genie:") || strings.TrimSpace(strings.TrimPrefix(r.ValidationReceiptRef, "test-genie:")) == "" {
		return fmt.Errorf("receipt %s: validation_receipt_ref must name a Test Genie receipt", r.Ref)
	}
	return nil
}

func (r Receipt) validateV2Driver(m *Matrix) error {
	wantDriver, err := driverRefForMatrix(m, CellID{CaseID: r.CaseID, Lane: r.Lane, TargetGround: r.Target.Ground})
	if err != nil {
		return fmt.Errorf("receipt %s: %w", r.Ref, err)
	}
	if r.DriverRef != wantDriver {
		return fmt.Errorf("receipt %s: schema v2 driver_ref must be %q for this target-ground cell", r.Ref, wantDriver)
	}
	return nil
}

func (r Receipt) validateV2Contract(caseRow Case) error {
	if !sameStringSlice(r.RequirementRefs, caseRow.RequirementRefs) {
		return fmt.Errorf("receipt %s: requirement_refs must exactly match matrix case %s", r.Ref, r.CaseID)
	}
	if len(r.Assertions) != 1 || strings.TrimSpace(r.Assertions[0]) != caseRow.Assertion {
		return fmt.Errorf("receipt %s: schema v2 assertions must contain the exact frozen matrix assertion for %s", r.Ref, r.CaseID)
	}
	if len(r.OperationRefs) == 0 {
		return fmt.Errorf("receipt %s: schema v2 requires at least one operation_ref", r.Ref)
	}
	for _, ref := range r.OperationRefs {
		if !strings.HasPrefix(ref, "operation:") || strings.TrimSpace(strings.TrimPrefix(ref, "operation:")) == "" {
			return fmt.Errorf("receipt %s: operation_refs must use operation:<id>", r.Ref)
		}
	}
	return nil
}

func (r Receipt) validatePassedV2() error {
	if err := r.validatePassedDigestsAndArtifacts(); err != nil {
		return err
	}
	if err := r.validatePassedBindings(); err != nil {
		return err
	}
	return nil
}

func (r Receipt) validatePassedDigestsAndArtifacts() error {
	if !isSHA256Digest(r.Candidate.ReleaseDigest) || !isSHA256Digest(r.Candidate.ConfigurationDigest) || !isSHA256Digest(r.Candidate.ClosureDigest) {
		return fmt.Errorf("receipt %s: passed verdict requires release, configuration and closure sha256 digests", r.Ref)
	}
	if len(r.ArtifactRefs) == 0 || len(r.Assertions) == 0 {
		return fmt.Errorf("receipt %s: passed verdict requires artifact references and assertions", r.Ref)
	}
	for _, ref := range r.ArtifactRefs {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("receipt %s: artifact_refs cannot contain an empty value", r.Ref)
		}
	}
	return nil
}

func (r Receipt) validatePassedBindings() error {
	if !hasBoundTargetReceipt(r.ArtifactRefs, r.OperationRefs) {
		return fmt.Errorf("receipt %s: passed verdict requires a target-receipt artifact bound to an operation_ref", r.Ref)
	}
	if !hasAssertionArtifact(r.ArtifactRefs, r.DriverRef) {
		return fmt.Errorf("receipt %s: passed verdict requires a case-driver assertion artifact", r.Ref)
	}
	for _, assertion := range r.Assertions {
		if strings.TrimSpace(assertion) == "" {
			return fmt.Errorf("receipt %s: assertions cannot contain an empty value", r.Ref)
		}
	}
	return nil
}

func hasBoundTargetReceipt(artifacts, operations []string) bool {
	for _, artifact := range artifacts {
		for _, operation := range operations {
			operationID := strings.TrimPrefix(operation, "operation:")
			if targetReceiptArtifactBinds(artifact, operationID) {
				return true
			}
		}
	}
	return false
}

func hasAssertionArtifact(artifacts []string, driverRef string) bool {
	prefix := "assertion-artifact:" + driverRef + ":"
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact, prefix) && isSHA256Digest(strings.TrimPrefix(artifact, prefix)) {
			return true
		}
	}
	return false
}

func targetReceiptArtifactBinds(artifact, operationID string) bool {
	prefix := "target-receipt://" + operationID + "/"
	if !strings.HasPrefix(artifact, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(artifact, prefix), "?fence=")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return false
	}
	fenceValue := strings.SplitN(parts[1], "&", 2)[0]
	fence, err := strconv.ParseUint(fenceValue, 10, 64)
	return err == nil && fence > 0
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func containsLane(values []Lane, target Lane) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsGround(values []TargetGround, target TargetGround) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isSHA256Digest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return false
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// LoadEvidenceDir reads every *.json receipt in dir. A missing directory is
// treated as empty evidence, not an error: absence of evidence is a valid,
// explicitly not-ready state.
func LoadEvidenceDir(dir string, m *Matrix) ([]Receipt, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read evidence dir: %w", err)
	}
	var receipts []Receipt
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read receipt %s: %w", e.Name(), err)
		}
		r, err := ParseReceipt(data, m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		r.Ref = strings.TrimSuffix(e.Name(), ".json")
		receipts = append(receipts, r)
	}
	sort.Slice(receipts, func(i, j int) bool { return receipts[i].Ref < receipts[j].Ref })
	return receipts, nil
}

// ParseReceipt decodes and validates one receipt.
func ParseReceipt(data []byte, m *Matrix) (Receipt, error) {
	var r Receipt
	if err := json.Unmarshal(data, &r); err != nil {
		return Receipt{}, fmt.Errorf("parse receipt: %w", err)
	}
	if err := r.Validate(m); err != nil {
		return Receipt{}, err
	}
	return r, nil
}
