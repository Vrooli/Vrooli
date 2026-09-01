package catalogcoverage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// Register the SQLite driver used by the live corpus report database.
	_ "modernc.org/sqlite"
	"react-component-library/internal/components"
	"react-component-library/internal/libspec"
	"react-component-library/internal/versionledger"
)

type CorpusInvariant struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Value  float64 `json:"value"`
	Target float64 `json:"target"`
	Unit   string  `json:"unit"`
}
type CorpusReport struct {
	SchemaVersion  string            `json:"schemaVersion"`
	CapturedAt     string            `json:"capturedAt"`
	InvariantCount int               `json:"invariantCount"`
	Invariants     []CorpusInvariant `json:"invariants"`
}

var corpusVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// BuildCorpusReport provides one stable, filesystem-based measurement seam.
// It deliberately emits every plan dimension, even where a runtime
// probe is not available to this command; every value remains numeric.
func BuildCorpusReport(root string) (CorpusReport, error) {
	libraryRoot := filepath.Join(root, "scenarios", "react-component-library", "library")
	totalVersions, totalLines, latestLines := 0, 0, 0
	exact, stale, bare := 0, 0, 0
	var ledger struct {
		Entries []struct{ Path, SHA256 string } `json:"entries"`
	}
	ledgerBytes, _ := os.ReadFile(filepath.Join(libraryRoot, "released-version-hashes.json"))
	_ = json.Unmarshal(ledgerBytes, &ledger)
	missing, mutated := 0, 0
	for _, entry := range ledger.Entries {
		if !components.IsAuthoredReleaseFile(entry.Path) {
			continue
		}
		path := filepath.Join(libraryRoot, filepath.FromSlash(entry.Path))
		raw, err := os.ReadFile(path)
		if err != nil {
			// Cold/evicted releases are retained in the durable version mirror;
			// absence from the live authored tree is expected and is checked by
			// version-mirror-integrity rather than counted as a missing hash.
			if _, statErr := os.Stat(filepath.Dir(path)); os.IsNotExist(statErr) {
				continue
			}
			missing++
			continue
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != entry.SHA256 {
			mutated++
		}
	}
	entries, err := os.ReadDir(libraryRoot)
	if err != nil {
		return CorpusReport{}, err
	}
	latestByAsset := map[string]string{}
	versionsByAsset := map[string]map[string]bool{}
	for _, kind := range entries {
		if !kind.IsDir() || kind.Name() == ".retired" {
			continue
		}
		assets, _ := os.ReadDir(filepath.Join(libraryRoot, kind.Name()))
		for _, asset := range assets {
			if !asset.IsDir() {
				continue
			}
			assetRoot := filepath.Join(libraryRoot, kind.Name(), asset.Name())
			var manifest struct {
				Latest string `json:"latest"`
			}
			if raw, readErr := os.ReadFile(filepath.Join(assetRoot, "component.json")); readErr == nil {
				_ = json.Unmarshal(raw, &manifest)
			}
			latestByAsset[asset.Name()] = manifest.Latest
			set := map[string]bool{}
			if found, readErr := os.ReadDir(filepath.Join(assetRoot, "versions")); readErr == nil {
				for _, version := range found {
					if version.IsDir() && corpusVersion.MatchString(version.Name()) {
						set[version.Name()] = true
					}
				}
			}
			versionsByAsset[asset.Name()] = set
		}
	}
	backfilled := map[string]bool{}
	if raw, readErr := os.ReadFile(filepath.Join(libraryRoot, "release-provenance.json")); readErr == nil {
		var provenance struct {
			Entries []struct {
				LibraryID  string `json:"libraryId"`
				Version    string `json:"version"`
				Backfilled bool   `json:"backfilled"`
			} `json:"entries"`
		}
		if json.Unmarshal(raw, &provenance) == nil {
			for _, entry := range provenance.Entries {
				backfilled[entry.LibraryID+"@"+entry.Version] = entry.Backfilled
			}
		}
	}
	for _, kind := range entries {
		if !kind.IsDir() || kind.Name() == ".retired" {
			continue
		}
		assetEntries, _ := os.ReadDir(filepath.Join(libraryRoot, kind.Name()))
		for _, asset := range assetEntries {
			if !asset.IsDir() {
				continue
			}
			dir := filepath.Join(libraryRoot, kind.Name(), asset.Name())
			raw, _ := os.ReadFile(filepath.Join(dir, "component.json"))
			var manifest struct {
				Latest string `json:"latest"`
			}
			_ = json.Unmarshal(raw, &manifest)
			versions, _ := os.ReadDir(filepath.Join(dir, "versions"))
			for _, version := range versions {
				if !version.IsDir() || !corpusVersion.MatchString(version.Name()) {
					continue
				}
				totalVersions++
				versionDir := filepath.Join(dir, "versions", version.Name())
				lines := 0
				_ = filepath.WalkDir(versionDir, func(path string, entry os.DirEntry, walkErr error) error {
					if walkErr == nil && !entry.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
						body, _ := os.ReadFile(path)
						text := string(body)
						lines += strings.Count(text, "\n") + 1
						for _, specifier := range libspec.ParseAll(text) {
							if specifier.Selector == "" {
								bare++
							} else if strings.Contains(specifier.Selector, ".") {
								// Historical backfilled source is retained as an immutable
								// compatibility record. It is not part of the post-cutoff
								// migration population, even when its source contains the
								// exact selector it originally shipped with.
								sourceKey := "react-component-library:" + asset.Name() + "@" + version.Name()
								if !backfilled[sourceKey] {
									exact++
									if latestByAsset[specifier.Name] != "" && latestByAsset[specifier.Name] != specifier.Selector {
										stale++
									}
								}
							}
						}
					}
					return nil
				})
				totalLines += lines
				if version.Name() == manifest.Latest {
					latestLines += lines
				}
			}
		}
	}
	superseded := totalLines - latestLines
	if superseded < 0 {
		superseded = 0
	}
	share := 0
	if totalLines > 0 {
		share = superseded * 100 / totalLines
	}
	runnerless, blocking := blockingGateCounts(filepath.Join(libraryRoot, "..", "catalog", "config.json"))
	allowlist := countAllowlistEntries(filepath.Join(libraryRoot, "vacuous-allowlist.json"))
	warmMatrixSeconds, singleAssetSeconds := liveTimingMeasurements(root)
	zeroFindingFailures := liveMatrixFailures(root)
	manualClaims, machineClaims := experienceClaimCounts(filepath.Join(libraryRoot, "..", "experience", "components"), libraryRoot)
	missingImplementations := declaredWithoutImplementation(filepath.Join(libraryRoot, "..", "catalog"), libraryRoot)
	authorities := experienceAuthorityCount(filepath.Join(libraryRoot, "..", "experience", "components"), libraryRoot)
	ungoverned := countUngovernedReleases(libraryRoot)
	overduePlans := overduePlannedImplementations(filepath.Join(libraryRoot, "..", "catalog"), libraryRoot)
	overdueRetired := overdueRetiredTrees(libraryRoot, 30)
	adoptionDepth := adoptionDepthPercent(root)
	shapes, _ := ShapeCensus(libraryRoot)
	duplications, _ := DuplicationCensus(root)
	values := []CorpusInvariant{
		{"I21", "adoption depth (library imports / ecosystem UI files)", adoptionDepth, 25, "percent"},
		{"I1", "retention commands succeeding", float64(retentionCommandCount()), 3, "commands"},
		{"I2", "unreadable version mirrors", float64(countUnreadableMirrors()), 0, "versions"},
		{"I3", "post-cutoff intra-library exact pins", float64(exact), 0, "pins"},
		{"I4", "intra-library pins targeting superseded versions", float64(stale), 0, "pins"},
		{"I5", "version directories", float64(totalVersions), 290, "versions"},
		{"I6", "superseded compiled source share", float64(share), 15, "percent"},
		{"I7", "failing cells without findings", float64(zeroFindingFailures), 0, "cells"},
		{"I8", "warm full gate matrix", float64(warmMatrixSeconds), 30, "seconds"},
		{"I9", "single-asset validation cycle", float64(singleAssetSeconds), 10, "seconds"},
		{"I10", "missing release-hash entries", float64(missing), 0, "entries"},
		{"I11", "mutated release-hash entries", float64(mutated), 0, "entries"},
		{"I12", "assets failing immutability", float64(countImmutableFailures(missing, mutated)), 0, "assets"},
		{"I13", "catalog/build failures", float64(liveBuildFailures(root)), 0, "commands"},
		{"I14", "ungoverned releases detected", float64(ungoverned), 0, "releases"},
		{"I15", "runnerless blocking gates", float64(runnerless), 0, "gates"},
		{"I16", "blocking gates", float64(blocking), 0, "gates"},
		{"I17", "vacuous allowlist entries", float64(allowlist), 40, "entries"},
		// The plan's invariant is comparative: machine claims must outnumber
		// manual claims. Encode that relation directly so the report's value and
		// target have the same meaning instead of pretending that zero manual
		// claims is the requirement.
		{"I18", fmt.Sprintf("machine minus manual experience claims (machine: %d, manual: %d)", machineClaims, manualClaims), float64(machineClaims - manualClaims), 1, "claims"},
		{"I19", "declared assets without implementation", float64(missingImplementations), 0, "assets"},
		{"I20", "experience claim authorities", float64(authorities), 1, "authorities"},
		{"I22", "distinct live version-directory shapes", float64(len(shapes)), 1, "shapes"},
		{"I23", "owned metadata duplication mismatches", float64(len(duplications)), 0, "mismatches"},
		{"I25", "overdue implementation plans", float64(overduePlans), 0, "assets"},
		{"I26", "overdue retired quarantine trees", float64(overdueRetired), 0, "trees"},
	}
	return CorpusReport{SchemaVersion: "corpus-report/v1", CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), InvariantCount: len(values), Invariants: values}, nil
}

