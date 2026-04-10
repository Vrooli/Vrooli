package dochealing

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem is the testability seam for filesystem operations.
type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldpath, newpath string) error
	Stat(path string) (os.FileInfo, error)
}

// OSFileSystem is the production implementation backed by the real OS.
type OSFileSystem struct{}

func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (OSFileSystem) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (OSFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// AutoFix moves misplaced docs to their canonical locations.
// It is synchronous, deterministic, and does not use an agent.
func (s *Service) AutoFix(ctx context.Context, scenarioName string, dryRun bool) (*AutoFixResult, error) {
	_ = ctx
	scenarioName = strings.TrimSpace(scenarioName)
	if scenarioName == "" {
		return nil, ErrScenarioRequired
	}
	if s.health == nil {
		return nil, ErrHealthUnavailable
	}

	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}

	healthResult, err := s.health.ValidateScenario(ctx, scenarioName)
	if err != nil {
		return nil, err
	}
	healthBefore := healthResult.Validation.HealthScore

	fs := s.fs
	if fs == nil {
		fs = OSFileSystem{}
	}

	var moved []MovedDoc
	var skipped []SkippedDoc

	for _, mp := range healthResult.Validation.MisplacedDocs {
		fromAbs := filepath.Join(scenarioPath, mp.ActualPath)
		toAbs := filepath.Join(scenarioPath, mp.ExpectedPath)

		if _, err := fs.Stat(toAbs); err == nil {
			skipped = append(skipped, SkippedDoc{
				FromPath: mp.ActualPath,
				ToPath:   mp.ExpectedPath,
				DocType:  string(mp.DocType),
				Reason:   "destination already exists",
			})
			continue
		}

		if dryRun {
			moved = append(moved, MovedDoc{
				FromPath: mp.ActualPath,
				ToPath:   mp.ExpectedPath,
				DocType:  string(mp.DocType),
			})
			continue
		}

		if err := fs.MkdirAll(filepath.Dir(toAbs), 0o755); err != nil {
			skipped = append(skipped, SkippedDoc{
				FromPath: mp.ActualPath,
				ToPath:   mp.ExpectedPath,
				DocType:  string(mp.DocType),
				Reason:   "mkdir failed: " + err.Error(),
			})
			continue
		}

		if err := fs.Rename(fromAbs, toAbs); err != nil {
			skipped = append(skipped, SkippedDoc{
				FromPath: mp.ActualPath,
				ToPath:   mp.ExpectedPath,
				DocType:  string(mp.DocType),
				Reason:   "rename failed: " + err.Error(),
			})
			continue
		}

		moved = append(moved, MovedDoc{
			FromPath: mp.ActualPath,
			ToPath:   mp.ExpectedPath,
			DocType:  string(mp.DocType),
		})
	}

	healthAfter := healthBefore
	if !dryRun && len(moved) > 0 {
		afterResult, err := s.health.ValidateScenario(ctx, scenarioName)
		if err == nil && afterResult != nil && afterResult.Validation != nil {
			healthAfter = afterResult.Validation.HealthScore
		}
	}

	return &AutoFixResult{
		ScenarioName: scenarioName,
		Moved:        moved,
		Skipped:      skipped,
		HealthBefore: healthBefore,
		HealthAfter:  healthAfter,
	}, nil
}
