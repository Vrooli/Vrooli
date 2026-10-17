// Package mocks provides in-memory fakes for validation_run's seams.
package mocks

import (
	"context"
	"errors"
	"sync"
	"time"

	manifest "development-toolchain-validator/internal/manifest"
	vrun "development-toolchain-validator/internal/validation_run"
)

// FakeAgentManager is a programmable fake for AgentManagerClient.
type FakeAgentManager struct {
	mu sync.Mutex

	StartErr      error
	StartRunID    string
	WaitErr       error
	WaitResult    vrun.RunSummary
	StartCalls    int
	WaitCalls     int
	LastStartSpec vrun.SandboxedRunSpec
}

var _ vrun.AgentManagerClient = (*FakeAgentManager)(nil)

func (f *FakeAgentManager) StartSandboxedRun(_ context.Context, spec vrun.SandboxedRunSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StartCalls++
	f.LastStartSpec = spec
	if f.StartErr != nil {
		return "", f.StartErr
	}
	if f.StartRunID == "" {
		f.StartRunID = "fake-run-1"
	}
	return f.StartRunID, nil
}

func (f *FakeAgentManager) WaitForTerminal(_ context.Context, _ string, _ time.Duration) (vrun.RunSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.WaitCalls++
	if f.WaitErr != nil {
		return vrun.RunSummary{}, f.WaitErr
	}
	return f.WaitResult, nil
}

// FakeToolRunner is a programmable fake for ToolRunner.
type FakeToolRunner struct {
	Result vrun.ToolResult
	Err    error
	Calls  int

	// LastTool / LastSlug / LastPath capture the most recent invocation
	// arguments so tests can assert correct targeting.
	LastTool string
	LastSlug string
	LastPath string
}

var _ vrun.ToolRunner = (*FakeToolRunner)(nil)

func (f *FakeToolRunner) Invoke(_ context.Context, toolName, goldenSlug, goldenPath string) (vrun.ToolResult, error) {
	f.Calls++
	f.LastTool = toolName
	f.LastSlug = goldenSlug
	f.LastPath = goldenPath
	if f.Err != nil {
		return vrun.ToolResult{}, f.Err
	}
	return f.Result, nil
}

// FakeWorkspaceSandbox is a programmable fake for WorkspaceSandboxClient.
type FakeWorkspaceSandbox struct {
	Files map[string]string
	Err   error
}

var _ vrun.WorkspaceSandboxClient = (*FakeWorkspaceSandbox)(nil)

func (f *FakeWorkspaceSandbox) FetchPathContent(_ context.Context, _, path string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	c, ok := f.Files[path]
	if !ok {
		return "", errors.New("no such path")
	}
	return c, nil
}

// FakeGoldenMaterializer is a programmable fake for GoldenMaterializer.
type FakeGoldenMaterializer struct {
	Goldens map[string]vrun.MaterializedGolden
	Err     error
}

var _ vrun.GoldenMaterializer = (*FakeGoldenMaterializer)(nil)

func (f *FakeGoldenMaterializer) Materialize(_ context.Context, goldenSlug string) (vrun.MaterializedGolden, error) {
	if f.Err != nil {
		return vrun.MaterializedGolden{}, f.Err
	}
	if f.Goldens != nil {
		if g, ok := f.Goldens[goldenSlug]; ok {
			return g, nil
		}
	}
	return vrun.MaterializedGolden{GoldenSlug: goldenSlug, PhysicalPath: "/tmp/" + goldenSlug, LogicalRoot: goldenSlug}, nil
}

// FakeManifestSource is a programmable fake for ManifestSource.
type FakeManifestSource struct {
	Manifests map[[2]string]manifest.Manifest
	Err       error
}

var _ vrun.ManifestSource = (*FakeManifestSource)(nil)

func (f *FakeManifestSource) GetManifest(_ context.Context, skillID, goldenSlug string) (manifest.Manifest, error) {
	if f.Err != nil {
		return manifest.Manifest{}, f.Err
	}
	m, ok := f.Manifests[[2]string{skillID, goldenSlug}]
	if !ok {
		return manifest.Manifest{}, manifest.ErrManifestNotFound{SkillID: skillID, GoldenSlug: goldenSlug}
	}
	return m, nil
}
