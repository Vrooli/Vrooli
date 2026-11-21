package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const ArtifactCatalogSchemaVersion = 1

const (
	ArtifactKindCommandOutput  = "command.output"
	ArtifactKindCommandLog     = "command.log"
	ArtifactKindFindingsReport = "findings.report"
	ArtifactKindCoverageReport = "coverage.report"
	ArtifactKindScreenshot     = "screenshot"
	ArtifactKindVisualDiff     = "visual.diff"
	ArtifactKindWorkflowVideo  = "workflow.video"
	ArtifactKindTrace          = "trace"
	ArtifactKindHAR            = "har"
	ArtifactKindConsole        = "console"
	ArtifactKindNetwork        = "network"
	ArtifactKindDOM            = "dom"
	ArtifactKindPhaseResult    = "phase.result"
	ArtifactKindGenericFile    = "generic.file"

	ArtifactAccessStream = "stream"

	ArtifactProvenanceCatalog = "catalog"
	ArtifactProvenanceLegacy  = "legacy_discovery"
	artifactProvenanceRuntime = "runtime_emission"
)

var artifactCatalogMu sync.Mutex

var (
	ErrArtifactCatalogNotFound           = errors.New("run artifact catalog not found")
	ErrInvalidArtifactCatalog            = errors.New("invalid run artifact catalog")
	ErrUnsupportedArtifactCatalogVersion = errors.New("unsupported run artifact catalog version")
	ErrArtifactNotFound                  = errors.New("run artifact not found")
	ErrUnsafeArtifact                    = errors.New("unsafe run artifact")
)

// ArtifactPhaseDeclaration lets the descriptor snapshot assign producer
// metadata without teaching discovery about current phase names.
type ArtifactPhaseDeclaration struct {
	Phase         string
	EvidenceKinds []string
}

type ArtifactRelationship struct {
	Type             string `json:"type"`
	TargetArtifactID string `json:"target_artifact_id"`
}

