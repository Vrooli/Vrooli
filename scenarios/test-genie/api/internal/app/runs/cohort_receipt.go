package runs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/executionevidence"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
)

const cohortReceiptMaxBytes = 1 << 20

// ExperienceCohortInput contains only the caller's joins to ordinary durable
// runs. Verdicts and artifact references are resolved again from Test Genie's
// stored run index and per-run evidence manifest before the receipt is written.
type ExperienceCohortInput struct {
	Scenarios []ExperienceCohortRunInput `json:"scenarios"`
	Cancelled bool                       `json:"cancelled,omitempty"`
}

type ExperienceCohortRunInput struct {
	Scenario      string `json:"scenario"`
	RunID         string `json:"runId,omitempty"`
	BuildIdentity string `json:"buildIdentity,omitempty"`
	InvokerStatus string `json:"invokerStatus,omitempty"`
}

type ExperienceCohortReceipt struct {
	SchemaVersion int                           `json:"schemaVersion"`
	ReceiptID     string                        `json:"receiptId"`
	InputDigest   string                        `json:"inputDigest"`
	Phase         string                        `json:"phase"`
	Status        string                        `json:"status"`
	Cancelled     bool                          `json:"cancelled,omitempty"`
	CreatedAt     time.Time                     `json:"createdAt"`
	Scenarios     []ExperienceCohortRunEvidence `json:"scenarios"`
}

type ExperienceCohortRunEvidence struct {
	Scenario         string                          `json:"scenario"`
	RunID            string                          `json:"runId,omitempty"`
	BuildIdentity    string                          `json:"buildIdentity,omitempty"`
	SourceGitSHA     string                          `json:"sourceGitSha,omitempty"`
	SourceTreeDigest string                          `json:"sourceTreeDigest,omitempty"`
	InvokerStatus    string                          `json:"invokerStatus,omitempty"`
	Verdict          string                          `json:"verdict,omitempty"`
	Findings         *executionevidence.ArtifactRef  `json:"findings,omitempty"`
	Evidence         []executionevidence.ArtifactRef `json:"evidenceRefs,omitempty"`
	Missing          []string                        `json:"missingProvenance,omitempty"`
}

type StoredExperienceCohortReceipt struct {
	Receipt  ExperienceCohortReceipt       `json:"receipt"`
	Artifact executionevidence.ArtifactRef `json:"artifact"`
	ReadURL  string                        `json:"readUrl"`
}

