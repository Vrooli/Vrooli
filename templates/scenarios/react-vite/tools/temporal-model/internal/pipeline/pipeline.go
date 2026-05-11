package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"react-vite-temporal-model/internal/artifact"
	"react-vite-temporal-model/internal/codegen"
	"react-vite-temporal-model/internal/contract"
	"react-vite-temporal-model/internal/filesystem"
	"react-vite-temporal-model/internal/model"
	"react-vite-temporal-model/internal/quint"
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
			if err := validateReplayFixture(flow, options.Root); err != nil {
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
	return nil
}

type OutputFile struct {
	Path string
	Data []byte
}

func buildOutputPlan(ctx context.Context, root string, flow model.Flow, quintVersion string, runner quint.Runner) ([]OutputFile, error) {
	rendered := quint.Render(flow)
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
	renderedCode, err := codegen.Render(flow, built)
	if err != nil {
		return nil, err
	}
	files := []OutputFile{
		{Path: flow.Outputs.ModelPath, Data: []byte(rendered)},
		{Path: flow.Outputs.ArtifactPath, Data: artifactData},
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
	target := filesystem.Abs(root, file.Path)
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
		return fmt.Errorf("%s is missing. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	if string(current) != string(next) {
		return fmt.Errorf("%s is stale. Run cd tools/temporal-model && GOWORK=off go run . generate --root ../.. --flow %s", filepath.ToSlash(path), flowID)
	}
	return nil
}

func validateReplayFixture(flow model.Flow, root string) error {
	return contract.ValidateReplayFixturePaths(
		root,
		flow.FlowID,
		flow.Replay.Kind,
		flow.Outputs.ReplayTestPath,
		flow.Replay.FixtureModule,
		flow.Replay.FixtureExport,
	)
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
