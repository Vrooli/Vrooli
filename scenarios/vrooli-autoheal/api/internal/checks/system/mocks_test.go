package system

import (
	"context"
	"vrooli-autoheal/internal/checks"
)

// mockFSReader implements checks.FileSystemReader for testing.
type mockFSReader struct {
	statfsResult *checks.StatfsResult
	statfsErr    error
}

func (m *mockFSReader) Statfs(path string) (*checks.StatfsResult, error) {
	return m.statfsResult, m.statfsErr
}

// multiCallFSReader returns different results per path.
type multiCallFSReader struct {
	results map[string]*checks.StatfsResult
}

func (m *multiCallFSReader) Statfs(path string) (*checks.StatfsResult, error) {
	if result, ok := m.results[path]; ok {
		return result, nil
	}
	return nil, context.DeadlineExceeded
}

// mockPortReader implements checks.PortReader for testing.
type mockPortReader struct {
	portInfo *checks.PortInfo
	err      error
}

func (m *mockPortReader) ReadPortStats() (*checks.PortInfo, error) {
	return m.portInfo, m.err
}

// mockExecutor implements checks.CommandExecutor for testing.
type mockExecutor struct {
	combinedOutput []byte
	combinedErr    error
	output         []byte
	outputErr      error
	runErr         error
}

func (m *mockExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.combinedOutput, m.combinedErr
}

func (m *mockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.output, m.outputErr
}

func (m *mockExecutor) Run(ctx context.Context, name string, args ...string) error {
	return m.runErr
}

// mockProcReader implements checks.ProcReader for testing.
type mockProcReader struct {
	memInfo      *checks.MemInfo
	memInfoErr   error
	processes    []checks.ProcessInfo
	processesErr error
	environByPID map[int]map[string]string
	cmdlineByPID map[int]string
}

func (m *mockProcReader) ReadMeminfo() (*checks.MemInfo, error) {
	return m.memInfo, m.memInfoErr
}

func (m *mockProcReader) ListProcesses() ([]checks.ProcessInfo, error) {
	return m.processes, m.processesErr
}

func (m *mockProcReader) ReadProcessEnviron(pid int) (map[string]string, error) {
	if m.environByPID != nil {
		if env, ok := m.environByPID[pid]; ok {
			return env, nil
		}
	}
	return map[string]string{}, nil
}

func (m *mockProcReader) ReadProcessCmdline(pid int) (string, error) {
	if m.cmdlineByPID != nil {
		if cmdline, ok := m.cmdlineByPID[pid]; ok {
			return cmdline, nil
		}
	}

	for _, proc := range m.processes {
		if proc.PID == pid {
			return proc.Cmdline, nil
		}
	}

	return "", nil
}
