package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"flow-verifier/internal/codegen"
	"flow-verifier/internal/flows/contract"
	"flow-verifier/internal/flows/model"
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
	for _, file := range files {
		if err := writeOrCheck(fs, root, file, flowID, mode); err != nil {
			return err
		}
	}
	return nil
}

func writeOrCheck(fs FileSystem, root string, file OutputFile, flowID string, mode Mode) error {
	target := fsadapter.Abs(root, file.Path)
	if mode == ModeCheck {
		return AssertFresh(fs, target, file.Data, flowID)
	}
	if err := fs.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return fs.WriteFile(target, file.Data, 0o644)
}

func AssertFresh(fs FileSystem, path string, next []byte, flowID string) error {
	current, err := fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s is missing. Run: flow-verifier verify run --flow %s", filepath.ToSlash(path), flowID)
	}
	if string(current) != string(next) {
		return fmt.Errorf("%s is stale. Run: flow-verifier verify run --flow %s", filepath.ToSlash(path), flowID)
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
	data, err := os.ReadFile(filepath.Join(root, "api", "go.mod"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(rest), nil
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
