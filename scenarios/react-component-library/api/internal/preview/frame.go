package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
)

type frameCatalogDocument struct {
	Kind  string `json:"kind"`
	Asset struct {
		ID      string   `json:"id"`
		Kind    string   `json:"kind"`
		Targets []string `json:"targets"`
	} `json:"asset"`
	Regions []struct {
		ID      string `json:"id"`
		Accepts string `json:"accepts"`
	} `json:"regions"`
	Expects []struct {
		Capability    string   `json:"capability"`
		TypeArguments []string `json:"typeArguments"`
	} `json:"expects"`
	Fixture struct {
		DataShapes  []string       `json:"dataShapes"`
		RecordCount map[string]int `json:"recordCount"`
		Satisfies   *struct {
			Capability    string   `json:"capability"`
			TypeArguments []string `json:"typeArguments"`
		} `json:"satisfies"`
	} `json:"fixture"`
}

// DeterministicFixturePayload is the small, serializable data contract passed
// to a Preview frame or harness. It is deliberately not a live provider: the
// clock, seed, ordering, and records are fixed so two captures of the same
// reference are comparable.
type DeterministicFixturePayload struct {
	Asset      string           `json:"asset"`
	Version    string           `json:"version"`
	State      string           `json:"state"`
	Seed       string           `json:"seed"`
	Clock      string           `json:"clock"`
	DataShapes []string         `json:"dataShapes,omitempty"`
	Records    []map[string]any `json:"records,omitempty"`
	Series     []map[string]any `json:"series,omitempty"`
	Nodes      []map[string]any `json:"nodes,omitempty"`
	Error      string           `json:"error,omitempty"`
}

// ResolveDeterministicFixture returns typed-enough representative records for
// the catalog fixture families used by Preview. It is intentionally bounded;
// a story that needs a new domain must add a versioned fixture family instead
// of reaching into a production service.
func ResolveDeterministicFixture(asset, version, state string) (DeterministicFixturePayload, error) {
	asset = strings.TrimSpace(asset)
	version = strings.TrimSpace(version)
	state = strings.TrimSpace(state)
	if state == "" {
		state = "typical"
	}
	if state == "ready" {
		state = "typical"
	}
	if version == "" {
		return DeterministicFixturePayload{}, frameBundleError(asset, "fixture version is required")
	}
	fixture := DeterministicFixturePayload{Asset: asset, Version: version, State: state, Seed: "rcl-fixture-v1", Clock: "2026-01-15T12:00:00Z"}
	switch asset {
	case "fixtures.resource-collection":
		fixture.DataShapes = []string{"empty", "typical", "overflow", "volume", "failure"}
		if state == "empty" {
			return fixture, nil
		}
		fixture.Records = []map[string]any{
			{"id": "resource-001", "name": "api-gateway", "owner": "Platform", "status": "Ready", "updated": "2026-01-15T11:42:00Z"},
			{"id": "resource-002", "name": "event-stream", "owner": "Data", "status": "Needs attention", "updated": "2026-01-15T10:18:00Z"},
			{"id": "resource-003", "name": "worker-cluster", "owner": "Operations", "status": "Paused", "updated": "2026-01-14T22:05:00Z"},
		}
		if state == "overflow" {
			fixture.Records = append(fixture.Records, map[string]any{"id": "resource-long", "name": "resource-with-a-deliberately-long-name-that-must-wrap-without-breaking-the-row", "owner": "Platform", "status": "Ready", "updated": fixture.Clock})
		}
		if state == "failure" || state == "permission" {
			fixture.Error = "Resource data is unavailable. Retry to restore the collection."
		}
	case "fixtures.user-directory":
		fixture.DataShapes = []string{"empty", "typical", "overflow", "failure"}
		fixture.Records = []map[string]any{
			{"id": "user-001", "name": "Avery Chen", "role": "Administrator", "status": "Active"},
			{"id": "user-002", "name": "Mina Patel", "role": "Reviewer", "status": "Active"},
			{"id": "user-003", "name": "Noah Williams", "role": "Operator", "status": "Invited"},
		}
	case "fixtures.time-series":
		fixture.DataShapes = []string{"empty", "typical", "overflow", "volume", "failure"}
		if state != "empty" {
			fixture.Series = []map[string]any{
				{"at": "2026-01-15T11:00:00Z", "value": 42.0},
				{"at": "2026-01-15T11:15:00Z", "value": 47.0},
				{"at": "2026-01-15T11:30:00Z", "value": 39.0},
				{"at": "2026-01-15T11:45:00Z", "value": 81.0},
			}
		}
		if state == "failure" {
			fixture.Error = "Metric history is unavailable. Retry to restore the series."
		}
	case "fixtures.navigation-tree":
		fixture.DataShapes = []string{"empty", "typical", "overflow", "failure"}
		fixture.Nodes = []map[string]any{
			{"id": "overview", "label": "Overview", "current": true, "disabled": false, "badge": ""},
			{"id": "resources", "label": "Resources", "current": false, "disabled": false, "badge": "2"},
			{"id": "activity", "label": "Activity", "current": false, "disabled": false, "badge": ""},
			{"id": "settings", "label": "Settings", "current": false, "disabled": false, "badge": ""},
		}
		if state == "failure" {
			fixture.Error = "Navigation data is unavailable. Retry to restore workspace sections."
		}
	case "fixtures.status-health":
		fixture.DataShapes = []string{"typical", "partial", "failure", "recovery"}
		fixture.Records = []map[string]any{
			{"id": "api", "label": "API gateway", "status": "healthy", "detail": "99.98% available"},
			{"id": "events", "label": "Event stream", "status": "degraded", "detail": "Backlog above target"},
			{"id": "workers", "label": "Worker pool", "status": "recovering", "detail": "3 of 4 nodes ready"},
		}
		if state == "failure" {
			fixture.Error = "Health signals are unavailable. Retry to restore status."
		}
	case "fixtures.error-recovery":
		fixture.DataShapes = []string{"failure", "permission", "timeout", "recovery"}
		fixture.Records = []map[string]any{{"id": "request-001", "reason": "timeout", "message": "The operation took longer than expected.", "action": "Retry"}}
		fixture.Error = "The operation took longer than expected."
	default:
		return DeterministicFixturePayload{}, frameBundleError(asset, "fixture family has no deterministic Preview implementation")
	}
	if !contains([]string{"empty", "typical", "overflow", "volume", "failure", "partial", "recovery", "permission", "timeout"}, state) {
		return DeterministicFixturePayload{}, frameBundleError(asset, fmt.Sprintf("fixture state %q is not supported", state))
	}
	return fixture, nil
}

