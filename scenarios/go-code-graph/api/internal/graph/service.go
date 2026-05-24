package graph

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Service orchestrates the Extract flow: validate input → acquire the
// per-path mutex → call the PackagesLoader seam → normalize → return.
//
// All non-trivial work flows through seams (PackagesLoader, PathMutex)
// so the Service itself is pure orchestration and exhaustively
// testable with FakeLoader.
type Service struct {
	loader PackagesLoader
	mu     *PathMutex
}

// NewService wires the production Service. Both arguments are required;
// the caller (api/main.go) owns construction so the mutex can be shared
// with the rewrite domain (OT-P0-006).
func NewService(loader PackagesLoader, mu *PathMutex) *Service {
	return &Service{loader: loader, mu: mu}
}

// Extract validates the input, locks the absolute scenario path,
// invokes the loader, and normalizes the result. Errors are typed
// ExtractError so handlers can map them to Connect codes via
// ErrorToConnectCode.
func (s *Service) Extract(ctx context.Context, in ExtractInput) (Graph, []Warning, error) {
	if strings.TrimSpace(in.ScenarioPath) == "" {
		return Graph{}, nil, ExtractError{
			Kind:    ExtractErrorInvalidInput,
			Message: "scenario_path is required",
		}
	}

	abs, err := filepath.Abs(in.ScenarioPath)
	if err != nil {
		return Graph{}, nil, ExtractError{
			Kind:    ExtractErrorPathUnreadable,
			Path:    in.ScenarioPath,
			Message: "resolve absolute path",
			Cause:   err,
		}
	}

	if err := preflightProject(abs); err != nil {
		return Graph{}, nil, err
	}

	unlock := s.mu.Lock(abs)
	defer unlock()

	pkgs, err := s.loader.Load(ctx, abs, LoadOptions{IncludeVendor: in.IncludeVendor})
	if err != nil {
		return Graph{}, nil, ExtractError{
			Kind:    ExtractErrorInternal,
			Path:    abs,
			Message: "packages loader",
			Cause:   err,
		}
	}

	graph, warnings := Normalize(pkgs, abs)
	return graph, warnings, nil
}

// preflightProject inspects the scenario path BEFORE the loader runs
// so we can return precise typed errors for the catastrophic cases
// (no go.mod, multiple go.mod, go.work, path unreadable). The loader
// itself would fail with a generic error.
func preflightProject(abs string) error {
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Message: "path does not exist", Cause: err}
		}
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Cause: err}
	}
	if !info.IsDir() {
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Message: "scenario_path is not a directory"}
	}

	if _, err := os.Stat(filepath.Join(abs, "go.work")); err == nil {
		return ExtractError{Kind: ExtractErrorWorkspaceUnsupported, Path: abs, Message: "go.work present"}
	}

	goMods, err := findGoMods(abs)
	if err != nil {
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Cause: err}
	}
	switch len(goMods) {
	case 0:
		return ExtractError{Kind: ExtractErrorNoGoMod, Path: abs, Message: "no go.mod under scenario_path"}
	case 1:
		// Common case.
	default:
		return ExtractError{Kind: ExtractErrorMultipleGoMod, Path: abs, Message: "multiple go.mod files found"}
	}
	return nil
}

// findGoMods returns the list of go.mod files under abs, descending
// into subdirectories but skipping vendor/, testdata/, and any path
// whose basename starts with "." (the loader skips those anyway).
func findGoMods(abs string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "testdata" || (len(base) > 1 && base[0] == '.' && p != abs) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			found = append(found, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}
