package storagepaths

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
)

const (
	AppID      = "vrooli"
	ScenarioID = "scenario-to-desktop"
)

// Locator resolves canonical runtime storage paths for scenario-to-desktop.
type Locator struct {
	resolver *storage.Resolver
	opts     storage.Options
}

// NewLocator creates a new locator using the default runtime profile.
func NewLocator() (*Locator, error) {
	return NewLocatorWith(storage.ResolverConfig{
		AppID:   AppID,
		Profile: storage.ProfileAuto,
	}, storage.Options{
		ScenarioID: ScenarioID,
	})
}

// NewLocatorWith creates a locator with explicit resolver config and options.
func NewLocatorWith(cfg storage.ResolverConfig, opts storage.Options) (*Locator, error) {
	if opts.ScenarioID == "" {
		opts.ScenarioID = ScenarioID
	}
	if cfg.AppID == "" {
		cfg.AppID = AppID
	}
	if cfg.Profile == "" {
		cfg.Profile = storage.ProfileAuto
	}
	resolver, err := storage.NewResolver(cfg)
	if err != nil {
		return nil, err
	}
	return &Locator{resolver: resolver, opts: opts}, nil
}

func (l *Locator) Resolver() *storage.Resolver {
	return l.resolver
}

func (l *Locator) Options() storage.Options {
	return l.opts
}

func (l *Locator) Paths() (storage.Paths, error) {
	return l.resolver.Resolve(l.opts)
}

func (l *Locator) EnsureAll() (storage.Paths, error) {
	return storage.EnsureAllDirs(l.resolver, l.opts, 0)
}

func (l *Locator) DataRoot() (string, error) {
	return l.ensure(storage.ClassData)
}

func (l *Locator) path(class storage.Class, rel string) (string, error) {
	return l.resolver.Path(l.opts, class, rel)
}

func (l *Locator) ensure(class storage.Class) (string, error) {
	return storage.EnsureClassDir(l.resolver, l.opts, class, 0)
}

func (l *Locator) DeployTargetsPath() (string, error) {
	return l.path(storage.ClassConfig, "deploy-targets.json")
}

func (l *Locator) TelemetryDir() (string, error) {
	return l.path(storage.ClassLogs, filepath.Join("deployment", "telemetry"))
}

func (l *Locator) TelemetryFilePath(scenario string) (string, error) {
	if scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}
	return l.path(storage.ClassLogs, filepath.Join("deployment", "telemetry", scenario+".jsonl"))
}

func (l *Locator) RecordsPath() (string, error) {
	return l.path(storage.ClassData, "desktop_records_v2.json")
}

func (l *Locator) SmokeTestsPath() (string, error) {
	return l.path(storage.ClassData, "smoke_tests_v2.json")
}

func (l *Locator) InvestigationsDir() (string, error) {
	return l.path(storage.ClassData, "investigations")
}

func (l *Locator) ScenarioStateDir() (string, error) {
	return l.path(storage.ClassState, "scenario-state")
}

func (l *Locator) PipelineStateDir() (string, error) {
	return l.path(storage.ClassState, "pipelines")
}

func (l *Locator) PipelineIndexDir() (string, error) {
	return l.path(storage.ClassState, "indexes")
}

func (l *Locator) LiveDesktopDir() (string, error) {
	return l.path(storage.ClassState, "livedesktop")
}

func (l *Locator) CapturesMetaPath() (string, error) {
	return l.path(storage.ClassData, filepath.Join("captures", "captures_meta.json"))
}

func (l *Locator) CapturesDir() (string, error) {
	return l.path(storage.ClassData, filepath.Join("captures", "files"))
}

func (l *Locator) StagingRoot() (string, error) {
	return l.path(storage.ClassCache, "staging")
}

func (l *Locator) EnsureTelemetryDir() (string, error) {
	path, err := l.TelemetryDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsureScenarioStateDir() (string, error) {
	path, err := l.ScenarioStateDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsurePipelineStateDir() (string, error) {
	path, err := l.PipelineStateDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsurePipelineIndexDir() (string, error) {
	path, err := l.PipelineIndexDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsureInvestigationsDir() (string, error) {
	path, err := l.InvestigationsDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsureCapturesDir() (string, error) {
	path, err := l.CapturesDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsureLiveDesktopDir() (string, error) {
	path, err := l.LiveDesktopDir()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func (l *Locator) EnsureStagingRoot() (string, error) {
	path, err := l.StagingRoot()
	if err != nil {
		return "", err
	}
	return ensurePath(path)
}

func ensurePath(path string) (string, error) {
	if err := os.MkdirAll(path, storage.DefaultDirPerm); err != nil {
		return "", err
	}
	return path, nil
}
