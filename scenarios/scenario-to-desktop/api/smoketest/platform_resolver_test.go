package smoketest_test

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestPlatformResolver_CurrentPlatform(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()
	resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)

	platform := resolver.CurrentPlatform()

	// The actual result depends on runtime.GOOS
	switch runtime.GOOS {
	case "windows":
		if platform != "win" {
			t.Errorf("CurrentPlatform() = %q, want %q on Windows", platform, "win")
		}
	case "darwin":
		if platform != "mac" {
			t.Errorf("CurrentPlatform() = %q, want %q on macOS", platform, "mac")
		}
	default:
		if platform != "linux" {
			t.Errorf("CurrentPlatform() = %q, want %q on Linux", platform, "linux")
		}
	}
}

func TestPlatformResolver_ResolveCommand_Linux(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()

	tests := []struct {
		name         string
		artifactPath string
		setupFS      func(*mocks.MockFileSystem)
		wantDisplay  string
		wantErr      bool
	}{
		{
			name:         "valid AppImage",
			artifactPath: "/path/to/MyApp.AppImage",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddFileInfo("/path/to/MyApp.AppImage", &mocks.MockFileInfo{
					NameVal: "MyApp.AppImage",
					ModeVal: 0o644, // Not executable yet
				})
			},
			wantDisplay: "/path/to/MyApp.AppImage --smoke-test (with AppImage extract fallback)",
			wantErr:     false,
		},
		{
			name:         "already executable AppImage",
			artifactPath: "/path/to/MyApp.AppImage",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddFileInfo("/path/to/MyApp.AppImage", &mocks.MockFileInfo{
					NameVal: "MyApp.AppImage",
					ModeVal: 0o755, // Already executable
				})
			},
			wantDisplay: "/path/to/MyApp.AppImage --smoke-test (with AppImage extract fallback)",
			wantErr:     false,
		},
		{
			name:         "unsupported artifact type",
			artifactPath: "/path/to/MyApp.deb",
			setupFS:      func(fs *mocks.MockFileSystem) {},
			wantErr:      true,
		},
		{
			name:         "stat error",
			artifactPath: "/path/to/MyApp.AppImage",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.StatFunc = func(path string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := mocks.NewMockFileSystem()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}

			resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
			cmd, args, display, err := resolver.ResolveCommand("linux", tt.artifactPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Linux AppImage smoke test uses a shell wrapper with an extract+run fallback
			// to avoid requiring FUSE mounts.
			if cmd != "sh" {
				t.Errorf("ResolveCommand() cmd = %q, want %q", cmd, "sh")
			}
			if len(args) < 5 {
				t.Errorf("ResolveCommand() args too short = %v", args)
			} else {
				if args[0] != "-c" {
					t.Errorf("ResolveCommand() args[0] = %q, want %q", args[0], "-c")
				}
				if args[1] == "" || !strings.Contains(args[1], "--appimage-extract") {
					t.Errorf("ResolveCommand() args[1] missing extract fallback script")
				}
				if !strings.Contains(args[1], "APPIMAGE_EXTRACT_AND_RUN=1") {
					t.Errorf("ResolveCommand() args[1] must avoid FUSE mounts")
				}
				if args[2] != "sh" {
					t.Errorf("ResolveCommand() args[2] = %q, want %q", args[2], "sh")
				}
				if args[3] != tt.artifactPath {
					t.Errorf("ResolveCommand() args[3] = %q, want %q", args[3], tt.artifactPath)
				}
				if args[4] != "--smoke-test" {
					t.Errorf("ResolveCommand() args[4] = %q, want %q", args[4], "--smoke-test")
				}
			}
			if display != tt.wantDisplay {
				t.Errorf("ResolveCommand() display = %q, want %q", display, tt.wantDisplay)
			}
		})
	}
}

func TestPlatformResolver_ResolveCommand_AcceptsConcretePipelineTarget(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()
	artifact := "/path/to/MyApp.AppImage"
	fs.AddFileInfo(artifact, &mocks.MockFileInfo{NameVal: "MyApp.AppImage", ModeVal: 0o755})
	resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)

	if _, _, _, err := resolver.ResolveCommand("linux-amd64", artifact); err != nil {
		t.Fatalf("ResolveCommand(linux-amd64) = %v", err)
	}
}

