package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"storage-manager/internal/providers"

	corestorage "github.com/vrooli/api-core/storage"
)

// storageInventoryResponse extends the owner declaration inventory with
// service-owned inventories. The core owner schema remains stable while the
// model rows provide the identity needed for attribution.
type storageInventoryResponse struct {
	corestorage.OwnerInventory
	OllamaModels *OllamaStorageInventory `json:"ollama_models,omitempty"`
}

// OllamaStorageInventory is a logical model attribution plus a logical file
// length cross-check. Model sizes and digests come only from /api/tags; shared
// Ollama blobs are never guessed into individual model paths. Allocated
// physical bytes are unknown without an owner measurement of allocated blocks.
type OllamaStorageInventory struct {
	Source                             string               `json:"source"`
	ModelRoot                          string               `json:"model_root"`
	Models                             []OllamaStorageModel `json:"models"`
	TotalModelBytes                    int64                `json:"total_model_bytes"`
	TotalModelBytesKnown               bool                 `json:"total_model_bytes_known"`
	LogicalFileLengthBytes             int64                `json:"logical_file_length_bytes"`
	LogicalFileLengthBytesKnown        bool                 `json:"logical_file_length_bytes_known"`
	UnattributedLogicalFileLengthBytes int64                `json:"unattributed_logical_file_length_bytes"`
	UnattributedLogicalFileLengthKnown bool                 `json:"unattributed_logical_file_length_known"`
	PhysicalBytesKnown                 bool                 `json:"physical_bytes_known"`
	PhysicalBytes                      *int64               `json:"physical_bytes"`     // nil until an owner measurement exists
	UnattributedBytes                  *int64               `json:"unattributed_bytes"` // nil until allocated bytes can be attributed
	UnattributedPaths                  []string             `json:"unattributed_paths,omitempty"`
	AccountingNote                     string               `json:"accounting_note,omitempty"`
	InventoryError                     string               `json:"inventory_error,omitempty"`
	Roles                              []OllamaStorageRole  `json:"roles,omitempty"`
	RoleAccountingKnown                bool                 `json:"role_accounting_known"`
	RoleAccountingNote                 string               `json:"role_accounting_note,omitempty"`
	PolicyReachabilityKnown            bool                 `json:"policy_reachability_known"`
}

type OllamaStorageModel struct {
	Name                     string   `json:"name"`
	Digest                   string   `json:"digest,omitempty"`
	Size                     int64    `json:"size"`
	PolicyReachable          bool     `json:"policy_reachable"`
	InstallProvenanceStatus  string   `json:"install_provenance_status"`
	ProvenanceRegistry       string   `json:"provenance_registry,omitempty"`
	ProvenanceRepository     string   `json:"provenance_repository,omitempty"`
	ProvenanceTag            string   `json:"provenance_tag,omitempty"`
	ProvenanceManifestDigest string   `json:"provenance_manifest_digest,omitempty"`
	ProvenanceRecordedAt     string   `json:"provenance_recorded_at,omitempty"`
	Regenerable              bool     `json:"regenerable"`
	RegenerableReason        string   `json:"regenerable_reason"`
	Roles                    []string `json:"roles,omitempty"`
	PolicyReachabilityKnown  bool     `json:"policy_reachability_known"`
}

// OllamaStorageRole reports logical tag bytes reachable from one configured
// inference role. A tag referenced by multiple roles is counted once in each
// role, so role totals are attribution views and must not be summed as unique
// storage or physical bytes.
type OllamaStorageRole struct {
	Name                   string   `json:"name"`
	PrimaryModel           string   `json:"primary_model,omitempty"`
	FallbackModels         []string `json:"fallback_models,omitempty"`
	Models                 []string `json:"models"`
	MissingModels          []string `json:"missing_models,omitempty"`
	AmbiguousModels        []string `json:"ambiguous_models,omitempty"`
	LogicalModelBytes      int64    `json:"logical_model_bytes"`
	LogicalModelBytesKnown bool     `json:"logical_model_bytes_known"`
}

