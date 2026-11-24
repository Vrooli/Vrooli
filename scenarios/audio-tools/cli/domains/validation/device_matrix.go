package validation

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

const defaultDeviceMatrixPath = "coverage/device-qualification-matrix.json"

const (
	matrixPolicyRequiredWhenClaimed                      = "required_when_claimed"
	matrixPolicyConditionalCandidateWhenRuntimeAvailable = "conditional_candidate_when_runtime_available"
)

type deviceMatrixDeclaration struct {
	SchemaVersion int             `json:"schema_version"`
	Boundary      string          `json:"qualification_boundary"`
	Targets       []matrixTarget  `json:"targets"`
	Profiles      []matrixProfile `json:"profiles"`
}

type matrixTarget struct {
	ID                  string `json:"id"`
	Capability          string `json:"capability"`
	Provider            string `json:"provider"`
	ModelID             string `json:"model_id"`
	RuntimeModelID      string `json:"runtime_model_id,omitempty"`
	QualificationPolicy string `json:"qualification_policy"`
}

type matrixProfile struct {
	ID           string `json:"id"`
	Profile      string `json:"profile"`
	OS           string `json:"os"`
	Browser      string `json:"browser"`
	Host         string `json:"host"`
	Accelerator  string `json:"accelerator"`
	CPULimit     int    `json:"cpu_limit,omitempty"`
	MemoryLimitB int64  `json:"memory_limit_bytes,omitempty"`
}

type deviceMatrixCell struct {
	ID              string        `json:"id"`
	DeviceProfile   matrixProfile `json:"device_profile"`
	Target          matrixTarget  `json:"target"`
	ExecutionState  string        `json:"execution_state"`
	SupportState    string        `json:"support_state"`
	Qualification   string        `json:"qualification"`
	Qualified       bool          `json:"product_qualified"`
	MissingEvidence string        `json:"missing_evidence"`
}

// deviceMatrixReceipt is an operator supplied, per-cell observation. A receipt
// can report an unavailable result, but only a passing product-path observation
// with pinned platform, browser, device, model and artifact identity qualifies.
type deviceMatrixReceipt struct {
	CellID       string `json:"cell_id"`
	Outcome      string `json:"outcome"` // passed, unavailable, or failed
	Reason       string `json:"reason,omitempty"`
	ObservedAt   string `json:"observed_at"`
	OSVersion    string `json:"os_version"`
	Architecture string `json:"architecture"`
	CPUCount     int    `json:"cpu_count"`
	MemoryBytes  int64  `json:"memory_total_bytes"`
	Accelerator  string `json:"accelerator_identity"`
	ResourceCPUs int    `json:"resource_cpu_limit"`
	ResourceMem  int64  `json:"resource_memory_limit_bytes"`
	Browser      string `json:"browser"`
	BrowserVer   string `json:"browser_version"`
	Device       string `json:"device_identity"`
	Provider     string `json:"provider"`
	ModelID      string `json:"model_id"`
	EngineVer    string `json:"engine_version"`
	ModelRev     string `json:"model_revision"`
	ProductPath  string `json:"product_path"`
	ProductState string `json:"product_path_status"`
	ArtifactRef  string `json:"artifact_ref"`
	ArtifactHash string `json:"artifact_sha256"`
}

type deviceMatrixReceiptFile struct {
	SchemaVersion int                   `json:"schema_version"`
	Receipts      []deviceMatrixReceipt `json:"receipts"`
}

var matrixSHA256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func (h *handlers) runDeviceMatrix(ctx cliapp.RunContext) error {
	root, err := h.scenarioRoot()
	if err != nil {
		return err
	}
	path := strings.TrimSpace(ctx.Flag("matrix-path"))
	if path == "" {
		path = filepath.Join(root, defaultDeviceMatrixPath)
	}
	declaration, err := loadDeviceMatrix(path)
	if err != nil {
		return err
	}
	cells := enumerateDeviceMatrix(declaration)
	receiptPath := strings.TrimSpace(ctx.Flag("receipt-path"))
	if receiptPath != "" {
		if !filepath.IsAbs(receiptPath) {
			receiptPath = filepath.Join(root, receiptPath)
		}
		data, readErr := os.ReadFile(receiptPath)
		if readErr != nil {
			return fmt.Errorf("read device matrix receipts %q: %w", receiptPath, readErr)
		}
		cells, err = evaluateDeviceMatrix(declaration, data)
		if err != nil {
			return err
		}
	}
	qualified := 0
	for _, cell := range cells {
		if cell.Qualified {
			qualified++
		}
	}
	result := map[string]any{
		"schema_version":         declaration.SchemaVersion,
		"qualification_boundary": declaration.Boundary,
		"matrix_complete":        qualified == len(cells) && len(cells) > 0,
		"qualified_cell_count":   qualified,
		"missing_cells":          cells,
		"receipt_path":           receiptPath,
		"note":                   "unavailable, failed, missing, or non-product-path cells remain unqualified; matrix completeness requires every declared cell to pass",
	}
	if ctx.JSON() {
		return json.NewEncoder(ctx.Stdout()).Encode(result)
	}
	fmt.Fprintf(ctx.Stdout(), "D1-D6 product qualification matrix: %d/%d cells qualified\n", qualified, len(cells))
	for _, cell := range cells {
		fmt.Fprintf(ctx.Stdout(), "%s %s: %s / %s / %s / %s (%s; %s)\n", strings.ToUpper(cell.Qualification), cell.ID, cell.DeviceProfile.Profile, cell.DeviceProfile.OS, cell.DeviceProfile.Browser, cell.Target.ID, cell.ExecutionState, cell.MissingEvidence)
	}
	return nil
}

