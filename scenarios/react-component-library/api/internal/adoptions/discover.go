package adoptions

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/components"

	"github.com/google/uuid"
)

// DefaultDiscoverThreshold is the Sørensen–Dice line-similarity a header-less
// file must reach against a library version to surface as a discovery
// candidate. It is deliberately below the ~0.9 a verbatim header-stripped copy
// scores: discovery surfaces borderline matches for human review rather than
// hiding them, and ConfirmDiscovery — never Discover — performs the write.
const DefaultDiscoverThreshold = 0.6

// minSharedLines guards against tiny files matching purely on shared
// boilerplate (imports, closing braces). A candidate must share at least this
// many normalized lines with the library version regardless of the ratio.
const minSharedLines = 4

// Discover walks scenario UI trees for header-less .ts/.tsx files and scores
// each against every released library component version. It is strictly
// read-only: matches are returned as candidates with similarity evidence, and
// the operator confirms a candidate with ConfirmDiscovery. Auto-writing is
// intentionally impossible here because generic primitives (Input, Button…)
// content-collide with the library's canonical shape.
func (s *service) Discover(ctx context.Context, in DiscoverInput) (DiscoverResult, error) {
	scanner, ok := s.files.(ScenarioCandidateScanner)
	if !ok {
		return DiscoverResult{}, fmt.Errorf("adoptions discover: candidate scanner not configured")
	}
	threshold := in.MinSimilarity
	if threshold <= 0 {
		threshold = DefaultDiscoverThreshold
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	result := DiscoverResult{MinSimilarity: threshold}

	candidates, err := scanner.ScanUntagged(ctx)
	if err != nil {
		return DiscoverResult{}, err
	}
	scenarioFilter := strings.TrimSpace(in.Scenario)
	filtered := candidates[:0]
	for _, c := range candidates {
		if scenarioFilter != "" && c.Scenario != scenarioFilter {
			continue
		}
		filtered = append(filtered, c)
	}
	candidates = filtered
	result.Scanned = len(candidates)
	if len(candidates) == 0 {
		return result, nil
	}

	// Files already tracked by an adoption record are not drift-blind even if
	// their on-disk header was removed; exclude them so discovery only reports
	// genuinely unknown copies.
	existing, err := s.repo.List(ctx, ListQuery{Limit: 100000})
	if err != nil {
		return result, err
	}
	recorded := map[string]bool{}
	for _, row := range existing {
		recorded[provenancePathKey(row.Scenario, row.AdoptedPath)] = true
		for _, file := range row.Files {
			recorded[provenancePathKey(row.Scenario, file.AdoptedPath)] = true
		}
	}

	corpus, err := s.buildLibraryCorpus(ctx)
	if err != nil {
		return result, err
	}

	for _, cand := range candidates {
		if recorded[provenancePathKey(cand.Scenario, cand.AdoptedPath)] {
			continue
		}
		candLines := normalizeLines(string(cand.Content))
		if len(candLines) == 0 {
			continue
		}
		best, ok := corpus.bestMatch(candLines, cand.AdoptedPath)
		if !ok || best.similarity < threshold || best.shared < minSharedLines {
			continue
		}
		result.Candidates = append(result.Candidates, DiscoveryCandidate{
			Scenario:       cand.Scenario,
			AdoptedPath:    cand.AdoptedPath,
			ComponentID:    best.componentID,
			LibraryID:      best.libraryID,
			Version:        best.version,
			DisplayName:    best.displayName,
			Similarity:     best.similarity,
			SharedLines:    best.shared,
			CandidateLines: len(candLines),
			SourceLines:    best.sourceLines,
			BasenameMatch:  best.basenameMatch,
			Evidence:       best.evidence(cand.AdoptedPath, len(candLines)),
		})
	}

	sort.Slice(result.Candidates, func(i, j int) bool {
		a, b := result.Candidates[i], result.Candidates[j]
		if a.Similarity != b.Similarity {
			return a.Similarity > b.Similarity
		}
		if a.Scenario != b.Scenario {
			return a.Scenario < b.Scenario
		}
		return a.AdoptedPath < b.AdoptedPath
	})
	if len(result.Candidates) > limit {
		result.Candidates = result.Candidates[:limit]
	}
	return result, nil
}

// ConfirmDiscovery injects a provenance header into a header-less file and
// creates its adoption record, attributed to the caller-named component +
// version. It re-reads the file (never trusting a stale candidate), verifies
// the file is still header-less, and recomputes the similarity so the stored
// evidence reflects the confirmed bytes.
func (s *service) ConfirmDiscovery(ctx context.Context, in ConfirmDiscoveryInput) (ConfirmDiscoveryResult, error) {
	scenario := strings.TrimSpace(in.Scenario)
	adoptedPath := strings.TrimSpace(in.AdoptedPath)
	componentID := strings.TrimSpace(in.ComponentID)
	version := strings.TrimSpace(in.Version)
	if scenario == "" || adoptedPath == "" || componentID == "" || version == "" {
		return ConfirmDiscoveryResult{}, ErrInvalidAdoption{Field: "confirm_discovery", Reason: "scenario, adopted_path, component_id and version are all required"}
	}

	current, err := s.files.Read(ctx, scenario, adoptedPath)
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	if provenanceField(string(current), "@vrooliComponentSource") != "" {
		return ConfirmDiscoveryResult{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "file already carries a provenance header; use reconcile or refresh"}
	}

	rows, err := s.repo.List(ctx, ListQuery{Scenario: scenario, Limit: 100000})
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	for _, row := range rows {
		if provenancePathKey(row.Scenario, row.AdoptedPath) == provenancePathKey(scenario, adoptedPath) {
			return ConfirmDiscoveryResult{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "an adoption record already tracks this path"}
		}
		for _, file := range row.Files {
			if provenancePathKey(row.Scenario, file.AdoptedPath) == provenancePathKey(scenario, adoptedPath) {
				return ConfirmDiscoveryResult{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "an adoption record already tracks this path"}
			}
		}
	}

	component, err := s.library.Get(ctx, componentID)
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	v, err := s.library.GetVersion(ctx, componentID, version)
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}

	candLines := normalizeLines(string(current))
	libFile, score, shared, ok := bestVersionFile(v, candLines)
	if !ok {
		return ConfirmDiscoveryResult{}, ErrInvalidAdoption{Field: "component_id", Reason: "component version has no comparable source file"}
	}
	_ = shared

	adoptionID := newAdoptionID()
	now := s.clock.Now().UTC()
	fv := v
	fv.LibraryID = component.LibraryID
	fv.Version = version
	fv.ContentSHA256 = libFile.ContentSHA256
	body := formatProvenance(fv, adoptionID, now) + stripSourceHeader(string(current))
	writtenPath, err := s.files.Write(ctx, scenario, adoptedPath, []byte(body))
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	snapshot := hashBytes([]byte(body))
	adoptionFile := AdoptionFile{LibraryPath: libFile.Path, AdoptedPath: adoptedPath, SourceSHA256: libFile.ContentSHA256, AdoptedSnapshotSHA256: snapshot}
	created, err := s.repo.Create(ctx, CreateInput{
		ID:                    adoptionID,
		ComponentID:           component.ID,
		LibraryID:             component.LibraryID,
		Scenario:              scenario,
		AdoptedPath:           adoptedPath,
		AdoptedVersion:        version,
		SourceSHA256:          libFile.ContentSHA256,
		AdoptedSnapshotSHA256: snapshot,
		Files:                 []AdoptionFile{adoptionFile},
	})
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	libraryStatus, localStatus, detail := s.computeStatus(ctx, created)
	if _, err := s.repo.ApplyRefresh(ctx, []RefreshUpdate{{ID: created.ID, LibraryVersionStatus: libraryStatus, LocalStatus: localStatus, StatusDetail: detail, RefreshedAt: now}}); err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	refreshed, err := s.repo.Get(ctx, created.ID)
	if err != nil {
		return ConfirmDiscoveryResult{}, err
	}
	return ConfirmDiscoveryResult{Adoption: refreshed, WrittenPath: writtenPath, Similarity: score}, nil
}

