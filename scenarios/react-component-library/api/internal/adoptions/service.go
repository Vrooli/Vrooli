package adoptions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"react-component-library/internal/clock"
	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/uimanifest"

	"github.com/google/uuid"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy lives next to the only code that applies it.
const defaultListLimit = 200

// Service is the application-layer surface handlers depend on. Owns
// the Refresh drift policy and any cross-handler validation that does
// not belong in transport.
type Service interface {
	Create(ctx context.Context, in CreateInput) (Adoption, error)
	Apply(ctx context.Context, in ApplyInput) (ApplyResult, error)
	Reapply(ctx context.Context, in ReapplyInput) (Adoption, string, error)
	Reconcile(ctx context.Context, in ReconcileInput) (ReconcileResult, error)
	Reconverge(ctx context.Context, in ReconvergeInput) (ReconvergeResult, error)
	Discover(ctx context.Context, in DiscoverInput) (DiscoverResult, error)
	ConfirmDiscovery(ctx context.Context, in ConfirmDiscoveryInput) (ConfirmDiscoveryResult, error)
	List(ctx context.Context, q ListQuery) ([]Adoption, error)
	ListEffective(ctx context.Context, componentID string, limit int) ([]EffectiveAdoption, error)
	Get(ctx context.Context, id string) (Adoption, error)
	Delete(ctx context.Context, id string) error
	Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error)
}

// RefreshSummary is the counter rollup returned by Refresh — used by
// CLI/UI to render a one-line outcome alongside the per-row table.
type RefreshSummary struct {
	LibraryCurrent    int
	LibraryBehind     int
	LibraryDeprecated int
	LibraryMissing    int
	LibraryUnknown    int
	LocalClean        int
	LocalModified     int
	LocalMissing      int
	LocalUnknown      int
}

// LibraryReader is the components-side seam Refresh needs. Hides the
// full components.Service surface so tests don't need to build one.
type LibraryReader interface {
	Get(ctx context.Context, id string) (components.Component, error)
	List(ctx context.Context, q components.SearchQuery) ([]components.Component, error)
	GetContent(ctx context.Context, id string) (components.Content, error)
	GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error)
	// ListVersions returns every indexed version of a component (entry + unit
	// file bodies populated) so discovery can score a header-less file against
	// each release, not just the latest. limit <= 0 means "all".
	ListVersions(ctx context.Context, componentID string, limit int) ([]components.ComponentVersion, error)
}

// ProvenanceFile is one source file found by a read-only provenance scan.
type ProvenanceFile struct {
	Scenario    string
	AdoptedPath string
	LibraryID   string
	Version     string
	AdoptionID  string
	Content     []byte
}

type ScenarioProvenanceScanner interface {
	ScanProvenance(ctx context.Context) ([]ProvenanceFile, error)
}

// CandidateFile is one header-less source file found by the discovery scan.
// It is the inverse of ProvenanceFile: no @vrooliComponentSource header, so
// only its path and raw content are known — discovery must infer the library
// origin by content similarity.
type CandidateFile struct {
	Scenario    string
	AdoptedPath string
	Content     []byte
}

// ScenarioCandidateScanner walks scenario UI trees and returns the header-less
// source files discovery scores against library versions. It reuses the same
// filesystem walk as ScanProvenance so there is a single scanner, not two.
type ScenarioCandidateScanner interface {
	ScanUntagged(ctx context.Context) ([]CandidateFile, error)
}

// ScenarioFileReader is the target-scenario-tree seam Refresh uses to
// read adopted files. Production maps adopted_path under a configured
// scenarios root with a path-traversal guard; tests inject a fake.
type ScenarioFileReader interface {
	Read(ctx context.Context, scenario, adoptedPath string) ([]byte, error)
}

type ScenarioFileWriter interface {
	ScenarioFileReader
	Exists(ctx context.Context, scenario, adoptedPath string) (bool, error)
	Write(ctx context.Context, scenario, adoptedPath string, content []byte) (string, error)
}

// ScenarioImportSiteFinder reports source files that directly import a target
// within a scenario tree. It is deliberately optional so narrow test fakes can
// retain their read/write-only contract.
type ScenarioImportSiteFinder interface {
	FindImportSites(ctx context.Context, scenario, adoptedPath string) ([]string, error)
}

// ScenarioTokenNamespaceReader resolves the semantic Tailwind vocabulary of a
// target scenario. Adoption owns the translation because the copied file must
// be valid before it reaches the consumer's content glob.
type ScenarioTokenNamespaceReader interface {
	TokenNamespace(ctx context.Context, scenario string) (string, error)
}

type ScenarioTokenMappingReader interface {
	TokenMapping(ctx context.Context, scenario string) (TokenMapping, error)
}

// DependencyValidator and StyleFitValidator keep adoption enforcement at the
// service boundary. Transport callers cannot skip these checks, while tests
// can supply narrow fakes without constructing sibling services.
type DependencyValidator interface {
	ValidateAdoption(ctx context.Context, componentID, version, scenario string) (deps.Verdict, error)
}

type StyleFitValidator interface {
	ValidateStyleFit(ctx context.Context, componentID, version, scenario string) (components.StyleFitVerdict, error)
}

// ErrAdoptedFileMissing is the typed sentinel ScenarioFileReader
// implementations return when the adopted_path does not exist. Refresh
// translates it to StatusUnknown rather than failing the whole batch.
type ErrAdoptedFileMissing struct {
	Scenario    string
	AdoptedPath string
}

func (e ErrAdoptedFileMissing) Error() string {
	return fmt.Sprintf("adopted file %q in scenario %q missing", e.AdoptedPath, e.Scenario)
}

type service struct {
	repo           Repository
	library        LibraryReader
	files          ScenarioFileWriter
	clock          clock.Clock
	reporter       DriftReporter
	logger         *log.Logger
	deps           DependencyValidator
	styles         StyleFitValidator
	tokens         ScenarioTokenNamespaceReader
	mappings       ScenarioTokenMappingReader
	manifestLoader uimanifest.Loader
}

// NewService constructs the production Service. reporter may be nil
// when the swarm-manager integration is disabled (e.g. tests that don't
// exercise drift reporting); a nil reporter is treated as a no-op so
// the rest of the Refresh path is unaffected.
func NewService(repo Repository, lib LibraryReader, files ScenarioFileWriter, clk clock.Clock) Service {
	return &service{repo: repo, library: lib, files: files, clock: clk}
}