type ollamaInstallProvenance struct {
	Name           string    `json:"name"`
	Digest         string    `json:"digest"`
	Registry       string    `json:"registry"`
	Repository     string    `json:"repository"`
	Tag            string    `json:"tag"`
	ManifestDigest string    `json:"manifest_digest"`
	VerifiedAt     time.Time `json:"verified_at"`
}

type ollamaInstallProvenanceDocument struct {
	SchemaVersion int                       `json:"schema_version"`
	Models        []ollamaInstallProvenance `json:"models"`
}

// NewOllamaInventoryFromEnvironment wires the service boundary using the
// same environment contract as the Ollama resource.
func NewOllamaInventoryFromEnvironment() providers.OllamaModelInventory {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL")), "/")
	if base == "" {
		host := strings.TrimSpace(os.Getenv("OLLAMA_HOST"))
		if host == "" {
			host = "127.0.0.1"
		}
		port := strings.TrimSpace(os.Getenv("OLLAMA_PORT"))
		if port == "" {
			port = "11434"
		}
		if strings.Contains(host, ":") {
			base = "http://" + host
		} else {
			base = "http://" + host + ":" + port
		}
	}
	return providers.NewHTTPOllamaModelInventory(base)
}

// DefaultOllamaModelRoot follows the portable resource declaration and keeps
// an explicit OLLAMA_MODELS relocation override authoritative.
func DefaultOllamaModelRoot() string {
	if root := strings.TrimSpace(os.Getenv("OLLAMA_MODELS")); root != "" {
		return root
	}
	base := strings.TrimSpace(os.Getenv("USER_DATA_DIR"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "vrooli", "resources", "ollama", "models")
}

// buildOllamaStorageInventory intentionally returns a useful report even if
// the optional filesystem root is unavailable. The service inventory remains
// authoritative, while the error makes the missing physical reconciliation
// visible to operators instead of silently claiming closed accounting.
func buildOllamaStorageInventory(ctx context.Context, client providers.OllamaModelInventory, root, policyPath string) OllamaStorageInventory {
	out := OllamaStorageInventory{Source: "ollama /api/tags", ModelRoot: root, Models: []OllamaStorageModel{}}
	provenance, provenanceErr := loadOllamaInstallProvenance(root)
	models, err := client.ListModels(ctx)
	if err != nil {
		out.InventoryError = err.Error()
		return out
	}
	policyRefs, roleErr := buildOllamaRoleAccounting(policyPath, models)
	policyKnown := policyRefs != nil
	modelRoles := map[string][]string{}
	for _, role := range policyRefs {
		for _, name := range role.Models {
			modelRoles[name] = append(modelRoles[name], role.Name)
		}
	}
	out.Roles = policyRefs
	out.RoleAccountingKnown = roleErr == nil
	out.PolicyReachabilityKnown = policyKnown
	if roleErr == nil {
		out.RoleAccountingNote = "role totals count each distinct referenced tag once per role; tags shared by roles are counted in each role, so these logical totals are non-additive and do not represent unique or allocated bytes"
	} else {
		out.InventoryError = joinInventoryError(out.InventoryError, roleErr.Error())
		out.RoleAccountingNote = "role accounting is unknown because the role policy or one or more referenced model sizes could not be verified"
	}
	out.TotalModelBytesKnown = true
	seenModelNames := make(map[string]struct{}, len(models))
	for _, model := range models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		row := OllamaStorageModel{
			Name: name, Digest: model.Digest, Size: model.Size,
			PolicyReachable: len(modelRoles[name]) > 0, Regenerable: false,
			PolicyReachabilityKnown: policyKnown,
			Roles:                   append([]string(nil), modelRoles[name]...),
		}
		if entry, ok := provenance[name]; ok && entry.Digest == model.Digest && entry.ManifestDigest == model.Digest && entry.Registry == "registry.ollama.ai" && !entry.VerifiedAt.IsZero() {
			row.InstallProvenanceStatus = "matched"
			row.ProvenanceRegistry = entry.Registry
			row.ProvenanceRepository = entry.Repository
			row.ProvenanceTag = entry.Tag
			row.ProvenanceManifestDigest = entry.ManifestDigest
			row.ProvenanceRecordedAt = entry.VerifiedAt.UTC().Format(time.RFC3339Nano)
			row.RegenerableReason = "install provenance matches the current model digest, but same-artifact restoration and retention authorization are not proven"
		} else {
			row.InstallProvenanceStatus = "unknown"
			row.RegenerableReason = "installation provenance is absent or does not match the current digest; a same-artifact registry re-pull is unverified"
		}
		out.Models = append(out.Models, row)
		if _, duplicate := seenModelNames[name]; duplicate {
			out.TotalModelBytesKnown = false
			out.InventoryError = joinInventoryError(out.InventoryError, fmt.Sprintf("Ollama /api/tags returned duplicate normalized model name %q; tag-size accounting is unknown", name))
			continue
		}
		seenModelNames[name] = struct{}{}
		total, ok := addNonnegativeBytes(out.TotalModelBytes, model.Size)
		if !ok {
			out.TotalModelBytesKnown = false
			out.InventoryError = joinInventoryError(out.InventoryError, fmt.Sprintf("Ollama tag %q has a negative logical size or the model-tag byte total overflowed", model.Name))
			continue
		}
		out.TotalModelBytes = total
	}
	sort.Slice(out.Models, func(i, j int) bool { return out.Models[i].Name < out.Models[j].Name })
	if provenanceErr != nil {
		out.InventoryError = joinInventoryError(out.InventoryError, provenanceErr.Error())
	}
	if root == "" {
		out.AccountingNote = "model root was not configured; allocated physical bytes are unknown and service-reported model bytes remain logical tag attribution only"
		return out
	}
	fileLengths, walkErr := directoryLogicalFileLengthBytes(root)
	if walkErr != nil {
		out.InventoryError = joinInventoryError(out.InventoryError, walkErr.Error())
		out.AccountingNote = "logical file length is unknown because the model-root walk failed; allocated physical bytes and reclaimable bytes are unknown"
		return out
	}
	out.LogicalFileLengthBytes = fileLengths
	out.LogicalFileLengthBytesKnown = true
	if !out.TotalModelBytesKnown {
		out.AccountingNote = "model-tag logical byte total is unknown because a reported size was negative or the aggregate overflowed; allocated physical bytes and reclaimable bytes are also unknown"
		return out
	}
	out.UnattributedLogicalFileLengthKnown = true
	if fileLengths > out.TotalModelBytes {
		out.UnattributedLogicalFileLengthBytes = fileLengths - out.TotalModelBytes
		out.UnattributedPaths = []string{root}
		out.AccountingNote = "directory walk sums logical file lengths, not allocated blocks; sparse files and hardlinks can make it differ from disk use, so physical allocated bytes and reclaimable bytes remain unknown"
	} else if fileLengths < out.TotalModelBytes {
		out.AccountingNote = "logical tag sizes exceed summed file lengths; shared or sparse Ollama blobs may explain the difference, but physical allocated bytes and reclaimable bytes remain unknown"
	} else {
		out.AccountingNote = "directory walk sums logical file lengths, not allocated blocks; physical allocated bytes and reclaimable bytes remain unknown"
	}
	return out
}

