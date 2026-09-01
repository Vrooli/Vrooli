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

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/themes"
	"react-component-library/internal/uimanifest"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy lives next to the only code that applies it.
const defaultListLimit = 200

// refreshListLimit is deliberately larger than the interactive list limit.
// Refresh is the fleet-wide reconciliation operation; silently truncating it
// would leave a false impression that the registry is fully classified.
const refreshListLimit = 100000

// Service is the application-layer surface handlers depend on. Owns
// the Refresh drift policy and any cross-handler validation that does
// not belong in transport.
type Service interface {
	Create(ctx context.Context, in CreateInput) (Adoption, error)
	Link(ctx context.Context, in LinkInput) (LinkResult, error)
	Eject(ctx context.Context, in EjectInput) (ApplyResult, error)
	Apply(ctx context.Context, in ApplyInput) (ApplyResult, error)
	BatchApply(ctx context.Context, in BatchApplyInput) (BatchApplyResult, error)
	Preflight(ctx context.Context, in PreflightInput) (PreflightResult, error)
	SyncScenarioTokens(ctx context.Context, in TokenSyncInput) (TokenSyncResult, error)
	PruneScenarioTokens(ctx context.Context, in TokenPruneInput) (TokenPruneResult, error)
	Reapply(ctx context.Context, in ReapplyInput) (Adoption, string, error)
	Reconcile(ctx context.Context, in ReconcileInput) (ReconcileResult, error)
	Reconverge(ctx context.Context, in ReconvergeInput) (ReconvergeResult, error)
	Discover(ctx context.Context, in DiscoverInput) (DiscoverResult, error)
	ConfirmDiscovery(ctx context.Context, in ConfirmDiscoveryInput) (ConfirmDiscoveryResult, error)
	List(ctx context.Context, q ListQuery) ([]Adoption, error)
	ListEffective(ctx context.Context, componentID string, limit int) ([]EffectiveAdoption, error)
	Get(ctx context.Context, id string) (Adoption, error)
	Delete(ctx context.Context, id string) error
	DeleteWithOptions(ctx context.Context, id string, confirmRemoveFiles bool) (DeleteResult, error)
	Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error)
	ForkReport(ctx context.Context, componentID string, apply bool) ([]Adoption, RefreshSummary, error)
}

