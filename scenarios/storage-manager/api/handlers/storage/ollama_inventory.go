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

	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/providers"
)

// storageInventoryResponse extends the owner declaration inventory with
// service-owned inventories. The core owner schema remains stable while the
// model rows provide the identity needed for attribution.
type storageInventoryResponse struct {
	corestorage.OwnerInventory
	OllamaModels *OllamaStorageInventory `json:"ollama_models,omitempty"`
}

// OllamaStorageInventory is a logical model attribution plus a physical
// accounting check. Model sizes and digests come only from /api/tags; shared
// Ollama blobs are never guessed into individual model paths. Any physical
// bytes that cannot be attributed by the service report are called out with a
// concrete root path.
type OllamaStorageInventory struct {
	Source            string               `json:"source"`
	ModelRoot         string               `json:"model_root"`
	Models            []OllamaStorageModel `json:"models"`
	TotalModelBytes   int64                `json:"total_model_bytes"`
	PhysicalBytes     int64                `json:"physical_bytes"`
	UnattributedBytes int64                `json:"unattributed_bytes"`
	UnattributedPaths []string             `json:"unattributed_paths,omitempty"`
	AccountingNote    string               `json:"accounting_note,omitempty"`
	InventoryError    string               `json:"inventory_error,omitempty"`
}

type OllamaStorageModel struct {
	Name              string `json:"name"`
	Digest            string `json:"digest,omitempty"`
	Size              int64  `json:"size"`
	PolicyReachable   bool   `json:"policy_reachable"`
	Regenerable       bool   `json:"regenerable"`
	RegenerableReason string `json:"regenerable_reason"`
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
	policyRefs, policyErr := loadOllamaPolicyReferences(policyPath)
	models, err := client.ListModels(ctx)
	if err != nil {
		out.InventoryError = err.Error()
		return out
	}
	for _, model := range models {
		if strings.TrimSpace(model.Name) == "" {
			continue
		}
		row := OllamaStorageModel{
			Name: model.Name, Digest: model.Digest, Size: model.Size,
			PolicyReachable: policyRefs[model.Name], Regenerable: true,
			RegenerableReason: "model weights can be re-pulled from the Ollama registry",
		}
		out.Models = append(out.Models, row)
		out.TotalModelBytes += model.Size
	}
	sort.Slice(out.Models, func(i, j int) bool { return out.Models[i].Name < out.Models[j].Name })
	if policyErr != nil {
		out.InventoryError = policyErr.Error()
	}
	if root == "" {
		out.AccountingNote = "physical model root was not configured; service-reported model bytes remain logical attribution only"
		return out
	}
	physical, walkErr := directoryBytes(root)
	if walkErr != nil {
		out.InventoryError = joinInventoryError(out.InventoryError, walkErr.Error())
		return out
	}
	out.PhysicalBytes = physical
	if physical > out.TotalModelBytes {
		out.UnattributedBytes = physical - out.TotalModelBytes
		out.UnattributedPaths = []string{root}
		out.AccountingNote = "Ollama reports logical model sizes and stores shared blobs; the physical remainder is deliberately unattributed rather than double-counted"
	} else if physical < out.TotalModelBytes {
		out.AccountingNote = "logical model bytes exceed the physical walk because Ollama model blobs may be shared or sparse; physical bytes are not assigned twice"
	}
	return out
}

func loadOllamaPolicyReferences(path string) (map[string]bool, error) {
	refs := map[string]bool{}
	data, err := os.ReadFile(path)
	if err != nil {
		return refs, fmt.Errorf("read Ollama model policy: %w", err)
	}
	var policy struct {
		Roles map[string]struct {
			Model     string   `json:"model"`
			Fallbacks []string `json:"fallbacks"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		return refs, fmt.Errorf("decode Ollama model policy: %w", err)
	}
	for _, role := range policy.Roles {
		if role.Model != "" {
			refs[role.Model] = true
		}
		for _, fallback := range role.Fallbacks {
			if fallback != "" {
				refs[fallback] = true
			}
		}
	}
	return refs, nil
}

func directoryBytes(root string) (int64, error) {
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
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk Ollama model root %q: %w", root, err)
	}
	return total, nil
}

func joinInventoryError(existing, next string) string {
	if existing == "" {
		return next
	}
	return existing + "; " + next
}