func evaluateDeviceMatrix(declaration deviceMatrixDeclaration, data []byte) ([]deviceMatrixCell, error) {
	var file deviceMatrixReceiptFile
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode device matrix receipts: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("decode device matrix receipts: expected exactly one JSON document")
	}
	if file.SchemaVersion != 1 {
		return nil, fmt.Errorf("device matrix receipts must use schema_version 1")
	}
	cells := enumerateDeviceMatrix(declaration)
	byID := make(map[string]int, len(cells))
	for i := range cells {
		byID[cells[i].ID] = i
	}
	seen := make(map[string]bool, len(file.Receipts))
	for _, receipt := range file.Receipts {
		index, ok := byID[receipt.CellID]
		if !ok {
			return nil, fmt.Errorf("device matrix receipt references undeclared cell %q", receipt.CellID)
		}
		if seen[receipt.CellID] {
			return nil, fmt.Errorf("duplicate device matrix receipt for cell %q", receipt.CellID)
		}
		seen[receipt.CellID] = true
		cell := &cells[index]
		cell.ExecutionState = "reported"
		cell.SupportState = receipt.Outcome
		cell.MissingEvidence = ""
		switch receipt.Outcome {
		case "unavailable":
			cell.Qualification = "unqualified"
			cell.MissingEvidence = "unavailable: " + nonempty(receipt.Reason, "receipt must explain why this cell is unavailable")
		case "failed":
			cell.Qualification = "unqualified"
			cell.MissingEvidence = "failed: " + nonempty(receipt.Reason, "failed receipts require a reason")
		case "passed":
			if declaration.Boundary != "product_path_receipts" {
				return nil, fmt.Errorf("passed receipt for %q is forbidden by inventory_only qualification boundary", receipt.CellID)
			}
			target, profile := cell.Target, cell.DeviceProfile
			if !concreteMatrixIdentity(receipt.Device) || !concreteMatrixIdentity(receipt.BrowserVer) || !concreteMatrixIdentity(receipt.EngineVer) || !concreteMatrixIdentity(receipt.ModelRev) || !concreteMatrixIdentity(receipt.ArtifactRef) || (!strings.EqualFold(profile.Accelerator, "CPU only") && !concreteMatrixIdentity(receipt.Accelerator)) {
				return nil, fmt.Errorf("passed receipt for %q uses a placeholder device, browser, engine, model, accelerator, or artifact identity", receipt.CellID)
			}
			if receipt.Reason != "" || receipt.ObservedAt == "" || receipt.OSVersion == "" || receipt.Architecture == "" || receipt.CPUCount < 1 || receipt.MemoryBytes < 1 || receipt.Accelerator == "" || receipt.Browser == "" || receipt.BrowserVer == "" || receipt.Device == "" || receipt.Provider == "" || receipt.ModelID == "" || receipt.EngineVer == "" || receipt.ModelRev == "" || receipt.ProductPath == "" || receipt.ProductState != "passed" || receipt.ArtifactRef == "" || receipt.ArtifactHash == "" {
				return nil, fmt.Errorf("passed receipt for %q is missing pinned provenance, product path, or artifact identity", receipt.CellID)
			}
			if _, err := time.Parse(time.RFC3339, receipt.ObservedAt); err != nil {
				return nil, fmt.Errorf("passed receipt for %q has invalid observed_at timestamp", receipt.CellID)
			}
			if !matrixSHA256Pattern.MatchString(receipt.ArtifactHash) {
				return nil, fmt.Errorf("passed receipt for %q must include a 64-character hexadecimal artifact_sha256", receipt.CellID)
			}
			browserMatches := strings.EqualFold(receipt.Browser, profile.Browser) || (strings.Contains(strings.ToLower(profile.Browser), "pwa") && strings.EqualFold(receipt.Browser, "Safari"))
			osName := strings.Fields(profile.OS)[0]
			if receipt.Provider != target.Provider || receipt.ModelID != target.ModelID || !browserMatches || !strings.Contains(strings.ToLower(receipt.OSVersion), strings.ToLower(osName)) || (len(strings.Fields(profile.OS)) > 1 && !strings.EqualFold(receipt.Architecture, strings.Fields(profile.OS)[1])) {
				return nil, fmt.Errorf("passed receipt for %q does not match declared OS/browser/provider/model", receipt.CellID)
			}
			if (profile.CPULimit > 0 && receipt.ResourceCPUs != profile.CPULimit) || (profile.MemoryLimitB > 0 && receipt.ResourceMem != profile.MemoryLimitB) {
				return nil, fmt.Errorf("passed receipt for %q does not match declared CPU/memory limits", receipt.CellID)
			}
			if strings.EqualFold(profile.Accelerator, "CPU only") && !strings.EqualFold(receipt.Accelerator, "none") {
				return nil, fmt.Errorf("passed receipt for %q claims accelerator on a CPU-only profile", receipt.CellID)
			}
			cell.Qualification, cell.Qualified = "qualified", true
		default:
			return nil, fmt.Errorf("receipt for %q has unsupported outcome %q", receipt.CellID, receipt.Outcome)
		}
	}
	return cells, nil
}

