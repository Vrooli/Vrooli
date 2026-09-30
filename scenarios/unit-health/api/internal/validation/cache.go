package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/evidence"
	"unit-health/internal/executor"
)

const defaultAdapterVersion = "unit-health-kernel/2.0.0"

func (s *Service) evidenceKey(invRoot, targetKind string, workspaces []Workspace) (evidence.Key, error) {
	return s.evidenceKeyForMode(invRoot, targetKind, workspaces, false)
}

func (s *Service) evidenceKeyForMode(invRoot, targetKind string, workspaces []Workspace, fastTestOnly bool) (evidence.Key, error) {
	var roots []string
	var adapterIdentities []string
	var toolchainIdentities []string
	var runnerProfiles []string
	var workspaceScope []string
	coverageMode := "test"
	for _, workspace := range workspaces {
		roots = append(roots, workspace.RootPath)
		workspaceScope = append(workspaceScope, workspace.ID+"="+filepath.Clean(workspace.RootPath))
		if workspace.AdapterID != "" {
			adapterIdentities = append(adapterIdentities, workspace.AdapterID+"@"+workspace.AdapterVersion)
		}
		if workspace.ToolchainIdentity != "" {
			toolchainIdentities = append(toolchainIdentities, workspace.ToolchainIdentity)
		}
		if workspace.RunnerProfile != "" {
			runnerProfiles = append(runnerProfiles, workspace.ID+"="+workspace.RunnerProfile)
		}
		if !fastTestOnly && workspace.CoverageCommand != "" {
			coverageMode = "coverage"
		}
	}
	sort.Strings(adapterIdentities)
	sort.Strings(toolchainIdentities)
	sort.Strings(runnerProfiles)
	sort.Strings(workspaceScope)
	if s.ToolchainIdentity != "" {
		toolchainIdentities = append(toolchainIdentities, s.ToolchainIdentity)
		sort.Strings(toolchainIdentities)
	}
	sourceDigest, configDigest, lockDigest, err := digestWorkspaceInputs(roots)
	if err != nil {
		return evidence.Key{}, err
	}
	runnerProfile := strings.Join(runnerProfiles, ",")
	if s.RunnerProfile != "" {
		if runnerProfile != "" {
			runnerProfile += ","
		}
		runnerProfile += s.RunnerProfile
	}
	hermetic := executor.HostHermeticCapabilities()
	return evidence.NewKey(evidence.KeyInput{
		SourceDigest:         sourceDigest,
		ConfigDigest:         configDigest,
		DependencyLockDigest: lockDigest,
		ToolchainIdentity:    strings.Join(toolchainIdentities, ","),
		AdapterID:            strings.Join(adapterIdentities, ","),
		AdapterVersion:       defaultAdapterVersion,
		PolicyDigest:         s.PolicyDigest,
		RunnerProfile:        runnerProfile,
		OS:                   runtime.GOOS,
		Architecture:         runtime.GOARCH,
		Environment: map[string]string{
			"target_kind":                targetKind,
			"root":                       filepath.Clean(invRoot),
			"workspace_scope":            strings.Join(workspaceScope, ","),
			"hermetic_network_deny":      strconv.FormatBool(hermetic.NetworkDeny),
			"hermetic_declared_net":      strconv.FormatBool(hermetic.AllowDeclaredNet),
			"hermetic_workspace_ro":      strconv.FormatBool(hermetic.WorkspaceReadonly),
			"hermetic_child_leak":        strconv.FormatBool(hermetic.ChildLeakDetection),
			"hermetic_open_handles":      strconv.FormatBool(hermetic.OpenHandleDetection),
			"hermetic_order_independent": strconv.FormatBool(hermetic.OrderIndependent),
			"hermetic_environment":       strconv.FormatBool(hermetic.RestoreEnvironment),
		},
		CoverageMode:   coverageMode,
		ArtifactSchema: "unit-health.response.v1",
	})
}

func digestWorkspaceInputs(roots []string) (source, config, lock string, err error) {
	sort.Strings(roots)
	sourceHash, configHash, lockHash := sha256.New(), sha256.New(), sha256.New()
	for _, root := range roots {
		root = filepath.Clean(root)
		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != root && ignoredDigestDir(entry.Name()) {
					return fs.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() || ignoredDigestFile(path) {
				return nil
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			marker := []byte(filepath.ToSlash(rel) + "\x00")
			_, _ = sourceHash.Write(marker)
			_, _ = sourceHash.Write(raw)
			base := filepath.Base(path)
			if adapterregistry.IsConfigFile(base) {
				_, _ = configHash.Write(marker)
				_, _ = configHash.Write(raw)
			}
			if adapterregistry.IsLockFile(base) {
				_, _ = lockHash.Write(marker)
				_, _ = lockHash.Write(raw)
			}
			return nil
		})
		if walkErr != nil {
			return "", "", "", fmt.Errorf("digest workspace %q: %w", root, walkErr)
		}
	}
	return hex.EncodeToString(sourceHash.Sum(nil)), hex.EncodeToString(configHash.Sum(nil)), hex.EncodeToString(lockHash.Sum(nil)), nil
}

func ignoredDigestDir(name string) bool {
	switch name {
	case ".git", "node_modules", "coverage", "dist", "build", ".cache", "vendor":
		return true
	}
	return false
}

func ignoredDigestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, ".log") || strings.HasSuffix(base, ".tmp") || base == "coverage.out"
}