// PersistExperienceCohortReceipt stores the combined record as immutable
// evidence beneath the first known canonical run. It indexes the receipt from
// that run's existing evidence manifest when one is available. Receipt IDs are
// content-derived from the request, so a retry can verify/reuse the exact file
// while a changed request receives a distinct path.
func (s *Service) PersistExperienceCohortReceipt(ctx context.Context, input ExperienceCohortInput) (StoredExperienceCohortReceipt, error) {
	if s == nil || s.artifactRoot == nil {
		return StoredExperienceCohortReceipt{}, errors.New("run evidence store is unavailable")
	}
	if len(input.Scenarios) == 0 || len(input.Scenarios) > 20 {
		return StoredExperienceCohortReceipt{}, errors.New("cohort requires 1 to 20 scenarios")
	}
	seen := map[string]struct{}{}
	reqBytes, err := json.Marshal(input)
	if err != nil {
		return StoredExperienceCohortReceipt{}, fmt.Errorf("encode cohort input: %w", err)
	}
	inputDigestBytes := sha256.Sum256(reqBytes)
	inputDigest := hex.EncodeToString(inputDigestBytes[:])
	// Serialize receipt publication in-process. Files are still immutable across
	// processes; conflicting concurrent writes fail closed at the evidence writer.
	s.cohortReceiptMu.Lock()
	defer s.cohortReceiptMu.Unlock()

	receipt := ExperienceCohortReceipt{SchemaVersion: 1, InputDigest: inputDigest, Phase: "experience", Status: "passed", Cancelled: input.Cancelled, CreatedAt: time.Now().UTC(), Scenarios: make([]ExperienceCohortRunEvidence, 0, len(input.Scenarios))}
	for _, requested := range input.Scenarios {
		if err := ctx.Err(); err != nil {
			return StoredExperienceCohortReceipt{}, err
		}
		name := strings.TrimSpace(requested.Scenario)
		if name == "" {
			return StoredExperienceCohortReceipt{}, errors.New("cohort scenario name is required")
		}
		if _, duplicate := seen[name]; duplicate {
			return StoredExperienceCohortReceipt{}, fmt.Errorf("duplicate cohort scenario %q", name)
		}
		seen[name] = struct{}{}
		row := ExperienceCohortRunEvidence{Scenario: name, RunID: strings.TrimSpace(requested.RunID), BuildIdentity: strings.TrimSpace(requested.BuildIdentity), InvokerStatus: strings.TrimSpace(requested.InvokerStatus)}
		if row.InvokerStatus != "" && row.InvokerStatus != "pass" && row.InvokerStatus != "fail" && row.InvokerStatus != "not_evaluated" && row.InvokerStatus != "error" && row.InvokerStatus != "cancelled" {
			return StoredExperienceCohortReceipt{}, fmt.Errorf("scenario %q has invalid invoker status", name)
		}
		if row.BuildIdentity != "" && !validCohortBuildIdentity(row.BuildIdentity) {
			row.BuildIdentity = ""
			row.Missing = append(row.Missing, "invalid_returned_build_identity")
		}
		if row.BuildIdentity == "" {
			row.Missing = appendCohortMissing(row.Missing, "returned_build_identity")
		}
		if row.RunID == "" {
			row.Missing = append(row.Missing, "run_id", "run_index_record", "canonical_verdict", "findings_artifact")
			if row.InvokerStatus == "" {
				row.InvokerStatus = "not_started_or_unknown"
			}
			receipt.Status = "partial"
			receipt.Scenarios = append(receipt.Scenarios, row)
			continue
		}
		if !validStrictIdentity(row.RunID) {
			row.Missing = append(row.Missing, "run_id_valid")
			receipt.Status = "partial"
			receipt.Scenarios = append(receipt.Scenarios, row)
			continue
		}
		sourceDir, artifactRoot, err := s.targetDirs(name)
		if err != nil {
			return StoredExperienceCohortReceipt{}, err
		}
		_ = sourceDir
		record, findErr := sharedruns.NewIndex(artifactRoot).Find(row.RunID)
		if findErr != nil {
			row.Missing = append(row.Missing, "run_index_record")
			receipt.Status = "partial"
			receipt.Scenarios = append(receipt.Scenarios, row)
			continue
		}
		record.NormalizeTargetIdentity()
		if record.TargetKind != "scenario" || record.TargetID != name {
			row.Missing = append(row.Missing, "run_target_identity_match")
			receipt.Status = "partial"
			receipt.Scenarios = append(receipt.Scenarios, row)
			continue
		}
		if record.Status != sharedruns.StatusPassed && record.Status != sharedruns.StatusFailed && record.Status != sharedruns.StatusAborted {
			row.Missing = append(row.Missing, "terminal_run_record")
			receipt.Status = "partial"
			receipt.Scenarios = append(receipt.Scenarios, row)
			continue
		}
		row.SourceGitSHA = strings.TrimSpace(record.GitSha)
		row.SourceTreeDigest = strings.TrimSpace(record.TreeDigest)
		if row.SourceGitSHA == "" && row.SourceTreeDigest == "" {
			row.Missing = append(row.Missing, "canonical_source_build_identity")
		}
		runRoot := sharedartifacts.RunDir(artifactRoot, row.RunID)
		manifest, manifestErr := executionevidence.ReadManifest(runRoot)
		if manifestErr != nil {
			row.Missing = append(row.Missing, "evidence_manifest", "canonical_verdict", "findings_artifact")
			receipt.Status = "partial"
		} else if manifest.RunID != row.RunID || manifest.Scenario != name {
			row.Missing = append(row.Missing, "evidence_manifest_identity_match", "canonical_verdict", "findings_artifact")
			receipt.Status = "partial"
		} else {
			row.Findings = &manifest.Findings
			phaseFound := false
			for _, phase := range manifest.Phases {
				if phase.Name != "experience" {
					continue
				}
				phaseFound = true
				row.Verdict = normalizeCohortVerdict(phase.Status)
				if phase.Findings != nil {
					row.Evidence = appendCohortArtifact(row.Evidence, *phase.Findings)
				}
				if row.Verdict == "" {
					row.Missing = append(row.Missing, "canonical_experience_verdict")
				}
				break
			}
			if !phaseFound {
				row.Missing = append(row.Missing, "experience_phase_evidence")
			}
			for _, ref := range manifest.Artifacts {
				if ref.Phase == "experience" && ref.Kind != "experience_cohort_receipt" {
					row.Evidence = appendCohortArtifact(row.Evidence, ref)
				}
			}
		}
		if row.Verdict != "passed" {
			receipt.Status = "failed"
		}
		receipt.Scenarios = append(receipt.Scenarios, row)
	}
	if input.Cancelled {
		receipt.Status = "cancelled"
	}
	storeRoot, err := s.artifactRoot("test-genie")
	if err != nil {
		return StoredExperienceCohortReceipt{}, fmt.Errorf("resolve Test Genie cohort evidence root: %w", err)
	}
	receipt.ReceiptID, err = experienceCohortReceiptID(inputDigest, receipt.Scenarios)
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	payload, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	writer, err := executionevidence.NewWriter(storeRoot)
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	relative := filepath.ToSlash(filepath.Join("cohort-receipts", receipt.ReceiptID+".json"))
	filePath := filepath.Join(storeRoot, filepath.FromSlash(relative))
	var artifact executionevidence.ArtifactRef
	if existing, readErr := os.ReadFile(filePath); readErr == nil {
		var prior ExperienceCohortReceipt
		if err := json.Unmarshal(existing, &prior); err != nil || prior.ReceiptID != receipt.ReceiptID || prior.InputDigest != inputDigest {
			return StoredExperienceCohortReceipt{}, executionevidence.ErrArtifactConflict
		}
		priorID, identityErr := experienceCohortReceiptID(prior.InputDigest, prior.Scenarios)
		if identityErr != nil || priorID != prior.ReceiptID {
			return StoredExperienceCohortReceipt{}, executionevidence.ErrArtifactConflict
		}
		receipt = prior
		artifact, err = writer.ReferenceExisting(receipt.ReceiptID, "experience_cohort_receipt", relative, "application/json", "experience")
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return StoredExperienceCohortReceipt{}, readErr
	} else {
		artifact, err = writer.StreamArtifact(ctx, receipt.ReceiptID, "experience_cohort_receipt", relative, "application/json", "experience", strings.NewReader(string(payload)), cohortReceiptMaxBytes)
	}
	if errors.Is(err, executionevidence.ErrArtifactConflict) {
		existing, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return StoredExperienceCohortReceipt{}, errors.Join(err, readErr)
		}
		var prior ExperienceCohortReceipt
		if decodeErr := json.Unmarshal(existing, &prior); decodeErr != nil || prior.ReceiptID != receipt.ReceiptID || prior.InputDigest != inputDigest {
			return StoredExperienceCohortReceipt{}, executionevidence.ErrArtifactConflict
		}
		priorID, identityErr := experienceCohortReceiptID(prior.InputDigest, prior.Scenarios)
		if identityErr != nil || priorID != prior.ReceiptID {
			return StoredExperienceCohortReceipt{}, executionevidence.ErrArtifactConflict
		}
		receipt = prior
		artifact, err = writer.ReferenceExisting(receipt.ReceiptID, "experience_cohort_receipt", relative, "application/json", "experience")
	}
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	return StoredExperienceCohortReceipt{Receipt: receipt, Artifact: artifact, ReadURL: "/api/v1/experience-cohort-receipts/" + receipt.ReceiptID}, nil
}