// BatchApply validates every root and its closure before writing anything.
// Shared dependency files are materialized once, while every root receives
// provenance rows for the files in its effective closure. The production
// repository commits those rows in one SQLite transaction; scenario files are
// restored from snapshots if either the write or transaction fails.
func (s *service) BatchApply(ctx context.Context, in BatchApplyInput) (BatchApplyResult, error) {
	if len(in.Items) == 0 {
		return BatchApplyResult{}, ErrInvalidAdoption{Field: "items", Reason: "at least one batch item is required"}
	}

	type preparedItem struct {
		input          BatchApplyItem
		root           components.Component
		version        components.ComponentVersion
		closure        components.ClosureReport
		plans          []adoptionPlan
		verdict        AdoptionVerdict
		adoptionID     string
		experiencePath string
		entrySnapshot  string
		writtenPath    string
		importSites    []string
	}
	type targetOwner struct {
		itemIndex int
		assetID   string
		version   string
		isRoot    bool
	}
	type uniqueFile struct {
		scenario string
		path     string
		file     adoptionUnitFile
		plan     adoptionPlan
		owner    int
		body     string
		written  string
	}

	prepared := make([]preparedItem, 0, len(in.Items))
	seenDependency := map[string]struct {
		rootID  string
		version string
	}{}
	shared := map[string]struct{}{}
	seenTargets := map[string]targetOwner{}
	targetControls := map[string]struct {
		replaceExisting  bool
		confirmOverwrite bool
	}{}
	uniqueFiles := map[string]*uniqueFile{}
	snapshots := map[string]batchFileSnapshot{}

	for index, raw := range in.Items {
		item := raw
		item.ComponentID = strings.TrimSpace(item.ComponentID)
		item.Scenario = strings.TrimSpace(item.Scenario)
		item.AdoptedPath = strings.TrimSpace(item.AdoptedPath)
		if item.ComponentID == "" || item.Scenario == "" || item.AdoptedPath == "" {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "component_id/scenario/adopted_path", Reason: "all are required"}
		}
		root, err := s.library.Get(ctx, item.ComponentID)
		if err != nil {
			return BatchApplyResult{}, err
		}
		versionName := strings.TrimSpace(item.Version)
		if versionName == "" {
			versionName = firstNonEmpty(root.LatestVersion, root.Version)
		}
		version, err := s.library.GetVersion(ctx, root.ID, versionName)
		if err != nil {
			return BatchApplyResult{}, err
		}
		closure, err := s.resolveAdoptionClosure(ctx, root, version, item.Scenario, item.IncludeSuggestions)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if _, err := s.requireTokenVerdict(ctx, root.ID, item.Scenario, closure, item.OverrideValidation); err != nil {
			return BatchApplyResult{}, err
		}
		verdict, err := s.adoptionVerdict(ctx, root, version, closure, item.Scenario)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if verdict.Blocking() && !item.OverrideValidation {
			return BatchApplyResult{}, readinessBlockedError(root.ID, item.Scenario, verdict)
		}
		plans := adoptionPlansForClosure(closure.Assets, item.AdoptedPath)
		if err := ensureDistinctAdoptionTargets(plans); err != nil {
			return BatchApplyResult{}, err
		}
		prepared = append(prepared, preparedItem{input: item, root: root, version: version, closure: closure, plans: plans, verdict: verdict, adoptionID: uuid.NewString()})
		current := &prepared[len(prepared)-1]
		if version.ExperienceContract != "" {
			current.experiencePath = cmpExperiencePath(root)
		}
		if err := s.captureBatchSnapshots(ctx, item.Scenario, current.experiencePath, plans, snapshots); err != nil {
			return BatchApplyResult{}, err
		}

		for _, plan := range plans {
			key := plan.Asset.ID
			if prior, ok := seenDependency[key]; ok {
				if prior.version != plan.Version.Version {
					return BatchApplyResult{}, ErrBatchDependencyConflict{Dependency: plan.Asset.LibraryID, FirstRoot: prior.rootID, FirstVersion: prior.version, SecondRoot: root.ID, SecondVersion: plan.Version.Version}
				}
				if prior.rootID != root.ID {
					shared[plan.Asset.LibraryID] = struct{}{}
				}
			} else {
				seenDependency[key] = struct {
					rootID  string
					version string
				}{rootID: root.ID, version: plan.Version.Version}
			}
			for _, file := range plan.Files {
				targetKey := batchSnapshotKey(item.Scenario, file.AdoptedPath)
				owner := targetOwner{itemIndex: index, assetID: plan.Asset.ID, version: plan.Version.Version, isRoot: plan.Asset.ID == root.ID}
				if prior, ok := seenTargets[targetKey]; ok {
					sharedAsset := prior.assetID == owner.assetID && prior.version == owner.version && !prior.isRoot && !owner.isRoot
					if !sharedAsset {
						return BatchApplyResult{}, fmt.Errorf("batch target collision at %q between %s and %s", file.AdoptedPath, prepared[prior.itemIndex].root.ID, root.ID)
					}
				} else {
					seenTargets[targetKey] = owner
					targetControls[targetKey] = struct {
						replaceExisting  bool
						confirmOverwrite bool
					}{replaceExisting: item.ReplaceExisting, confirmOverwrite: item.ConfirmOverwrite}
					uniqueFiles[targetKey] = &uniqueFile{scenario: item.Scenario, path: file.AdoptedPath, file: file, plan: plan, owner: index}
				}
				if prior, exists := seenTargets[targetKey]; exists && prior.itemIndex != index {
					controls := targetControls[targetKey]
					controls.replaceExisting = controls.replaceExisting && item.ReplaceExisting
					controls.confirmOverwrite = controls.confirmOverwrite && item.ConfirmOverwrite
					targetControls[targetKey] = controls
				}
			}
		}
	}

	// Validate existing targets once, using the strictest overwrite controls
	// from all roots that reference a shared target.
	for key := range seenTargets {
		file := uniqueFiles[key]
		if file == nil {
			continue
		}
		controls := targetControls[key]
		exists, err := s.files.Exists(ctx, file.scenario, file.path)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if !exists {
			continue
		}
		if !controls.replaceExisting {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "target file already exists; set replace_existing to replace it"}
		}
		existing, err := s.files.Read(ctx, file.scenario, file.path)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if hashBytes([]byte(stripSourceHeader(string(existing)))) != hashBytes([]byte(stripSourceHeader(file.file.Content))) && !controls.confirmOverwrite {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing target differs from the ingested library source"}
		}
	}

	// Experience contracts are root-owned files, so validate them separately.
	for i := range prepared {
		if prepared[i].experiencePath == "" {
			continue
		}
		exists, err := s.files.Exists(ctx, prepared[i].input.Scenario, prepared[i].experiencePath)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if !exists || prepared[i].input.ReplaceExisting {
			continue
		}
		return BatchApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "component experience contract already exists; set replace_existing to replace it"}
	}

	mappings := map[string]TokenMapping{}
	for i := range prepared {
		if _, ok := mappings[prepared[i].input.Scenario]; ok {
			continue
		}
		mapping, err := s.resolveTokenMapping(ctx, prepared[i].input.Scenario)
		if err != nil {
			return BatchApplyResult{}, err
		}
		mappings[prepared[i].input.Scenario] = mapping
	}

	keys := make([]string, 0, len(uniqueFiles))
	for key := range uniqueFiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		unique := uniqueFiles[key]
		item := &prepared[unique.owner]
		mapping := mappings[unique.scenario]
		translated, translations, err := TranslateDesignTokens(stripSourceHeader(unique.file.Content), mapping.Namespace, mapping)
		if err != nil {
			if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
				return BatchApplyResult{}, fmt.Errorf("batch translation failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return BatchApplyResult{}, err
		}
		now := s.clock.Now().UTC()
		body := formatProvenance(unique.plan.Version, item.adoptionID, now, hashBytes([]byte(translated)), formatTokenTranslations(translations)) + translated
		path, actual, err := s.writeAdoptedSource(ctx, unique.scenario, unique.path, []byte(body))
		if err != nil {
			if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
				return BatchApplyResult{}, fmt.Errorf("batch write failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return BatchApplyResult{}, err
		}
		unique.written, unique.body = path, actual
	}
	for i := range prepared {
		for _, plan := range prepared[i].plans {
			for _, file := range plan.Files {
				key := batchSnapshotKey(prepared[i].input.Scenario, file.AdoptedPath)
				unique := uniqueFiles[key]
				if unique == nil {
					continue
				}
				if file.IsEntry && plan.Asset.ID == prepared[i].root.ID {
					prepared[i].entrySnapshot = adoptedSnapshotHash(unique.body)
					prepared[i].writtenPath = unique.written
				}
				if finder, ok := s.files.(ScenarioImportSiteFinder); ok {
					sites, err := finder.FindImportSites(ctx, prepared[i].input.Scenario, file.AdoptedPath)
					if err != nil {
						if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
							return BatchApplyResult{}, fmt.Errorf("batch import-site scan failed: %w (rollback failed: %v)", err, rollbackErr)
						}
						return BatchApplyResult{}, err
					}
					prepared[i].importSites = append(prepared[i].importSites, sites...)
				}
			}
		}
		if prepared[i].experiencePath != "" {
			if _, err := s.files.Write(ctx, prepared[i].input.Scenario, prepared[i].experiencePath, []byte(prepared[i].version.ExperienceContract)); err != nil {
				if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
					return BatchApplyResult{}, fmt.Errorf("batch experience write failed: %w (rollback failed: %v)", err, rollbackErr)
				}
				return BatchApplyResult{}, fmt.Errorf("write component experience contract: %w", err)
			}
		}
	}

	inputs := make([]CreateInput, 0, len(prepared))
	for i := range prepared {
		adoptionFiles := make([]AdoptionFile, 0)
		for _, plan := range prepared[i].plans {
			for _, file := range plan.Files {
				unique := uniqueFiles[batchSnapshotKey(prepared[i].input.Scenario, file.AdoptedPath)]
				adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: adoptedSnapshotHash(unique.body), SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
			}
		}
		inputs = append(inputs, CreateInput{ID: prepared[i].adoptionID, ComponentID: prepared[i].root.ID, LibraryID: prepared[i].root.LibraryID, Scenario: prepared[i].input.Scenario, AdoptedPath: prepared[i].input.AdoptedPath, AdoptedVersion: prepared[i].version.Version, SourceSHA256: prepared[i].version.ContentSHA256, AdoptedSnapshotSHA256: prepared[i].entrySnapshot, IncludeSuggestions: append([]string(nil), prepared[i].input.IncludeSuggestions...), ForkReason: prepared[i].input.ForkReason, ExtensionPoints: append([]string(nil), prepared[i].input.ExtensionPoints...), Files: adoptionFiles})
	}
	var created []Adoption
	var err error
	if batchRepo, ok := s.repo.(BatchCreator); ok {
		created, err = batchRepo.CreateBatch(ctx, inputs)
	} else {
		created = make([]Adoption, 0, len(inputs))
		for _, input := range inputs {
			var adoption Adoption
			adoption, err = s.repo.Create(ctx, input)
			if err != nil {
				for _, prior := range created {
					_ = s.repo.Delete(ctx, prior.ID)
				}
				break
			}
			created = append(created, adoption)
		}
	}
	if err != nil {
		if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
			return BatchApplyResult{}, fmt.Errorf("batch persistence failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return BatchApplyResult{}, err
	}

	result := BatchApplyResult{Results: make([]ApplyResult, 0, len(prepared))}
	for i := range prepared {
		copied := make([]string, 0, len(prepared[i].closure.Assets))
		for _, asset := range prepared[i].closure.Assets {
			copied = append(copied, asset.Asset.LibraryID)
		}
		result.Results = append(result.Results, ApplyResult{Adoption: created[i], WrittenPath: prepared[i].writtenPath, ExperiencePath: prepared[i].experiencePath, ImportSites: prepared[i].importSites, StyleFitAffinity: components.DesignAffinity(prepared[i].verdict.StyleFit), StyleFitDetail: prepared[i].verdict.StyleFitDetail, CopiedAssets: copied, SatisfiedPorts: prepared[i].closure.SatisfiedPorts, AvailableSuggestions: prepared[i].closure.AvailableSuggestions})
	}
	for dependency := range shared {
		result.SharedDependencies = append(result.SharedDependencies, dependency)
	}
	sort.Strings(result.SharedDependencies)
	return result, nil
}