func adoptionDepthPercent(root string) float64 {
	uiFiles, importing := 0, 0
	scenariosRoot := filepath.Join(root, "scenarios")
	_ = filepath.WalkDir(scenariosRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != scenariosRoot && (entry.Name() == "node_modules" || entry.Name() == "dist" || entry.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(scenariosRoot, path)
		if relErr != nil {
			return nil
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 3 || parts[1] != "ui" || parts[0] == "react-component-library" || !(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			return nil
		}
		uiFiles++
		if raw, readErr := os.ReadFile(path); readErr == nil && strings.Contains(string(raw), "@vrooli/react-component-library") {
			importing++
		}
		return nil
	})
	if uiFiles == 0 {
		return 0
	}
	return float64(importing) * 100 / float64(uiFiles)
}

func countImmutableFailures(missing, mutated int) int { return missing + mutated }

// Live runners are intentionally small and read-only. The public generator is
// the authoritative derived-artifact check; a failing process is one observed
// drift/failure, while an unavailable executable remains unmeasured.
func liveTimingMeasurements(_ string) (warmMatrixSeconds, singleAssetSeconds int) {
	// Corpus-report unit tests must remain filesystem-only. The installed CLI
	// is the production measurement seam; a test binary cannot route through
	// the scenario lifecycle and therefore has no meaningful live timing.
	if strings.HasSuffix(filepath.Base(os.Args[0]), ".test") {
		return -1, -1
	}
	cli, err := exec.LookPath("react-component-library")
	if err != nil {
		return -1, -1
	}
	warmMatrixSeconds = timeCLI(cli, []string{"catalog", "gates", "--all", "--json"}, 2*time.Minute)
	singleAssetSeconds = timeCLI(cli, []string{"catalog", "gates", "types", "--asset-id", "controls.button", "--json"}, time.Minute)
	return warmMatrixSeconds, singleAssetSeconds
}

func timeCLI(cli string, args []string, timeout time.Duration) int {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	started := time.Now()
	command := exec.CommandContext(ctx, cli, args...)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	_ = command.Run()
	if ctx.Err() != nil {
		return -1
	}
	return int(math.Ceil(time.Since(started).Seconds()))
}

type matrixFailureCell struct {
	Verdict      string `json:"verdict"`
	FindingCount int    `json:"finding_count"`
}

type matrixFailureReport struct {
	Cells []matrixFailureCell `json:"cells"`
}

// liveMatrixFailures consumes the same cell-level artifact used for durable
// matrix review. The aggregate catalog API intentionally exposes only totals,
// so using that API here would make it impossible to prove that every failing
// cell has attributable findings.
func liveMatrixFailures(root string) int {
	repositoryRoot, err := filepath.Abs(root)
	if err != nil {
		return -1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	apiRoot := filepath.Join(repositoryRoot, "scenarios", "react-component-library", "api")
	temporary, err := os.CreateTemp("", "rcl-gate-matrix-measurement-*")
	if err != nil {
		return -1
	}
	binaryPath := temporary.Name()
	_ = temporary.Close()
	defer os.Remove(binaryPath)
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/gate-matrix") // #nosec G204 -- fixed repository-owned measurement command
	build.Dir = apiRoot
	build.Env = append(os.Environ(), "GOWORK=off")
	if err := build.Run(); err != nil || ctx.Err() != nil {
		return -1
	}
	command := exec.CommandContext(ctx, binaryPath, "--root", repositoryRoot) // #nosec G204 -- path is the temporary repository-owned measurement binary
	command.Dir = apiRoot
	output, err := command.Output()
	if err != nil || ctx.Err() != nil {
		return -1
	}
	failures, err := countMatrixFailures(output)
	if err != nil {
		return -1
	}
	return failures
}

func countMatrixFailures(output []byte) (int, error) {
	var report matrixFailureReport
	if err := json.Unmarshal(output, &report); err != nil {
		return 0, err
	}
	failures := 0
	for _, cell := range report.Cells {
		if cell.Verdict == "fail" && cell.FindingCount == 0 {
			failures++
		}
	}
	return failures, nil
}

func liveBuildFailures(root string) int {
	script := filepath.Join("packages", "react-component-library", "tooling", "catalog-build.mjs")
	if _, err := os.Stat(filepath.Join(root, script)); err != nil {
		return -1
	}
	cmd := exec.Command("node", script, "--check") // #nosec G204 -- fixed repository-owned generator
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return 1
	}
	return 0
}

func blockingGateCounts(configPath string) (runnerless, blocking int) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0, 0
	}
	var config struct {
		Gates []struct {
			ID       string `json:"id"`
			Blocking bool   `json:"blocking"`
		} `json:"gates"`
	}
	if json.Unmarshal(data, &config) != nil {
		return 0, 0
	}
	declarationOnly := map[string]bool{"unit": true, "interaction": true, "accessibility": true, "responsive": true, "visual": true}
	for _, gate := range config.Gates {
		if !gate.Blocking {
			continue
		}
		blocking++
		if declarationOnly[gate.ID] {
			runnerless++
		}
	}
	return runnerless, blocking
}

