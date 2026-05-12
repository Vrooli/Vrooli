package adoptions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/clock"
	"react-component-library/internal/components"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy lives next to the only code that applies it.
const defaultListLimit = 200

// Service is the application-layer surface handlers depend on. Owns
// the Refresh drift policy and any cross-handler validation that does
// not belong in transport.
type Service interface {
	Create(ctx context.Context, in CreateInput) (Adoption, error)
	List(ctx context.Context, q ListQuery) ([]Adoption, error)
	Get(ctx context.Context, id string) (Adoption, error)
	Delete(ctx context.Context, id string) error
	Refresh(ctx context.Context, componentID string) ([]Adoption, RefreshSummary, error)
}

// RefreshSummary is the counter rollup returned by Refresh — used by
// CLI/UI to render a one-line outcome alongside the per-row table.
type RefreshSummary struct {
	Current  int
	Behind   int
	Modified int
	Unknown  int
}

// LibraryReader is the components-side seam Refresh needs. Hides the
// full components.Service surface so tests don't need to build one.
type LibraryReader interface {
	Get(ctx context.Context, id string) (components.Component, error)
	GetContent(ctx context.Context, id string) (components.Content, error)
}

// ScenarioFileReader is the target-scenario-tree seam Refresh uses to
// read adopted files. Production maps adopted_path under a configured
// scenarios root with a path-traversal guard; tests inject a fake.
type ScenarioFileReader interface {
	Read(ctx context.Context, scenario, adoptedPath string) ([]byte, error)
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
	repo    Repository
	library LibraryReader
	files   ScenarioFileReader
	clock   clock.Clock
}

// NewService constructs the production Service.
func NewService(repo Repository, lib LibraryReader, files ScenarioFileReader, clk clock.Clock) Service {
	return &service{repo: repo, library: lib, files: files, clock: clk}
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
		status, detail := s.computeStatus(ctx, row)
		rows[i].Status = status
		rows[i].StatusDetail = detail
		rows[i].RefreshedAt = now
		updates = append(updates, RefreshUpdate{
			ID: row.ID, Status: status, StatusDetail: detail, RefreshedAt: now,
		})
		switch status {
		case StatusCurrent:
			summary.Current++
		case StatusBehind:
			summary.Behind++
		case StatusModified:
			summary.Modified++
		case StatusUnknown:
			summary.Unknown++
		}
	}
	if _, err := s.repo.ApplyRefresh(ctx, updates); err != nil {
		return nil, RefreshSummary{}, err
	}
	return rows, summary, nil
}

// computeStatus is the AD-002 drift decision tree. Pure function of
// the row + the two reader seams.
func (s *service) computeStatus(ctx context.Context, row Adoption) (Status, string) {
	cmp, err := s.library.Get(ctx, row.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return StatusUnknown, "component removed from library"
		}
		return StatusUnknown, fmt.Sprintf("library lookup failed: %v", err)
	}
	adoptedBytes, err := s.files.Read(ctx, row.Scenario, row.AdoptedPath)
	if err != nil {
		var missing ErrAdoptedFileMissing
		if errors.As(err, &missing) {
			return StatusUnknown, "adopted file missing"
		}
		return StatusUnknown, fmt.Sprintf("adopted file read failed: %v", err)
	}
	adoptedSHA := hashBytes(adoptedBytes)
	content, err := s.library.GetContent(ctx, row.ComponentID)
	if err != nil {
		return StatusUnknown, fmt.Sprintf("library content read failed: %v", err)
	}
	librarySHA := content.SHA256
	if adoptedSHA == librarySHA {
		// Bytes match the library's current snapshot. Even if
		// adopted_version is stale, the copy is materially current.
		return StatusCurrent, ""
	}
	if row.AdoptedSnapshotSHA256 != "" && adoptedSHA == row.AdoptedSnapshotSHA256 {
		// Adopted bytes still look like the snapshot captured at
		// create time; library has moved on without local edits.
		detail := fmt.Sprintf("library at %s", emptyOrVersion(cmp.Version))
		return StatusBehind, detail
	}
	return StatusModified, "adopted bytes diverge from any known snapshot"
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
	base := filepath.Join(r.root, scenario)
	cleaned := filepath.Clean(filepath.Join(base, adoptedPath))
	rel, err := filepath.Rel(base, cleaned)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return nil, fmt.Errorf("adopted_path %q escapes scenario root", adoptedPath)
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