type ArtifactComparison struct {
	Semantics string            `json:"semantics,omitempty"`
	Analyzer  string            `json:"analyzer,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ArtifactRef is persisted privately. StorageRoot and StoragePath never cross
// the API boundary; public callers receive an explicit projection containing
// the remaining safe metadata and opaque ID.
type ArtifactRef struct {
	ID               string                 `json:"id"`
	Kind             string                 `json:"kind"`
	MediaType        string                 `json:"media_type"`
	Label            string                 `json:"label"`
	ProducingPhase   string                 `json:"producing_phase,omitempty"`
	SizeBytes        int64                  `json:"size_bytes"`
	CreatedAt        string                 `json:"created_at"`
	AccessCapability string                 `json:"access_capability"`
	Metadata         map[string]string      `json:"metadata,omitempty"`
	Relationships    []ArtifactRelationship `json:"relationships,omitempty"`
	Comparison       *ArtifactComparison    `json:"comparison,omitempty"`
	Provenance       string                 `json:"provenance"`
	StorageRoot      string                 `json:"storage_root"`
	StoragePath      string                 `json:"storage_path"`
}

type ArtifactCatalog struct {
	SchemaVersion    int           `json:"schema_version"`
	Digest           string        `json:"digest"`
	RunID            string        `json:"run_id"`
	GeneratedAt      string        `json:"generated_at"`
	LegacyDiscovered bool          `json:"legacy_discovered,omitempty"`
	Artifacts        []ArtifactRef `json:"artifacts"`
}

// ArtifactRegistration is the internal runtime-emission seam. Storage fields
// remain private; Kind intentionally stays open for future provider evidence.
type ArtifactRegistration struct {
	Kind           string
	MediaType      string
	Label          string
	ProducingPhase string
	Metadata       map[string]string
	Relationships  []ArtifactRelationship
	Comparison     *ArtifactComparison
	StorageRoot    string
	StoragePath    string
}

// RegisterArtifact atomically adds or replaces one runtime-emitted instance.
// Final discovery refreshes preserve these explicit semantics while validating
// that the bytes remain in an allowed run-owned root.
func RegisterArtifact(scenarioDir, runID string, registration ArtifactRegistration) (ArtifactRef, error) {
	artifactCatalogMu.Lock()
	defer artifactCatalogMu.Unlock()
	storageRoot := strings.TrimSpace(registration.StorageRoot)
	storagePath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(registration.StoragePath))))
	root := RunDir(scenarioDir, runID)
	if storageRoot == "logs" {
		root = RunLogsDir(scenarioDir, runID)
	} else if storageRoot != "run" {
		return ArtifactRef{}, fmt.Errorf("%w: invalid storage root", ErrInvalidArtifactCatalog)
	}
	resolved, err := validateContainedRegularFile(root, storagePath)
	if err != nil {
		return ArtifactRef{}, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return ArtifactRef{}, err
	}
	kind := strings.ToLower(strings.TrimSpace(registration.Kind))
	if kind == "" {
		return ArtifactRef{}, fmt.Errorf("%w: kind is required", ErrInvalidArtifactCatalog)
	}
	mediaType := strings.TrimSpace(registration.MediaType)
	if mediaType == "" {
		mediaType = artifactMediaType(kind, storagePath)
	}
	label := strings.TrimSpace(registration.Label)
	if label == "" {
		label = artifactLabel(storagePath)
	}
	ref := ArtifactRef{
		ID: artifactID(runID, storageRoot, storagePath), Kind: kind, MediaType: mediaType,
		Label: label, ProducingPhase: strings.TrimSpace(registration.ProducingPhase), SizeBytes: info.Size(),
		CreatedAt: info.ModTime().UTC().Format(time.RFC3339Nano), AccessCapability: ArtifactAccessStream,
		Metadata: cloneMetadata(registration.Metadata), Relationships: append([]ArtifactRelationship(nil), registration.Relationships...),
		Comparison: registration.Comparison, Provenance: artifactProvenanceRuntime,
		StorageRoot: storageRoot, StoragePath: storagePath,
	}
	catalog, readErr := ReadArtifactCatalog(scenarioDir, runID)
	if errors.Is(readErr, ErrArtifactCatalogNotFound) {
		catalog, readErr = DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Now().UTC(), false)
	}
	if readErr != nil {
		return ArtifactRef{}, readErr
	}
	replaced := false
	for i := range catalog.Artifacts {
		if catalog.Artifacts[i].ID == ref.ID {
			catalog.Artifacts[i] = ref
			replaced = true
			break
		}
	}
	if !replaced {
		catalog.Artifacts = append(catalog.Artifacts, ref)
	}
	sort.Slice(catalog.Artifacts, func(i, j int) bool {
		if catalog.Artifacts[i].Kind != catalog.Artifacts[j].Kind {
			return catalog.Artifacts[i].Kind < catalog.Artifacts[j].Kind
		}
		return catalog.Artifacts[i].ID < catalog.Artifacts[j].ID
	})
	catalog.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	catalog.LegacyDiscovered = false
	catalog.Digest = ""
	if err := WriteArtifactCatalog(scenarioDir, catalog); err != nil {
		return ArtifactRef{}, err
	}
	return ref, nil
}

// RefreshArtifactCatalog discovers all current run-owned files while retaining
// explicit runtime-emitted kinds and metadata for matching opaque IDs.
func RefreshArtifactCatalog(scenarioDir, runID string, declarations []ArtifactPhaseDeclaration, generatedAt time.Time) (ArtifactCatalog, error) {
	artifactCatalogMu.Lock()
	defer artifactCatalogMu.Unlock()
	discovered, err := DiscoverArtifactCatalog(scenarioDir, runID, declarations, generatedAt, false)
	if err != nil {
		return ArtifactCatalog{}, err
	}
	existing, readErr := ReadArtifactCatalog(scenarioDir, runID)
	if readErr != nil && !errors.Is(readErr, ErrArtifactCatalogNotFound) {
		return ArtifactCatalog{}, readErr
	}
	if readErr == nil {
		runtimeByID := map[string]ArtifactRef{}
		for _, ref := range existing.Artifacts {
			if ref.Provenance == artifactProvenanceRuntime {
				runtimeByID[ref.ID] = ref
			}
		}
		for i := range discovered.Artifacts {
			if registered, ok := runtimeByID[discovered.Artifacts[i].ID]; ok {
				// Refresh byte-derived fields while preserving emitter-owned type,
				// presentation, relationships, and comparison semantics.
				registered.SizeBytes = discovered.Artifacts[i].SizeBytes
				registered.CreatedAt = discovered.Artifacts[i].CreatedAt
				discovered.Artifacts[i] = registered
			}
		}
	}
	sort.Slice(discovered.Artifacts, func(i, j int) bool {
		if discovered.Artifacts[i].Kind != discovered.Artifacts[j].Kind {
			return discovered.Artifacts[i].Kind < discovered.Artifacts[j].Kind
		}
		return discovered.Artifacts[i].ID < discovered.Artifacts[j].ID
	})
	discovered.Digest = ""
	if err := WriteArtifactCatalog(scenarioDir, discovered); err != nil {
		return ArtifactCatalog{}, err
	}
	return discovered, nil
}

func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// DiscoverArtifactCatalog inventories existing run-owned bytes without moving
// or copying them. It is used both at finalization and as the read-only legacy
// path for runs that predate catalogs.
func DiscoverArtifactCatalog(scenarioDir, runID string, declarations []ArtifactPhaseDeclaration, generatedAt time.Time, legacy bool) (ArtifactCatalog, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return ArtifactCatalog{}, fmt.Errorf("%w: run id is required", ErrInvalidArtifactCatalog)
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	catalog := ArtifactCatalog{
		SchemaVersion:    ArtifactCatalogSchemaVersion,
		RunID:            runID,
		GeneratedAt:      generatedAt.UTC().Format(time.RFC3339Nano),
		LegacyDiscovered: legacy,
		Artifacts:        []ArtifactRef{},
	}
	kindPhases := declarationIndex(declarations)
	roots := []struct {
		name string
		path string
	}{
		{name: "run", path: RunDir(scenarioDir, runID)},
		{name: "logs", path: RunLogsDir(scenarioDir, runID)},
	}
	for _, root := range roots {
		if err := discoverArtifactRoot(&catalog, runID, root.name, root.path, kindPhases, legacy); err != nil {
			return ArtifactCatalog{}, err
		}
	}
	if legacy {
		// A legacy catalog is a read-only projection, not a newly persisted
		// artifact. Anchor its generation time to the newest discovered byte so
		// repeated reads of unchanged historical evidence have one stable digest.
		// Empty legacy runs use the Unix epoch rather than wall-clock time.
		catalog.GeneratedAt = stableLegacyCatalogTime(catalog.Artifacts).Format(time.RFC3339Nano)
	}
	sort.Slice(catalog.Artifacts, func(i, j int) bool {
		if catalog.Artifacts[i].Kind != catalog.Artifacts[j].Kind {
			return catalog.Artifacts[i].Kind < catalog.Artifacts[j].Kind
		}
		return catalog.Artifacts[i].ID < catalog.Artifacts[j].ID
	})
	linkRelatedArtifacts(catalog.Artifacts)
	digest, err := artifactCatalogDigest(catalog)
	if err != nil {
		return ArtifactCatalog{}, err
	}
	catalog.Digest = digest
	return catalog, nil
}

func stableLegacyCatalogTime(artifacts []ArtifactRef) time.Time {
	latest := time.Unix(0, 0).UTC()
	for _, artifact := range artifacts {
		createdAt, err := time.Parse(time.RFC3339Nano, artifact.CreatedAt)
		if err == nil && createdAt.After(latest) {
			latest = createdAt
		}
	}
	return latest
}

func declarationIndex(declarations []ArtifactPhaseDeclaration) map[string][]string {
	index := map[string][]string{}
	for _, declaration := range declarations {
		phase := strings.TrimSpace(declaration.Phase)
		if phase == "" {
			continue
		}
		for _, kind := range declaration.EvidenceKinds {
			kind = strings.ToLower(strings.TrimSpace(kind))
			if kind != "" {
				index[kind] = append(index[kind], phase)
			}
		}
	}
	return index
}

func discoverArtifactRoot(catalog *ArtifactCatalog, runID, storageRoot, root string, kindPhases map[string][]string, legacy bool) error {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect artifact root: %w", err)
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if storageRoot == "run" && isCatalogInternalFile(rel) {
			return nil
		}
		resolved, err := validateContainedRegularFile(root, rel)
		if err != nil {
			// Unsafe symlinks and special files are not advertised as evidence.
			return nil
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil
		}
		kind := classifyArtifact(storageRoot, rel)
		ref := ArtifactRef{
			ID:               artifactID(runID, storageRoot, rel),
			Kind:             kind,
			MediaType:        artifactMediaType(kind, rel),
			Label:            artifactLabel(rel),
			ProducingPhase:   producingPhase(storageRoot, rel, kind, kindPhases),
			SizeBytes:        info.Size(),
			CreatedAt:        info.ModTime().UTC().Format(time.RFC3339Nano),
			AccessCapability: ArtifactAccessStream,
			Metadata:         artifactMetadata(root, storageRoot, rel),
			Provenance:       ArtifactProvenanceCatalog,
			StorageRoot:      storageRoot,
			StoragePath:      rel,
		}
		if legacy {
			ref.Provenance = ArtifactProvenanceLegacy
		}
		if kind == ArtifactKindScreenshot || kind == ArtifactKindVisualDiff {
			// The concrete analyzer is resolved from the provider descriptor by
			// the comparison service. This catalog stores a capability token so
			// artifact discovery remains provider-agnostic.
			ref.Comparison = &ArtifactComparison{Semantics: "advisory", Analyzer: "visual-comparison-provider"}
		}
		catalog.Artifacts = append(catalog.Artifacts, ref)
		return nil
	})
}

func isCatalogInternalFile(rel string) bool {
	switch filepath.Base(filepath.FromSlash(rel)) {
	case ArtifactCatalogFile, RunSnapshotFile, DescriptorSnapshotFile:
		return true
	default:
		return false
	}
}

func classifyArtifact(storageRoot, rel string) string {
	lower := strings.ToLower(filepath.ToSlash(rel))
	base := strings.ToLower(filepath.Base(filepath.FromSlash(rel)))
	ext := strings.ToLower(filepath.Ext(base))
	if storageRoot == "logs" {
		return ArtifactKindCommandLog
	}
	if strings.HasPrefix(lower, PhaseResultsSubdir+"/") && ext == ".json" {
		return ArtifactKindPhaseResult
	}
	if base == FindingsArtifactFile {
		return ArtifactKindFindingsReport
	}
	if strings.Contains(lower, "visual-diff") || strings.Contains(lower, "visual_diff") {
		return ArtifactKindVisualDiff
	}
	if base == "screenshot.png" || strings.Contains(lower, "/screenshots/") {
		return ArtifactKindScreenshot
	}
	if strings.Contains(lower, "/video/") || ext == ".webm" || strings.HasSuffix(base, "-webm") || ext == ".mp4" {
		return ArtifactKindWorkflowVideo
	}
	if strings.Contains(lower, "/trace/") || strings.Contains(base, ".trace.") || ext == ".trace" || ext == ".zip" && strings.Contains(lower, "trace") {
		return ArtifactKindTrace
	}
	if ext == ".har" || base == "har.json" || strings.HasSuffix(base, ".har.json") || strings.Contains(lower, "/har/") {
		return ArtifactKindHAR
	}
	if strings.HasPrefix(base, "console.") {
		return ArtifactKindConsole
	}
	if strings.HasPrefix(base, "network.") {
		return ArtifactKindNetwork
	}
	if strings.HasPrefix(base, "dom.") {
		return ArtifactKindDOM
	}
	if strings.Contains(lower, "coverage") || base == "lcov.info" || strings.HasSuffix(base, ".coverprofile") {
		return ArtifactKindCoverageReport
	}
	if ext == ".log" || ext == ".txt" || ext == ".jsonl" {
		return ArtifactKindCommandOutput
	}
	return ArtifactKindGenericFile
}

func artifactMediaType(kind, rel string) string {
	switch kind {
	case ArtifactKindWorkflowVideo:
		if strings.EqualFold(filepath.Ext(rel), ".mp4") {
			return "video/mp4"
		}
		return "video/webm"
	case ArtifactKindScreenshot:
		return "image/png"
	case ArtifactKindVisualDiff:
		if detected := mime.TypeByExtension(filepath.Ext(rel)); detected != "" {
			return detected
		}
		return "application/octet-stream"
	case ArtifactKindDOM:
		if strings.EqualFold(filepath.Ext(rel), ".html") || strings.EqualFold(filepath.Ext(rel), ".htm") {
			return "text/html; charset=utf-8"
		}
	case ArtifactKindCommandLog, ArtifactKindCommandOutput:
		return "text/plain; charset=utf-8"
	case ArtifactKindTrace:
		if strings.EqualFold(filepath.Ext(rel), ".zip") || filepath.Ext(rel) == "" {
			return "application/zip"
		}
		if strings.EqualFold(filepath.Ext(rel), ".jsonl") {
			return "application/x-ndjson"
		}
	}
	if detected := mime.TypeByExtension(filepath.Ext(rel)); detected != "" {
		return detected
	}
	return "application/octet-stream"
}

func artifactLabel(rel string) string {
	base := filepath.Base(filepath.FromSlash(rel))
	if base == "" || base == "." {
		return "Artifact"
	}
	return base
}

func artifactMetadata(root, storageRoot, rel string) map[string]string {
	metadata := map[string]string{}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if storageRoot == "run" && len(parts) >= 3 && parts[0] == AutomationSubdir {
		metadata["workflow"] = parts[1]
	}
	if storageRoot == "run" && len(parts) >= 4 && parts[0] == UISmokeSubdir && parts[1] == UISmokePagesSubdir {
		metadata["page_key"] = parts[2]
		pagePath := filepath.Join(root, filepath.FromSlash(strings.Join(parts[:3], "/")), visualPageFile)
		if raw, err := os.ReadFile(pagePath); err == nil {
			var page visualPageMeta
			if json.Unmarshal(raw, &page) == nil {
				if page.Page != "" {
					metadata["page"] = page.Page
				}
				if page.Label != "" {
					metadata["page_label"] = page.Label
				}
			}
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func producingPhase(storageRoot, rel, kind string, kindPhases map[string][]string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if storageRoot == "logs" && len(parts) > 0 {
		return strings.TrimSuffix(parts[0], filepath.Ext(parts[0]))
	}
	if len(parts) == 2 && parts[0] == PhaseResultsSubdir {
		return strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
	}
	if phases := kindPhases[kind]; len(phases) == 1 {
		return phases[0]
	}
	return ""
}

func linkRelatedArtifacts(artifacts []ArtifactRef) {
	groups := map[string][]int{}
	for i := range artifacts {
		ref := artifacts[i]
		group := ""
		if pageKey := ref.Metadata["page_key"]; pageKey != "" {
			group = "page:" + pageKey
		} else if workflow := ref.Metadata["workflow"]; workflow != "" {
			group = "workflow:" + workflow
		}
		if group == "" {
			continue
		}
		groups[group] = append(groups[group], i)
	}
	for _, indices := range groups {
		if len(indices) < 2 {
			continue
		}
		primary := indices[0]
		for _, index := range indices {
			if artifacts[index].Kind == ArtifactKindWorkflowVideo || artifacts[index].Kind == ArtifactKindScreenshot {
				primary = index
				break
			}
		}
		for _, index := range indices {
			if index == primary {
				continue
			}
			artifacts[index].Relationships = append(artifacts[index].Relationships, ArtifactRelationship{Type: "related", TargetArtifactID: artifacts[primary].ID})
			artifacts[primary].Relationships = append(artifacts[primary].Relationships, ArtifactRelationship{Type: "related", TargetArtifactID: artifacts[index].ID})
		}
	}
}

func artifactID(runID, storageRoot, rel string) string {
	sum := sha256.Sum256([]byte(runID + "\x00" + storageRoot + "\x00" + filepath.ToSlash(rel)))
	return "artifact_" + hex.EncodeToString(sum[:16])
}

func WriteArtifactCatalog(scenarioDir string, catalog ArtifactCatalog) error {
	if err := validateArtifactCatalog(catalog); err != nil {
		return err
	}
	expected, err := artifactCatalogDigest(catalog)
	if err != nil {
		return err
	}
	if catalog.Digest == "" {
		catalog.Digest = expected
	}
	if catalog.Digest != expected {
		return fmt.Errorf("%w: digest mismatch", ErrInvalidArtifactCatalog)
	}
	raw, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal artifact catalog: %w", err)
	}
	return writeAtomicArtifactCatalog(RunArtifactCatalogPath(scenarioDir, catalog.RunID), raw)
}

func ReadArtifactCatalog(scenarioDir, runID string) (ArtifactCatalog, error) {
	raw, err := os.ReadFile(RunArtifactCatalogPath(scenarioDir, runID))
	if err != nil {
		if os.IsNotExist(err) {
			return ArtifactCatalog{}, ErrArtifactCatalogNotFound
		}
		return ArtifactCatalog{}, fmt.Errorf("read artifact catalog: %w", err)
	}
	var catalog ArtifactCatalog
	if err := json.Unmarshal(raw, &catalog); err != nil {
		return ArtifactCatalog{}, fmt.Errorf("%w: decode: %v", ErrInvalidArtifactCatalog, err)
	}
	if err := validateArtifactCatalog(catalog); err != nil {
		return ArtifactCatalog{}, err
	}
	if catalog.RunID != strings.TrimSpace(runID) {
		return ArtifactCatalog{}, fmt.Errorf("%w: catalog run_id %q does not match requested run %q", ErrInvalidArtifactCatalog, catalog.RunID, strings.TrimSpace(runID))
	}
	expected, err := artifactCatalogDigest(catalog)
	if err != nil {
		return ArtifactCatalog{}, err
	}
	if catalog.Digest == "" || catalog.Digest != expected {
		return ArtifactCatalog{}, fmt.Errorf("%w: digest mismatch", ErrInvalidArtifactCatalog)
	}
	return catalog, nil
}

func validateArtifactCatalog(catalog ArtifactCatalog) error {
	if catalog.SchemaVersion != ArtifactCatalogSchemaVersion {
		return fmt.Errorf("%w: got %d", ErrUnsupportedArtifactCatalogVersion, catalog.SchemaVersion)
	}
	if strings.TrimSpace(catalog.RunID) == "" {
		return fmt.Errorf("%w: run_id is required", ErrInvalidArtifactCatalog)
	}
	if _, err := time.Parse(time.RFC3339Nano, catalog.GeneratedAt); err != nil {
		return fmt.Errorf("%w: invalid generated_at", ErrInvalidArtifactCatalog)
	}
	seen := map[string]struct{}{}
	for _, ref := range catalog.Artifacts {
		if ref.ID == "" || ref.ID != artifactID(catalog.RunID, ref.StorageRoot, ref.StoragePath) {
			return fmt.Errorf("%w: invalid opaque id", ErrInvalidArtifactCatalog)
		}
		if _, duplicate := seen[ref.ID]; duplicate {
			return fmt.Errorf("%w: duplicate artifact id %q", ErrInvalidArtifactCatalog, ref.ID)
		}
		seen[ref.ID] = struct{}{}
		if ref.Kind == "" || strings.TrimSpace(ref.Label) == "" || ref.AccessCapability != ArtifactAccessStream {
			return fmt.Errorf("%w: incomplete artifact %q", ErrInvalidArtifactCatalog, ref.ID)
		}
		if _, err := time.Parse(time.RFC3339Nano, ref.CreatedAt); err != nil {
			return fmt.Errorf("%w: invalid created_at for %q", ErrInvalidArtifactCatalog, ref.ID)
		}
		switch ref.Provenance {
		case ArtifactProvenanceCatalog, ArtifactProvenanceLegacy, artifactProvenanceRuntime:
		default:
			return fmt.Errorf("%w: invalid provenance for %q", ErrInvalidArtifactCatalog, ref.ID)
		}
		if _, _, err := mime.ParseMediaType(ref.MediaType); err != nil {
			return fmt.Errorf("%w: invalid media type for %q", ErrInvalidArtifactCatalog, ref.ID)
		}
		if ref.StorageRoot != "run" && ref.StorageRoot != "logs" {
			return fmt.Errorf("%w: invalid storage root", ErrInvalidArtifactCatalog)
		}
		cleanPath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(ref.StoragePath)))
		if ref.StoragePath == "" || filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("%w: invalid storage path", ErrInvalidArtifactCatalog)
		}
	}
	return nil
}

func artifactCatalogDigest(catalog ArtifactCatalog) (string, error) {
	payload := struct {
		SchemaVersion    int           `json:"schema_version"`
		RunID            string        `json:"run_id"`
		GeneratedAt      string        `json:"generated_at"`
		LegacyDiscovered bool          `json:"legacy_discovered,omitempty"`
		Artifacts        []ArtifactRef `json:"artifacts"`
	}{catalog.SchemaVersion, catalog.RunID, catalog.GeneratedAt, catalog.LegacyDiscovered, catalog.Artifacts}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal artifact catalog digest: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "ac:sha256:" + hex.EncodeToString(sum[:]), nil
}

func writeAtomicArtifactCatalog(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create artifact catalog dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".artifact-catalog-*.tmp")
	if err != nil {
		return fmt.Errorf("create artifact catalog tmp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write artifact catalog tmp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod artifact catalog tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync artifact catalog tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close artifact catalog tmp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace artifact catalog: %w", err)
	}
	if handle, err := os.Open(dir); err == nil {
		_ = handle.Sync()
		_ = handle.Close()
	}
	return nil
}

// ResolveCatalogArtifact resolves bytes only after an opaque ID is found in
// this run's verified catalog. A foreign run's ID therefore cannot be reused.
func ResolveCatalogArtifact(scenarioDir, runID, artifactID string, declarations []ArtifactPhaseDeclaration) (ArtifactRef, string, error) {
	catalog, err := ReadArtifactCatalog(scenarioDir, runID)
	if errors.Is(err, ErrArtifactCatalogNotFound) {
		catalog, err = DiscoverArtifactCatalog(scenarioDir, runID, declarations, time.Now().UTC(), true)
	}
	if err != nil {
		return ArtifactRef{}, "", err
	}
	artifactID = strings.TrimSpace(artifactID)
	for _, ref := range catalog.Artifacts {
		if ref.ID != artifactID {
			continue
		}
		root := RunDir(scenarioDir, runID)
		if ref.StorageRoot == "logs" {
			root = RunLogsDir(scenarioDir, runID)
		}
		path, err := validateContainedRegularFile(root, ref.StoragePath)
		if err != nil {
			return ArtifactRef{}, "", err
		}
		return ref, path, nil
	}
	return ArtifactRef{}, "", ErrArtifactNotFound
}

func validateContainedRegularFile(root, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		return "", fmt.Errorf("%w: invalid artifact locator", ErrUnsafeArtifact)
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: invalid artifact root", ErrUnsafeArtifact)
	}
	candidate := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(rel)))
	if !pathWithin(cleanRoot, candidate) {
		return "", fmt.Errorf("%w: artifact escapes run root", ErrUnsafeArtifact)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrArtifactNotFound
		}
		return "", fmt.Errorf("%w: resolve artifact: %v", ErrUnsafeArtifact, err)
	}
	if !pathWithin(cleanRoot, resolved) {
		return "", fmt.Errorf("%w: symlink escapes run root", ErrUnsafeArtifact)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrArtifactNotFound
		}
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%w: artifact is not a regular file", ErrUnsafeArtifact)
	}
	return resolved, nil
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}
