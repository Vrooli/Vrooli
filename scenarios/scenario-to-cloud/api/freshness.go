package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
)

const (
	defaultScenarioVersion = "1.0.0"
	versionSourceService   = "service_json"
	versionSourceUI        = "ui_package_json"
	versionSourceDefault   = "default"
	noteVersionFallback    = "Scenario version not detected from service.json or ui/package.json; falling back to bundle SHA comparison"
)

func (s *Server) evaluateDeploymentFreshness(_ context.Context, dep *domain.Deployment, manifest domain.CloudManifest) *domain.FreshnessStatus {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return &domain.FreshnessStatus{
			Status:            domain.FreshnessUnknown,
			Summary:           "Unable to determine repository root for freshness checks",
			VersionStatus:     domain.FreshnessUnknown,
			FingerprintStatus: domain.FreshnessUnknown,
			Notes:             []string{fmt.Sprintf("Repository lookup failed: %v", err)},
		}
	}
	// Health is an interactive observation surface. It reports live reachability
	// and observation age on the request path; a full deterministic bundle
	// fingerprint can traverse a large scenario tree and belongs to the
	// explicit release/freshness workflow, not every health refresh.
	return evaluateFreshnessVersionOnly(repoRoot, dep, manifest)
}

func evaluateFreshness(repoRoot string, dep *domain.Deployment, manifest domain.CloudManifest) *domain.FreshnessStatus {
	return evaluateFreshnessWithFingerprint(repoRoot, dep, manifest, true)
}

func evaluateFreshnessVersionOnly(repoRoot string, dep *domain.Deployment, manifest domain.CloudManifest) *domain.FreshnessStatus {
	return evaluateFreshnessWithFingerprint(repoRoot, dep, manifest, false)
}

func evaluateFreshnessWithFingerprint(repoRoot string, dep *domain.Deployment, manifest domain.CloudManifest, includeFingerprint bool) *domain.FreshnessStatus {
	result := &domain.FreshnessStatus{
		Status:            domain.FreshnessUnknown,
		Summary:           "Freshness could not be determined",
		VersionStatus:     domain.FreshnessUnknown,
		FingerprintStatus: domain.FreshnessUnknown,
	}

	localVersion, versionSource, versionErr := resolveScenarioVersion(repoRoot, manifest.Scenario.ID)
	if versionErr != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("Local version lookup failed: %v", versionErr))
	} else {
		result.LocalVersion = localVersion
		result.VersionSource = versionSource
		if versionSource == versionSourceDefault {
			result.Notes = append(result.Notes, noteVersionFallback)
		}

		deployedVersion := strings.TrimSpace(manifest.Scenario.Ref)
		if deployedVersion == "" {
			result.Notes = append(result.Notes, "Deployment manifest has no scenario.ref snapshot")
		} else {
			result.DeployedVersion = deployedVersion
			if deployedVersion == localVersion {
				result.VersionStatus = domain.FreshnessCurrent
			} else {
				result.VersionStatus = domain.FreshnessOutdated
				result.Notes = append(result.Notes, "Local scenario version differs from deployed snapshot")
			}
		}
	}

	if !includeFingerprint {
		result.Notes = append(result.Notes, "Bundle fingerprint comparison is deferred to the release freshness workflow")
		if result.VersionStatus == domain.FreshnessCurrent || result.VersionStatus == domain.FreshnessOutdated {
			result.Status = result.VersionStatus
			if result.Status == domain.FreshnessCurrent {
				result.Summary = "Deployment version matches local scenario state; bundle fingerprint deferred"
			} else {
				result.Summary = "Deployment version differs from local scenario state"
			}
		}
		return result
	}

	deployedSHA := ""
	if dep != nil && dep.BundleSHA256 != nil {
		deployedSHA = strings.TrimSpace(*dep.BundleSHA256)
	}
	if deployedSHA == "" {
		result.Notes = append(result.Notes, "Deployment has no stored bundle SHA256")
	} else {
		result.DeployedBundleSHA256 = deployedSHA
		localSHA, _, hashErr := bundle.CalculateBundleSHA(repoRoot, manifest)
		if hashErr != nil {
			result.Notes = append(result.Notes, fmt.Sprintf("Local bundle fingerprint failed: %v", hashErr))
		} else {
			result.LocalBundleSHA256 = localSHA
			if localSHA == deployedSHA {
				result.FingerprintStatus = domain.FreshnessCurrent
			} else {
				result.FingerprintStatus = domain.FreshnessOutdated
				result.Notes = append(result.Notes, "Local deterministic bundle SHA differs from deployed bundle SHA")
			}
		}
	}

	switch {
	case result.VersionStatus == domain.FreshnessOutdated || result.FingerprintStatus == domain.FreshnessOutdated:
		result.Status = domain.FreshnessOutdated
		result.Summary = "Deployment is healthy but outdated relative to local scenario state"
	case result.VersionStatus == domain.FreshnessCurrent || result.FingerprintStatus == domain.FreshnessCurrent:
		result.Status = domain.FreshnessCurrent
		result.Summary = "Deployment is current with local scenario state"
	default:
		result.Status = domain.FreshnessUnknown
		result.Summary = "Freshness unknown (missing version or fingerprint data)"
	}

	return result
}

func resolveScenarioVersion(repoRoot, scenarioID string) (string, string, error) {
	if strings.TrimSpace(scenarioID) == "" {
		return "", "", fmt.Errorf("scenario id is required")
	}

	servicePath, err := bundle.ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err == nil {
		if version, err := readServiceVersion(servicePath); err == nil && version != "" {
			return version, versionSourceService, nil
		}
	}
	scenarioRoot, err := bundle.ResolveScenarioPath(repoRoot, scenarioID)
	if err != nil {
		return defaultScenarioVersion, versionSourceDefault, nil
	}
	uiPackagePath := filepath.Join(scenarioRoot, "ui", "package.json")
	if version, err := readUIPackageVersion(uiPackagePath); err == nil && version != "" {
		return version, versionSourceUI, nil
	}

	return defaultScenarioVersion, versionSourceDefault, nil
}

func readServiceVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var payload struct {
		Service struct {
			Version string `json:"version"`
		} `json:"service"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Service.Version), nil
}

func readUIPackageVersion(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var payload struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Version), nil
}