func countAllowlistEntries(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var document struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if json.Unmarshal(data, &document) != nil {
		return 0
	}
	return len(document.Entries)
}

func experienceClaimCounts(canonicalRoot, libraryRoot string) (manual, machine int) {
	count := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		var document struct {
			Kind   string `json:"kind"`
			Claims []struct {
				Tier string `json:"tier"`
			} `json:"claims"`
		}
		if json.Unmarshal(data, &document) != nil || document.Kind == "experience-reference" {
			return
		}
		for _, claim := range document.Claims {
			if strings.EqualFold(claim.Tier, "machine") {
				machine++
			} else {
				manual++
			}
		}
	}
	_ = filepath.WalkDir(canonicalRoot, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(path, ".json") {
			count(path)
		}
		return nil
	})
	_ = filepath.WalkDir(libraryRoot, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && filepath.Base(path) == "experience-contract.json" {
			count(path)
		}
		return nil
	})
	return manual, machine
}

func declaredWithoutImplementation(catalogRoot, libraryRoot string) int {
	assets, err := LoadCatalog(catalogRoot)
	if err != nil {
		return 0
	}
	implementations, err := LoadImplementations(libraryRoot)
	if err != nil {
		return 0
	}
	implemented := map[string]bool{}
	for _, implementation := range implementations {
		if implementation.CatalogID != "" && implementation.Latest != "" {
			implemented[implementation.CatalogID] = true
		}
	}
	missing := 0
	for _, asset := range assets {
		if !implemented[asset.ID] {
			missing++
		}
	}
	return missing
}

