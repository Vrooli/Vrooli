package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"flow-verifier/internal/flows/kinds/temporal/codegen"
	"flow-verifier/internal/flows/kinds/temporal/contract"
	"flow-verifier/internal/flows/kinds/temporal/model"
	"flow-verifier/internal/fsadapter"
	"flow-verifier/internal/verification/artifact"
	"flow-verifier/internal/verification/lint"
	"flow-verifier/internal/verification/quint"
)

type Mode int

const (
	ModeGenerate Mode = iota
	ModeCheck
)

type FileSystem interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
}

type Options struct {
	Root   string
	Flows  []model.Flow
	Mode   Mode
	Runner quint.Runner
	FS     FileSystem
	Stdout io.Writer
}

func Run(ctx context.Context, options Options) error {
	stdout := options.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	fs := options.FS
	if fs == nil {
		fs = osFileSystem{}
	}
	runner := options.Runner
	if runner == nil {
		runner = quint.ExecRunner{}
	}
	if options.Mode == ModeCheck {
		for _, flow := range options.Flows {
			if err := contract.ValidateConventionalFiles(options.Root, contractFromFlow(flow)); err != nil {
				return err
			}
		}
	}

	version, err := runner.Run(ctx, quint.Command{Args: []string{"quint", "--version"}, Dir: options.Root})
	if err != nil {
		return err
	}
	quintVersion := strings.TrimSpace(version.Stdout)
	if quintVersion == "" {
		return fmt.Errorf("quint --version returned an empty version")
	}

	wrote := 0
	for _, flow := range options.Flows {
		files, err := buildOutputPlan(ctx, options.Root, flow, quintVersion, runner)
		if err != nil {
			return err
		}
		if err := applyOutputPlan(fs, options.Root, flow.FlowID, options.Mode, files); err != nil {
			return err
		}
		if options.Mode == ModeCheck {
			fmt.Fprintf(stdout, "fresh %s\n", flow.FlowID)
			continue
		}
		for _, file := range files {
			fmt.Fprintf(stdout, "wrote %s\n", file.Path)
		}
		wrote++
	}
	if options.Mode == ModeGenerate {
		fmt.Fprintf(stdout, "generated %d temporal flow(s)\n", wrote)
	}
	if options.Mode == ModeCheck {
		if err := lint.CheckAll(options.Root, options.Flows); err != nil {
			return err
		}
	}
	return nil
}

type OutputFile struct {
	Path string
	Data []byte
}

func buildOutputPlan(ctx context.Context, root string, flow model.Flow, quintVersion string, runner quint.Runner) ([]OutputFile, error) {
	rendered := quint.Render(flow)
	// Quint reads the model from disk, so ensure the model file
	// exists at its canonical path before running the verifier.
	// This is idempotent — write-and-overwrite — and required so a
	// fresh scenario can bootstrap from contract alone.
	modelAbs := fsadapter.Abs(root, flow.Layout.ModelPath)
	if err := os.MkdirAll(filepath.Dir(modelAbs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(modelAbs, []byte(rendered), 0o644); err != nil {
		return nil, err
	}
	built, err := artifact.Build(ctx, flow, artifact.BuildOptions{
		Root:         root,
		Rendered:     rendered,
		QuintVersion: quintVersion,
		RunQuint:     true,
		Runner:       runner,
	})
	if err != nil {
		return nil, err
	}
	artifactData, err := artifact.CanonicalJSON(built)
	if err != nil {
		return nil, err
	}
	modulePath, err := detectGoModulePath(root)
	if err != nil {
		return nil, err
	}
	renderedCode, err := codegen.Render(flow, built, codegen.Options{GoModulePath: modulePath})
	if err != nil {
		return nil, err
	}
	files := []OutputFile{
		{Path: flow.Layout.ModelPath, Data: []byte(rendered)},
		{Path: flow.Layout.ArtifactPath, Data: artifactData},
	}
	for _, file := range renderedCode.Files {
		files = append(files, OutputFile{Path: file.Path, Data: file.Data})
	}
	return files, nil
}

func applyOutputPlan(fs FileSystem, root string, flowID string, mode Mode, files []OutputFile) error {
	if mode != ModeCheck {
		for _, file := range files {
			target := fsadapter.Abs(root, file.Path)
			if err := fs.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := fs.WriteFile(target, file.Data, 0o644); err != nil {
				return err
			}
		}
		return nil
	}
	// Check mode aggregates every missing/stale artifact for this flow
	// into a single FreshnessError so the recorder can persist a
	// structured list instead of a single string-matched message.
	freshness := &FreshnessError{FlowID: flowID}
	for _, file := range files {
		target := fsadapter.Abs(root, file.Path)
		rel := normaliseArtifactPath(root, target)
		current, err := fs.ReadFile(target)
		if err != nil {
			freshness.Missing = append(freshness.Missing, rel)
			continue
		}
		if string(current) != string(file.Data) {
			freshness.Stale = append(freshness.Stale, rel)
		}
	}
	if len(freshness.Missing) > 0 {
		freshness.Kind = FreshnessMissing
		return freshness
	}
	if len(freshness.Stale) > 0 {
		freshness.Kind = FreshnessStale
		return freshness
	}
	return nil
}

// AssertFresh remains for callers that check a single artifact (notably
// the pipeline_test fakes). It now reports through *FreshnessError so
// recorders can inspect the typed payload uniformly.
func AssertFresh(fs FileSystem, path string, next []byte, flowID string) error {
	rel := filepath.ToSlash(filepath.Base(path))
	current, err := fs.ReadFile(path)
	if err != nil {
		return &FreshnessError{FlowID: flowID, Kind: FreshnessMissing, Missing: []string{rel}}
	}
	if string(current) != string(next) {
		return &FreshnessError{FlowID: flowID, Kind: FreshnessStale, Stale: []string{rel}}
	}
	return nil
}

// contractFromFlow rebuilds the minimal contract surface
// ValidateConventionalFiles needs from a compiled Flow. The
// validation only consults FlowID and Layout, so we don't need to
// rehydrate the rest.
func contractFromFlow(flow model.Flow) contract.Contract {
	return contract.Contract{
		FlowID: flow.FlowID,
		Layout: flow.Layout,
	}
}

// detectGoModulePath reads root/api/go.mod and returns the module name
// so codegen can emit correct import paths. Returns the empty string if
// the file is absent (the codegen falls back to the {{SCENARIO_ID}}
// template placeholder in that case).
func detectGoModulePath(root string) (string, error) {
	// Most scenarios live under <root>/api; the in-tree built-in flow
	// runs with root already pointing at the api dir, so probe both.
	for _, rel := range []string{filepath.Join("api", "go.mod"), "go.mod"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if rest, ok := strings.CutPrefix(line, "module "); ok {
				return strings.TrimSpace(rest), nil
			}
		}
	}
	return "", nil
}

type osFileSystem struct{}

func (osFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