// newAdoptionID is a seam so tests can assert deterministic IDs; production
// uses a fresh UUID. Kept package-private and overridable in tests only.
var newAdoptionID = func() string { return uuid.NewString() }

// libraryCorpus indexes every comparable version file across the library so a
// candidate is scored once per file, not re-fetched per candidate.
type libraryCorpus struct {
	files []corpusFile
}

type corpusFile struct {
	componentID string
	libraryID   string
	version     string
	displayName string
	path        string
	lines       []string
}

func (s *service) buildLibraryCorpus(ctx context.Context) (libraryCorpus, error) {
	list, err := s.library.List(ctx, components.SearchQuery{Limit: 1000})
	if err != nil {
		return libraryCorpus{}, err
	}
	var corpus libraryCorpus
	for _, component := range list {
		versions, err := s.library.ListVersions(ctx, component.ID, 0)
		if err != nil {
			return libraryCorpus{}, err
		}
		for _, v := range versions {
			for _, vf := range comparableVersionFiles(v) {
				corpus.files = append(corpus.files, corpusFile{
					componentID: component.ID,
					libraryID:   component.LibraryID,
					version:     v.Version,
					displayName: component.DisplayName,
					path:        vf.Path,
					lines:       normalizeLines(vf.Content),
				})
			}
		}
	}
	return corpus, nil
}

