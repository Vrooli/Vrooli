// Package artifacts owns the generated/ lifecycle: report what exists on
// disk for a flow, drive regeneration through pipeline.Verify, and clear
// the directory when the UI's Clear affordance fires. The seam is
// layout.GeneratedDir(flowDir); no caller computes generated paths
// directly.
//
// Generation delegates to a pipeline.Generator-like callback rather than
// importing pipeline.Verify directly — that keeps the artifacts package
// unit-testable without spinning up quint and lets the handler wire the
// production pipeline in module.go.
package artifacts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"flow-verifier/internal/flows/discovery"
	"flow-verifier/internal/flows/layout"
	"flow-verifier/internal/flows/model"
)

// Status describes the on-disk state of one flow's generated/ tree.
// The UI's ArtifactStatusPill maps Status straight to a colour.
type Status string

const (
	StatusFresh   Status = "fresh"
	StatusMissing Status = "missing"
)

// File is one row in the artifacts inspection response.
type File struct {
	Path   string    `json:"path"`
	Exists bool      `json:"exists"`
	Size   int64     `json:"size,omitempty"`
	MTime  time.Time `json:"mtime,omitempty"`
}

// Report is the per-flow inspection result returned by the GET route.
// Missing lists the artifact paths the UI's "Generate" CTA will write.
type Report struct {
	FlowID       string   `json:"flowId"`
	ScenarioPath string   `json:"scenarioPath"`
	GeneratedDir string   `json:"generatedDir"`
	Status       Status   `json:"status"`
	Files        []File   `json:"files"`
	Missing      []string `json:"missing"`
}

// ClearResult enumerates the paths a successful clear removed.
type ClearResult struct {
	FlowID  string   `json:"flowId"`
	Removed []string `json:"removed"`
}

// ErrFlowNotFound is returned when a flow id does not resolve under the
// supplied root. Handlers translate it to 404.
var ErrFlowNotFound = errors.New("flow not found")

// ErrPathTraversal is returned when a computed generated/ path resolves
// outside the supplied root. Handlers translate it to 409 so a crafted
// flow id can't make the API delete arbitrary files.
var ErrPathTraversal = errors.New("generated directory resolves outside scenario root")

// Generator is the seam used to (re)materialise a flow's generated/
// tree. The production implementation in module.go wraps
// pipeline.Verify(... ModeGenerate ...); tests substitute a stub.
type Generator interface {
	Generate(ctx context.Context, root, flowID string) error
}

// Service is the in-process surface handlers call. Root is fixed at
// construction time so handlers don't have to thread it through every
// request.
type Service struct {
	gen Generator
}

// NewService wires the service. gen may be nil if only the inspection
// surface is needed; Generate calls return an error in that case.
func NewService(gen Generator) *Service {
	return &Service{gen: gen}
}

// Status inspects the generated/ tree for one flow and returns the
// Report the GET handler serialises.
func (s *Service) Status(root, flowID string) (Report, error) {
	flow, err := findFlow(root, flowID)
	if err != nil {
		return Report{}, err
	}
	return s.statusForFlow(root, flow)
}

// Generate (re)materialises one flow's generated/ tree via the configured
// Generator. The caller resolves root; this method does not pick it.
func (s *Service) Generate(ctx context.Context, root, flowID string) (Report, error) {
	if s.gen == nil {
		return Report{}, fmt.Errorf("artifacts: generator not configured")
	}
	if _, err := findFlow(root, flowID); err != nil {
		return Report{}, err
	}
	if err := s.gen.Generate(ctx, root, flowID); err != nil {
		return Report{}, err
	}
	return s.Status(root, flowID)
}

// Clear removes every file under <root>/<flow-dir>/generated/. Refuses
// any flow whose generated dir doesn't resolve under root.
func (s *Service) Clear(root, flowID string) (ClearResult, error) {
	flow, err := findFlow(root, flowID)
	if err != nil {
		return ClearResult{}, err
	}
	generatedAbs, err := generatedAbs(root, flow)
	if err != nil {
		return ClearResult{}, err
	}
	removed, err := clearDir(generatedAbs)
	if err != nil {
		return ClearResult{}, err
	}
	return ClearResult{FlowID: flow.FlowID, Removed: removed}, nil
}

