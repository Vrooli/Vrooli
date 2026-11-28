package main

import (
	"bytes"
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"
)

const rh3BenignFixtureMarker = ".gct-rh3-benign-security"
const rh4BenignFixtureMarker = ".gct-rh4-benign-security"

// rh3BenignCandidateSecurity is deliberately narrow: it accepts only the
// literal, harmless blobs used by the RH3 live fixture. It proves that the
// real commit admission path can consume a complete scanner result, but it is
// not evidence of staged-secret detection and is unavailable outside a pinned
// development test lease under a disposable temp directory.
type rh3BenignCandidateSecurity struct{}

type unavailableCandidateSecurity struct{}

func (s unavailableCandidateSecurity) Scan(context.Context, map[string][]byte) (CandidateSecurityResult, error) {
	return CandidateSecurityResult{FailureCode: "scanner_unavailable"}, nil
}

// selectCommitCandidateSecurity chooses the configured production scanner or,
// only when none is configured, the narrowly scoped RH3 fixture provider.
// Keep the pointer check before interface conversion: a nil *CandidateSecurityClient
// stored in CandidateSecurityScanner would be a non-nil interface and suppress
// the fixture path.
func selectCommitCandidateSecurity(
	ctx context.Context,
	configured *CandidateSecurityClient,
	db *database.RoutedDB,
	roots *filerouting.RoutedRoots,
	git GitRunner,
	repoPath string,
) (CandidateSecurityScanner, bool, string) {
	if configured != nil {
		return configured, false, ""
	}
	provider, reason := rh3FixtureCandidateSecurityReason(ctx, db, roots, git, repoPath)
	if provider == nil && database.IsTestMode(ctx) {
		return unavailableCandidateSecurity{}, false, reason
	}
	return provider, provider != nil, reason
}

func (rh3BenignCandidateSecurity) Scan(_ context.Context, files map[string][]byte) (CandidateSecurityResult, error) {
	allowed := map[string]string{
		"tracked.txt":            "first candidate\n",
		"later.txt":              "candidate B\n",
		"during-check.txt":       "candidate C\n",
		"unusual name ☃.txt":     "first candidate\n",
		"path with spaces/δ.txt": "candidate B\n",
	}
	if len(files) == 0 {
		return CandidateSecurityResult{FixtureOnly: true, FailureCode: "empty_fixture_candidate"}, nil
	}
	for path, content := range files {
		if allowed[path] == "" || !bytes.Equal(content, []byte(allowed[path])) {
			return CandidateSecurityResult{FixtureOnly: true, FailureCode: "non_allowlisted_fixture_blob"}, nil
		}
	}
	return CandidateSecurityResult{Complete: true, FixtureOnly: true}, nil
}

// rh3FixtureCandidateSecurity returns the synthetic provider only when all
// three fixture identities agree: the request is pinned test mode, the leased
// SQLite DB lives below an RH3-owned temporary directory with an exact lease
// marker, and the repository plus Git metadata resolve inside that directory.
// A linked worktree that points to an external common Git directory is refused.
func rh3FixtureCandidateSecurity(
	ctx context.Context,
	db *database.RoutedDB,
	roots *filerouting.RoutedRoots,
	git GitRunner,
	repoPath string,
) CandidateSecurityScanner {
	provider, _ := rh3FixtureCandidateSecurityReason(ctx, db, roots, git, repoPath)
	return provider
}

func rh3FixtureCandidateSecurityReason(
	ctx context.Context,
	db *database.RoutedDB,
	roots *filerouting.RoutedRoots,
	git GitRunner,
	repoPath string,
) (CandidateSecurityScanner, string) {
	reject := func(reason string) (CandidateSecurityScanner, string) { return nil, reason }
	if !projectmeta.IsDevelopment() || !database.IsTestMode(ctx) || db == nil || roots == nil || git == nil {
		return reject("development test lease is not pinned")
	}
	if _, err := roots.Pick(ctx, storage.ClassData); err != nil {
		return reject("file-root lease is not pinned")
	}
	leaseID := db.ActiveLeaseID()
	if leaseID == "" {
		return reject("database lease is not active")
	}
	state, err := db.TestPoolRestartState(leaseID)
	if err != nil {
		return reject("database lease state is unavailable")
	}
	dsn, err := url.Parse(state.DSN)
	if err != nil || dsn.Scheme != "file" || dsn.Host != "" {
		return reject("database lease is not a local SQLite fixture")
	}
	dbPath := filepath.FromSlash(dsn.Path)
	if !filepath.IsAbs(dbPath) {
		return reject("database fixture path is not absolute")
	}
	canonicalDB, err := filepath.EvalSymlinks(dbPath)
	if err != nil {
		return reject("database fixture file is unavailable")
	}
	fixtureRoot, err := filepath.EvalSymlinks(filepath.Dir(canonicalDB))
	fixtureName := filepath.Base(fixtureRoot)
	if err != nil || (!strings.HasPrefix(fixtureName, "gct-rh3-live-") && !strings.HasPrefix(fixtureName, "gct-rh4-live-")) {
		return reject("database fixture directory is not an RH3 temp root")
	}
	tempRoot, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil || !pathIsWithin(tempRoot, fixtureRoot) {
		return reject("database fixture is outside the system temp root")
	}
	markerName := rh3BenignFixtureMarker
	if strings.HasPrefix(fixtureName, "gct-rh4-live-") {
		markerName = rh4BenignFixtureMarker
	}
	markerPath := filepath.Join(fixtureRoot, markerName)
	markerInfo, err := os.Lstat(markerPath)
	if err != nil || !markerInfo.Mode().IsRegular() || markerInfo.Mode()&os.ModeSymlink != 0 {
		return reject("exact fixture marker is unavailable")
	}
	marker, err := os.ReadFile(markerPath)
	if err != nil || strings.TrimSpace(string(marker)) != leaseID {
		return reject("fixture marker does not bind the active lease")
	}

	repoRoot, err := git.RevParse(ctx, repoPath, "--show-toplevel")
	if err != nil {
		return reject("repository root could not be resolved")
	}
	gitDir, err := git.RevParse(ctx, repoPath, "--absolute-git-dir")
	if err != nil {
		return reject("repository Git directory could not be resolved")
	}
	commonDir, err := git.RevParse(ctx, repoPath, "--git-common-dir")
	if err != nil {
		return reject("repository common Git directory could not be resolved")
	}
	paths := []string{repoPath, strings.TrimSpace(string(repoRoot)), strings.TrimSpace(string(gitDir)), strings.TrimSpace(string(commonDir))}
	for index, path := range paths {
		if path == "" {
			return reject("repository Git metadata path is empty")
		}
		if index == len(paths)-1 && !filepath.IsAbs(path) {
			path = filepath.Join(strings.TrimSpace(string(repoRoot)), path)
		}
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil || !pathIsWithin(fixtureRoot, canonical) {
			return reject("repository or Git metadata escapes fixture root")
		}
	}
	return rh3BenignCandidateSecurity{}, ""
}

func pathIsWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

var _ CandidateSecurityScanner = rh3BenignCandidateSecurity{}