// matchEvidence is the best library file for a candidate plus the similarity
// numbers used to build human-readable evidence.
type matchEvidence struct {
	componentID   string
	libraryID     string
	version       string
	displayName   string
	path          string
	similarity    float64
	shared        int
	sourceLines   int
	basenameMatch bool
}

func (m matchEvidence) evidence(adoptedPath string, candLines int) []string {
	return []string{
		fmt.Sprintf("Sørensen–Dice line similarity %.3f against %s@%s (%s)", m.similarity, m.libraryID, m.version, m.path),
		fmt.Sprintf("shared %d normalized lines; candidate has %d, source has %d", m.shared, candLines, m.sourceLines),
		fmt.Sprintf("basename %q vs %q matches: %t", filepath.Base(adoptedPath), m.path, m.basenameMatch),
	}
}

func (c libraryCorpus) bestMatch(candLines []string, adoptedPath string) (matchEvidence, bool) {
	base := strings.ToLower(filepath.Base(adoptedPath))
	best := matchEvidence{}
	found := false
	for _, f := range c.files {
		if len(f.lines) == 0 {
			continue
		}
		score, shared := diceSimilarity(candLines, f.lines)
		if score <= 0 {
			continue
		}
		basename := strings.EqualFold(base, strings.ToLower(f.path))
		better := score > best.similarity
		// Tie-break toward a basename match so button.tsx prefers Button.tsx
		// over an equally-scoring sibling primitive.
		if score == best.similarity && basename && !best.basenameMatch {
			better = true
		}
		if !found || better {
			best = matchEvidence{
				componentID:   f.componentID,
				libraryID:     f.libraryID,
				version:       f.version,
				displayName:   f.displayName,
				path:          f.path,
				similarity:    score,
				shared:        shared,
				sourceLines:   len(f.lines),
				basenameMatch: basename,
			}
			found = true
		}
	}
	return best, found
}

// bestVersionFile picks the single version file most similar to the candidate,
// used at confirm time to attribute the per-file source hash.
func bestVersionFile(v components.ComponentVersion, candLines []string) (components.ComponentVersionFile, float64, int, bool) {
	var best components.ComponentVersionFile
	bestScore := -1.0
	bestShared := 0
	found := false
	for _, vf := range comparableVersionFiles(v) {
		score, shared := diceSimilarity(candLines, normalizeLines(vf.Content))
		if !found || score > bestScore {
			best, bestScore, bestShared, found = vf, score, shared, true
		}
	}
	if !found {
		return components.ComponentVersionFile{}, 0, 0, false
	}
	return best, bestScore, bestShared, true
}

// comparableVersionFiles returns the version's unit files, falling back to the
// mirrored entry body when the version has no per-file rows.
func comparableVersionFiles(v components.ComponentVersion) []components.ComponentVersionFile {
	if len(v.Files) > 0 {
		return v.Files
	}
	if strings.TrimSpace(v.Content) == "" {
		return nil
	}
	name := filepath.Base(v.SourcePath)
	if name == "" || name == "." {
		name = v.Version
	}
	return []components.ComponentVersionFile{{Path: name, Content: v.Content, ContentSHA256: v.ContentSHA256, IsEntry: true}}
}

// normalizeLines reduces source text to the multiset of meaningful lines:
// provenance/JSDoc header stripped, each line trimmed with internal whitespace
// collapsed, blank lines dropped. This defeats reformatting and header noise
// without hiding real content differences.
func normalizeLines(src string) []string {
	src = stripSourceHeader(src)
	var out []string
	for _, raw := range strings.Split(src, "\n") {
		line := strings.Join(strings.Fields(raw), " ")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// diceSimilarity is the Sørensen–Dice coefficient over line multisets:
// 2*|A∩B| / (|A|+|B|). Symmetric, bounded [0,1], robust to insertions and
// reordering. Returns the score and the shared-line count.
func diceSimilarity(a, b []string) (float64, int) {
	if len(a) == 0 || len(b) == 0 {
		return 0, 0
	}
	counts := make(map[string]int, len(a))
	for _, line := range a {
		counts[line]++
	}
	shared := 0
	for _, line := range b {
		if counts[line] > 0 {
			counts[line]--
			shared++
		}
	}
	return 2 * float64(shared) / float64(len(a)+len(b)), shared
}