func (s *service) bundleFrame(ctx context.Context, frame *components.StoryFrame) (string, string, string, []deps.Declaration, error) {
	if frame == nil {
		return "", "", "", nil, nil
	}
	frameDoc, err := s.readCatalogFrame(frame.Asset)
	if err != nil {
		return "", "", "", nil, err
	}
	if frameDoc.Kind != "catalog-asset" || frameDoc.Asset.ID != frame.Asset {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset is not a catalog asset")
	}
	if !contains(frameDoc.Asset.Targets, "react-vite") {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset does not target react-vite")
	}
	regionDeclared := false
	for _, region := range frameDoc.Regions {
		if region.ID == frame.Region {
			regionDeclared = true
			break
		}
	}
	if !regionDeclared {
		return "", "", "", nil, frameBundleError(frame.Asset, fmt.Sprintf("frame region %q is not declared", frame.Region))
	}
	for _, region := range frameDoc.Regions {
		if region.ID == frame.Region && region.Accepts != "" && region.Accepts != frame.Capability {
			return "", "", "", nil, frameBundleError(frame.Asset, fmt.Sprintf("frame region %q requires subject capability %q", frame.Region, region.Accepts))
		}
	}
	var fixtureDoc frameCatalogDocument
	if strings.TrimSpace(frame.Fixture) != "" {
		fixtureDoc, err = s.readCatalogFrame(frame.Fixture)
		if err != nil {
			return "", "", "", nil, err
		}
		if fixtureDoc.Asset.Kind != "fixture" {
			return "", "", "", nil, frameBundleError(frame.Fixture, "frame fixture must be a fixture asset")
		}
	} else if frameRequiresFixture(frameDoc.Expects) {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame requires a fixture")
	}
	for _, expect := range frameDoc.Expects {
		if expect.Capability != "data-source" {
			continue
		}
		if fixtureDoc.Fixture.Satisfies == nil || fixtureDoc.Fixture.Satisfies.Capability != "data-source" || !compatibleFrameTypes(expect.TypeArguments, fixtureDoc.Fixture.Satisfies.TypeArguments) {
			return "", "", "", nil, frameBundleError(frame.Fixture, "frame fixture does not satisfy the frame asset's data-source port")
		}
	}
	componentsList, err := s.components.List(ctx, components.SearchQuery{Limit: 10000})
	if err != nil {
		return "", "", "", nil, err
	}
	var implementations []components.Component
	for index := range componentsList {
		candidate, getErr := s.components.Get(ctx, componentsList[index].ID)
		if getErr == nil && candidate.CatalogID == frame.Asset {
			implementations = append(implementations, candidate)
		}
	}
	if len(implementations) == 0 {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset has no react-vite implementation")
	}
	if len(implementations) > 1 {
		return "", "", "", nil, frameBundleError(frame.Asset, "frame asset has ambiguous react-vite implementations")
	}
	implementation := &implementations[0]
	version := strings.TrimSpace(frame.Version)
	if version == "" {
		// Schema-v3 stories predate immutable frame references. Keep them
		// renderable during migration, but make the fallback explicit and
		// deterministic for the current catalog projection.
		version = implementation.LatestVersion
	}
	if len(implementation.Dependencies) > 0 {
		if _, err := components.ResolveDependencyClosure(ctx, s.components, implementation.ID, version); err != nil {
			return "", "", "", nil, err
		}
	}
	content, err := s.components.GetVersionContent(ctx, implementation.ID, version)
	if err != nil {
		return "", "", "", nil, err
	}
	js, _, err := s.bundler.BuildBundle(ctx, stampPreviewSource(content.Body, content.SourcePath, frame.Asset, version), content.SourcePath)
	if err != nil {
		return "", "", "", nil, err
	}
	var declarations []deps.Declaration
	if s.deps != nil {
		var depErr error
		declarations, depErr = s.deps.ListForComponentVersion(ctx, implementation.ID, version)
		if depErr != nil {
			return "", "", "", nil, depErr
		}
	}
	fixtureJSON := []byte(nil)
	if strings.TrimSpace(frame.Fixture) != "" {
		fixtureJSON, err = json.Marshal(struct {
			Asset       string   `json:"asset"`
			DataShapes  []string `json:"dataShapes,omitempty"`
			RecordCount any      `json:"recordCount,omitempty"`
		}{Asset: fixtureDoc.Asset.ID, DataShapes: fixtureDoc.Fixture.DataShapes, RecordCount: fixtureDoc.Fixture.RecordCount})
		if err != nil {
			return "", "", "", nil, err
		}
	}
	return js, string(fixtureJSON), content.SourcePath, declarations, nil
}

