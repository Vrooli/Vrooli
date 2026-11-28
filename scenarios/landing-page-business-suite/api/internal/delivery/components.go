package delivery

import (
	"context"
	"encoding/json"
	"fmt"

	component "github.com/vrooli/vrooli/packages/desktop-component"
)

// ComponentAssets uses the existing artifact catalog and request-routed store;
// component metadata has no second database or private download registry.
func (s *CatalogService) ComponentAssets(ctx context.Context, bundle string) ([]Asset, error) {
	rows, err := s.ListAssetsContext(ctx, bundle)
	if err != nil {
		return nil, err
	}
	result := make([]Asset, 0)
	for _, row := range rows {
		if row.Metadata["kind"] == "component" {
			result = append(result, row)
		}
	}
	return result, nil
}

func ComponentEntry(asset Asset) (component.SignedEntry, error) {
	var entry component.SignedEntry
	raw, err := json.Marshal(asset.Metadata["component"])
	if err != nil {
		return entry, err
	}
	if err = json.Unmarshal(raw, &entry); err != nil {
		return entry, err
	}
	if _, err = entry.Entry.CanonicalBytes(); err != nil {
		return entry, err
	}
	if entry.Entry.Version != asset.ReleaseVersion || entry.Entry.SHA256 != asset.Checksum {
		return entry, fmt.Errorf("component catalog release tuple mismatch")
	}
	return entry, nil
}
