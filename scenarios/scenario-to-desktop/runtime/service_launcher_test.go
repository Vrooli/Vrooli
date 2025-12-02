package bundleruntime

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// MockProcess implements Process for testing.
type MockProcess struct {
	pid      int
	waitErr  error
	signaled bool
	killed   bool
	waitCh   chan struct{}
}

func NewMockProcess(pid int) *MockProcess {
	return &MockProcess{
		pid:    pid,
		waitCh: make(chan struct{}),
	}
}

func (p *MockProcess) Pid() int { return p.pid }

func (p *MockProcess) Wait() error {
	<-p.waitCh
	return p.waitErr
}

func (p *MockProcess) Signal(sig os.Signal) error {
	p.signaled = true
	return nil
}

func (p *MockProcess) Kill() error {
	p.killed = true
	// Only close channel once - use select to avoid panic
	select {
	case <-p.waitCh:
		// Already closed
	default:
		close(p.waitCh)
	}
	return nil
}

// MockProcessRunner implements ProcessRunner for testing.
type MockProcessRunner struct {
	processes    []*MockProcess
	shouldFail   bool
	startedCmds  []string
	currentIndex int
}

func (r *MockProcessRunner) Start(ctx context.Context, cmd string, args []string, env []string, dir string, stdout, stderr io.Writer) (Process, error) {
	r.startedCmds = append(r.startedCmds, cmd)
	if r.shouldFail {
		return nil, errors.New("start failed")
	}
	if r.currentIndex < len(r.processes) {
		proc := r.processes[r.currentIndex]
		r.currentIndex++
		return proc, nil
	}
	// Return a default process
	return NewMockProcess(12345), nil
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want *int
	}{
		{"nil error", nil, intPtr(0)},
		{"generic error", errors.New("failed"), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := exitCode(tt.err)
			if tt.want == nil {
				if got != nil {
					t.Errorf("exitCode() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("exitCode() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("exitCode() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func TestStartService_DryRun(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			DryRun: true,
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{
						ID: "api",
						Binaries: map[string]manifest.Binary{
							"linux_amd64": {Path: "bin/api"},
						},
					},
				},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	svc := manifest.Service{
		ID: "api",
		Binaries: map[string]manifest.Binary{
			"linux_amd64": {Path: "bin/api"},
		},
	}
	ctx := context.Background()

	if err := s.startService(ctx, svc); err != nil {
		t.Fatalf("startService() error = %v", err)
	}

	status := s.serviceStatus["api"]
	if !status.Ready {
		t.Error("startService() dry-run should set Ready=true")
	}
	if status.Message != "dry-run" {
		t.Errorf("startService() dry-run message = %q, want %q", status.Message, "dry-run")
	}
}

func TestStartService_NoBinary(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	svc := manifest.Service{ID: "api", Binaries: map[string]manifest.Binary{}} // No matching binary
	ctx := context.Background()

	err := s.startService(ctx, svc)
	if err == nil {
		t.Error("startService() expected error for missing binary")
	}
}

func TestPrepareServiceDirs(t *testing.T) {
	tmp := t.TempDir()
	mockFS := NewMockFileSystem()

	s := &Supervisor{
		fs:      mockFS,
		appData: tmp,
	}

	svc := manifest.Service{
		ID:       "api",
		DataDirs: []string{"data", "cache", "logs"},
	}

	if err := s.prepareServiceDirs(svc); err != nil {
		t.Fatalf("prepareServiceDirs() error = %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		tmp + "/data",
		tmp + "/cache",
		tmp + "/logs",
	}
	for _, dir := range expectedDirs {
		if !mockFS.dirs[dir] {
			t.Errorf("prepareServiceDirs() didn't create %q", dir)
		}
	}
}

func TestPrepareServiceDirs_WithLogDir(t *testing.T) {
	tmp := t.TempDir()

	s := &Supervisor{
		fs:      RealFileSystem{},
		appData: tmp,
	}

	svc := manifest.Service{
		ID:     "api",
		LogDir: "logs/api.log",
	}

	if err := s.prepareServiceDirs(svc); err != nil {
		t.Fatalf("prepareServiceDirs() error = %v", err)
	}

	// Verify log file was touched
	logPath := tmp + "/logs/api.log"
	if _, err := s.fs.Stat(logPath); err != nil {
		t.Errorf("prepareServiceDirs() didn't create log file: %v", err)
	}
}

func TestLogWriter(t *testing.T) {
	tmp := t.TempDir()

	s := &Supervisor{
		fs:      RealFileSystem{},
		appData: tmp,
	}

	tests := []struct {
		name      string
		logDir    string
		wantNil   bool
		wantPath  string
		wantError bool
		setupDir  bool
	}{
		{"empty log dir", "", true, "", false, false},
		{"valid log dir", "logs/api.log", false, tmp + "/logs/api.log", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parent directory if needed
			if tt.setupDir && tt.logDir != "" {
				logPath := tmp + "/" + tt.logDir
				if err := s.fs.MkdirAll(logPath[:len(logPath)-len("/api.log")], 0o755); err != nil {
					t.Fatalf("MkdirAll() error = %v", err)
				}
			}

			svc := manifest.Service{ID: "api", LogDir: tt.logDir}
			writer, path, err := s.logWriter(svc)

			if (err != nil) != tt.wantError {
				t.Errorf("logWriter() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantNil && writer != nil {
				t.Error("logWriter() returned non-nil writer, expected nil")
			}

			if !tt.wantNil && writer == nil {
				t.Error("logWriter() returned nil writer")
			}

			if path != tt.wantPath {
				t.Errorf("logWriter() path = %q, want %q", path, tt.wantPath)
			}

			if writer != nil {
				writer.Close()
			}
		})
	}
}

func TestGracefulStop(t *testing.T) {
	mockClock := NewMockClock(time.Now())
	s := &Supervisor{clock: mockClock}

	proc := NewMockProcess(123)
	svcProc := &serviceProcess{proc: proc}

	ctx := context.Background()

	// Simulate process exiting after signal - close before calling gracefulStop
	// to simulate a process that exits immediately
	close(proc.waitCh)

	s.gracefulStop(ctx, svcProc)

	if !proc.signaled {
		t.Error("gracefulStop() didn't signal process")
	}
}

func TestGracefulStop_Timeout(t *testing.T) {
	mockClock := NewMockClock(time.Now())
	s := &Supervisor{clock: mockClock}

	proc := NewMockProcess(123)
	svcProc := &serviceProcess{proc: proc}

	ctx := context.Background()

	// Don't close waitCh, so process doesn't exit gracefully
	s.gracefulStop(ctx, svcProc)

	if !proc.signaled {
		t.Error("gracefulStop() didn't signal process")
	}
	if !proc.killed {
		t.Error("gracefulStop() didn't kill process after timeout")
	}
}

func TestServiceProcess(t *testing.T) {
	proc := NewMockProcess(12345)
	svcProc := &serviceProcess{
		proc:    proc,
		logPath: "/var/log/api.log",
		service: manifest.Service{ID: "api"},
		started: time.Now(),
	}

	if svcProc.proc.Pid() != 12345 {
		t.Errorf("serviceProcess.proc.Pid() = %d, want 12345", svcProc.proc.Pid())
	}
	if svcProc.logPath != "/var/log/api.log" {
		t.Errorf("serviceProcess.logPath = %q, want %q", svcProc.logPath, "/var/log/api.log")
	}
}

func TestLaunchServices_EmptyManifest(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{},
			},
		},
	}

	ctx := context.Background()
	err := s.launchServices(ctx)
	if err != nil {
		t.Errorf("launchServices() error = %v for empty services", err)
	}
}

func TestLaunchServices_ContextCanceled(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "api"},
				},
			},
		},
		serviceStatus: make(map[string]ServiceStatus),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := s.launchServices(ctx)
	if err == nil {
		t.Error("launchServices() expected error for canceled context")
	}
}

func TestStopServices_NoProcesses(t *testing.T) {
	s := &Supervisor{
		opts: Options{
			Manifest: &manifest.Manifest{
				Services: []manifest.Service{
					{ID: "api"},
				},
			},
		},
		procs: make(map[string]*serviceProcess),
		clock: NewMockClock(time.Now()),
	}

	ctx := context.Background()

	// Should not panic
	s.stopServices(ctx)
}