func frameRequiresFixture(expects []struct {
	Capability    string   `json:"capability"`
	TypeArguments []string `json:"typeArguments"`
}) bool {
	for _, expect := range expects {
		if expect.Capability == "data-source" {
			return true
		}
	}
	return false
}

func (s *service) readCatalogFrame(id string) (frameCatalogDocument, error) {
	if strings.TrimSpace(s.repoRoot) == "" {
		return frameCatalogDocument{}, frameBundleError(id, "preview repository root is not configured")
	}
	parts := strings.Split(strings.TrimSpace(id), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(id, `/\\`) {
		return frameCatalogDocument{}, frameBundleError(id, "invalid catalog asset id")
	}
	path := filepath.Join(s.repoRoot, "scenarios", "react-component-library", "catalog", "assets", parts[0], parts[1]+".json")
	if _, err := os.Stat(path); err != nil {
		return frameCatalogDocument{}, frameBundleError(id, "catalog asset was not found")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return frameCatalogDocument{}, fmt.Errorf("read catalog asset %q: %w", id, err)
	}
	var doc frameCatalogDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return frameCatalogDocument{}, fmt.Errorf("parse catalog asset %q: %w", id, err)
	}
	return doc, nil
}

func frameBundleError(sourcePath, detail string) error {
	return ErrBundle{SourcePath: sourcePath, Messages: []string{detail}}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func compatibleFrameTypes(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	for index := range expected {
		if strings.HasPrefix(expected[index], "T") {
			continue
		}
		if expected[index] != actual[index] {
			return false
		}
	}
	return true
}
