package adoptions

import (
	"context"
	"fmt"
	"log"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/themes"

	"github.com/vrooli/api-core/uimanifest"

	"github.com/vrooli/api-core/schedule"
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

type ScenarioTokenFallbackExemptionReader interface {
	TokenFallbackExemptions(ctx context.Context) (map[string]string, error)
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
