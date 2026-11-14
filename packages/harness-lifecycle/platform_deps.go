package agentinstall

import (
	"context"
	"io/fs"
	"os"
	"runtime"
	"time"

	"github.com/vrooli/cliresolve"
	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
)

// platformDeps is a per-call seam for host services used by agentinstall. It
// deliberately exposes operations rather than package types, keeping tests
// independent from process globals while production retains the same owners.
type platformDeps struct {
	goos            string
	goarch          string
	windows         bool
	darwin          bool
	loadManifest    func(string) (harnessManifest, error)
	resourceDataDir func(home, resource string) (string, error)
	commandOutput   func(context.Context, string, ...string) ([]byte, error)
	commandCombined func(context.Context, string, ...string) ([]byte, error)
	homeDir         func() (string, error)
	mkdirAll        func(string, fs.FileMode) error
	commitOwnedPath func(string, string) error
	writeOwnedFile  func(string, []byte, fs.FileMode) error
	ensureOwnedDir  func(string) (string, error)
	acquireLock     func(context.Context, string) (func(), error)
	downloadTimeout time.Duration
	controlTimeout  time.Duration
	permDir         fs.FileMode
	permPrivate     fs.FileMode
	permOwnerWrite  fs.FileMode
}

func defaultPlatformDeps() platformDeps {
	return platformDeps{
		goos:    runtime.GOOS,
		goarch:  runtime.GOARCH,
		windows: runtime.GOOS == string(hostreqspec.PlatformWindows),
		darwin:  runtime.GOOS == string(hostreqspec.PlatformDarwin),
		loadManifest: func(path string) (harnessManifest, error) {
			m, err := manifestpkg.Load(path)
			return harnessManifest{Name: m.Name, Acquisition: m.Acquisition}, err
		},
		resourceDataDir: func(home, resource string) (string, error) {
			resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", UserHomeDir: func() (string, error) { return home, nil }, EnvGet: os.Getenv})
			if err != nil {
				return "", err
			}
			paths, err := resolver.Resolve(storage.Options{ResourceID: resource})
			if err != nil {
				return "", err
			}
			return paths.DataDir, nil
		},
		commandOutput: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if home, err := os.UserHomeDir(); err == nil {
				if installed, err := cliresolve.Executable(home, name); err == nil {
					name = installed
				}
			}
			return shell.NewCommandContext(ctx, name, args...).Output()
		},
		commandCombined: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if home, err := os.UserHomeDir(); err == nil {
				if installed, err := cliresolve.Executable(home, name); err == nil {
					name = installed
				}
			}
			command := shell.NewCommandContext(ctx, name, args...)
			if isVersionProbe(args) && os.Geteuid() == 0 {
				// A privileged run (`sudo vrooli setup`) must not let an agent
				// write its state into the operator's home: it would leave
				// root-owned files that the agent later fails to clean up.
				scratch, err := os.MkdirTemp("", "vrooli-version-probe-")
				if err != nil {
					return nil, err
				}
				defer os.RemoveAll(scratch)
				command.Env = append(os.Environ(), "HOME="+scratch, "CODEX_HOME="+scratch)
			}
			return command.CombinedOutput()
		},
		homeDir:         config.HomeDir,
		mkdirAll:        os.MkdirAll,
		commitOwnedPath: config.CommitOwnedPathAtomic,
		writeOwnedFile:  config.WriteOwnedFileAtomic,
		ensureOwnedDir:  config.EnsureOwnedDir,
		acquireLock:     artifactlock.AcquireContext,
		downloadTimeout: tuning.AgentInstallDownloadTimeout(),
		controlTimeout:  tuning.ControlPlaneClientTimeout(),
		permDir:         tuning.PermDir,
		permPrivate:     tuning.PermPrivateDir,
		permOwnerWrite:  tuning.PermOwnerWrite,
	}
}

func defaultHarnessLoadDepsFromPlatform() harnessLoadDeps {
	platform := defaultPlatformDeps()
	return harnessLoadDeps{loadManifest: platform.loadManifest, homeDir: platform.homeDir}
}

func (d platformDeps) withDefaults() platformDeps {
	defaults := defaultPlatformDeps()
	if d.loadManifest == nil {
		d.loadManifest = defaults.loadManifest
	}
	if d.goos == "" {
		d.goos = defaults.goos
	}
	if d.goarch == "" {
		d.goarch = defaults.goarch
	}
	if d.resourceDataDir == nil {
		d.resourceDataDir = defaults.resourceDataDir
	}
	if d.commandOutput == nil {
		d.commandOutput = defaults.commandOutput
	}
	if d.commandCombined == nil {
		d.commandCombined = defaults.commandCombined
	}
	if d.homeDir == nil {
		d.homeDir = defaults.homeDir
	}
	if d.mkdirAll == nil {
		d.mkdirAll = defaults.mkdirAll
	}
	if d.commitOwnedPath == nil {
		d.commitOwnedPath = defaults.commitOwnedPath
	}
	if d.writeOwnedFile == nil {
		d.writeOwnedFile = defaults.writeOwnedFile
	}
	if d.ensureOwnedDir == nil {
		d.ensureOwnedDir = defaults.ensureOwnedDir
	}
	if d.acquireLock == nil {
		d.acquireLock = defaults.acquireLock
	}
	if d.downloadTimeout == 0 {
		d.downloadTimeout = defaults.downloadTimeout
	}
	if d.controlTimeout == 0 {
		d.controlTimeout = defaults.controlTimeout
	}
	if d.permDir == 0 {
		d.permDir = defaults.permDir
	}
	if d.permPrivate == 0 {
		d.permPrivate = defaults.permPrivate
	}
	if d.permOwnerWrite == 0 {
		d.permOwnerWrite = defaults.permOwnerWrite
	}
	return d
}

// isVersionProbe reports whether args ask an agent only for its version.
func isVersionProbe(args []string) bool { return len(args) == 1 && args[0] == "--version" }