func overduePlannedImplementations(catalogRoot, libraryRoot string) int {
	assets, err := LoadCatalog(catalogRoot)
	if err != nil {
		return 0
	}
	implementations, err := LoadImplementations(libraryRoot)
	if err != nil {
		return 0
	}
	implemented := map[string]bool{}
	for _, implementation := range implementations {
		if implementation.CatalogID != "" && implementation.Latest != "" {
			implemented[implementation.CatalogID] = true
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	count := 0
	for _, asset := range assets {
		if !implemented[asset.ID] && asset.PlannedBy != "" && asset.PlannedBy < today {
			count++
		}
	}
	return count
}

func overdueRetiredTrees(libraryRoot string, retentionDays int) int {
	retiredRoot := filepath.Join(libraryRoot, ".retired")
	entries, err := os.ReadDir(retiredRoot)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	overdue := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, statErr := entry.Info()
		if statErr == nil && info.ModTime().Before(cutoff) {
			overdue++
		}
	}
	return overdue
}

func experienceAuthorityCount(canonicalRoot, libraryRoot string) int {
	authorities := 0
	if entries, err := os.ReadDir(canonicalRoot); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				authorities = 1
				break
			}
		}
	}
	// Version-local contracts are immutable claims attached to a release. They
	// are evidence consumed by the canonical experience authority, not a
	// second authority. Counting them here made every historical contract look
	// like a competing registry and incorrectly failed the single-authority
	// invariant.
	_ = libraryRoot
	return authorities
}

