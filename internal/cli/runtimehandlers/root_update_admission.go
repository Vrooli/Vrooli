package runtimehandlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

// These helpers qualify source/checkouts for admission. They do not define or
// interpret persisted recovery records and stay outside the state contract.
func pinnedRuntimeSnapshotSource(home, candidate string, resolve func() (string, error)) (string, func() error, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return "", nil, err
	}
	base := canonicalPath(filepath.Join(state, "root-rebaseline", "snapshots"))
	clean := canonicalPath(candidate)
	rel, err := filepath.Rel(base, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", nil, nil
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) != 2 || parts[1] != "vrooli" || !(strings.HasPrefix(parts[0], "rebaseline-bundle.") || strings.HasPrefix(parts[0], "rebaseline-hub.")) {
		return "", nil, fmt.Errorf("invalid runtime rebaseline snapshot path")
	}
	cwd, err := repoRootFromWorkingDirectory()
	if err != nil {
		return "", nil, err
	}
	source, err := pinnedRebaselineSource(resolve, filepath.Join(cwd, ".snapshot-source-binding"))
	if err != nil {
		return "", nil, err
	}
	check := func() error {
		id, err := coreactivation.ProbeRootIdentity(context.Background(), clean)
		if err != nil {
			return fmt.Errorf("probe prepared snapshot: %w", err)
		}
		fp, err := buildinfo.ComputeSourceFingerprintForPaths(source, buildinfo.FingerprintTargetsForExecutable("vrooli")...)
		if err != nil {
			return err
		}
		if id.GetBuildIdentity() == "" || id.GetBuildIdentity() != fp {
			return fmt.Errorf("prepared snapshot was not built from the current checkout; prepare a new bundle")
		}
		return nil
	}
	if err := check(); err != nil {
		return "", nil, err
	}
	return source, check, nil
}

func pinnedRebaselineSource(resolveSourceRoot func() (string, error), candidate string) (string, error) {
	cwdRoot, err := repoRootFromWorkingDirectory()
	if err != nil {
		return "", fmt.Errorf("run root rebaseline from the source checkout that built the candidate: %w", err)
	}
	cwdRoot = canonicalPath(cwdRoot)
	if resolved, err := resolveSourceRoot(); err == nil && canonicalPath(resolved) != cwdRoot {
		return "", fmt.Errorf("the source root override (VROOLI_SOURCE_ROOT or VROOLI_ROOT) names %s, but this command runs from %s; unset it or run from that checkout", resolved, cwdRoot)
	}
	if rel, err := filepath.Rel(cwdRoot, canonicalPath(candidate)); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("the candidate %s was not built in the checkout this command runs from (%s)", candidate, cwdRoot)
	}
	return cwdRoot, nil
}

var repoRootFromWorkingDirectory = repocontract.FindRepoRootFromCWD

func canonicalPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(path)
}