func TestPlatformResolver_ResolveCommand_Windows(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()
	resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)

	tests := []struct {
		name         string
		artifactPath string
		wantCmd      string
		wantErr      bool
	}{
		{
			name:         "valid exe",
			artifactPath: "C:\\path\\to\\MyApp.exe",
			wantCmd:      "C:\\path\\to\\MyApp.exe",
			wantErr:      false,
		},
		{
			name:         "exe lowercase",
			artifactPath: "C:\\path\\to\\myapp.EXE",
			wantCmd:      "C:\\path\\to\\myapp.EXE",
			wantErr:      false,
		},
		{
			name:         "unsupported artifact",
			artifactPath: "C:\\path\\to\\myapp.msi",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, _, err := resolver.ResolveCommand("win", tt.artifactPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cmd != tt.wantCmd {
				t.Errorf("ResolveCommand() cmd = %q, want %q", cmd, tt.wantCmd)
			}
		})
	}
}

func TestPlatformResolver_ResolveCommand_Mac(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()

	tests := []struct {
		name         string
		artifactPath string
		setupFS      func(*mocks.MockFileSystem)
		wantCmd      string
		wantErr      bool
	}{
		{
			name:         "valid app bundle",
			artifactPath: "/path/to/MyApp.app",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddDirEntries("/path/to/MyApp.app/Contents/MacOS", []os.DirEntry{
					&mocks.MockDirEntry{EntryName: "MyApp", EntryIsDir: false},
				})
				fs.AddFileInfo("/path/to/MyApp.app/Contents/MacOS/MyApp", &mocks.MockFileInfo{
					NameVal: "MyApp",
					ModeVal: 0o755,
				})
			},
			wantCmd: "/path/to/MyApp.app/Contents/MacOS/MyApp",
			wantErr: false,
		},
		{
			name:         "unsupported artifact",
			artifactPath: "/path/to/MyApp.dmg",
			setupFS:      func(fs *mocks.MockFileSystem) {},
			wantErr:      true,
		},
		{
			name:         "empty MacOS directory",
			artifactPath: "/path/to/MyApp.app",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddDirEntries("/path/to/MyApp.app/Contents/MacOS", []os.DirEntry{})
			},
			wantErr: true,
		},
		{
			name:         "MacOS dir read error",
			artifactPath: "/path/to/MyApp.app",
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.ReadDirFunc = func(path string) ([]os.DirEntry, error) {
					return nil, os.ErrNotExist
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := mocks.NewMockFileSystem()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}

			resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
			cmd, _, _, err := resolver.ResolveCommand("mac", tt.artifactPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if cmd != tt.wantCmd {
				t.Errorf("ResolveCommand() cmd = %q, want %q", cmd, tt.wantCmd)
			}
		})
	}
}

func TestPlatformResolver_ResolveCommand_UnsupportedPlatform(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()
	resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)

	_, _, _, err := resolver.ResolveCommand("freebsd", "/path/to/app")
	if err == nil {
		t.Error("ResolveCommand() expected error for unsupported platform")
	}
}

func TestPlatformResolver_RequiresHeadlessWrapper(t *testing.T) {
	if runtime.GOOS != "linux" {
		repocontracttest.SkipPlatform(t, "RequiresHeadlessWrapper tests only run on Linux")
	}

	config := smoketest.DefaultConfig()

	tests := []struct {
		name       string
		setupEnv   func(*mocks.MockEnvironmentReader)
		setupExec  func(*mocks.MockProcessExecutor)
		wantNeeded bool
		wantCmd    string
		wantErr    bool
	}{
		{
			name: "DISPLAY set - no wrapper needed",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.SetEnv("DISPLAY", ":0")
			},
			setupExec:  func(exec *mocks.MockProcessExecutor) {},
			wantNeeded: false,
		},
		{
			name:     "no DISPLAY, xvfb-run available",
			setupEnv: func(env *mocks.MockEnvironmentReader) {},
			setupExec: func(exec *mocks.MockProcessExecutor) {
				exec.AddLookPath("xvfb-run", "/usr/bin/xvfb-run")
			},
			wantNeeded: true,
			wantCmd:    "xvfb-run",
		},
		{
			name:     "no DISPLAY, xvfb-run unavailable",
			setupEnv: func(env *mocks.MockEnvironmentReader) {},
			setupExec: func(exec *mocks.MockProcessExecutor) {
				exec.AddLookPathError("xvfb-run", os.ErrNotExist)
			},
			wantNeeded: true,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			executor := mocks.NewMockProcessExecutor()
			fs := mocks.NewMockFileSystem()

			if tt.setupEnv != nil {
				tt.setupEnv(envReader)
			}
			if tt.setupExec != nil {
				tt.setupExec(executor)
			}

			resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
			needed, cmd, _, err := resolver.RequiresHeadlessWrapper()

			if (err != nil) != tt.wantErr {
				t.Errorf("RequiresHeadlessWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if needed != tt.wantNeeded {
				t.Errorf("RequiresHeadlessWrapper() needed = %v, want %v", needed, tt.wantNeeded)
			}
			if !tt.wantErr && tt.wantNeeded && cmd != tt.wantCmd {
				t.Errorf("RequiresHeadlessWrapper() cmd = %q, want %q", cmd, tt.wantCmd)
			}
		})
	}
}