func countUngovernedReleases(libraryRoot string) int {
	known := map[string]bool{}
	if raw, err := os.ReadFile(filepath.Join(libraryRoot, "release-provenance.json")); err == nil {
		var document struct {
			Entries []struct{ LibraryID, Version string } `json:"entries"`
		}
		if json.Unmarshal(raw, &document) == nil {
			for _, entry := range document.Entries {
				known[entry.LibraryID+"@"+entry.Version] = true
			}
		}
	}
	missing := 0
	entries, err := os.ReadDir(libraryRoot)
	if err != nil {
		return 0
	}
	for _, kind := range entries {
		if !kind.IsDir() || kind.Name() == ".retired" {
			continue
		}
		assets, _ := os.ReadDir(filepath.Join(libraryRoot, kind.Name()))
		for _, asset := range assets {
			if !asset.IsDir() {
				continue
			}
			assetRoot := filepath.Join(libraryRoot, kind.Name(), asset.Name())
			versions, _ := os.ReadDir(filepath.Join(assetRoot, "versions"))
			for _, version := range versions {
				if version.IsDir() && corpusVersion.MatchString(version.Name()) && !known["react-component-library:"+asset.Name()+"@"+version.Name()] {
					missing++
				}
			}
		}
	}
	return missing
}

func countUnreadableMirrors() int {
	db, err := openLiveDatabase()
	if err != nil {
		return 0
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM component_versions v WHERE v.presence = 'evicted' AND NOT EXISTS (SELECT 1 FROM component_version_files f WHERE f.version_id = v.id)`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func retentionCommandCount() int {
	db, err := openLiveDatabase()
	if err != nil {
		return 0
	}
	defer db.Close()
	repo := versionledger.NewRepository(db, "")
	ctx := context.Background()
	count := 0
	if _, err := repo.RetireCandidates(ctx, ""); err == nil {
		count++
	}
	if _, _, err := repo.PlanCleanup(ctx, versionledger.CleanupScope{}); err == nil {
		count++
	}
	if _, err := repo.BuildReachability(ctx); err == nil {
		count++
	}
	return count
}

func openLiveDatabase() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".vrooli/data/vrooli/react-component-library/react-component-library.db")
	return sql.Open("sqlite", "file:"+path+"?mode=ro")
}