// SetDriftReporter installs the swarm-manager drift reporter on an
// existing service. Kept as a setter (vs. NewService param) so handler
// wiring can construct the service first and inject the reporter once
// the swarm-manager CLI path has been resolved.
func SetDriftReporter(svc Service, r DriftReporter, logger *log.Logger) {
	if s, ok := svc.(*service); ok {
		s.reporter = r
		if logger != nil {
			s.logger = logger
		}
	}
}

// SetValidationGates installs the two authoritative pre-apply checks. The
// validators are configured in main after all sibling services exist; keeping
// this separate from NewService preserves the focused construction seam used
// by refresh-only tests.
func SetValidationGates(svc Service, dependency DependencyValidator, style StyleFitValidator) {
	if s, ok := svc.(*service); ok {
		s.deps = dependency
		s.styles = style
	}
}

func SetTokenNamespaceReader(svc Service, reader ScenarioTokenNamespaceReader) {
	if s, ok := svc.(*service); ok {
		s.tokens = reader
		if mappingReader, ok := reader.(ScenarioTokenMappingReader); ok {
			s.mappings = mappingReader
		}
	}
}

// SetManifestLoader supplies the target template's composition contract to
// adoption closure resolution. Keeping it optional preserves focused service
// tests while production always wires the same loader used by placement.
func SetManifestLoader(svc Service, loader uimanifest.Loader) {
	if s, ok := svc.(*service); ok {
		s.manifestLoader = loader
	}
}

// DriftReporter is the seam the service uses to file a backlog item
// when an adoption first transitions to behind/modified. Production
// wires SwarmManagerCLIReporter; tests inject a fake.
//
// Implementations MUST be idempotent at the caller-visible level: if
// Report returns the same ref for the same DriftEvent, callers will
// see deduplication automatically. In practice, the service only
// invokes Report when DriftBacklogRef is empty, so even a non-
// idempotent reporter is safe under normal use.
type DriftReporter interface {
	Report(ctx context.Context, ev DriftEvent) (DriftReport, error)
}

// DriftEvent is the payload Refresh hands to the reporter when it
// detects a fresh drift transition. Fields are flat so a CLI invoker
// can format `--data` JSON without reaching back into the service.
type DriftEvent struct {
	AdoptionID           string
	ComponentID          string
	LibraryID            string
	Scenario             string
	AdoptedPath          string
	AdoptedVersion       string
	LibraryVersion       string
	LibraryVersionStatus LibraryVersionStatus
	LocalStatus          LocalStatus
	StatusDetail         string
}

// DriftReport is the reporter's return shape. Ref is the
// `<kind>/<name>` identifier the service stores on the adoption to
// dedupe future Refresh calls.
type DriftReport struct {
	Ref string
}

var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Adoption, error) {
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.AdoptedPath = strings.TrimSpace(in.AdoptedPath)
	if in.ComponentID == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if in.Scenario == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if in.AdoptedPath == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	// Validate the component exists and echo its libraryId. Snapshot
	// hash is best-effort: if the adopted file is unreadable at create
	// time we still record the row (operator can refresh later).
	cmp, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return Adoption{}, ErrInvalidAdoption{Field: "component_id", Reason: "no component with that id"}
		}
		return Adoption{}, fmt.Errorf("validate component %q: %w", in.ComponentID, err)
	}
	if in.LibraryID == "" {
		in.LibraryID = cmp.LibraryID
	}
	if in.AdoptedSnapshotSHA256 == "" {
		if bytes, err := s.files.Read(ctx, in.Scenario, in.AdoptedPath); err == nil {
			in.AdoptedSnapshotSHA256 = hashBytes(bytes)
		}
	}
	return s.repo.Create(ctx, in)
}

