package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// catalogAssetMetadata is the subset of the published catalog document used
// to fill optional component-manifest fields.
type catalogAssetMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slot        string `json:"slot"`
	Domain      string `json:"domain"`
}

func catalogAssetMetadataFor(libraryRoot, catalogID string) (*catalogAssetMetadata, error) {
	if strings.TrimSpace(catalogID) == "" {
		return nil, nil
	}
	root := repositoryRootFromLibrary(libraryRoot)
	if root == "" {
		return nil, nil
	}
	paths, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range paths {
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		var doc struct {
			Asset catalogAssetMetadata `json:"asset"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parse catalog asset %s: %w", path, err)
		}
		if doc.Asset.ID == catalogID {
			return &doc.Asset, nil
		}
	}
	return nil, nil
}

func catalogComponentSlot(slot string) string {
	switch strings.ToLower(strings.TrimSpace(slot)) {
	case "ui-primitive", "surface", "control":
		return string(ComponentSlotUIPrimitive)
	case "ui-pattern", "pattern", "page-template", "layout-shell":
		return string(ComponentSlotUIPattern)
	case "layout-nav", "navigation":
		return string(ComponentSlotLayoutNav)
	default:
		return ""
	}
}
