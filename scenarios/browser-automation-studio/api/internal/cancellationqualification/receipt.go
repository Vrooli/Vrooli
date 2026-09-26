package cancellationqualification

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ReceiptSchemaVersion = 1
	ContractRow          = "cancellation-recovery"
	EvidenceGlob         = ".vrooli/runtime/rehabilitation-evidence/cancellation-recovery-*.json"
	ObservationsEnv      = "BAS_J07_OBSERVATIONS"
)

var RequiredSourceFiles = []string{
	"api/automation/executor/session_lifecycle.go",
	"api/automation/executor/session_lifecycle_test.go",
	"api/services/workflow/executions.go",
	"api/services/workflow/execution_results_test.go",
	"api/services/recovery/service.go",
	"api/services/recovery/service_test.go",
	"api/handlers/executions/service.go",
	"api/handlers/executions/service_test.go",
	"api/cmd/cancellation-qualification/main.go",
	"api/cmd/cancellation-qualification/main_test.go",
	"api/cmd/cancellation-qualification/qualification.mjs",
	"api/internal/cancellationqualification/receipt.go",
	"api/internal/cancellationqualification/receipt_test.go",
	"api/cmd/cancellation-restart-cohort/qualification.mjs",
	"playwright-driver/src/session/manager.ts",
	"playwright-driver/src/routes/session-run.ts",
	"playwright-driver/src/routes/session-close.ts",
	"playwright-driver/tests/integration/typed-action-semantics.test.ts",
	"playwright-driver/tests/unit/idempotency/session-idempotency.test.ts",
	"playwright-driver/tests/unit/routes/session-run.test.ts",
}

var requiredCases = []string{"cancellation", "timeout", "driverDeath", "apiRestart", "retriedStart"}

// Receipt is the independent fixture's aggregate observation for BAS-RH-J07.
type Receipt struct {
	SchemaVersion int               `json:"schemaVersion"`
	ContractRow   string            `json:"contractRow"`
	ContractSHA   string            `json:"contractSha256"`
	SourceFiles   map[string]string `json:"sourceFiles"`
	Runtime       struct {
		BuildIdentity string `json:"buildIdentity"`
	} `json:"runtime"`
	Cases map[string]CaseObservation `json:"cases"`
}

// CaseObservation is written only after the named test has asserted each field.
type CaseObservation struct {
	OwnerTest                string  `json:"ownerTest"`
	Observed                 bool    `json:"observed"`
	Passed                   bool    `json:"passed"`
	ExternalEffects          int     `json:"externalEffects"`
	TerminalStatus           string  `json:"terminalStatus"`
	LiveResourcesBeforeClose int     `json:"liveResourcesBeforeClose"`
	LiveResourcesAfterClose  int     `json:"liveResourcesAfterClose"`
	InputStoppedMS           float64 `json:"inputStoppedMs"`
	CleanupMS                float64 `json:"cleanupMs"`
	RecoveryMS               float64 `json:"recoveryMs"`
	UncertainEffect          bool    `json:"uncertainEffect"`
	RetryAdmitted            bool    `json:"retryAdmitted"`
}

// ObservationRecord is emitted by an owner test only after its assertions pass.
// The qualification command consumes these records; it does not infer values
// from test names or exit status.
type ObservationRecord struct {
	Case        string          `json:"case"`
	Observation CaseObservation `json:"observation"`
}

// RecordObservation appends one validated owner-test observation when the
// qualification command provides an output path. Ordinary test runs are
// side-effect free.
func RecordObservation(ownerTest, caseName string, observation CaseObservation) error {
	path := strings.TrimSpace(os.Getenv(ObservationsEnv))
	if path == "" {
		return nil
	}
	observation.OwnerTest = strings.TrimSpace(ownerTest)
	if observation.OwnerTest == "" {
		return fmt.Errorf("owner test name is required for %s observation", caseName)
	}
	if err := validateObservation(caseName, observation); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open J07 observation file: %w", err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(ObservationRecord{Case: caseName, Observation: observation}); err != nil {
		return fmt.Errorf("write J07 observation: %w", err)
	}
	return nil
}

// NewReceipt binds a new receipt to the exact contract, owner sources and
// deployed API candidate observed by the focused qualification test.
func NewReceipt(scenarioRoot, buildIdentity string) (Receipt, error) {
	contractPath := filepath.Join(scenarioRoot, "docs/internal/REFRACTOR_CONTRACT.json")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		return Receipt{}, fmt.Errorf("read rehabilitation contract: %w", err)
	}
	contractSum := sha256.Sum256(contract)
	receipt := Receipt{
		SchemaVersion: ReceiptSchemaVersion,
		ContractRow:   ContractRow,
		ContractSHA:   hex.EncodeToString(contractSum[:]),
		SourceFiles:   make(map[string]string, len(RequiredSourceFiles)),
		Runtime: struct {
			BuildIdentity string `json:"buildIdentity"`
		}{BuildIdentity: buildIdentity},
		Cases: make(map[string]CaseObservation, len(requiredCases)),
	}
	for _, rel := range RequiredSourceFiles {
		path := filepath.Join(scenarioRoot, filepath.FromSlash(rel))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return Receipt{}, fmt.Errorf("read owner source %s: %w", rel, readErr)
		}
		sum := sha256.Sum256(data)
		receipt.SourceFiles[rel] = hex.EncodeToString(sum[:])
	}
	return receipt, nil
}

