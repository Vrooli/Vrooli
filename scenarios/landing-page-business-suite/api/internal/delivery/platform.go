package delivery

import (
	"fmt"
	"strings"

	"github.com/vrooli/api-core/targetmodel"
)

// NormalizeCatalogPlatform maps a producer-supplied desktop target identifier
// (for example "linux-x64" or "darwin-arm64") onto the operating-system
// platform and architecture the download catalog and electron update feed use.
//
// The catalog is operating-system scoped: `download_assets.platform` only
// accepts windows, mac, and linux, and the update feed is addressed by
// latest.yml, latest-mac.yml, and latest-linux.yml. Producers such as
// scenario-to-desktop reason in architecture-qualified target identifiers, so
// the owner normalizes at its boundary instead of requiring every caller to
// speak two vocabularies. The architecture is retained in artifact metadata so
// "which machine" remains answerable.
func NormalizeCatalogPlatform(platform string) (osPlatform, architecture string, err error) {
	trimmed := strings.TrimSpace(platform)
	if trimmed == "" {
		return "", "", fmt.Errorf("platform is required")
	}
	osPlatform, architecture, err = targetmodel.ProjectDesktopPlatform(trimmed)
	if err != nil {
		return "", "", err
	}
	return osPlatform, architecture, nil
}

// lookupCatalogPlatform normalizes a platform used for a catalog read. An empty
// value stays empty so optional readers keep their existing behavior.
func lookupCatalogPlatform(platform string) (string, error) {
	if strings.TrimSpace(platform) == "" {
		return "", nil
	}
	osPlatform, _, err := NormalizeCatalogPlatform(platform)
	return osPlatform, err
}

// catalogArtifactMetadata returns artifact metadata with the canonical catalog
// platform, the resolved architecture, and the original producer target id.
// It never discards producer metadata: release identity fields are preserved.
func catalogArtifactMetadata(metadata map[string]interface{}, targetID, osPlatform, architecture string) map[string]interface{} {
	result := make(map[string]interface{}, len(metadata)+3)
	for key, value := range metadata {
		result[key] = value
	}
	result["platform"] = osPlatform
	if strings.TrimSpace(targetID) != "" {
		result["target_id"] = strings.TrimSpace(targetID)
	}
	if architecture != "" {
		result["architecture"] = architecture
	} else {
		delete(result, "architecture")
	}
	return result
}