// StatusForScenario walks every flow rooted at root and returns one
// Report per flow.
func (s *Service) StatusForScenario(root string) ([]Report, error) {
	all, err := discovery.FindContracts(rootAbs(root))
	if err != nil {
		return nil, err
	}
	out := make([]Report, 0, len(all))
	for _, flow := range all {
		report, err := s.statusForFlow(root, flow)
		if err != nil {
			return nil, err
		}
		out = append(out, report)
	}
	return out, nil
}

// GenerateForScenario walks every flow rooted at root and regenerates
// each. Serial; first failure aborts and returns the partial result.
func (s *Service) GenerateForScenario(ctx context.Context, root string) ([]Report, error) {
	if s.gen == nil {
		return nil, fmt.Errorf("artifacts: generator not configured")
	}
	all, err := discovery.FindContracts(rootAbs(root))
	if err != nil {
		return nil, err
	}
	out := make([]Report, 0, len(all))
	for _, flow := range all {
		if err := s.gen.Generate(ctx, root, flow.FlowID); err != nil {
			return out, err
		}
		report, err := s.statusForFlow(root, flow)
		if err != nil {
			return out, err
		}
		out = append(out, report)
	}
	return out, nil
}

// ClearForScenario removes the generated/ tree for every flow rooted at
// root and returns the union of cleared file paths.
func (s *Service) ClearForScenario(root string) ([]ClearResult, error) {
	all, err := discovery.FindContracts(rootAbs(root))
	if err != nil {
		return nil, err
	}
	out := make([]ClearResult, 0, len(all))
	for _, flow := range all {
		generatedAbs, err := generatedAbs(root, flow)
		if err != nil {
			return out, err
		}
		removed, err := clearDir(generatedAbs)
		if err != nil {
			return out, err
		}
		out = append(out, ClearResult{FlowID: flow.FlowID, Removed: removed})
	}
	return out, nil
}

func (s *Service) statusForFlow(root string, flow model.Flow) (Report, error) {
	generatedAbs, err := generatedAbs(root, flow)
	if err != nil {
		return Report{}, err
	}
	wanted := []string{
		flow.Layout.ModelPath,
		flow.Layout.ArtifactPath,
		flow.Layout.RuntimePath,
		flow.Layout.ReplayHelperPath,
	}
	files := make([]File, 0, len(wanted))
	missing := make([]string, 0)
	for _, rel := range wanted {
		abs := filepath.Join(rootAbs(root), filepath.FromSlash(rel))
		f := File{Path: rel}
		if info, err := os.Stat(abs); err == nil {
			f.Exists = true
			f.Size = info.Size()
			f.MTime = info.ModTime().UTC()
		} else {
			missing = append(missing, rel)
		}
		files = append(files, f)
	}
	sort.Strings(missing)
	status := StatusFresh
	if len(missing) > 0 {
		status = StatusMissing
	}
	return Report{
		FlowID:       flow.FlowID,
		ScenarioPath: rootAbs(root),
		GeneratedDir: generatedAbs,
		Status:       status,
		Files:        files,
		Missing:      missing,
	}, nil
}

func findFlow(root, flowID string) (model.Flow, error) {
	if flowID == "" {
		return model.Flow{}, fmt.Errorf("flow id is required")
	}
	flowsList, err := discovery.FindContracts(rootAbs(root))
	if err != nil {
		return model.Flow{}, err
	}
	for _, f := range flowsList {
		if f.FlowID == flowID {
			return f, nil
		}
	}
	return model.Flow{}, fmt.Errorf("%w: %s", ErrFlowNotFound, flowID)
}

func generatedAbs(root string, flow model.Flow) (string, error) {
	rootA := rootAbs(root)
	dir := flow.Layout.BaseDir + "/" + layout.GeneratedDirName
	abs := filepath.Join(rootA, filepath.FromSlash(dir))
	resolved, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	// Guard: the resolved path must sit under root. Without this a
	// crafted contract path could escape via ".." segments and DELETE
	// arbitrary directories.
	rel, err := filepath.Rel(rootA, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", ErrPathTraversal
	}
	return resolved, nil
}

func clearDir(generatedAbs string) ([]string, error) {
	removed := []string{}
	if _, err := os.Stat(generatedAbs); os.IsNotExist(err) {
		return removed, nil
	}
	entries, err := os.ReadDir(generatedAbs)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(generatedAbs, e.Name())
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, e.Name())
	}
	// If the directory is now empty, remove it too.
	if rest, err := os.ReadDir(generatedAbs); err == nil && len(rest) == 0 {
		_ = os.Remove(generatedAbs)
	}
	sort.Strings(removed)
	return removed, nil
}

func rootAbs(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return abs
}
