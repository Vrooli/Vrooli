package paths

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
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
	return resolveScenarioDataRoot(log, "BAS_RECORDINGS_ROOT", "recordings", "recordings")
}

// ResolveSessionProfilesRoot returns an absolute path for storing persisted browser session profiles.
func ResolveSessionProfilesRoot(log *logrus.Logger) string {
	return resolveScenarioDataRoot(log, "BAS_SESSION_PROFILES_ROOT", "session profiles", "session-profiles")
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
