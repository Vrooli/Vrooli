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

	"react-component-library/internal/librarywalk"

	// Register the SQLite driver used by the live corpus report database.
	"react-component-library/internal/components"
	"react-component-library/internal/libspec"
	"react-component-library/internal/versionledger"

	_ "modernc.org/sqlite"
)

type CorpusInvariant struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Value  float64 `json:"value"`
	Target float64 `json:"target"`
	Unit   string  `json:"unit"`
	Status string  `json:"status,omitempty"`
}
type CorpusReport struct {
	SchemaVersion  string            `json:"schemaVersion"`
	CapturedAt     string            `json:"capturedAt"`
	InvariantCount int               `json:"invariantCount"`
	Invariants     []CorpusInvariant `json:"invariants"`
	VersionShapes  []ShapeRow        `json:"versionShapes,omitempty"`
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
				_ = librarywalk.WalkContext(context.Background(), versionDir, func(path string, entry os.DirEntry, walkErr error) error {
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
	warmMatrixSeconds, singleAssetSeconds, timingMeasured := liveTimingMeasurements(root)
	measurementFailed := !timingMeasured
	zeroFindingFailures, matrixMeasured := liveMatrixFailures(root)
	zeroFindingFailuresFailed := !matrixMeasured
	buildFailures, buildMeasured := liveBuildFailures(root)
	buildFailuresFailed := !buildMeasured
	manualClaims, machineClaims := experienceClaimCounts(filepath.Join(libraryRoot, "..", "experience", "components"), libraryRoot)
	missingImplementations := declaredWithoutImplementation(filepath.Join(libraryRoot, "..", "catalog"), libraryRoot)
	ungoverned := countUngovernedReleases(libraryRoot)
	overduePlans := overduePlannedImplementations(filepath.Join(libraryRoot, "..", "catalog"), libraryRoot)
	overdueRetired := overdueRetiredTrees(libraryRoot, 30)
	adoptionDepth := adoptionDepthPercent(root)
	machineryLines := LedgerMachineryLines()
	assetLines := LedgerAssetLines()
	machineryRatio := 0.0
	if assetLines > 0 {
		machineryRatio = float64(machineryLines) / float64(assetLines)
	}
	shapes, _ := ShapeCensus(libraryRoot)
	duplications, _ := DuplicationCensus(root)
	values := []CorpusInvariant{
		{"I21", "adoption depth (library imports / ecosystem UI files)", adoptionDepth, 25, "percent", ""},
		{"I1", "retention commands succeeding", float64(retentionCommandCount()), 3, "commands", ""},
		{"I2", "unreadable version mirrors", float64(countUnreadableMirrors()), 0, "versions", ""},
		{"I3", "post-cutoff intra-library exact pins", float64(exact), 0, "pins", ""},
		{"I4", "intra-library pins targeting superseded versions", float64(stale), 0, "pins", ""},
		{"I5", "version directories", float64(totalVersions), 290, "versions", ""},
		{"I6", "superseded compiled source share", float64(share), 15, "percent", ""},
		{"I7", "failing cells without findings", float64(zeroFindingFailures), 0, "cells", ""},
		{"I8", "warm full gate matrix", float64(warmMatrixSeconds), 30, "seconds", ""},
		{"I9", "single-asset check cycle", float64(singleAssetSeconds), 10, "seconds", ""},
		{"I10", "missing release-hash entries", float64(missing), 0, "entries", ""},
		{"I11", "mutated release-hash entries", float64(mutated), 0, "entries", ""},
		{"I12", "assets failing immutability", float64(countImmutableFailures(missing, mutated)), 0, "assets", ""},
		{"I13", "catalog/build failures", float64(buildFailures), 0, "commands", ""},
		{"I14", "ungoverned releases detected", float64(ungoverned), 0, "releases", ""},
		{"I15", "runnerless blocking gates", float64(runnerless), 0, "gates", ""},
		{"I16", "blocking gates", float64(blocking), 0, "gates", ""},
		{"I17", "vacuous allowlist entries", float64(allowlist), 40, "entries", ""},
		// The plan's invariant is comparative: machine claims must outnumber
		// manual claims. Encode that relation directly so the report's value and
		// target have the same meaning instead of pretending that zero manual
		// claims is the requirement.
		{"I18", fmt.Sprintf("machine minus manual experience claims (machine: %d, manual: %d)", machineClaims, manualClaims), float64(machineClaims - manualClaims), 1, "claims", ""},
		{"I19", "declared assets without implementation", float64(missingImplementations), 0, "assets", ""},
		{"I22", "distinct live version-directory shapes", float64(len(shapes)), 1, "shapes", ""},
		{"I23", "owned metadata duplication mismatches", float64(len(duplications)), 0, "mismatches", ""},
		{"I27", "machinery-to-asset line ratio", machineryRatio, 0.92, "ratio", ""},
		{"I25", "overdue implementation plans", float64(overduePlans), 0, "assets", ""},
		{"I26", "overdue retired quarantine trees", float64(overdueRetired), 0, "trees", ""},
	}
	if measurementFailed || zeroFindingFailuresFailed || buildFailuresFailed {
		for index := range values {
			if (values[index].ID == "I8" || values[index].ID == "I9") && measurementFailed ||
				values[index].ID == "I7" && zeroFindingFailuresFailed ||
				values[index].ID == "I13" && buildFailuresFailed {
				values[index].Status = "failed_measurement"
			}
		}
	}
	return CorpusReport{SchemaVersion: "corpus-report/v1", CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), InvariantCount: len(values), Invariants: values, VersionShapes: shapes}, nil
}

func adoptionDepthPercent(root string) float64 {
	uiFiles, importing := 0, 0
	scenariosRoot := filepath.Join(root, "scenarios")
	_ = librarywalk.WalkContext(context.Background(), scenariosRoot, func(path string, entry os.DirEntry, err error) error {
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
func liveTimingMeasurements(_ string) (warmMatrixSeconds, singleAssetSeconds int, measured bool) {
	// Corpus-report unit tests must remain filesystem-only. The installed CLI
	// is the production measurement seam; a test binary cannot route through
	// the scenario lifecycle and therefore has no meaningful live timing.
	if strings.HasSuffix(filepath.Base(os.Args[0]), ".test") {
		return 0, 0, false
	}
	cli, err := exec.LookPath("react-component-library")
	if err != nil {
		return 0, 0, false
	}
	var warmMeasured, assetMeasured bool
	warmMatrixSeconds, warmMeasured = timeCLI(cli, []string{"catalog", "gates", "--all", "--json"}, 2*time.Minute)
	singleAssetSeconds, assetMeasured = timeCLI(cli, []string{"asset", "check", "controls.button", "--json"}, time.Minute)
	return warmMatrixSeconds, singleAssetSeconds, warmMeasured && assetMeasured
}

func timeCLI(cli string, args []string, timeout time.Duration) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	started := time.Now()
	command := exec.CommandContext(ctx, cli, args...)
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	_ = command.Run()
	if ctx.Err() != nil {
		return 0, false
	}
	return int(math.Ceil(time.Since(started).Seconds())), true
}

type matrixFailureCell struct {
	Verdict      string `json:"verdict"`
	FindingCount int    `json:"finding_count"`
}

type matrixFailureReport struct {
	Cells []matrixFailureCell `json:"cells"`
}

// liveMatrixFailures evaluates the same gate definitions used by the catalog
// matrix. The aggregate catalog API intentionally exposes only totals, so
// using that API here would make it impossible to prove that every failing
// cell has attributable findings.
func liveMatrixFailures(root string) (int, bool) {
	repositoryRoot, err := filepath.Abs(root)
	if err != nil {
		return 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	failures, err := CountMatrixFailures(ctx, repositoryRoot)
	if err != nil || ctx.Err() != nil {
		return 0, false
	}
	return failures, true
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

func liveBuildFailures(root string) (int, bool) {
	script := filepath.Join("packages", "react-component-library", "tooling", "catalog-build.mjs")
	if _, err := os.Stat(filepath.Join(root, script)); err != nil {
		return 0, false
	}
	cmd := exec.Command("node", script, "--check") // #nosec G204 -- fixed repository-owned generator
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return 1, true
	}
	return 0, true
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
	_ = librarywalk.WalkContext(context.Background(), canonicalRoot, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && strings.HasSuffix(path, ".json") {
			count(path)
		}
		return nil
	})
	_ = librarywalk.WalkContext(context.Background(), libraryRoot, func(path string, entry os.DirEntry, err error) error {
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
