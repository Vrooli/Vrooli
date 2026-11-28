package wordpress

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const imageSetSchemaVersion = 1

var digestReference = regexp.MustCompile(`^[^\s@]+@sha256:[0-9a-f]{64}$`)

type imageSetFile struct {
	SchemaVersion int                   `json:"schemaVersion"`
	Images        map[string]ImageEntry `json:"images"`
}

// ImageEntry pins one owner-controlled OCI image and the local architectures
// for which the manifest was checked.
type ImageEntry struct {
	Version   string   `json:"version"`
	Reference string   `json:"reference"`
	Platforms []string `json:"platforms"`
}

// ImageSet is the validated immutable profile used by acquisition and start.
type ImageSet struct {
	Site     ImageEntry
	Database ImageEntry
	CLI      ImageEntry
}

// LoadImageSet reads only the closed F11 image profile. Callers never select
// image references, registries, commands, or platforms at runtime.
func LoadImageSet(path string) (ImageSet, error) {
	file, err := os.Open(path)
	if err != nil {
		return ImageSet{}, fmt.Errorf("open WordPress image profile: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, 64<<10))
	decoder.DisallowUnknownFields()
	var body imageSetFile
	if err := decoder.Decode(&body); err != nil {
		return ImageSet{}, fmt.Errorf("decode WordPress image profile: %w", err)
	}
	if body.SchemaVersion != imageSetSchemaVersion {
		return ImageSet{}, fmt.Errorf("unsupported WordPress image profile schema %d", body.SchemaVersion)
	}
	if len(body.Images) != 3 {
		return ImageSet{}, fmt.Errorf("WordPress image profile must declare exactly site, database, and cli images")
	}
	set := ImageSet{Site: body.Images["site"], Database: body.Images["database"], CLI: body.Images["cli"]}
	for role, entry := range map[string]ImageEntry{"site": set.Site, "database": set.Database, "cli": set.CLI} {
		if err := validateImageEntry(role, entry); err != nil {
			return ImageSet{}, err
		}
	}
	return set, nil
}

func validateImageEntry(role string, entry ImageEntry) error {
	if strings.TrimSpace(entry.Version) == "" || !digestReference.MatchString(entry.Reference) {
		return fmt.Errorf("WordPress %s image must have a version and an immutable sha256 reference", role)
	}
	if len(entry.Platforms) == 0 {
		return fmt.Errorf("WordPress %s image must declare supported platforms", role)
	}
	seen := map[string]bool{}
	for _, platform := range entry.Platforms {
		platform = strings.TrimSpace(platform)
		if platform != "linux/amd64" && platform != "linux/arm64/v8" {
			return fmt.Errorf("WordPress %s image declares unsupported platform %q", role, platform)
		}
		if seen[platform] {
			return fmt.Errorf("WordPress %s image repeats platform %q", role, platform)
		}
		seen[platform] = true
	}
	return nil
}

func (set ImageSet) entries() []ImageEntry {
	return []ImageEntry{set.Site, set.Database, set.CLI}
}
