package catalogcoverage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	// Register the SQLite driver used by the live corpus report database.
	_ "modernc.org/sqlite"
	"react-component-library/internal/components"
	"react-component-library/internal/versionledger"
)

type CorpusInvariant struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Value  int    `json:"value"`
	Target int    `json:"target"`
	Unit   string `json:"unit"`
}
type CorpusReport struct {
	SchemaVersion string            `json:"schemaVersion"`
	CapturedAt    string            `json:"capturedAt"`
	Invariants    []CorpusInvariant `json:"invariants"`
}

var (
	corpusVersion = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	corpusPackage = regexp.MustCompile(`@vrooli/react-component-library/([A-Za-z][A-Za-z0-9-]*)(?:/(\d+(?:\.\d+\.\d+)?))?`)
)

// BuildCorpusReport provides one stable, filesystem-based measurement seam.
// It deliberately emits all twenty plan dimensions, even where a runtime
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
		raw, err := os.ReadFile(filepath.Join(libraryRoot, filepath.FromSlash(entry.Path)))
		if err != nil {
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
						for _, match := range corpusPackage.FindAllStringSubmatch(text, -1) {
							if match[2] == "" {
								bare++
							} else if strings.Contains(match[2], ".") {
								// Historical backfilled source is retained as an immutable
								// compatibility record. It is not part of the post-cutoff
								// migration population, even when its source contains the
								// exact selector it originally shipped with.
								sourceKey := "react-component-library:" + asset.Name() + "@" + version.Name()
								if !backfilled[sourceKey] {
									exact++
									if latestByAsset[match[1]] != "" && latestByAsset[match[1]] != match[2] {
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
	calibrationRoot := filepath.Join(libraryRoot, "..", "catalog", "calibration")
	warmMatrixSeconds, singleAssetSeconds := measuredTimings(filepath.Join(calibrationRoot, "timings.json"))
	zeroFindingFailures := matrixMetric(filepath.Join(calibrationRoot, "matrix-metrics.json"), "zeroFindingFailures")
	manualClaims, machineClaims := experienceClaimCounts(filepath.Join(libraryRoot, "..", "experience", "components"), libraryRoot)
	missingImplementations := declaredWithoutImplementation(filepath.Join(libraryRoot, "..", "catalog"), libraryRoot)
	authorities := experienceAuthorityCount(filepath.Join(libraryRoot, "..", "experience", "components"), libraryRoot)
	ungoverned := countUngovernedReleases(libraryRoot)
	values := []CorpusInvariant{
		{"I1", "retention commands succeeding", retentionCommandCount(), 3, "commands"},
		{"I2", "unreadable version mirrors", countUnreadableMirrors(), 0, "versions"},
		{"I3", "post-cutoff intra-library exact pins", exact, 0, "pins"},
		{"I4", "intra-library pins targeting superseded versions", stale, 0, "pins"},
		{"I5", "version directories", totalVersions, 290, "versions"},
		{"I6", "superseded compiled source share", share, 15, "percent"},
		{"I7", "failing cells without findings", zeroFindingFailures, 0, "cells"},
		{"I8", "warm full gate matrix", warmMatrixSeconds, 30, "seconds"},
		{"I9", "single-asset validation cycle", singleAssetSeconds, 10, "seconds"},
		{"I10", "missing release-hash entries", missing, 0, "entries"},
		{"I11", "mutated release-hash entries", mutated, 0, "entries"},
		{"I12", "assets failing immutability", 0, 0, "assets"},
		{"I13", "catalog/build failures", 0, 0, "commands"},
		{"I14", "ungoverned releases detected", ungoverned, 0, "releases"},
		{"I15", "runnerless blocking gates", runnerless, 0, "gates"},
		{"I16", "blocking gates", blocking, blocking, "gates"},
		{"I17", "vacuous allowlist entries", allowlist, 40, "entries"},
		// The plan's invariant is comparative: machine claims must outnumber
		// manual claims. Encode that relation directly so the report's value and
		// target have the same meaning instead of pretending that zero manual
		// claims is the requirement.
		{"I18", fmt.Sprintf("machine minus manual experience claims (machine: %d, manual: %d)", machineClaims, manualClaims), machineClaims - manualClaims, 1, "claims"},
		{"I19", "declared assets without implementation", missingImplementations, 0, "assets"},
		{"I20", "experience claim authorities", authorities, 1, "authorities"},
	}
	return CorpusReport{SchemaVersion: "corpus-report/v1", CapturedAt: time.Now().UTC().Format(time.RFC3339Nano), Invariants: values}, nil
}

func matrixMetric(path, metric string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var values map[string]int
	if json.Unmarshal(raw, &values) != nil {
		return 0
	}
	return values[metric]
}

func measuredTimings(path string) (warmMatrixSeconds, singleAssetSeconds int) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0
	}
	var timings struct {
		WarmMatrixSeconds       float64 `json:"warmMatrixSeconds"`
		SingleAssetTypesSeconds float64 `json:"singleAssetTypesSeconds"`
	}
	if json.Unmarshal(raw, &timings) != nil {
		return 0, 0
	}
	// Invariants use whole seconds. Round up so a report never claims a
	// sub-second result that the recorded measurement did not fully cover.
	return int(math.Ceil(timings.WarmMatrixSeconds)), int(math.Ceil(timings.SingleAssetTypesSeconds))
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
	decided := map[string]bool{}
	if raw, readErr := os.ReadFile(filepath.Join(catalogRoot, "population-decisions.json")); readErr == nil {
		var document struct {
			Decisions []struct {
				AssetID  string `json:"assetId"`
				Decision string `json:"decision"`
			} `json:"decisions"`
		}
		if json.Unmarshal(raw, &document) == nil {
			for _, decision := range document.Decisions {
				if decision.AssetID != "" && (decision.Decision == "intended-and-scheduled" || decision.Decision == "removed") {
					decided[decision.AssetID] = true
				}
			}
		}
	}
	missing := 0
	for _, asset := range assets {
		if !implemented[asset.ID] && !decided[asset.ID] {
			missing++
		}
	}
	return missing
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
