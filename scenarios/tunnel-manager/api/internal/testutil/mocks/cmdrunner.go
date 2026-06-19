package mocks

import (
	"context"
	"sync"

	"tunnel-manager/internal/cmdrunner"
)

// CmdCall records one invocation of a FakeCmdRunner.
type CmdCall struct {
	Name string
	Args []string
}

// FakeCmdRunner is the testutil counterpart to cmdrunner.Default. It
// records every invocation and returns a scripted output/error so tests
// assert the exact argv (e.g. "systemctl restart cloudflared") and drive
// failure paths without shelling out to the host.
type FakeCmdRunner struct {
	mu sync.Mutex

	// Out is returned from every call.
	Out []byte
	// Err is returned from every call.
	Err error
	// Calls records every invocation in order.
	Calls []CmdCall
}

// Run is the cmdrunner.Runner implementation. Use it as the seam value:
//
//	fake := &mocks.FakeCmdRunner{}
//	svc := tunnel.NewService(repo, fake.Run)
func (f *FakeCmdRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, CmdCall{Name: name, Args: append([]string(nil), args...)})
	return f.Out, f.Err
}

// CallCount returns how many times Run was invoked.
func (f *FakeCmdRunner) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Calls)
}

// Compile-time guarantee that FakeCmdRunner.Run satisfies cmdrunner.Runner.
var _ cmdrunner.Runner = (*FakeCmdRunner)(nil).Run
