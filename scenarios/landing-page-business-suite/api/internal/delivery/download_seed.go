package delivery

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// The startup catalog is an owner-specific seed, not a public marketing
// fallback. It contains delivery records only and cannot recreate landing
// sections or a public resolver.
//
//go:embed download_seed.json
var downloadSeedJSON []byte

// DefaultDownloadSeed returns a fresh copy of the exact baseline delivery
// records formerly carried by the landing fallback payload.
func DefaultDownloadSeed() ([]App, error) {
	var apps []App
	if err := json.Unmarshal(downloadSeedJSON, &apps); err != nil {
		return nil, fmt.Errorf("decode embedded delivery seed: %w", err)
	}
	for i := range apps {
		if apps[i].Platforms == nil {
			apps[i].Platforms = []Asset{}
		}
	}
	return apps, nil
}