func (s *service) Apply(ctx context.Context, in ApplyInput) (ApplyResult, error) {
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.AdoptedPath = strings.TrimSpace(in.AdoptedPath)
	if in.ComponentID == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if in.Scenario == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if in.AdoptedPath == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	cmp, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return ApplyResult{}, ErrInvalidAdoption{Field: "component_id", Reason: "no component with that id"}
		}
		return ApplyResult{}, err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = cmp.LatestVersion
	}
	if version == "" {
		version = cmp.Version
	}
	if version == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "version", Reason: "component has no latest version"}
	}
	styleFit, err := s.validateAdoption(ctx, in.ComponentID, version, in.Scenario, in.OverrideValidation)
	if err != nil {
		return ApplyResult{}, err
	}
	exists, err := s.files.Exists(ctx, in.Scenario, in.AdoptedPath)
	if err != nil {
		return ApplyResult{}, err
	}
	v, err := s.library.GetVersion(ctx, in.ComponentID, version)
	if err != nil {
		return ApplyResult{}, err
	}
	_ = exists // entry existence is checked together with every unit target below.
	closure, err := s.resolveAdoptionClosure(ctx, cmp, v, in.Scenario, in.IncludeSuggestions)
	if err != nil {
		return ApplyResult{}, err
	}
	plans := adoptionPlansForClosure(closure.Assets, in.AdoptedPath)
	if err := ensureDistinctAdoptionTargets(plans); err != nil {
		return ApplyResult{}, err
	}
	var importSites []string
	now := s.clock.Now().UTC()
	for _, plan := range plans {
		for _, file := range plan.Files {
			exists, err := s.files.Exists(ctx, in.Scenario, file.AdoptedPath)
			if err != nil {
				return ApplyResult{}, err
			}
			if !exists {
				continue
			}
			if !in.ReplaceExisting {
				return ApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "target file already exists; set replace_existing to replace it"}
			}
			existing, err := s.files.Read(ctx, in.Scenario, file.AdoptedPath)
			if err != nil {
				return ApplyResult{}, err
			}
			if hashBytes([]byte(stripSourceHeader(string(existing)))) != hashBytes([]byte(stripSourceHeader(file.Content))) && !in.ConfirmOverwrite {
				return ApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing target differs from the ingested library source"}
			}
		}
	}

	// One adoption represents the operator's root apply. Dependency assets are
	// materialized as files attributed to that root, not independent direct
	// adoptions. This preserves the parent relationship required to distinguish
	// direct component usage from mediated hook usage.
	adoptionID := uuid.NewString()
	adoptionFiles := make([]AdoptionFile, 0)
	written := ""
	entrySnapshot := ""
	tokenMapping, err := s.resolveTokenMapping(ctx, in.Scenario)
	if err != nil {
		return ApplyResult{}, err
	}
	for _, plan := range plans {
		for _, file := range plan.Files {
			fv := plan.Version
			fv.Content, fv.ContentSHA256 = file.Content, file.ContentSHA256
			translated, translations, err := TranslateDesignTokens(stripSourceHeader(file.Content), tokenMapping.Namespace, tokenMapping)
			if err != nil {
				return ApplyResult{}, err
			}
			translationNote := formatTokenTranslations(translations)
			body := formatProvenance(fv, adoptionID, now, hashBytes([]byte(translated)), translationNote) + translated
			path, formattedBody, err := s.writeAdoptedSource(ctx, in.Scenario, file.AdoptedPath, []byte(body))
			if err != nil {
				return ApplyResult{}, err
			}
			body = formattedBody
			if file.IsEntry && plan.Asset.ID == cmp.ID {
				entrySnapshot = adoptedSnapshotHash(body)
				if plan.Asset.ID == cmp.ID {
					written = path
				}
			}
			adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: adoptedSnapshotHash(body), SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
			if finder, ok := s.files.(ScenarioImportSiteFinder); ok {
				sites, err := finder.FindImportSites(ctx, in.Scenario, file.AdoptedPath)
				if err != nil {
					return ApplyResult{}, err
				}
				importSites = append(importSites, sites...)
			}
		}
	}
	experiencePath := ""
	if strings.TrimSpace(v.ExperienceContract) != "" {
		experiencePath = componentExperiencePath(cmp.Slug)
		experienceExists, err := s.files.Exists(ctx, in.Scenario, experiencePath)
		if err != nil {
			return ApplyResult{}, err
		}
		if experienceExists {
			if !in.ReplaceExisting {
				return ApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "component experience contract already exists; set replace_existing to replace it"}
			}
			existing, err := s.files.Read(ctx, in.Scenario, experiencePath)
			if err != nil {
				return ApplyResult{}, err
			}
			if string(existing) != v.ExperienceContract && !in.ConfirmOverwrite {
				return ApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing component experience contract differs from the catalog source"}
			}
		}
		if _, err := s.files.Write(ctx, in.Scenario, experiencePath, []byte(v.ExperienceContract)); err != nil {
			return ApplyResult{}, fmt.Errorf("write component experience contract: %w", err)
		}
	}
	root, err := s.repo.Create(ctx, CreateInput{ID: adoptionID, ComponentID: cmp.ID, LibraryID: cmp.LibraryID, Scenario: in.Scenario, AdoptedPath: in.AdoptedPath, AdoptedVersion: version, SourceSHA256: v.ContentSHA256, AdoptedSnapshotSHA256: entrySnapshot, Files: adoptionFiles})
	if err != nil {
		return ApplyResult{}, err
	}
	copied := make([]string, 0, len(closure.Assets))
	for _, asset := range closure.Assets {
		copied = append(copied, asset.Asset.LibraryID)
	}
	result := ApplyResult{Adoption: root, WrittenPath: written, ExperiencePath: experiencePath, ImportSites: importSites, CopiedAssets: copied, SatisfiedPorts: closure.SatisfiedPorts, AvailableSuggestions: closure.AvailableSuggestions}
	if styleFit != nil {
		result.StyleFitAffinity = styleFit.Affinity
		result.StyleFitDetail = styleFit.Detail
	}
	return result, nil
}

type adoptionPlan struct {
	Asset       components.Component
	Version     components.ComponentVersion
	EntryTarget string
	Files       []adoptionUnitFile
}

func (s *service) resolveAdoptionClosure(ctx context.Context, root components.Component, version components.ComponentVersion, scenario string, includeSuggestions []string) (components.ClosureReport, error) {
	if len(root.Dependencies) == 0 {
		return components.ClosureReport{Assets: []components.ResolvedAsset{{Asset: root, Version: version}}}, nil
	}
	reader, ok := any(s.library).(components.DependencyReader)
	if !ok {
		return components.ClosureReport{}, fmt.Errorf("asset dependency reader is not configured")
	}
	var provides []string
	if s.manifestLoader != nil {
		manifest, err := s.manifestLoader.Load(scenario)
		if err != nil {
			return components.ClosureReport{}, fmt.Errorf("load target UI manifest: %w", err)
		}
		provides = manifest.Provides
	}
	return components.ResolveDependencyClosureReport(ctx, reader, root.ID, version.Version, includeSuggestions, provides, nil)
}

func adoptionPlansForClosure(closure []components.ResolvedAsset, rootTarget string) []adoptionPlan {
	targets := make(map[string]string, len(closure)*2)
	for _, resolved := range closure {
		target := rootTarget
		if resolved.Asset.ID != closure[len(closure)-1].Asset.ID {
			target = dependencyEntryTarget(rootTarget, resolved.Asset, resolved.Version)
		}
		targets[moduleKey(resolved.Version.SourcePath)] = target
		targets[moduleKey(resolved.Asset.DisplayName)] = target
	}
	plans := make([]adoptionPlan, 0, len(closure))
	for _, resolved := range closure {
		target := rootTarget
		if resolved.Asset.ID != closure[len(closure)-1].Asset.ID {
			target = dependencyEntryTarget(rootTarget, resolved.Asset, resolved.Version)
		}
		files := adoptionUnitFiles(resolved.Version, target)
		for i := range files {
			files[i].Content = rewriteUnitImports(files[i].Content, files[i].AdoptedPath, targets)
		}
		plans = append(plans, adoptionPlan{Asset: resolved.Asset, Version: resolved.Version, EntryTarget: target, Files: files})
	}
	return plans
}

