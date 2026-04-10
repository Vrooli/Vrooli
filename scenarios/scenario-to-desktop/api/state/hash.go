package state

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sort"
)

// ComputeManifestHash computes SHA256 of manifest file content.
// Returns the hash, file modification time (unix), and any error.
func ComputeManifestHash(manifestPath string) (hash string, mtime int64, err error) {
	if manifestPath == "" {
		return "", 0, nil
	}

	info, err := os.Stat(manifestPath)
	if err != nil {
		return "", 0, err
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", 0, err
	}

	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), info.ModTime().Unix(), nil
}

// ComputeStateHash computes SHA256 of the entire scenario state (excluding the hash field).
func ComputeStateHash(state *ScenarioState) string {
	if state == nil {
		return ""
	}

	// Create a copy without the hash field
	temp := *state
	temp.Hash = ""

	data, err := json.Marshal(temp)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ComputeFingerprintHash creates a short hash of an input fingerprint for comparison.
func ComputeFingerprintHash(fp *InputFingerprint) string {
	if fp == nil {
		return ""
	}

	data, err := json.Marshal(fp)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:8]) // Short hash for display
}

// ComputeFormStateHash creates a hash of the form state for change detection.
func ComputeFormStateHash(fs *FormState) string {
	if fs == nil {
		return ""
	}

	data, err := json.Marshal(fs)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ExtractBundleFingerprint extracts bundle-stage-relevant fields from form state.
func ExtractBundleFingerprint(fs *FormState, manifestHash string, manifestMtime int64) InputFingerprint {
	return InputFingerprint{
		ManifestPath:  fs.BundleManifestPath,
		ManifestHash:  manifestHash,
		ManifestMtime: manifestMtime,
	}
}

// ExtractPreflightFingerprint extracts preflight-stage-relevant fields.
func ExtractPreflightFingerprint(fs *FormState, manifestHash string) InputFingerprint {
	var secretKeys []string
	if fs.PreflightSecrets != nil {
		secretKeys = make([]string, 0, len(fs.PreflightSecrets))
		for k := range fs.PreflightSecrets {
			secretKeys = append(secretKeys, k)
		}
		sort.Strings(secretKeys)
	}

	return InputFingerprint{
		ManifestPath:        fs.BundleManifestPath,
		ManifestHash:        manifestHash,
		PreflightSecretKeys: secretKeys,
		StartServices:       fs.PreflightStartServices,
	}
}

// ExtractGenerateFingerprint extracts generate-stage-relevant fields.
func ExtractGenerateFingerprint(fs *FormState) InputFingerprint {
	return InputFingerprint{
		TemplateType:   fs.SelectedTemplate,
		Framework:      fs.Framework,
		DeploymentMode: fs.DeploymentMode,
		AppDisplayName: fs.AppDisplayName,
		AppDescription: fs.AppDescription,
		IconPath:       fs.IconPath,
	}
}

// ExtractBuildFingerprint extracts build-stage-relevant fields.
func ExtractBuildFingerprint(fs *FormState, signingConfigHash string) InputFingerprint {
	var platforms []string
	if fs.Platforms.Win {
		platforms = append(platforms, "win")
	}
	if fs.Platforms.Mac {
		platforms = append(platforms, "mac")
	}
	if fs.Platforms.Linux {
		platforms = append(platforms, "linux")
	}
	sort.Strings(platforms)

	return InputFingerprint{
		Platforms:         platforms,
		SigningEnabled:    fs.SigningEnabledForBuild,
		SigningConfigHash: signingConfigHash,
		OutputLocation:    fs.LocationMode,
	}
}

// ExtractSmokeTestFingerprint extracts smoke-test-stage-relevant fields.
func ExtractSmokeTestFingerprint(platform string) InputFingerprint {
	return InputFingerprint{
		SmokeTestPlatform: platform,
	}
}

// FingerprintsEqual compares two fingerprints for equality.
// Returns true if all non-empty fields match.
func FingerprintsEqual(a, b *InputFingerprint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return ComputeFingerprintHash(a) == ComputeFingerprintHash(b)
}

// ManifestFingerprintChanged checks if manifest-related fields changed.
func ManifestFingerprintChanged(stored, current *InputFingerprint) bool {
	if stored == nil || current == nil {
		return stored != current
	}

	// Path change is always significant
	if stored.ManifestPath != current.ManifestPath {
		return true
	}

	// Hash change means content changed
	if stored.ManifestHash != "" && current.ManifestHash != "" {
		return stored.ManifestHash != current.ManifestHash
	}

	return false
}