type batchFileSnapshot struct {
	scenario string
	path     string
	existed  bool
	content  []byte
}

func batchSnapshotKey(scenario, path string) string { return scenario + "::" + path }

func (s *service) captureBatchSnapshots(ctx context.Context, scenario, experiencePath string, plans []adoptionPlan, snapshots map[string]batchFileSnapshot) error {
	paths := make([]string, 0)
	for _, plan := range plans {
		for _, file := range plan.Files {
			paths = append(paths, file.AdoptedPath)
		}
	}
	if experiencePath != "" {
		paths = append(paths, experiencePath)
	}
	for _, path := range paths {
		key := batchSnapshotKey(scenario, path)
		if _, seen := snapshots[key]; seen {
			continue
		}
		exists, err := s.files.Exists(ctx, scenario, path)
		if err != nil {
			return err
		}
		snapshot := batchFileSnapshot{scenario: scenario, path: path, existed: exists}
		if exists {
			content, err := s.files.Read(ctx, scenario, path)
			if err != nil {
				return err
			}
			snapshot.content = append([]byte(nil), content...)
		}
		snapshots[key] = snapshot
	}
	return nil
}

func (s *service) rollbackBatch(ctx context.Context, snapshots map[string]batchFileSnapshot, results []ApplyResult) error {
	if len(results) == 0 && len(snapshots) == 0 {
		return nil
	}
	var firstErr error
	for _, result := range results {
		if err := s.repo.Delete(ctx, result.Adoption.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, snapshot := range snapshots {
		var err error
		if snapshot.existed {
			if raw, ok := s.files.(interface {
				WriteRaw(context.Context, string, string, []byte) error
			}); ok {
				err = raw.WriteRaw(ctx, snapshot.scenario, snapshot.path, snapshot.content)
			} else {
				_, err = s.files.Write(ctx, snapshot.scenario, snapshot.path, snapshot.content)
			}
		} else if remover, ok := s.files.(ScenarioFileRemover); ok {
			err = remover.Remove(ctx, snapshot.scenario, snapshot.path)
		} else {
			return fmt.Errorf("file writer cannot remove newly created batch files")
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func cmpExperiencePath(root components.Component) string {
	if root.Slug == "" {
		return ""
	}
	return componentExperiencePath(root.Slug)
}

// RefreshSummary is the counter rollup returned by Refresh — used by
// CLI/UI to render a one-line outcome alongside the per-row table.
type RefreshSummary struct {
	LibraryCurrent       int
	LibraryBehind        int
	LibraryDeprecated    int
	LibraryMissing       int
	LibraryUnknown       int
	LibrarySourceDrifted int
	LocalClean           int
	LocalModified        int
	LocalMissing         int
	LocalUnknown         int
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

// VersionFileReader is an optional source-store seam used by link-time
// projection. The registry keeps released version rows immutable, while a
// source companion can be added alongside an existing release without
// changing that release's entry hash.
type VersionFileReader interface {
	GetVersionContentAt(ctx context.Context, componentID, version, path string) (components.Content, error)
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

type ScenarioFileRemover interface {
	Remove(ctx context.Context, scenario, adoptedPath string) error
}

// ScenarioImportSiteFinder reports source files that directly import a target
// within a scenario tree. It is deliberately optional so narrow test fakes can
// retain their read/write-only contract.
type ScenarioImportSiteFinder interface {
	FindImportSites(ctx context.Context, scenario, adoptedPath string) ([]string, error)
}

type ScenarioLibraryImportReader interface {
	ImportedLibrarySpecifiers(ctx context.Context, scenario string) ([]components.LibraryPackageSpecifier, error)
}

type ScenarioReferenceTokenReader interface {
	ReferenceTokens(ctx context.Context) (map[string]themes.DesignToken, error)
}

// ScenarioImportRewriter replaces imports of a copied root with the package
// subpath selected by Link. It is optional so focused service fakes can remain
// read/write-only; production must implement it because linking without
// updating call sites would delete the old file and leave a broken build.
type ScenarioImportRewriter interface {
	RewriteImportSites(ctx context.Context, scenario, adoptedPath, replacement string) ([]string, error)
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

// MaturityReader consumes the catalog's evidence-backed readiness rung. The
// catalog remains the authority for how evidence becomes a rung.
type MaturityReader interface {
	Maturity(ctx context.Context, component components.Component, version, scenario string) (MaturityVerdict, error)
}

// ContractCoverageReader consumes the catalog's persisted contract-gate
// observations. It keeps i18n and selector evidence on the same adoption
// preflight projection as maturity without rerunning gates during a write.
type ContractCoverageReader interface {
	GateVerdict(ctx context.Context, component components.Component, version, scenario, gate string) (string, error)
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
	clock          schedule.Clock
	reporter       DriftReporter
	logger         *log.Logger
	deps           DependencyValidator
	styles         StyleFitValidator
	maturity       MaturityReader
	coverage       ContractCoverageReader
	tokens         ScenarioTokenNamespaceReader
	mappings       ScenarioTokenMappingReader
	tokenInventory ScenarioTokenInventoryReader
	manifestLoader uimanifest.Loader
	presence       PresenceReconciler
}

// PresenceReconciler is the narrow lifecycle hook used after an adoption
// changes reachability. The version-ledger package supplies the production
// implementation; tests can inject a recorder or a failing reconciler.
type PresenceReconciler interface {
	ReconcilePresence(ctx context.Context, componentID string, apply bool) error
}

func (s *service) ensureVersionMaterialized(ctx context.Context, componentID, version string) error {
	materializer, ok := s.library.(components.Materializer)
	if !ok {
		return nil
	}
	if _, err := materializer.EnsureMaterialized(ctx, componentID, version, ""); err != nil {
		return fmt.Errorf("materialize %s@%s before adoption: %w", componentID, version, err)
	}
	return nil
}

// NewService constructs the production Service. reporter may be nil
// when the swarm-manager integration is disabled (e.g. tests that don't
// exercise drift reporting); a nil reporter is treated as a no-op so
// the rest of the Refresh path is unaffected.
func NewService(repo Repository, lib LibraryReader, files ScenarioFileWriter, clk schedule.Clock) Service {
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

func SetMaturityReader(svc Service, reader MaturityReader) {
	if s, ok := svc.(*service); ok {
		s.maturity = reader
	}
}

func SetContractCoverageReader(svc Service, reader ContractCoverageReader) {
	if s, ok := svc.(*service); ok {
		s.coverage = reader
	}
}

// SetPresenceReconciler wires the scoped tier update used by adoption delete
// and reconverge. A nil reconciler leaves the service compatible with focused
// callers that do not construct the version-ledger domain.
func SetPresenceReconciler(svc Service, reconciler PresenceReconciler) {
	if s, ok := svc.(*service); ok {
		s.presence = reconciler
	}
}

func SetTokenNamespaceReader(svc Service, reader ScenarioTokenNamespaceReader) {
	if s, ok := svc.(*service); ok {
		s.tokens = reader
		if mappingReader, ok := reader.(ScenarioTokenMappingReader); ok {
			s.mappings = mappingReader
		}
		if inventoryReader, ok := reader.(ScenarioTokenInventoryReader); ok {
			s.tokenInventory = inventoryReader
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
	explicitVersion := strings.TrimSpace(in.Version) != ""
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
	if err := s.ensureVersionMaterialized(ctx, cmp.ID, version); err != nil {
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
	if _, err := s.requireTokenVerdict(ctx, in.ComponentID, in.Scenario, closure, in.OverrideValidation); err != nil {
		return ApplyResult{}, err
	}
	verdict, err := s.adoptionVerdict(ctx, cmp, v, closure, in.Scenario)
	if err != nil {
		return ApplyResult{}, err
	}
	if verdict.Blocking() && (!in.OverrideValidation || (v.Status == components.VersionStatusDraft && !explicitVersion)) {
		return ApplyResult{}, readinessBlockedError(in.ComponentID, in.Scenario, verdict)
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
	root, err := s.repo.Create(ctx, CreateInput{ID: adoptionID, ComponentID: cmp.ID, LibraryID: cmp.LibraryID, Scenario: in.Scenario, AdoptedPath: in.AdoptedPath, AdoptedVersion: version, SourceSHA256: v.ContentSHA256, AdoptedSnapshotSHA256: entrySnapshot, IncludeSuggestions: append([]string(nil), in.IncludeSuggestions...), ForkReason: in.ForkReason, ExtensionPoints: append([]string(nil), in.ExtensionPoints...), Files: adoptionFiles})
	if err != nil {
		return ApplyResult{}, err
	}
	copied := make([]string, 0, len(closure.Assets))
	for _, asset := range closure.Assets {
		copied = append(copied, asset.Asset.LibraryID)
	}
	result := ApplyResult{Adoption: root, WrittenPath: written, ExperiencePath: experiencePath, ImportSites: importSites, CopiedAssets: copied, SatisfiedPorts: closure.SatisfiedPorts, AvailableSuggestions: closure.AvailableSuggestions}
	result.StyleFitAffinity = components.DesignAffinity(verdict.StyleFit)
	result.StyleFitDetail = verdict.StyleFitDetail
	return result, nil
}

func (s *service) Preflight(ctx context.Context, in PreflightInput) (PreflightResult, error) {
	componentID := strings.TrimSpace(in.ComponentID)
	scenario := strings.TrimSpace(in.Scenario)
	if componentID == "" || scenario == "" {
		return PreflightResult{}, ErrInvalidAdoption{Field: "component_id/scenario", Reason: "required"}
	}
	root, err := s.library.Get(ctx, componentID)
	if err != nil {
		return PreflightResult{}, err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = firstNonEmpty(root.LatestVersion, root.Version)
	}
	if err := s.ensureVersionMaterialized(ctx, componentID, version); err != nil {
		return PreflightResult{}, err
	}
	v, err := s.library.GetVersion(ctx, componentID, version)
	if err != nil {
		return PreflightResult{}, err
	}
	closure, err := s.resolveAdoptionClosure(ctx, root, v, scenario, nil)
	if err != nil {
		return PreflightResult{}, err
	}
	tokens, err := s.resolveTokenVerdict(ctx, closure, scenario)
	if err != nil {
		return PreflightResult{}, err
	}
	readiness, err := s.adoptionVerdict(ctx, root, v, closure, scenario)
	if err != nil {
		return PreflightResult{}, err
	}
	result := PreflightResult{ComponentID: componentID, Scenario: scenario, Version: version, Verdict: readiness, Tokens: tokens, Dependency: readiness.Dependency, StyleFit: readiness.StyleFit, I18n: readiness.I18n, Selectors: readiness.Selectors, Blocking: readiness.Blocking()}
	return result, nil
}

func (s *service) adoptionVerdict(ctx context.Context, root components.Component, version components.ComponentVersion, closure components.ClosureReport, scenario string) (AdoptionVerdict, error) {
	// A nil coverage reader is a deliberately reduced unit-test seam. Production
	// wiring always installs CatalogGateReader, which returns not-measured when
	// evidence is absent and therefore keeps the adoption blocking contract.
	result := AdoptionVerdict{Version: version.Status, I18n: "pass", Selectors: "pass"}
	if s.coverage != nil {
		result.I18n = "not-measured"
		result.Selectors = "not-measured"
	}
	if s.deps != nil {
		verdict, err := s.deps.ValidateAdoption(ctx, root.ID, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Dependency = string(verdict.Kind)
	}
	if s.styles != nil {
		verdict, err := s.styles.ValidateStyleFit(ctx, root.ID, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.StyleFit = string(verdict.Affinity)
		result.StyleFitDetail = verdict.Detail
	}
	if s.maturity != nil {
		maturity, err := s.maturity.Maturity(ctx, root, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Maturity = maturity
	}
	var err error
	if s.coverage != nil {
		result.I18n, err = s.coverage.GateVerdict(ctx, root, version.Version, scenario, "i18n")
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Selectors, err = s.coverage.GateVerdict(ctx, root, version.Version, scenario, "selector-coverage")
		if err != nil {
			return AdoptionVerdict{}, err
		}
	}
	result.Tokens, err = s.resolveTokenVerdict(ctx, closure, scenario)
	if err != nil {
		return AdoptionVerdict{}, err
	}
	if version.Status == components.VersionStatusDeprecated {
		result.Warnings = []string{"selected version is deprecated"}
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
		// Story metadata belongs to the library catalog, not the adopted
		// runtime closure. Dependency stories otherwise collide at the shared
		// target and make a valid Button adoption impossible.
		if file.Path == "story.json" || file.Path == "story.tsx" || file.Path == "experience-contract.json" {
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
	if localStatus == LocalStatusModified && !in.ConfirmLocalOverwrite && !in.DryRun {
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
	if err := s.ensureVersionMaterialized(ctx, row.ComponentID, version); err != nil {
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
	closure, err := s.resolveAdoptionClosure(ctx, root, v, row.Scenario, row.IncludeSuggestions)
	if err != nil {
		return Adoption{}, "", err
	}
	if _, err := s.requireTokenVerdict(ctx, row.ComponentID, row.Scenario, closure, in.OverrideValidation); err != nil {
		return Adoption{}, "", err
	}
	readiness, err := s.adoptionVerdict(ctx, root, v, closure, row.Scenario)
	if err != nil {
		return Adoption{}, "", err
	}
	if readiness.Blocking() && !in.OverrideValidation {
		return Adoption{}, "", readinessBlockedError(row.ComponentID, row.Scenario, readiness)
	}
	plans := adoptionPlansForClosure(closure.Assets, row.AdoptedPath)
	if in.DryRun {
		if _, err := s.resolveTokenMapping(ctx, row.Scenario); err != nil {
			return Adoption{}, "", err
		}
		return row, "", nil
	}
	newPaths := make(map[string]struct{})
	for _, plan := range plans {
		for _, file := range plan.Files {
			newPaths[file.AdoptedPath] = struct{}{}
		}
	}
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
	// Remove orphaned files only after the replacement unit and its registry
	// row have been written successfully. Other live adoptions retain shared
	// files; failed writes never delete the previous unit.
	if remover, ok := s.files.(ScenarioFileRemover); ok {
		ownedByOther := make(map[string]struct{})
		if rows, listErr := s.repo.List(ctx, ListQuery{Scenario: row.Scenario, Limit: 100000}); listErr == nil {
			for _, other := range rows {
				if other.ID == row.ID {
					continue
				}
				for _, file := range other.Files {
					ownedByOther[file.AdoptedPath] = struct{}{}
				}
			}
		}
		for _, oldFile := range row.Files {
			if _, stillPresent := newPaths[oldFile.AdoptedPath]; stillPresent {
				continue
			}
			if _, shared := ownedByOther[oldFile.AdoptedPath]; shared {
				continue
			}
			if err := remover.Remove(ctx, row.Scenario, oldFile.AdoptedPath); err != nil {
				return updated, written, fmt.Errorf("remove orphaned adopted file %q: %w", oldFile.AdoptedPath, err)
			}
		}
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
	_, err := s.DeleteWithOptions(ctx, id, false)
	return err
}

func (s *service) DeleteWithOptions(ctx context.Context, id string, confirmRemoveFiles bool) (DeleteResult, error) {
	row, err := s.repo.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return DeleteResult{}, err
	}
	files := append([]AdoptionFile(nil), row.Files...)
	if len(files) == 0 {
		files = []AdoptionFile{{AdoptedPath: row.AdoptedPath}}
	}
	rows, err := s.repo.List(ctx, ListQuery{Scenario: row.Scenario, Limit: 100000})
	if err != nil {
		return DeleteResult{}, err
	}
	owned := make(map[string]struct{})
	for _, other := range rows {
		if other.ID == row.ID {
			continue
		}
		for _, file := range other.Files {
			owned[file.AdoptedPath] = struct{}{}
		}
		if other.AdoptedPath != "" {
			owned[other.AdoptedPath] = struct{}{}
		}
	}
	result := DeleteResult{AdoptionID: row.ID}
	for _, file := range files {
		if _, exists := owned[file.AdoptedPath]; exists {
			continue
		}
		result.RemovableFiles = append(result.RemovableFiles, file.AdoptedPath)
	}
	if !confirmRemoveFiles && len(result.RemovableFiles) > 0 {
		result.RequiresConfirmation = true
		return result, ErrInvalidAdoption{Field: "confirm_remove_files", Reason: fmt.Sprintf("deletion would remove %s; re-run with --confirm-remove-files", strings.Join(result.RemovableFiles, ", "))}
	}
	if confirmRemoveFiles {
		if remover, ok := s.files.(ScenarioFileRemover); ok {
			for _, path := range result.RemovableFiles {
				if err := remover.Remove(ctx, row.Scenario, path); err != nil {
					return result, err
				}
				result.RemovedFiles = append(result.RemovedFiles, path)
			}
		}
	}
	if err := s.repo.Delete(ctx, row.ID); err != nil {
		return result, err
	}
	if s.presence != nil {
		if err := s.presence.ReconcilePresence(ctx, row.ComponentID, true); err != nil {
			return result, fmt.Errorf("reconcile presence after deleting adoption %s@%s: %w", row.LibraryID, row.AdoptedVersion, err)
		}
	}
	return result, nil
}

func (s *service) Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error) {
	return s.refresh(ctx, componentID, true)
}

// ForkReport classifies the complete adoption registry using the same
// evidence as Refresh. Dry-run is the default so operators can inspect every
// row before persisting the four-way disposition; apply only updates the
// registry metadata and never rewrites scenario source files.
func (s *service) ForkReport(ctx context.Context, componentID string, apply bool) ([]Adoption, RefreshSummary, error) {
	return s.refresh(ctx, componentID, apply)
}

func (s *service) refresh(ctx context.Context, componentID string, apply bool) ([]Adoption, RefreshSummary, error) {
	rows, err := s.repo.List(ctx, ListQuery{ComponentID: strings.TrimSpace(componentID), Limit: refreshListLimit})
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
		// Automated classifications are observations, not operator declarations.
		// Recompute them on refresh so improving the classifier repairs earlier
		// coarse results and every registry row has an explicit disposition. A
		// declared fork remains authoritative and is never overwritten here.
		if row.ForkStatus == ForkStatusDeclared {
			update.ForkStatus = ForkStatusDeclared
		} else if row.Mode == AdoptionModeLinked {
			update.ForkStatus = ForkStatusContractPreserved
		} else if localStatus == LocalStatusModified {
			update.ForkStatus = forkStatusForDisposition(s.classifyModified(ctx, row))
		} else if localStatus == LocalStatusClean && s.verifiedCleanAgainstLibrary(ctx, row) {
			update.ForkStatus = ForkStatusMechanicalTranslation
		} else {
			update.ForkStatus = ForkStatusLocalFork
		}
		rows[i].ForkStatus = update.ForkStatus
		// Drift policy:
		//   * status flips to behind/modified AND no backlog item filed
		//     yet → call the reporter and store the returned ref so the
		//     next refresh skips it.
		//   * status returns to current → clear the stored ref so a
		//     fresh drift files a new item rather than being silently
		//     swallowed.
		if apply && s.reporter != nil {
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
		case LibraryVersionStatusSourceDrifted:
			summary.LibrarySourceDrifted++
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
	if apply {
		if _, err := s.repo.ApplyRefresh(ctx, updates); err != nil {
			return nil, RefreshSummary{}, err
		}
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
	_ ScenarioFileReader                  = (*FSScenarioFileReader)(nil)
	_ ScenarioProvenanceScanner           = (*FSScenarioFileReader)(nil)
	_ ScenarioCandidateScanner            = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenNamespaceReader        = (*FSScenarioFileReader)(nil)
	_ ScenarioTokenInventoryReader        = (*FSScenarioFileReader)(nil)
	_ ScenarioRuntimeTokenInventoryReader = (*FSScenarioFileReader)(nil)
	_ ScenarioLibraryImportReader         = (*FSScenarioFileReader)(nil)
	_ ScenarioReferenceTokenReader        = (*FSScenarioFileReader)(nil)
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

// DeclaredTokens returns every custom property declared by the scenario UI.
// It intentionally scans source rather than only the canonical ramp so a
// consumer-owned declaration outside the managed region still satisfies a
// library requirement without being overwritten.
func (r *FSScenarioFileReader) DeclaredTokens(_ context.Context, scenario string) ([]string, error) {
	base := filepath.Join(r.root, scenario, "ui")
	declared := map[string]struct{}{}
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == "node_modules" || entry.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".css" && ext != ".ts" && ext != ".tsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range scenarioTokenDeclarationRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				declared[match[1]] = struct{}{}
			}
		}
		for _, match := range scenarioRuntimeTokenWriteRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				declared[match[1]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan declared tokens for scenario %q: %w", scenario, err)
	}
	result := make([]string, 0, len(declared))
	for property := range declared {
		result = append(result, property)
	}
	sort.Strings(result)
	return result, nil
}

// RuntimeWrittenTokens reports properties whose value is owned by application
// state rather than by the static design-token ramp. Token sync removes these
// from its managed region so a stale default cannot race the runtime writer.
func (r *FSScenarioFileReader) RuntimeWrittenTokens(_ context.Context, scenario string) ([]string, error) {
	base := filepath.Join(r.root, scenario, "ui", "src")
	written := map[string]struct{}{}
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range scenarioRuntimeTokenWriteRE.FindAllStringSubmatch(string(raw), -1) {
			if len(match) == 2 {
				written[match[1]] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan runtime token writers for scenario %q: %w", scenario, err)
	}
	result := make([]string, 0, len(written))
	for property := range written {
		result = append(result, property)
	}
	sort.Strings(result)
	return result, nil
}

func (r *FSScenarioFileReader) ImportedLibrarySpecifiers(_ context.Context, scenario string) ([]components.LibraryPackageSpecifier, error) {
	sources, err := r.walkScenarioSources()
	if err != nil {
		return nil, err
	}
	seen := map[components.LibraryPackageSpecifier]bool{}
	var result []components.LibraryPackageSpecifier
	for _, source := range sources {
		if source.scenario != scenario {
			continue
		}
		for _, specifier := range components.LibraryPackageSpecifiers(string(source.content)) {
			if !seen[specifier] {
				seen[specifier] = true
				result = append(result, specifier)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].RequestedVersion < result[j].RequestedVersion
	})
	return result, nil
}

func (r *FSScenarioFileReader) ReferenceTokens(_ context.Context) (map[string]themes.DesignToken, error) {
	path := filepath.Join(filepath.Dir(r.root), "templates", "design", "_base", "tokens.css")
	tokens, err := themes.ReadTokenFile(path)
	if err != nil {
		return nil, err
	}
	result := make(map[string]themes.DesignToken, len(tokens))
	for _, token := range tokens {
		if token.Tier == "" {
			return nil, fmt.Errorf("canonical token %s has no parseable tier annotation", token.Name)
		}
		result[token.Name] = token
	}
	return result, nil
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

// WriteRaw restores a batch snapshot without invoking the scenario formatter.
// Rollback must recover the exact bytes that existed before the batch began.
func (r *FSScenarioFileReader) WriteRaw(_ context.Context, scenario, adoptedPath string, content []byte) error {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cleaned), 0o755); err != nil {
		return fmt.Errorf("create restored file dir: %w", err)
	}
	if err := os.WriteFile(cleaned, content, 0o600); err != nil {
		return fmt.Errorf("restore adopted file %q: %w", adoptedPath, err)
	}
	return nil
}

func (r *FSScenarioFileReader) Remove(_ context.Context, scenario, adoptedPath string) error {
	cleaned, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return err
	}
	if err := os.Remove(cleaned); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove adopted file %q: %w", adoptedPath, err)
	}
	return nil
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

func (r *FSScenarioFileReader) RewriteImportSites(ctx context.Context, scenario, adoptedPath, replacement string) ([]string, error) {
	base := filepath.Join(r.root, scenario)
	updated := make([]string, 0)
	if strings.HasPrefix(replacement, linkedPackageName+"/") {
		modulePrefix := replacement
		if slash := strings.LastIndex(modulePrefix, "/"); slash >= 0 {
			modulePrefix = modulePrefix[:slash+1]
		}
		// A row records the adopter-owned provenance path, while source files
		// import the package subpath. Rewrite the whole component family, not
		// only the row's current exact version, so a repeated governed Link
		// repairs stale package imports as well.
		oldModulePattern := regexp.MustCompile(`(["'])` + regexp.QuoteMeta(modulePrefix) + `[^"']+(["'])`)
		err := filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || strings.Contains(filepath.ToSlash(path), "/node_modules/") || !isSourceFile(path) {
				return nil
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(body)
			rewritten := oldModulePattern.ReplaceAllString(text, `$1`+replacement+`$2`)
			if rewritten == text {
				return nil
			}
			if writeErr := os.WriteFile(path, []byte(rewritten), 0o644); writeErr != nil {
				return writeErr
			}
			rel, relErr := filepath.Rel(base, path)
			if relErr != nil {
				return relErr
			}
			updated = append(updated, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("rewrite package import sites for %q: %w", adoptedPath, err)
		}
	}
	if strings.HasPrefix(filepath.ToSlash(adoptedPath), "./") {
		sort.Strings(updated)
		return updated, nil
	}
	sites, err := r.FindImportSites(ctx, scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	target, err := r.resolve(scenario, adoptedPath)
	if err != nil {
		return nil, err
	}
	for _, site := range sites {
		absolute := filepath.Join(base, filepath.FromSlash(site))
		content, readErr := os.ReadFile(absolute)
		if readErr != nil {
			return nil, fmt.Errorf("read import site %q: %w", site, readErr)
		}
		relTarget, relErr := filepath.Rel(filepath.Dir(absolute), target)
		if relErr != nil {
			return nil, relErr
		}
		module := strings.TrimSuffix(filepath.ToSlash(relTarget), filepath.Ext(relTarget))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		body := string(content)
		for _, quote := range []string{"\"", "'"} {
			body = strings.ReplaceAll(body, quote+module+quote, quote+replacement+quote)
		}
		if body == string(content) {
			continue
		}
		if writeErr := os.WriteFile(absolute, []byte(body), 0o644); writeErr != nil {
			return nil, fmt.Errorf("rewrite import site %q: %w", site, writeErr)
		}
		updated = append(updated, site)
	}
	return updated, nil
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
	if row.SourceSHA256 != "" && version.ContentSHA256 != "" && row.SourceSHA256 != version.ContentSHA256 {
		return LibraryVersionStatusSourceDrifted, fmt.Sprintf("source bytes for released version %s changed: recorded %s, current %s", row.AdoptedVersion, row.SourceSHA256, version.ContentSHA256)
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