func loadOllamaInstallProvenance(modelRoot string) (map[string]ollamaInstallProvenance, error) {
	entries := map[string]ollamaInstallProvenance{}
	if strings.TrimSpace(modelRoot) == "" {
		return entries, fmt.Errorf("Ollama model root is not configured; install provenance is unavailable")
	}
	path := filepath.Join(filepath.Dir(modelRoot), "ollama-install-provenance.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return entries, nil
	}
	if err != nil {
		return entries, fmt.Errorf("inspect Ollama install provenance %q: %w", path, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return entries, fmt.Errorf("Ollama install provenance %q is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return entries, fmt.Errorf("read Ollama install provenance %q: %w", path, err)
	}
	var doc ollamaInstallProvenanceDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return entries, fmt.Errorf("decode Ollama install provenance %q: %w", path, err)
	}
	if doc.SchemaVersion != 1 {
		return entries, fmt.Errorf("unsupported Ollama install provenance schema version %d", doc.SchemaVersion)
	}
	for _, entry := range doc.Models {
		if strings.TrimSpace(entry.Name) == "" {
			continue
		}
		entries[entry.Name] = entry
	}
	return entries, nil
}

func buildOllamaRoleAccounting(path string, models []providers.OllamaModel) ([]OllamaStorageRole, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Ollama model policy: %w", err)
	}
	var policy struct {
		Roles map[string]struct {
			Model     string   `json:"model"`
			Fallbacks []string `json:"fallbacks"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("decode Ollama model policy: %w", err)
	}

	installed := make(map[string]int64, len(models))
	duplicateNames := make(map[string]bool)
	for _, model := range models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if _, exists := installed[name]; exists {
			duplicateNames[name] = true
			continue
		}
		installed[name] = model.Size
	}

	roleNames := make([]string, 0, len(policy.Roles))
	for name := range policy.Roles {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("Ollama model policy contains an empty role name")
		}
		roleNames = append(roleNames, name)
	}
	sort.Strings(roleNames)
	out := make([]OllamaStorageRole, 0, len(roleNames))
	var accountingErr error
	for _, name := range roleNames {
		configured := policy.Roles[name]
		role := OllamaStorageRole{
			Name: name, PrimaryModel: strings.TrimSpace(configured.Model),
			FallbackModels: make([]string, 0, len(configured.Fallbacks)), Models: []string{},
			MissingModels: []string{}, AmbiguousModels: []string{}, LogicalModelBytesKnown: true,
		}
		seen := map[string]bool{}
		if role.PrimaryModel != "" {
			role.Models = append(role.Models, role.PrimaryModel)
			seen[role.PrimaryModel] = true
		}
		for _, rawFallback := range configured.Fallbacks {
			fallback := strings.TrimSpace(rawFallback)
			if fallback == "" {
				continue
			}
			role.FallbackModels = append(role.FallbackModels, fallback)
			if !seen[fallback] {
				role.Models = append(role.Models, fallback)
				seen[fallback] = true
			}
		}
		for _, modelName := range role.Models {
			if duplicateNames[modelName] {
				role.AmbiguousModels = append(role.AmbiguousModels, modelName)
				role.LogicalModelBytesKnown = false
				accountingErr = fmt.Errorf("Ollama role accounting is ambiguous: normalized model name %q has duplicate /api/tags rows and is referenced by role %q", modelName, name)
				continue
			}
			size, ok := installed[modelName]
			if !ok {
				role.MissingModels = append(role.MissingModels, modelName)
				role.LogicalModelBytesKnown = false
				accountingErr = fmt.Errorf("Ollama role accounting is incomplete: configured model %q for role %q is not installed", modelName, name)
				continue
			}
			next, valid := addNonnegativeBytes(role.LogicalModelBytes, size)
			if !valid {
				role.LogicalModelBytesKnown = false
				accountingErr = fmt.Errorf("Ollama role accounting is invalid or overflows for role %q", name)
				continue
			}
			role.LogicalModelBytes = next
		}
		out = append(out, role)
	}
	return out, accountingErr
}

func directoryLogicalFileLengthBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		next, ok := addNonnegativeBytes(total, info.Size())
		if !ok {
			return fmt.Errorf("logical file length is negative or overflows at %q", path)
		}
		total = next
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("sum logical file lengths under Ollama model root %q: %w", root, err)
	}
	return total, nil
}

func addNonnegativeBytes(total, value int64) (int64, bool) {
	if total < 0 || value < 0 {
		return total, false
	}
	max := int64(^uint64(0) >> 1)
	if total > max-value {
		return max, false
	}
	return total + value, true
}

func joinInventoryError(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "; " + next
}
