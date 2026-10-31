package closure

import (
	"context"
	"strings"

	"scenario-to-cloud/domain"
)

// Request is the transport-neutral closure request shared by the API, CLI
// and UI. Every surface builds one of these and receives the same closure.
type Request struct {
	ScenarioID           string          `json:"scenario"`
	Environment          string          `json:"environment,omitempty"`
	OS                   string          `json:"os"`
	Arch                 string          `json:"arch"`
	Scope                string          `json:"scope,omitempty"`
	Overrides            RequestOverride `json:"overrides,omitempty"`
	TargetCapacity       *TargetCapacity `json:"target_capacity,omitempty"`
	ReleaseArtifactBytes []uint64        `json:"release_artifact_bytes,omitempty"`
}

// RequestOverride is the JSON form of Overrides.
type RequestOverride struct {
	SelectOptional    []string          `json:"select_optional,omitempty"`
	DeselectOptional  []string          `json:"deselect_optional,omitempty"`
	AutoRestart       map[string]bool   `json:"auto_restart,omitempty"`
	SupervisionMember map[string]bool   `json:"supervision_member,omitempty"`
	OperatingMode     map[string]string `json:"operating_mode,omitempty"`
}

func (o RequestOverride) overrides() Overrides {
	return Overrides{
		SelectOptional:    o.SelectOptional,
		DeselectOptional:  o.DeselectOptional,
		AutoRestart:       o.AutoRestart,
		SupervisionMember: o.SupervisionMember,
		OperatingMode:     o.OperatingMode,
	}
}

// Service binds the catalogs a deployment needs and resolves closures.
type Service struct {
	RepoRoot         string
	Catalog          Catalog
	Analyzer         AnalyzerClient
	HostRequirements HostRequirementsResolver
	// DefaultPlatform is used by callers that have not detected the target
	// platform yet (manifest refresh before the VPS is probed).
	DefaultPlatform Platform
}

// Derive resolves a closure for a Request.
func (s *Service) Derive(ctx context.Context, request Request) (domain.Closure, error) {
	return Resolve(ctx, Inputs{
		ScenarioID:           strings.TrimSpace(request.ScenarioID),
		Environment:          strings.TrimSpace(request.Environment),
		Platform:             Platform{OS: strings.TrimSpace(request.OS), Arch: strings.TrimSpace(request.Arch)},
		Scope:                Scope(strings.TrimSpace(request.Scope)),
		RepoRoot:             s.RepoRoot,
		Catalog:              s.Catalog,
		Analyzer:             s.Analyzer,
		HostRequirements:     s.HostRequirements,
		Overrides:            request.Overrides.overrides(),
		TargetCapacity:       request.TargetCapacity,
		ReleaseArtifactBytes: request.ReleaseArtifactBytes,
	})
}

// Resolve derives the bundle-scope closure for a scenario on the service's
// default platform. It satisfies the manifest refresher's ClosureSource.
func (s *Service) Resolve(ctx context.Context, scenarioID, environment string) (domain.Closure, error) {
	return s.Derive(ctx, Request{ScenarioID: scenarioID, Environment: environment, OS: s.DefaultPlatform.OS, Arch: s.DefaultPlatform.Arch, Scope: string(ScopeBundle)})
}