// Behavior-based tests that work on any platform using platform override
func TestPlatformResolver_HeadlessWrapper_WithOverride(t *testing.T) {
	config := smoketest.DefaultConfig()

	tests := []struct {
		name             string
		platformOverride string
		setupEnv         func(*mocks.MockEnvironmentReader)
		setupExec        func(*mocks.MockProcessExecutor)
		wantNeeded       bool
		wantCmd          string
		wantErr          bool
	}{
		{
			name:             "linux with DISPLAY - no wrapper",
			platformOverride: "linux",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.SetEnv("DISPLAY", ":0")
			},
			wantNeeded: false,
		},
		{
			name:             "linux without DISPLAY, xvfb available",
			platformOverride: "linux",
			setupEnv:         func(env *mocks.MockEnvironmentReader) {},
			setupExec: func(exec *mocks.MockProcessExecutor) {
				exec.AddLookPath("xvfb-run", "/usr/bin/xvfb-run")
			},
			wantNeeded: true,
			wantCmd:    "xvfb-run",
		},
		{
			name:             "linux without DISPLAY, xvfb unavailable",
			platformOverride: "linux",
			setupEnv:         func(env *mocks.MockEnvironmentReader) {},
			setupExec: func(exec *mocks.MockProcessExecutor) {
				exec.AddLookPathError("xvfb-run", os.ErrNotExist)
			},
			wantNeeded: true,
			wantErr:    true,
		},
		{
			name:             "mac - no wrapper needed",
			platformOverride: "mac",
			wantNeeded:       false,
		},
		{
			name:             "windows - no wrapper needed",
			platformOverride: "win",
			wantNeeded:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			executor := mocks.NewMockProcessExecutor()
			fs := mocks.NewMockFileSystem()

			if tt.setupEnv != nil {
				tt.setupEnv(envReader)
			}
			if tt.setupExec != nil {
				tt.setupExec(executor)
			}

			resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
			resolver.SetPlatformOverride(tt.platformOverride)
			needed, cmd, _, err := resolver.RequiresHeadlessWrapper()

			if (err != nil) != tt.wantErr {
				t.Errorf("RequiresHeadlessWrapper() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if needed != tt.wantNeeded {
				t.Errorf("RequiresHeadlessWrapper() needed = %v, want %v", needed, tt.wantNeeded)
			}
			if !tt.wantErr && tt.wantNeeded && cmd != tt.wantCmd {
				t.Errorf("RequiresHeadlessWrapper() cmd = %q, want %q", cmd, tt.wantCmd)
			}
		})
	}
}

