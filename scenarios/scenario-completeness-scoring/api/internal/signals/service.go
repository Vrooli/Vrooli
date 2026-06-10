package signals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// defaultCategory applies when .vrooli/service.json is absent or declares
// no category.
const defaultCategory = "utility"

// serviceCollector reads .vrooli/service.json for the scenario category.
type serviceCollector struct{}

func (serviceCollector) Name() string { return "service" }

func (serviceCollector) Collect(snap *Snapshot) error {
	path := filepath.Join(snap.Root, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No manifest is normal; the default category stands.
			snap.Category = defaultCategory
			return nil
		}
		return fmt.Errorf("read service.json: %w", err)
	}

	var cfg struct {
		Category string `json:"category"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("decode service.json: %w", err)
	}
	if cfg.Category == "" {
		cfg.Category = defaultCategory
	}
	snap.Category = cfg.Category
	return nil
}
