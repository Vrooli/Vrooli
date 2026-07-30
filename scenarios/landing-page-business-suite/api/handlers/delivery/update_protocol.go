package delivery

import (
	"fmt"

	internal "landing-page-business-suite-api/internal/delivery"
)

func ManifestFilenameToPlatform(filename string) string {
	switch filename {
	case "latest.yml":
		return "windows"
	case "latest-mac.yml":
		return "mac"
	case "latest-linux.yml":
		return "linux"
	default:
		return ""
	}
}

func ChannelToVariantKey(channel string) string {
	if channel == "stable" || channel == "" {
		return "default"
	}
	return channel
}

func BuildElectronManifest(artifact *internal.Artifact, releaseNotes string) []byte {
	out := fmt.Sprintf("version: %s\npath: %s\nsha512: %s\nreleaseDate: %s\nfiles:\n  - url: %s\n    sha512: %s\n    size: %d\n", artifact.ReleaseVersion, artifact.OriginalFilename, artifact.SHA512, artifact.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"), artifact.OriginalFilename, artifact.SHA512, artifact.SizeBytes)
	if releaseNotes != "" {
		out += fmt.Sprintf("releaseNotes: %s\n", releaseNotes)
	}
	return []byte(out)
}