func dependencyEntryTarget(rootTarget string, asset components.Component, version components.ComponentVersion) string {
	dir := filepath.ToSlash(filepath.Dir(rootTarget))
	if asset.AssetKind == components.AssetKindHook {
		if strings.HasSuffix(dir, "/components") || dir == "components" {
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
		dir = filepath.ToSlash(filepath.Join(dir, "hooks"))
	} else if asset.AssetKind == components.AssetKindFoundation {
		if strings.HasSuffix(dir, "/components") || dir == "components" {
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
		dir = filepath.ToSlash(filepath.Join(dir, "foundations"))
	} else if strings.HasSuffix(dir, "/components") || dir == "components" {
		dir = filepath.ToSlash(filepath.Dir(dir))
		dir = filepath.ToSlash(filepath.Join(dir, "components"))
	}
	ext := filepath.Ext(version.SourcePath)
	if ext == "" {
		ext = ".ts"
		if asset.AssetKind == components.AssetKindComponent {
			ext = ".tsx"
		}
	}
	return filepath.ToSlash(filepath.Join(dir, filepath.Base(version.SourcePath)))
}

func ensureDistinctAdoptionTargets(plans []adoptionPlan) error {
	seen := map[string]string{}
	for _, plan := range plans {
		for _, file := range plan.Files {
			if owner, exists := seen[file.AdoptedPath]; exists && owner != plan.Asset.LibraryID {
				return fmt.Errorf("asset dependency target collision at %q between %s and %s", file.AdoptedPath, owner, plan.Asset.LibraryID)
			}
			seen[file.AdoptedPath] = plan.Asset.LibraryID
		}
	}
	return nil
}

type adoptionUnitFile struct {
	components.ComponentVersionFile
	AdoptedPath string
}

func adoptionUnitFiles(v components.ComponentVersion, entryTarget string) []adoptionUnitFile {
	files := append([]components.ComponentVersionFile(nil), v.Files...)
	if len(files) == 0 {
		files = []components.ComponentVersionFile{{Path: filepath.Base(v.SourcePath), Content: v.Content, ContentSHA256: v.ContentSHA256, IsEntry: true}}
	}
	if files[0].Path == "" {
		files[0].Path = filepath.Base(entryTarget)
	}
	out := make([]adoptionUnitFile, 0, len(files))
	targets := make(map[string]string, len(files))
	for _, file := range files {
		if file.Path == "story.tsx" || file.Path == "experience-contract.json" {
			continue
		}
		target := filepath.ToSlash(filepath.Join(filepath.Dir(entryTarget), file.Path))
		if file.IsEntry {
			target = entryTarget
		} else if isHookCompanion(file.Path, entryTarget) {
			target = filepath.ToSlash(filepath.Join(filepath.Dir(filepath.Dir(entryTarget)), "hooks", file.Path))
		}
		targets[moduleKey(file.Path)] = target
		out = append(out, adoptionUnitFile{ComponentVersionFile: file, AdoptedPath: target})
	}
	for i := range out {
		out[i].Content = rewriteUnitImports(out[i].Content, out[i].AdoptedPath, targets)
	}
	return out
}

func isHookCompanion(path, entryTarget string) bool {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entryDir := filepath.ToSlash(filepath.Dir(entryTarget))
	return strings.HasPrefix(name, "use") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' &&
		(strings.HasSuffix(entryDir, "/components") || entryDir == "components")
}

func moduleKey(path string) string {
	base := filepath.ToSlash(filepath.Base(path))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

var unitImportRE = regexp.MustCompile(`["']\.[^"']+["']`)

func rewriteUnitImports(body, adoptedPath string, targets map[string]string) string {
	return unitImportRE.ReplaceAllStringFunc(body, func(match string) string {
		quote, specifier := match[:1], match[1:len(match)-1]
		target, ok := targets[moduleKey(specifier)]
		if !ok {
			return match
		}
		rel, err := filepath.Rel(filepath.Dir(adoptedPath), target)
		if err != nil {
			return match
		}
		module := strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		return quote + module + quote
	})
}

func (s *service) Reapply(ctx context.Context, in ReapplyInput) (Adoption, string, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return Adoption{}, "", err
	}
	_, localStatus, _ := s.computeStatus(ctx, row)
	if localStatus == LocalStatusModified && !in.ConfirmLocalOverwrite {
		return Adoption{}, "", ErrInvalidAdoption{Field: "confirm_local_overwrite", Reason: "adopted file has local modifications"}
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		cmp, err := s.library.Get(ctx, row.ComponentID)
		if err != nil {
			return Adoption{}, "", err
		}
		version = firstNonEmpty(cmp.LatestVersion, cmp.Version, row.AdoptedVersion)
	}
	if _, err := s.validateAdoption(ctx, row.ComponentID, version, row.Scenario, in.OverrideValidation); err != nil {
		return Adoption{}, "", err
	}
	root, err := s.library.Get(ctx, row.ComponentID)
	if err != nil {
		return Adoption{}, "", err
	}
	v, err := s.library.GetVersion(ctx, row.ComponentID, version)
	if err != nil {
		return Adoption{}, "", err
	}
	now := s.clock.Now().UTC()
	closure, err := s.resolveAdoptionClosure(ctx, root, v, row.Scenario, nil)
	if err != nil {
		return Adoption{}, "", err
	}
	plans := adoptionPlansForClosure(closure.Assets, row.AdoptedPath)
	adoptionFiles := make([]AdoptionFile, 0)
	written, entrySnapshot := "", ""
	tokenMapping, err := s.resolveTokenMapping(ctx, row.Scenario)
	if err != nil {
		return Adoption{}, "", err
	}
	for _, plan := range plans {
		for _, file := range plan.Files {
			fv := plan.Version
			fv.Content, fv.ContentSHA256 = file.Content, file.ContentSHA256
			translated, translations, err := TranslateDesignTokens(stripSourceHeader(file.Content), tokenMapping.Namespace, tokenMapping)
			if err != nil {
				return Adoption{}, "", err
			}
			body := formatProvenance(fv, row.ID, now, hashBytes([]byte(translated)), formatTokenTranslations(translations)) + translated
			path, formattedBody, err := s.writeAdoptedSource(ctx, row.Scenario, file.AdoptedPath, []byte(body))
			if err != nil {
				return Adoption{}, "", err
			}
			body = formattedBody
			snapshot := adoptedSnapshotHash(body)
			if file.IsEntry && plan.Asset.ID == row.ComponentID {
				written, entrySnapshot = path, snapshot
			}
			adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: snapshot, SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
		}
	}
	if strings.TrimSpace(v.ExperienceContract) != "" {
		experiencePath := componentExperiencePath(root.Slug)
		if _, err := s.files.Write(ctx, row.Scenario, experiencePath, []byte(v.ExperienceContract)); err != nil {
			return Adoption{}, "", fmt.Errorf("write component experience contract: %w", err)
		}
	}
	updated, err := s.repo.UpdateAppliedUnit(ctx, AppliedUnitUpdate{AppliedSnapshotUpdate: AppliedSnapshotUpdate{
		ID:                    row.ID,
		AdoptedVersion:        version,
		SourceSHA256:          v.ContentSHA256,
		AdoptedSnapshotSHA256: entrySnapshot,
		AppliedAt:             now,
	}, Files: adoptionFiles})
	if err != nil {
		return Adoption{}, "", err
	}
	return updated, written, nil
}

// componentExperiencePath is the single canonical location for a copied
// component-scope experience contract. Catalog slugs retain their display
// casing for library source paths, while experience ids and document names
// are stable kebab-case identifiers.
func componentExperiencePath(slug string) string {
	return filepath.ToSlash(filepath.Join("experience", "components", toKebab(slug)+".json"))
}

func (s *service) resolveTokenNamespace(ctx context.Context, scenario string) (string, error) {
	if s.tokens == nil {
		return "app", nil
	}
	namespace, err := s.tokens.TokenNamespace(ctx, scenario)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(namespace) == "" {
		return "app", nil
	}
	return strings.TrimSpace(namespace), nil
}

// writeAdoptedSource returns the exact bytes that the target scenario now
// owns. The production filesystem writer formats through the target UI's
// local toolchain, so snapshots must be derived from the post-format bytes;
// otherwise a successful reapply immediately reports itself as modified.
func (s *service) writeAdoptedSource(ctx context.Context, scenario, adoptedPath string, content []byte) (string, string, error) {
	path, err := s.files.Write(ctx, scenario, adoptedPath, content)
	if err != nil {
		return "", "", err
	}
	actual, err := s.files.Read(ctx, scenario, adoptedPath)
	if err != nil {
		return "", "", fmt.Errorf("read formatted adopted file %q: %w", adoptedPath, err)
	}
	return path, string(actual), nil
}

func (s *service) resolveTokenMapping(ctx context.Context, scenario string) (TokenMapping, error) {
	if s.mappings != nil {
		mapping, err := s.mappings.TokenMapping(ctx, scenario)
		if err != nil {
			return TokenMapping{}, err
		}
		return mapping, nil
	}
	// Unit-test services intentionally omit filesystem wiring. The production
	// handler installs FSScenarioFileReader through SetTokenNamespaceReader,
	// where a missing scenario-owned file is a hard error.
	return TokenMapping{Namespace: "app"}, nil
}

// validateAdoption deliberately executes both checks before deciding whether
// either blocking verdict is allowed. A discouraged style affinity is an
// explicit incompatibility, not advisory copy: applying it without an
// operator override would silently undermine the catalog's style contract.
// An override never bypasses execution of either server-side validation.
func (s *service) validateAdoption(ctx context.Context, componentID, version, scenario string, override bool) (*components.StyleFitVerdict, error) {
	blocked := false
	if s.deps != nil {
		verdict, err := s.deps.ValidateAdoption(ctx, componentID, version, scenario)
		if err != nil {
			return nil, fmt.Errorf("validate adoption dependencies: %w", err)
		}
		blocked = verdict.Kind == deps.VerdictBlock
	}
	var styleFit *components.StyleFitVerdict
	if s.styles != nil {
		verdict, err := s.styles.ValidateStyleFit(ctx, componentID, version, scenario)
		if err != nil {
			return nil, fmt.Errorf("validate adoption style fit: %w", err)
		}
		styleFit = &verdict
		blocked = blocked || verdict.Affinity == components.DesignAffinityDiscouraged
	}
	if blocked && !override {
		return styleFit, ErrAdoptionValidationBlocked{ComponentID: componentID, Version: version, Scenario: scenario}
	}
	return styleFit, nil
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Adoption, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	return s.repo.List(ctx, q)
}

func (s *service) ListEffective(ctx context.Context, componentID string, limit int) ([]EffectiveAdoption, error) {
	if strings.TrimSpace(componentID) == "" {
		return nil, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	return s.repo.ListEffective(ctx, componentID, limit)
}

func (s *service) Get(ctx context.Context, id string) (Adoption, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error) {
	rows, err := s.repo.List(ctx, ListQuery{ComponentID: strings.TrimSpace(componentID), Limit: defaultListLimit})
	if err != nil {
		return nil, RefreshSummary{}, err
	}
	now := s.clock.Now().UTC()
	updates := make([]RefreshUpdate, 0, len(rows))
	summary := RefreshSummary{}
	for i, row := range rows {
		libraryStatus, localStatus, detail := s.computeStatus(ctx, row)
		rows[i].LibraryVersionStatus = libraryStatus
		rows[i].LocalStatus = localStatus
		rows[i].StatusDetail = detail
		rows[i].RefreshedAt = now
		update := RefreshUpdate{
			ID: row.ID, LibraryVersionStatus: libraryStatus, LocalStatus: localStatus, StatusDetail: detail, RefreshedAt: now,
		}
		// Drift policy:
		//   * status flips to behind/modified AND no backlog item filed
		//     yet → call the reporter and store the returned ref so the
		//     next refresh skips it.
		//   * status returns to current → clear the stored ref so a
		//     fresh drift files a new item rather than being silently
		//     swallowed.
		if s.reporter != nil {
			switch {
			case (libraryStatus == LibraryVersionStatusBehind || localStatus == LocalStatusModified) && row.DriftBacklogRef == "":
				ev := s.buildDriftEvent(ctx, rows[i])
				report, err := s.reporter.Report(ctx, ev)
				if err != nil {
					s.logf("drift reporter for adoption %q failed: %v", row.ID, err)
				} else if ref := strings.TrimSpace(report.Ref); ref != "" {
					update.DriftBacklogRef = ref
					rows[i].DriftBacklogRef = ref
				}
			case libraryStatus == LibraryVersionStatusCurrent && localStatus == LocalStatusClean && row.DriftBacklogRef != "":
				update.ClearDriftBacklogRef = true
				rows[i].DriftBacklogRef = ""
			}
		}
		updates = append(updates, update)
		switch libraryStatus {
		case LibraryVersionStatusCurrent:
			summary.LibraryCurrent++
		case LibraryVersionStatusBehind:
			summary.LibraryBehind++
		case LibraryVersionStatusDeprecated:
			summary.LibraryDeprecated++
		case LibraryVersionStatusMissing:
			summary.LibraryMissing++
		case LibraryVersionStatusUnknown:
			summary.LibraryUnknown++
		}
		switch localStatus {
		case LocalStatusClean:
			summary.LocalClean++
		case LocalStatusModified:
			summary.LocalModified++
		case LocalStatusMissing:
			summary.LocalMissing++
		case LocalStatusUnknown:
			summary.LocalUnknown++
		}
	}
	if _, err := s.repo.ApplyRefresh(ctx, updates); err != nil {
		return nil, RefreshSummary{}, err
	}
	return rows, summary, nil
}

// computeStatus is the AD-002 drift decision tree. Pure function of
// the row + the two reader seams.
func (s *service) computeStatus(ctx context.Context, row Adoption) (LibraryVersionStatus, LocalStatus, string) {
	cmp, err := s.library.Get(ctx, row.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return LibraryVersionStatusMissing, LocalStatusUnknown, "component removed from library"
		}
		return LibraryVersionStatusUnknown, LocalStatusUnknown, fmt.Sprintf("library lookup failed: %v", err)
	}
	files := row.Files
	if len(files) == 0 {
		files = []AdoptionFile{{AdoptedPath: row.AdoptedPath, AdoptedSnapshotSHA256: row.AdoptedSnapshotSHA256}}
	}
	localStatus := LocalStatusClean
	detail := ""
	for _, file := range files {
		adoptedBytes, err := s.files.Read(ctx, row.Scenario, file.AdoptedPath)
		if err == nil {
			if file.AdoptedSnapshotSHA256 != "" && adoptedSnapshotHash(string(adoptedBytes)) != file.AdoptedSnapshotSHA256 {
				localStatus, detail = LocalStatusModified, fmt.Sprintf("adopted file %s diverges from applied snapshot", file.AdoptedPath)
				break
			}
			continue
		}
		var missing ErrAdoptedFileMissing
		if errors.As(err, &missing) {
			libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
			return libStatus, LocalStatusMissing, firstNonEmpty("adopted file missing: "+file.AdoptedPath, libDetail)
		}
		libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
		return libStatus, LocalStatusUnknown, firstNonEmpty(fmt.Sprintf("adopted file read failed: %v", err), libDetail)
	}
	libStatus, libDetail := s.libraryStatusFor(ctx, row, cmp)
	if libDetail != "" {
		detail = libDetail
	}
	return libStatus, localStatus, detail
}

// buildDriftEvent assembles the payload handed to the reporter. The
// LibraryVersion lookup is best-effort: if the library reader fails
// (it shouldn't, since computeStatus just succeeded for this row), the
// event still goes out without a library version.
func (s *service) buildDriftEvent(ctx context.Context, row Adoption) DriftEvent {
	ev := DriftEvent{
		AdoptionID:           row.ID,
		ComponentID:          row.ComponentID,
		LibraryID:            row.LibraryID,
		Scenario:             row.Scenario,
		AdoptedPath:          row.AdoptedPath,
		AdoptedVersion:       row.AdoptedVersion,
		LibraryVersionStatus: row.LibraryVersionStatus,
		LocalStatus:          row.LocalStatus,
		StatusDetail:         row.StatusDetail,
	}
	if cmp, err := s.library.Get(ctx, row.ComponentID); err == nil {
		ev.LibraryVersion = firstNonEmpty(cmp.LatestVersion, cmp.Version)
		if ev.LibraryID == "" {
			ev.LibraryID = cmp.LibraryID
		}
	}
	return ev
}

func (s *service) logf(format string, args ...any) {
	if s.logger == nil {
		log.Default().Printf(format, args...)
		return
	}
	s.logger.Printf(format, args...)
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// adoptedSnapshotHash deliberately excludes the generated provenance header.
// A dependency can be shared by multiple root adoptions in one scenario; its
// provenance owner may change while the translated source remains identical.
// Drift must describe source edits, not which root wrote the shared file last.
func adoptedSnapshotHash(body string) string {
	return hashBytes([]byte(stripSourceHeader(body)))
}

func emptyOrVersion(v string) string {
	if v == "" {
		return "(no version)"
	}
	return v
}

// FSScenarioFileReader is the production ScenarioFileReader. Maps
// adopted_path under root/<scenario>/ with a path-traversal guard.
type FSScenarioFileReader struct {
	root string
}

// NewFSScenarioFileReader constructs an FSScenarioFileReader rooted at
// root. Root must be absolute; callers resolve it via api-core/storage.
func NewFSScenarioFileReader(root string) *FSScenarioFileReader {
	return &FSScenarioFileReader{root: root}
}

var (
	_ ScenarioFileReader           = (*FSScenarioFileReader)(nil)
	_ ScenarioProvenanceScanner    = (*FSScenarioFileReader)(nil)
	_ ScenarioCandidateScanner     = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenNamespaceReader = (*FSScenarioFileReader)(nil)
)

// TokenNamespace reads the consumer's declared semantic namespace. The
// scenarios intentionally keep isolated Tailwind configs, so this is a
// filesystem fact rather than a library-global default.
func (r *FSScenarioFileReader) TokenNamespace(_ context.Context, scenario string) (string, error) {
	base := filepath.Join(r.root, scenario, "ui")
	raw, err := os.ReadFile(filepath.Join(base, "tailwind.config.ts"))
	if err != nil {
		if os.IsNotExist(err) {
			return "app", nil
		}
		return "", err
	}
	text := string(raw)
	switch {
	case strings.Contains(text, "wc:"):
		return "wc", nil
	case strings.Contains(text, "slate:"):
		return "slate", nil
	default:
		return "app", nil
	}
}

func (r *FSScenarioFileReader) TokenMapping(_ context.Context, scenario string) (TokenMapping, error) {
	path := filepath.Join(r.root, scenario, "ui", "token-map.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TokenMapping{}, fmt.Errorf("adoption token mapping missing for scenario %q at %s", scenario, path)
		}
		return TokenMapping{}, err
	}
	var mapping TokenMapping
	if err := json.Unmarshal(raw, &mapping); err != nil {
		return TokenMapping{}, fmt.Errorf("decode adoption token mapping for %q: %w", scenario, err)
	}
	if strings.TrimSpace(mapping.Namespace) == "" {
		return TokenMapping{}, fmt.Errorf("adoption token mapping for scenario %q is missing namespace", scenario)
	}
	if err := validateTokenMapping(mapping, []string{"app-danger", "app-info", "app-primary", "app-warning"}); err != nil {
		return TokenMapping{}, fmt.Errorf("validate adoption token mapping for %q: %w", scenario, err)
	}
	return mapping, nil
}

// scannedSource is one .ts/.tsx file yielded by walkScenarioSources: its
// scenario-relative path plus raw bytes. Both ScanProvenance and ScanUntagged
// classify these; neither re-walks the tree.
type scannedSource struct {
	scenario    string
	adoptedPath string
	content     []byte
}

// walkScenarioSources is the single filesystem walk shared by ScanProvenance
// (header-tagged files) and ScanUntagged (header-less candidates). It reads
// every non-symlink .ts/.tsx file under each scenario's ui/src tree, skipping
// node_modules. It never writes.
func (r *FSScenarioFileReader) walkScenarioSources() ([]scannedSource, error) {
	entries, err := os.ReadDir(r.root)
	if err != nil {
		return nil, fmt.Errorf("list scenarios for scan: %w", err)
	}
	var out []scannedSource
	for _, scenario := range entries {
		if !scenario.IsDir() {
			continue
		}
		base := filepath.Join(r.root, scenario.Name(), "ui", "src")
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".tsx")) {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(filepath.Join(r.root, scenario.Name()), path)
			if err != nil {
				return err
			}
			out = append(out, scannedSource{scenario: scenario.Name(), adoptedPath: filepath.ToSlash(rel), content: raw})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan sources for %s: %w", scenario.Name(), err)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].scenario != out[j].scenario {
			return out[i].scenario < out[j].scenario
		}
		return out[i].adoptedPath < out[j].adoptedPath
	})
	return out, nil
}