// GetExperienceCohortReceipt retrieves one immutable aggregate receipt by its
// content-derived ID from Test Genie's existing execution-evidence root.
func (s *Service) GetExperienceCohortReceipt(ctx context.Context, receiptID string) (StoredExperienceCohortReceipt, error) {
	if s == nil || s.artifactRoot == nil {
		return StoredExperienceCohortReceipt{}, errors.New("run evidence store is unavailable")
	}
	if !validCohortReceiptID(receiptID) {
		return StoredExperienceCohortReceipt{}, errors.New("invalid experience cohort receipt ID")
	}
	root, err := s.artifactRoot("test-genie")
	if err != nil {
		return StoredExperienceCohortReceipt{}, fmt.Errorf("resolve Test Genie cohort evidence root: %w", err)
	}
	relative := filepath.ToSlash(filepath.Join("cohort-receipts", receiptID+".json"))
	path := filepath.Join(root, filepath.FromSlash(relative))
	ctxReader, err := os.Open(path)
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	defer ctxReader.Close()
	payload, err := io.ReadAll(io.LimitReader(ctxReader, cohortReceiptMaxBytes+1))
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	if int64(len(payload)) > cohortReceiptMaxBytes {
		return StoredExperienceCohortReceipt{}, executionevidence.ErrArtifactTooLarge
	}
	if err := ctx.Err(); err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	var receipt ExperienceCohortReceipt
	if err := json.Unmarshal(payload, &receipt); err != nil || receipt.ReceiptID != receiptID || receipt.SchemaVersion != 1 {
		return StoredExperienceCohortReceipt{}, executionevidence.ErrCorruptEvidence
	}
	contentID, err := experienceCohortReceiptID(receipt.InputDigest, receipt.Scenarios)
	if err != nil || contentID != receiptID {
		return StoredExperienceCohortReceipt{}, executionevidence.ErrCorruptEvidence
	}
	writer, err := executionevidence.NewWriter(root)
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	artifact, err := writer.ReferenceExisting(receiptID, "experience_cohort_receipt", relative, "application/json", "experience")
	if err != nil {
		return StoredExperienceCohortReceipt{}, err
	}
	return StoredExperienceCohortReceipt{Receipt: receipt, Artifact: artifact, ReadURL: "/api/v1/experience-cohort-receipts/" + receiptID}, nil
}

