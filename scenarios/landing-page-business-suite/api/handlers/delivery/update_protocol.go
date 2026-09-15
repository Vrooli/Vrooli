package delivery

import (
	"fmt"
	"strconv"

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

func BuildElectronManifest(artifact *internal.Artifact, releaseNotes string, binding ...string) []byte {
	channel, appKey := "stable", artifact.AppKey
	if len(binding) > 0 && binding[0] != "" {
		channel = binding[0]
	}
	if len(binding) > 1 && binding[1] != "" {
		appKey = binding[1]
	}
	out := fmt.Sprintf("version: %s\npath: %s\nsha512: %s\nreleaseDate: %s\n", artifact.ReleaseVersion, artifact.OriginalFilename, artifact.SHA512, artifact.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"))
	out += fmt.Sprintf("vrooli:\n  applicationKey: %s\n  channel: %s\n  platform: %s\n  artifactRef: %s\n", strconv.Quote(appKey), strconv.Quote(channel), strconv.Quote(artifact.Platform), strconv.Quote("sha512:"+artifact.SHA512))
	out += fmt.Sprintf("files:\n  - url: %s\n    sha512: %s\n    size: %d\n", artifact.OriginalFilename, artifact.SHA512, artifact.SizeBytes)
	if releaseNotes != "" {
		out += fmt.Sprintf("releaseNotes: %s\n", releaseNotes)
	}
	return []byte(out)
}
