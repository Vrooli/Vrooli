package paths

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

const scenarioRoot = "browser-automation-studio"

// RecordingsRootProvider resolves recording artifacts under the request's
// primary or lease-owned data root. It preserves BAS_RECORDINGS_ROOT as the
// production location while making test-mode writes disposable.
type RecordingsRootProvider struct {
	roots  *filerouting.RoutedRoots
	subdir string
}

// NewRecordingsRootProvider constructs the single file-routing seam used by
// the API process. The data root is the parent of the configured recordings
// directory so a custom BAS_RECORDINGS_ROOT remains authoritative.
func NewRecordingsRootProvider(log *logrus.Logger) (*RecordingsRootProvider, error) {
	recordingsRoot := ResolveRecordingsRoot(log)
	if recordingsRoot == "" {
		return nil, fmt.Errorf("resolve recordings root")
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return nil, fmt.Errorf("create storage resolver: %w", err)
	}
	primary, err := resolver.Resolve(storage.Options{ScenarioID: scenarioRoot})
	if err != nil {
		return nil, fmt.Errorf("resolve storage roots: %w", err)
	}
	primary.DataDir = filepath.Dir(recordingsRoot)
	return &RecordingsRootProvider{roots: filerouting.New(primary), subdir: filepath.Base(recordingsRoot)}, nil
}

func (p *RecordingsRootProvider) Root(ctx context.Context) (string, error) {
	if p == nil || p.roots == nil {
		return "", fmt.Errorf("recordings root provider is nil")
	}
	base, err := p.roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, p.subdir), nil
}

// ProjectsRoot resolves project folders from the same request-aware data root
// as execution recordings. This keeps project-backed workflows inside the
// lease-owned filesystem during routed validation, instead of scanning or
// mutating the primary demo tree.
func (p *RecordingsRootProvider) ProjectsRoot(ctx context.Context) (string, error) {
	if p == nil || p.roots == nil {
		return "", fmt.Errorf("recordings root provider is nil")
	}
	base, err := p.roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "projects"), nil
}

func (p *RecordingsRootProvider) RecordWrite(ctx context.Context) {
	if p != nil && p.roots != nil {
		p.roots.RecordWrite(ctx)
	}
}

// FileRoots exposes the routing service's leased roots without leaking path
// selection to callers that only need the execution artifact root.
func (p *RecordingsRootProvider) FileRoots() *filerouting.RoutedRoots {
	if p == nil {
		return nil
	}
	return p.roots
}

// ResolveRecordingsRoot returns an absolute path for storing recording assets.
func ResolveRecordingsRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_RECORDINGS_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_RECORDINGS_ROOT without normalization")
		}
		return value
	}

	if resolved := resolveScenarioStoragePath("recordings"); resolved != "" {
		return resolved
	}

	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if scenarioDir, resolveErr := repocontract.ResolveScenarioPath(root, scenarioRoot); resolveErr == nil {
			return filepath.Join(scenarioDir, "data", "recordings")
		} else if log != nil {
			log.WithError(resolveErr).Warn("Failed to resolve recordings root from repo contract; falling back to cwd-derived path")
		}
	} else if log != nil {
		log.WithError(err).Warn("Failed to resolve repo root for recordings; falling back to cwd-derived path")
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for recordings root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "recordings")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				recordings := filepath.Join(dir, "data", "recordings")
				if abs, err := filepath.Abs(recordings); err == nil {
					return abs
				}
				return recordings
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "recordings")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

// ResolveSessionProfilesRoot returns an absolute path for storing persisted browser session profiles.
func ResolveSessionProfilesRoot(log *logrus.Logger) string {
	if value := strings.TrimSpace(os.Getenv("BAS_SESSION_PROFILES_ROOT")); value != "" {
		if abs, err := filepath.Abs(value); err == nil {
			return abs
		}
		if log != nil {
			log.WithField("path", value).Warn("Using BAS_SESSION_PROFILES_ROOT without normalization")
		}
		return value
	}

	if resolved := resolveScenarioStoragePath("session-profiles"); resolved != "" {
		return resolved
	}

	if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
		if scenarioDir, resolveErr := repocontract.ResolveScenarioPath(root, scenarioRoot); resolveErr == nil {
			return filepath.Join(scenarioDir, "data", "session-profiles")
		} else if log != nil {
			log.WithError(resolveErr).Warn("Failed to resolve session profiles root from repo contract; falling back to cwd-derived path")
		}
	} else if log != nil {
		log.WithError(err).Warn("Failed to resolve repo root for session profiles; falling back to cwd-derived path")
	}

	cwd, err := os.Getwd()
	if err != nil {
		if log != nil {
			log.WithError(err).Warn("Failed to resolve working directory for session profiles root; using relative default")
		}
		return filepath.Join("scenarios", scenarioRoot, "data", "session-profiles")
	}

	absCwd, err := filepath.Abs(cwd)
	if err == nil {
		for dir := absCwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
			parent := filepath.Dir(dir)
			if filepath.Base(dir) == scenarioRoot && filepath.Base(parent) == "scenarios" {
				root := filepath.Join(dir, "data", "session-profiles")
				if abs, err := filepath.Abs(root); err == nil {
					return abs
				}
				return root
			}
		}
	}

	root := filepath.Join(absCwd, "scenarios", scenarioRoot, "data", "session-profiles")
	if abs, err := filepath.Abs(root); err == nil {
		return abs
	}
	return root
}

func resolveScenarioStoragePath(rel string) string {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return ""
	}
	path, err := resolver.Path(storage.Options{ScenarioID: scenarioRoot}, storage.ClassData, rel)
	if err != nil {
		return ""
	}
	return path
}