func normalizeCohortVerdict(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "pass":
		return "passed"
	case "failed", "fail":
		return "failed"
	case "not_evaluated", "not-evaluated", "not evaluated":
		return "not_evaluated"
	case "aborted", "cancelled", "canceled":
		return "cancelled"
	default:
		return ""
	}
}

func appendCohortArtifact(existing []executionevidence.ArtifactRef, candidate executionevidence.ArtifactRef) []executionevidence.ArtifactRef {
	for _, ref := range existing {
		if ref.ID == candidate.ID && ref.SHA256 == candidate.SHA256 {
			return existing
		}
	}
	return append(existing, candidate)
}

func appendCohortMissing(existing []string, missing string) []string {
	for _, item := range existing {
		if item == missing {
			return existing
		}
	}
	return append(existing, missing)
}

func validCohortBuildIdentity(value string) bool {
	if value == "" || len(value) > 256 || value != strings.TrimSpace(value) {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:+-", r) {
			continue
		}
		return false
	}
	return true
}

func validCohortReceiptID(value string) bool {
	const prefix = "experience-cohort-"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, prefix))
	return err == nil
}

func experienceCohortReceiptID(inputDigest string, rows []ExperienceCohortRunEvidence) (string, error) {
	identityBytes, err := json.Marshal(struct {
		InputDigest string                        `json:"inputDigest"`
		Runs        []ExperienceCohortRunEvidence `json:"runs"`
	}{InputDigest: inputDigest, Runs: rows})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(identityBytes)
	return "experience-cohort-" + hex.EncodeToString(digest[:]), nil
}
