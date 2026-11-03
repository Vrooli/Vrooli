package journeys

import (
	"fmt"
	"strconv"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func scenarioAppInstallMode(planID string) string {
	switch strings.TrimSpace(planID) {
	case TemporaryAppTestPlanID:
		return "temporary"
	case DurableAppPlanID:
		return "durable"
	default:
		return ""
	}
}

func validateScenarioInstallArtifact(artifact deliveryramp.Artifact) (string, error) {
	metadata := artifact.Metadata
	if metadata == nil {
		return "", fmt.Errorf("scenario app install requires owner-produced artifact metadata")
	}
	refParts := strings.Split(strings.TrimSpace(artifact.ImmutableRef), ":")
	checksum := strings.TrimPrefix(strings.TrimSpace(artifact.Checksum), "sha256:")
	if len(refParts) != 2 || (refParts[0] != "android-debug" && refParts[0] != "android-release") || !isLowerHex(refParts[1], 64) || !isLowerHex(checksum, 64) || refParts[1] != checksum {
		return "", fmt.Errorf("scenario app install requires an immutable Android ref matching the APK/AAB digest")
	}
	packageName := strings.TrimSpace(metadata["package_name"])
	if !validAndroidPackageName(packageName) {
		return "", fmt.Errorf("scenario app install requires a fully-qualified package identity")
	}
	if strings.TrimSpace(metadata["version_name"]) == "" {
		return "", fmt.Errorf("scenario app install requires the artifact version name")
	}
	versionCode, err := strconv.ParseInt(strings.TrimSpace(metadata["version_code"]), 10, 64)
	if err != nil || versionCode <= 0 {
		return "", fmt.Errorf("scenario app install requires a positive artifact version code")
	}
	signing := strings.TrimSpace(metadata["signing"])
	if signing == "" || (refParts[0] == "android-release" && strings.TrimSpace(metadata["signing_identity"]) == "") {
		return "", fmt.Errorf("scenario app install requires an explicit artifact signing identity")
	}
	return packageName, nil
}

func isLowerHex(value string, size int) bool {
	if len(value) != size {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func validAndroidPackageName(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for index, r := range part {
			if index == 0 {
				if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
					return false
				}
				continue
			}
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}
