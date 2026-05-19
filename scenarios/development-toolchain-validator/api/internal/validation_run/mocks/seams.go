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

	StartErr        error
	StartRunID      string
	WaitErr         error
	WaitResult      vrun.RunSummary
	StartCalls      int
	WaitCalls       int
	LastStartSpec   vrun.SandboxedRunSpec
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
}

var _ vrun.ToolRunner = (*FakeToolRunner)(nil)

func (f *FakeToolRunner) Invoke(context.Context, string, string) (vrun.ToolResult, error) {
	f.Calls++
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

// FakeGoldenSource is a programmable fake for GoldenSource.
type FakeGoldenSource struct {
	Paths map[string]string
	Err   error
}

var _ vrun.GoldenSource = (*FakeGoldenSource)(nil)

func (f *FakeGoldenSource) GoldenPath(_ context.Context, goldenSlug string) (string, error) {
	if f.Err != nil {
		return "", f.Err
	}
	return f.Paths[goldenSlug], nil
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