// ScanProvenance reads only scenario UI source files. It never writes target
// scenarios; reconciliation owns the sole database mutation after this scan.
func (r *FSScenarioFileReader) ScanProvenance(_ context.Context) ([]ProvenanceFile, error) {
	sources, err := r.walkScenarioSources()
	if err != nil {
		return nil, err
	}
	var out []ProvenanceFile
	for _, src := range sources {
		libraryID := provenanceField(string(src.content), "@vrooliComponentSource")
		version := provenanceField(string(src.content), "@vrooliComponentVersion")
		if libraryID == "" || version == "" {
			continue
		}
		out = append(out, ProvenanceFile{Scenario: src.scenario, AdoptedPath: src.adoptedPath, LibraryID: libraryID, Version: version, AdoptionID: provenanceField(string(src.content), "@vrooliComponentAdoption"), Content: src.content})
	}
	return out, nil
}

// ScanUntagged returns the inverse set: every .ts/.tsx file that has NO
// @vrooliComponentSource/@vrooliComponentVersion header. These are the files
// discovery scores for similarity; a real vendored copy that lost its header
// is exactly the drift-blind case this closes.
func (r *FSScenarioFileReader) ScanUntagged(_ context.Context) ([]CandidateFile, error) {
	sources, err := r.walkScenarioSources()
	if err != nil {
		return nil, err
	}
	var out []CandidateFile
	for _, src := range sources {
		libraryID := provenanceField(string(src.content), "@vrooliComponentSource")
		version := provenanceField(string(src.content), "@vrooliComponentVersion")
		if libraryID != "" && version != "" {
			continue // already tagged; ScanProvenance/Reconcile owns these
		}
		out = append(out, CandidateFile{Scenario: src.scenario, AdoptedPath: src.adoptedPath, Content: src.content})
	}
	return out, nil
}

