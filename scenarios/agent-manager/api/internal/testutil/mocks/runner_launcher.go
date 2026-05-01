package mocks

import (
	"context"
	"sync"

	adapterrunner "agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

var (
	_ adapterrunner.Launcher               = (*FakeLauncher)(nil)
	_ adapterrunner.SandboxLauncherFactory = (*FakeSandboxLauncherFactory)(nil)
)

// FakeLauncher records launch requests and returns the configured process/error.
type FakeLauncher struct {
	mu sync.Mutex

	Tag       string
	Process   adapterrunner.LaunchedProcess
	LaunchErr error
	calls     []adapterrunner.LaunchRequest
}

func NewFakeLauncher(tag string) *FakeLauncher {
	return &FakeLauncher{Tag: tag}
}

func (f *FakeLauncher) Launch(_ context.Context, req adapterrunner.LaunchRequest) (adapterrunner.LaunchedProcess, error) {
	f.mu.Lock()
	f.calls = append(f.calls, req)
	f.mu.Unlock()
	return f.Process, f.LaunchErr
}

func (f *FakeLauncher) LaunchCalls() []adapterrunner.LaunchRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]adapterrunner.LaunchRequest(nil), f.calls...)
}

// FakeSandboxLauncherFactory records the sandbox ID it was asked about and
// returns Launcher.
type FakeSandboxLauncherFactory struct {
	mu sync.Mutex

	Launcher  adapterrunner.Launcher
	calledIDs []uuid.UUID
}

func NewFakeSandboxLauncherFactory(launcher adapterrunner.Launcher) *FakeSandboxLauncherFactory {
	return &FakeSandboxLauncherFactory{Launcher: launcher}
}

func (f *FakeSandboxLauncherFactory) LauncherFor(sandboxID uuid.UUID) adapterrunner.Launcher {
	f.mu.Lock()
	f.calledIDs = append(f.calledIDs, sandboxID)
	f.mu.Unlock()
	return f.Launcher
}

func (f *FakeSandboxLauncherFactory) CalledIDs() []uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]uuid.UUID(nil), f.calledIDs...)
}