// AssembleReceipt joins one owner-emitted observation for each required J07
// case to the current contract, sources, and managed build identity.
func AssembleReceipt(scenarioRoot, buildIdentity string, records []ObservationRecord) (Receipt, error) {
	receipt, err := NewReceipt(scenarioRoot, buildIdentity)
	if err != nil {
		return Receipt{}, err
	}
	required := make(map[string]bool, len(requiredCases))
	for _, name := range requiredCases {
		required[name] = true
	}
	for _, record := range records {
		if !required[record.Case] {
			return Receipt{}, fmt.Errorf("unexpected cancellation observation for case %s", record.Case)
		}
		if _, ok := receipt.Cases[record.Case]; ok {
			return Receipt{}, fmt.Errorf("duplicate cancellation observation for case %s", record.Case)
		}
		if err := validateObservation(record.Case, record.Observation); err != nil {
			return Receipt{}, err
		}
		receipt.Cases[record.Case] = record.Observation
	}
	if err := Validate(scenarioRoot, receipt, buildIdentity); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

// Validate rejects stale identity, missing cases and observations outside the
// published cancellation/recovery band.
func Validate(scenarioRoot string, receipt Receipt, liveBuild string) error {
	if receipt.SchemaVersion != ReceiptSchemaVersion || receipt.ContractRow != ContractRow {
		return fmt.Errorf("unsupported cancellation receipt identity")
	}
	if liveBuild == "" || receipt.Runtime.BuildIdentity != liveBuild {
		return fmt.Errorf("managed build identity does not match cancellation receipt")
	}
	contractPath := filepath.Join(scenarioRoot, "docs/internal/REFRACTOR_CONTRACT.json")
	contract, err := os.ReadFile(contractPath)
	if err != nil {
		return fmt.Errorf("read rehabilitation contract: %w", err)
	}
	contractSum := sha256.Sum256(contract)
	if receipt.ContractSHA != hex.EncodeToString(contractSum[:]) {
		return fmt.Errorf("contract digest does not match cancellation receipt")
	}
	for _, rel := range RequiredSourceFiles {
		want, ok := receipt.SourceFiles[rel]
		if !ok || !isSHA256(want) {
			return fmt.Errorf("cancellation receipt lacks source digest: %s", rel)
		}
		path := filepath.Join(scenarioRoot, filepath.FromSlash(rel))
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read owner source %s: %w", rel, readErr)
		}
		sum := sha256.Sum256(data)
		if want != hex.EncodeToString(sum[:]) {
			return fmt.Errorf("owner source digest mismatch: %s", rel)
		}
	}
	for _, name := range requiredCases {
		observation, ok := receipt.Cases[name]
		if !ok {
			return fmt.Errorf("cancellation receipt lacks case %s", name)
		}
		if strings.TrimSpace(observation.OwnerTest) == "" {
			return fmt.Errorf("cancellation receipt lacks owner test for case %s", name)
		}
		if err := validateObservation(name, observation); err != nil {
			return err
		}
	}
	return nil
}

// LatestCurrentReceipt loads the newest retained receipt that matches the
// current contract, source tree and live build. A stale newer file does not hide
// an older current one; every candidate is validated independently.
func LatestCurrentReceipt(scenarioRoot, liveBuild string) (Receipt, string, error) {
	pattern := filepath.Join(scenarioRoot, filepath.FromSlash(EvidenceGlob))
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return Receipt{}, "", err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var failures []error
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, readErr))
			continue
		}
		var receipt Receipt
		if unmarshalErr := json.Unmarshal(data, &receipt); unmarshalErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, unmarshalErr))
			continue
		}
		if validateErr := Validate(scenarioRoot, receipt, liveBuild); validateErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", path, validateErr))
			continue
		}
		return receipt, path, nil
	}
	if len(failures) == 0 {
		return Receipt{}, "", fmt.Errorf("no retained cancellation receipt for build %q", liveBuild)
	}
	return Receipt{}, "", fmt.Errorf("no current cancellation receipt for build %q: %v", liveBuild, failures[0])
}

func validateObservation(name string, observation CaseObservation) error {
	if !observation.Observed || !observation.Passed {
		return fmt.Errorf("cancellation case %s was not observed passing", name)
	}
	if observation.ExternalEffects != 1 || observation.TerminalStatus == "" ||
		observation.LiveResourcesBeforeClose < 1 || observation.LiveResourcesAfterClose != 0 {
		return fmt.Errorf("cancellation case %s lacks one-effect terminal cleanup evidence", name)
	}
	switch name {
	case "cancellation":
		if observation.TerminalStatus != "cancelled" {
			return fmt.Errorf("cancellation case ended %q, want cancelled", observation.TerminalStatus)
		}
	case "timeout":
		if observation.TerminalStatus != "failed" && observation.TerminalStatus != "cancelled" {
			return fmt.Errorf("timeout case ended %q, want failed or cancelled", observation.TerminalStatus)
		}
	default:
		if observation.TerminalStatus != "failed" {
			return fmt.Errorf("cancellation case %s ended %q, want failed", name, observation.TerminalStatus)
		}
	}
	if observation.CleanupMS < 0 || observation.CleanupMS > 5000 || observation.RecoveryMS < 0 || observation.RecoveryMS > 10000 {
		return fmt.Errorf("cancellation case %s exceeds cleanup or recovery band", name)
	}
	if (name == "cancellation" || name == "timeout") && (observation.InputStoppedMS < 0 || observation.InputStoppedMS > 1000) {
		return fmt.Errorf("cancellation case %s exceeds input-stop band", name)
	}
	if !observation.UncertainEffect || observation.RetryAdmitted {
		return fmt.Errorf("cancellation case %s replayed or failed to retain an uncertain effect", name)
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