func (r *FSScenarioFileReader) Read(_ context.Context, scenario, adoptedPath string) ([]byte, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	bytes, err := os.ReadFile(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrAdoptedFileMissing{Scenario: scenario, AdoptedPath: adoptedPath}
		}
		return nil, fmt.Errorf("read adopted file %q: %w", cleaned, err)
	}
	return bytes, nil
}

func (r *FSScenarioFileReader) Exists(_ context.Context, scenario, adoptedPath string) (bool, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(cleaned)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (r *FSScenarioFileReader) Write(ctx context.Context, scenario, adoptedPath string, content []byte) (string, error) {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(cleaned), 0o755); err != nil {
		return "", fmt.Errorf("create adopted file dir: %w", err)
	}
	if err := os.WriteFile(cleaned, content, 0o600); err != nil {
		return "", fmt.Errorf("write adopted file %q: %w", adoptedPath, err)
	}
	if err := r.formatWrittenSource(ctx, scenario, adoptedPath, cleaned); err != nil {
		return "", err
	}
	return cleaned, nil
}

// formatWrittenSource keeps copied source readable without imposing a
// library-global formatter on consumers. Each adopting scenario owns its UI
// toolchain; when that toolchain includes Prettier, use that exact binary and
// its config. Scenarios without a formatter remain adoptable, while the
// component-library scenario (and the standard UI template) get formatting
// automatically at the write boundary.
func (r *FSScenarioFileReader) formatWrittenSource(ctx context.Context, scenario, adoptedPath, absolutePath string) error {
	ext := strings.ToLower(filepath.Ext(adoptedPath))
	if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" && ext != ".css" && ext != ".json" {
		return nil
	}
	uiRoot := filepath.Join(r.root, scenario, "ui")
	candidates := []string{
		filepath.Join(uiRoot, "node_modules", ".bin", "prettier"),
		filepath.Join(uiRoot, "node_modules", ".bin", "prettier.cmd"),
		filepath.Join(uiRoot, "node_modules", "prettier", "bin", "prettier.cjs"),
	}
	formatter := ""
	for _, candidate := range candidates {
		if info, statErr := os.Stat(candidate); statErr == nil && !info.IsDir() {
			formatter = candidate
			break
		}
	}
	if formatter == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, formatter, "--write", absolutePath)
	cmd.Dir = uiRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("format adopted file %q with target Prettier: %w: %s", adoptedPath, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// FindImportSites performs a narrow, deterministic import-specifier scan. It
// intentionally reports only direct static/dynamic/require imports; callers
// get actionable replacement evidence without pretending to build a full TS
// dependency graph here.
func (r *FSScenarioFileReader) FindImportSites(_ context.Context, scenario, adoptedPath string) ([]string, error) {
	target, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Join(r.root, scenario)
	sites := make([]string, 0)
	err = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || path == target {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		relTarget, err := filepath.Rel(filepath.Dir(path), target)
		if err != nil {
			return err
		}
		module := strings.TrimSuffix(filepath.ToSlash(relTarget), filepath.Ext(relTarget))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(content)
		if strings.Contains(text, `"`+module+`"`) || strings.Contains(text, `'`+module+`'`) || strings.Contains(text, `"`+filepath.ToSlash(relTarget)+`"`) || strings.Contains(text, `'`+filepath.ToSlash(relTarget)+`'`) {
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			sites = append(sites, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("find import sites for %q: %w", adoptedPath, err)
	}
	sort.Strings(sites)
	return sites, nil
}

// resolve maps (scenario, adoptedPath) onto disk. Scenario is either a plain
// scenario name under root, or a template key ("../templates/scenarios/<id>")
// pointing at a vendored template copy; only adoptedPath is traversal-guarded
// against escaping the resolved scenario dir.
func (r *FSScenarioFileReader) resolve(scenario, adoptedPath string) (string, error) {
	base := filepath.Join(r.root, scenario)
	cleaned := filepath.Clean(filepath.Join(base, adoptedPath))
	rel, err := filepath.Rel(base, cleaned)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("adopted_path %q escapes scenario root", adoptedPath)
	}
	return cleaned, nil
}

func (s *service) libraryStatusFor(ctx context.Context, row Adoption, cmp components.Component) (LibraryVersionStatus, string) {
	status, detail, needsVersion := libraryStatusFor(row, cmp, components.ComponentVersionStatus(""))
	if !needsVersion {
		return status, detail
	}
	version, err := s.library.GetVersion(ctx, row.ComponentID, row.AdoptedVersion)
	if err != nil {
		return LibraryVersionStatusUnknown, fmt.Sprintf("adopted version %s not found in library", emptyOrVersion(row.AdoptedVersion))
	}
	status, detail, _ = libraryStatusFor(row, cmp, version.Status)
	return status, detail
}

func libraryStatusFor(row Adoption, cmp components.Component, adoptedStatus components.ComponentVersionStatus) (LibraryVersionStatus, string, bool) {
	latest := firstNonEmpty(cmp.LatestVersion, cmp.Version)
	if row.AdoptedVersion == "" || latest == "" {
		return LibraryVersionStatusUnknown, "", false
	}
	if adoptedStatus == "" {
		return LibraryVersionStatusUnknown, "", true
	}
	if adoptedStatus == components.VersionStatusDeprecated {
		return LibraryVersionStatusDeprecated, fmt.Sprintf("adopted version %s is deprecated", row.AdoptedVersion), false
	}
	if adoptedStatus == components.VersionStatusDraft {
		return LibraryVersionStatusUnknown, fmt.Sprintf("adopted version %s is a draft", row.AdoptedVersion), false
	}
	if row.AdoptedVersion == latest {
		return LibraryVersionStatusCurrent, "", false
	}
	cmpResult, ok := compareVersionStrings(row.AdoptedVersion, latest)
	if ok && cmpResult >= 0 {
		return LibraryVersionStatusCurrent, "", false
	}
	return LibraryVersionStatusBehind, fmt.Sprintf("library at %s", emptyOrVersion(latest)), false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type semver struct {
	major int
	minor int
	patch int
}

func parseSemver(raw string) (semver, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return semver{}, fmt.Errorf("empty version")
	}
	if i := strings.IndexAny(raw, "-+"); i >= 0 {
		raw = raw[:i]
	}
	parts := strings.SplitN(raw, ".", 3)
	out := semver{}
	dst := []*int{&out.major, &out.minor, &out.patch}
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "x" || part == "X" || part == "*" {
			continue
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			return semver{}, err
		}
		*dst[i] = value
	}
	return out, nil
}

func compareVersionStrings(a, b string) (int, bool) {
	av, err := parseSemver(a)
	if err != nil {
		return 0, false
	}
	bv, err := parseSemver(b)
	if err != nil {
		return 0, false
	}
	switch {
	case av.major != bv.major:
		if av.major < bv.major {
			return -1, true
		}
		return 1, true
	case av.minor != bv.minor:
		if av.minor < bv.minor {
			return -1, true
		}
		return 1, true
	case av.patch != bv.patch:
		if av.patch < bv.patch {
			return -1, true
		}
		return 1, true
	default:
		return 0, true
	}
}

func formatProvenance(v components.ComponentVersion, adoptionID string, appliedAt time.Time, driftHash, translationNote string) string {
	// JSDoc tag names align 1:1 with ui-health's ComponentProvenance proto:
	//   @vrooliComponentSource       -> library
	//   @vrooliComponentVersion      -> library_version
	//   @vrooliComponentAdoption     -> adoption_id
	//   @vrooliComponentAppliedAt    -> applied_at
	//   @vrooliComponentSourceSha256 -> source_sha256
	//   @vrooliComponentDriftHash    -> drift_hash (equal to source_sha256 at
	//                                   adoption time; recomputed at scan time)
	if driftHash == "" {
		driftHash = v.ContentSHA256
	}
	return fmt.Sprintf(`/**
 * @vrooliComponentSource %s
 * @vrooliComponentVersion %s
 * @vrooliComponentAdoption %s
 * @vrooliComponentAppliedAt %s
 * @vrooliComponentSourceSha256 %s
 * @vrooliComponentDriftHash %s
 * @vrooliComponentTokenTranslation %s
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
`, v.LibraryID, v.Version, adoptionID, appliedAt.UTC().Format(time.RFC3339), v.ContentSHA256, driftHash, translationNote)
}

func formatTokenTranslations(translations []TokenTranslation) string {
	if len(translations) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(translations))
	for _, translation := range translations {
		parts = append(parts, translation.From+"->"+translation.To)
	}
	return strings.Join(parts, ",")
}

func stripSourceHeader(src string) string {
	trimmed := strings.TrimLeft(src, "\ufeff \t\r\n")
	if strings.HasPrefix(trimmed, "// @vrooliComponent") || strings.HasPrefix(trimmed, "// @libraryId") {
		if end := strings.IndexByte(trimmed, '\n'); end >= 0 {
			return strings.TrimLeft(trimmed[end+1:], "\r\n")
		}
		return ""
	}
	if !strings.HasPrefix(trimmed, "/**") {
		return src
	}
	end := strings.Index(trimmed, "*/")
	if end < 0 {
		return src
	}
	body := trimmed[:end+2]
	if !strings.Contains(body, "@libraryId") && !strings.Contains(body, "@vrooliComponent") {
		return src
	}
	return strings.TrimLeft(trimmed[end+2:], "\r\n")
}
