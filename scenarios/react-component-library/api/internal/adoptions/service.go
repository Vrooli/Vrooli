package adoptions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"react-component-library/internal/clock"
	"react-component-library/internal/components"

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
	Apply(ctx context.Context, in ApplyInput) (Adoption, string, error)
	Reapply(ctx context.Context, in ReapplyInput) (Adoption, string, error)
	List(ctx context.Context, q ListQuery) ([]Adoption, error)
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
	GetContent(ctx context.Context, id string) (components.Content, error)
	GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error)
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
	repo     Repository
	library  LibraryReader
	files    ScenarioFileWriter
	clock    clock.Clock
	reporter DriftReporter
	logger   *log.Logger
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

func (s *service) Apply(ctx context.Context, in ApplyInput) (Adoption, string, error) {
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.AdoptedPath = strings.TrimSpace(in.AdoptedPath)
	if in.ComponentID == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if in.Scenario == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if in.AdoptedPath == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	cmp, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return Adoption{}, "", ErrInvalidAdoption{Field: "component_id", Reason: "no component with that id"}
		}
		return Adoption{}, "", err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = cmp.LatestVersion
	}
	if version == "" {
		version = cmp.Version
	}
	if version == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "version", Reason: "component has no latest version"}
	}
	exists, err := s.files.Exists(ctx, in.Scenario, in.AdoptedPath)
	if err != nil {
		return Adoption{}, "", err
	}
	if exists && !in.ConfirmOverwrite {
		return Adoption{}, "", ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "target file already exists"}
	}
	v, err := s.library.GetVersion(ctx, in.ComponentID, version)
	if err != nil {
		return Adoption{}, "", err
	}
	adoptionID := uuid.NewString()
	now := s.clock.Now().UTC()
	body := formatProvenance(v, adoptionID, now) + stripSourceHeader(v.Content)
	written, err := s.files.Write(ctx, in.Scenario, in.AdoptedPath, []byte(body))
	if err != nil {
		return Adoption{}, "", err
	}
	a, err := s.repo.Create(ctx, CreateInput{
		ID:          adoptionID,
		ComponentID: cmp.ID, LibraryID: cmp.LibraryID, Scenario: in.Scenario, AdoptedPath: in.AdoptedPath,
		AdoptedVersion: version, SourceSHA256: v.ContentSHA256, AdoptedSnapshotSHA256: hashBytes([]byte(body)),
	})
	if err != nil {
		return Adoption{}, "", err
	}
	return a, written, nil
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
	v, err := s.library.GetVersion(ctx, row.ComponentID, version)
	if err != nil {
		return Adoption{}, "", err
	}
	now := s.clock.Now().UTC()
	body := formatProvenance(v, row.ID, now) + stripSourceHeader(v.Content)
	written, err := s.files.Write(ctx, row.Scenario, row.AdoptedPath, []byte(body))
	if err != nil {
		return Adoption{}, "", err
	}
	updated, err := s.repo.UpdateAppliedSnapshot(ctx, AppliedSnapshotUpdate{
		ID:                    row.ID,
		AdoptedVersion:        version,
		SourceSHA256:          v.ContentSHA256,
		AdoptedSnapshotSHA256: hashBytes([]byte(body)),
		AppliedAt:             now,
	})
	if err != nil {
		return Adoption{}, "", err
	}
	return updated, written, nil
}

func (s *service) List(ctx context.Context, q ListQuery) ([]Adoption, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	return s.repo.List(ctx, q)
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
	adoptedBytes, err := s.files.Read(ctx, row.Scenario, row.AdoptedPath)
	if err != nil {
		var missing ErrAdoptedFileMissing
		if errors.As(err, &missing) {
			return libraryStatusFor(row, cmp), LocalStatusMissing, "adopted file missing"
		}
		return libraryStatusFor(row, cmp), LocalStatusUnknown, fmt.Sprintf("adopted file read failed: %v", err)
	}
	adoptedSHA := hashBytes(adoptedBytes)
	localStatus := LocalStatusModified
	detail := "adopted bytes diverge from applied snapshot"
	if row.AdoptedSnapshotSHA256 != "" && adoptedSHA == row.AdoptedSnapshotSHA256 {
		localStatus = LocalStatusClean
		detail = ""
	}
	libStatus := libraryStatusFor(row, cmp)
	if libStatus == LibraryVersionStatusBehind {
		detail = fmt.Sprintf("library at %s", emptyOrVersion(firstNonEmpty(cmp.LatestVersion, cmp.Version)))
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

var _ ScenarioFileReader = (*FSScenarioFileReader)(nil)

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

func (r *FSScenarioFileReader) Write(_ context.Context, scenario, adoptedPath string, content []byte) (string, error) {
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
	return cleaned, nil
}

func (r *FSScenarioFileReader) resolve(scenario, adoptedPath string) (string, error) {
	base := filepath.Join(r.root, scenario)
	cleaned := filepath.Clean(filepath.Join(base, adoptedPath))
	rel, err := filepath.Rel(base, cleaned)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("adopted_path %q escapes scenario root", adoptedPath)
	}
	return cleaned, nil
}

func libraryStatusFor(row Adoption, cmp components.Component) LibraryVersionStatus {
	latest := firstNonEmpty(cmp.LatestVersion, cmp.Version)
	if row.AdoptedVersion == "" || latest == "" {
		return LibraryVersionStatusUnknown
	}
	if row.AdoptedVersion == latest {
		return LibraryVersionStatusCurrent
	}
	return LibraryVersionStatusBehind
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func formatProvenance(v components.ComponentVersion, adoptionID string, appliedAt time.Time) string {
	// JSDoc tag names align 1:1 with ui-health's ComponentProvenance proto:
	//   @vrooliComponentSource       -> library
	//   @vrooliComponentVersion      -> library_version
	//   @vrooliComponentAdoption     -> adoption_id
	//   @vrooliComponentAppliedAt    -> applied_at
	//   @vrooliComponentSourceSha256 -> source_sha256
	//   @vrooliComponentDriftHash    -> drift_hash (equal to source_sha256 at
	//                                   adoption time; recomputed at scan time)
	return fmt.Sprintf(`/**
 * @vrooliComponentSource %s
 * @vrooliComponentVersion %s
 * @vrooliComponentAdoption %s
 * @vrooliComponentAppliedAt %s
 * @vrooliComponentSourceSha256 %s
 * @vrooliComponentDriftHash %s
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
`, v.LibraryID, v.Version, adoptionID, appliedAt.UTC().Format(time.RFC3339), v.ContentSHA256, v.ContentSHA256)
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