func TestPlatformResolver_ResolveCommand_CrossPlatform(t *testing.T) {
	config := smoketest.DefaultConfig()

	// Test Linux AppImage resolution with mocked filesystem
	t.Run("linux AppImage with mocked filesystem", func(t *testing.T) {
		executor := mocks.NewMockProcessExecutor()
		envReader := mocks.NewMockEnvironmentReader()
		fs := mocks.NewMockFileSystem()

		// Mock the file as non-executable initially
		fs.AddFileInfo("/path/to/MyApp.AppImage", &mocks.MockFileInfo{
			NameVal: "MyApp.AppImage",
			ModeVal: 0o644,
		})

		resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
		cmd, args, display, err := resolver.ResolveCommand("linux", "/path/to/MyApp.AppImage")
		if err != nil {
			t.Fatalf("ResolveCommand() error = %v", err)
		}
		if cmd != "sh" {
			t.Errorf("cmd = %q, want %q", cmd, "sh")
		}
		if len(args) < 5 || args[4] != "--smoke-test" {
			t.Errorf("args = %v, want shell wrapper ending with --smoke-test", args)
		}
		if display != "/path/to/MyApp.AppImage --smoke-test (with AppImage extract fallback)" {
			t.Errorf("display = %q, want %q", display, "/path/to/MyApp.AppImage --smoke-test (with AppImage extract fallback)")
		}

		// Verify chmod was called to make it executable
		if len(fs.ChmodCalls) != 1 {
			t.Errorf("Expected 1 chmod call, got %d", len(fs.ChmodCalls))
		}
	})

	// Test Mac .app bundle resolution with mocked filesystem
	t.Run("mac app bundle with mocked filesystem", func(t *testing.T) {
		executor := mocks.NewMockProcessExecutor()
		envReader := mocks.NewMockEnvironmentReader()
		fs := mocks.NewMockFileSystem()

		// Mock the directory structure
		fs.AddDirEntries("/path/to/MyApp.app/Contents/MacOS", []os.DirEntry{
			&mocks.MockDirEntry{EntryName: "MyApp", EntryIsDir: false},
		})
		fs.AddFileInfo("/path/to/MyApp.app/Contents/MacOS/MyApp", &mocks.MockFileInfo{
			NameVal: "MyApp",
			ModeVal: 0o755,
		})

		resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
		cmd, args, _, err := resolver.ResolveCommand("mac", "/path/to/MyApp.app")
		if err != nil {
			t.Fatalf("ResolveCommand() error = %v", err)
		}
		if cmd != "/path/to/MyApp.app/Contents/MacOS/MyApp" {
			t.Errorf("cmd = %q, want %q", cmd, "/path/to/MyApp.app/Contents/MacOS/MyApp")
		}
		if len(args) != 1 || args[0] != "--smoke-test" {
			t.Errorf("args = %v, want [--smoke-test]", args)
		}
	})

	// Test Windows .exe resolution
	t.Run("windows exe resolution", func(t *testing.T) {
		executor := mocks.NewMockProcessExecutor()
		envReader := mocks.NewMockEnvironmentReader()
		fs := mocks.NewMockFileSystem()

		resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)
		cmd, args, _, err := resolver.ResolveCommand("win", "C:\\path\\to\\MyApp.exe")
		if err != nil {
			t.Fatalf("ResolveCommand() error = %v", err)
		}
		if cmd != "C:\\path\\to\\MyApp.exe" {
			t.Errorf("cmd = %q, want %q", cmd, "C:\\path\\to\\MyApp.exe")
		}
		if len(args) != 1 || args[0] != "--smoke-test" {
			t.Errorf("args = %v, want [--smoke-test]", args)
		}
	})
}

func TestPlatformResolver_SetPlatformOverride(t *testing.T) {
	config := smoketest.DefaultConfig()
	executor := mocks.NewMockProcessExecutor()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()

	resolver := smoketest.NewPlatformResolver(executor, config, envReader, fs)

	// Test setting override
	resolver.SetPlatformOverride("mac")
	if resolver.CurrentPlatform() != "mac" {
		t.Errorf("CurrentPlatform() = %q after override, want %q", resolver.CurrentPlatform(), "mac")
	}

	resolver.SetPlatformOverride("win")
	if resolver.CurrentPlatform() != "win" {
		t.Errorf("CurrentPlatform() = %q after override, want %q", resolver.CurrentPlatform(), "win")
	}

	// Test clearing override
	resolver.SetPlatformOverride("")
	// Should return to actual platform
	expected := "linux"
	switch runtime.GOOS {
	case "windows":
		expected = "win"
	case "darwin":
		expected = "mac"
	}
	if resolver.CurrentPlatform() != expected {
		t.Errorf("CurrentPlatform() = %q after clearing override, want %q", resolver.CurrentPlatform(), expected)
	}
}