// concreteMatrixIdentity prevents inventory placeholders from being mistaken
// for pinned provenance in a product qualification receipt.
func concreteMatrixIdentity(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "unknown", "n/a", "na", "none", "unspecified", "not available", "host dependent", "tbd", "todo":
		return false
	default:
		return true
	}
}

func nonempty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func loadDeviceMatrix(path string) (deviceMatrixDeclaration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return deviceMatrixDeclaration{}, fmt.Errorf("read device matrix declaration %q: %w", path, err)
	}
	var declaration deviceMatrixDeclaration
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return deviceMatrixDeclaration{}, fmt.Errorf("decode device matrix declaration %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return deviceMatrixDeclaration{}, fmt.Errorf("decode device matrix declaration %q: expected exactly one JSON document", path)
	}
	if declaration.SchemaVersion != 1 || (declaration.Boundary != "inventory_only" && declaration.Boundary != "product_path_receipts") || len(declaration.Targets) == 0 || len(declaration.Profiles) == 0 {
		return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q must use schema_version 1, inventory_only or product_path_receipts boundary, and declare targets and profiles", path)
	}
	seenTargets := make(map[string]struct{}, len(declaration.Targets))
	for _, target := range declaration.Targets {
		if target.ID == "" || target.Capability == "" || target.Provider == "" || target.ModelID == "" || target.QualificationPolicy == "" {
			return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q contains an incomplete target", path)
		}
		if target.QualificationPolicy != matrixPolicyRequiredWhenClaimed && target.QualificationPolicy != matrixPolicyConditionalCandidateWhenRuntimeAvailable {
			return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q contains unsupported qualification policy %q", path, target.QualificationPolicy)
		}
		if _, exists := seenTargets[target.ID]; exists {
			return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q contains duplicate target id %q", path, target.ID)
		}
		seenTargets[target.ID] = struct{}{}
	}
	seenProfiles := make(map[string]struct{}, len(declaration.Profiles))
	for _, profile := range declaration.Profiles {
		if profile.ID == "" || profile.Profile == "" || profile.OS == "" || profile.Browser == "" {
			return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q contains an incomplete device profile", path)
		}
		if _, exists := seenProfiles[profile.ID]; exists {
			return deviceMatrixDeclaration{}, fmt.Errorf("device matrix declaration %q contains duplicate device profile id %q", path, profile.ID)
		}
		seenProfiles[profile.ID] = struct{}{}
	}
	return declaration, nil
}

func enumerateDeviceMatrix(declaration deviceMatrixDeclaration) []deviceMatrixCell {
	cells := make([]deviceMatrixCell, 0, len(declaration.Targets)*len(declaration.Profiles))
	for _, profile := range declaration.Profiles {
		for _, target := range declaration.Targets {
			missing := "native device/browser product-path receipt with pinned versions and passing inference/synthesis"
			if target.QualificationPolicy == matrixPolicyConditionalCandidateWhenRuntimeAvailable {
				missing = "runtime availability decision plus native device/browser product-path receipt with pinned versions and passing synthesis"
			}
			cells = append(cells, deviceMatrixCell{
				ID:              profile.ID + "/" + target.ID,
				DeviceProfile:   profile,
				Target:          target,
				ExecutionState:  "not_run",
				SupportState:    "unverified",
				Qualification:   "unqualified",
				Qualified:       false,
				MissingEvidence: missing,
			})
		}
	}
	return cells
}
