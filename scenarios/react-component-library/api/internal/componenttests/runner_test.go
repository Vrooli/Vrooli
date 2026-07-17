package componenttests

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
)

type assets struct {
	root, dependency components.Component
	versions         map[string]components.ComponentVersion
}

func (a assets) Get(_ context.Context, id string) (components.Component, error) {
	if id == a.root.ID {
		return a.root, nil
	}
	return components.Component{}, errors.New("not found")
}
func (a assets) GetByLibraryID(_ context.Context, id string) (components.Component, error) {
	if id == a.dependency.LibraryID {
		return a.dependency, nil
	}
	if id == a.root.LibraryID {
		return a.root, nil
	}
	return components.Component{}, errors.New("not found")
}
func (a assets) GetVersion(_ context.Context, id, version string) (components.ComponentVersion, error) {
	v, ok := a.versions[id+"@"+version]
	if !ok {
		return components.ComponentVersion{}, errors.New("not found")
	}
	return v, nil
}

type contracts map[string]Contract

func (c contracts) Load(a components.Component, v components.ComponentVersion) (Contract, error) {
	contract, ok := c[a.ID+"@"+v.Version]
	if !ok {
		return Contract{}, errors.New("missing test contract")
	}
	return contract, nil
}

type examples map[string][]components.ComponentExample

func (e examples) ListExamples(_ context.Context, query components.ExampleQuery) ([]components.ComponentExample, error) {
	return e[query.ComponentID+"@"+query.Version], nil
}

func TestRunnerRequiresPinAndReportsEachClosureAsset(t *testing.T) {
	hook := components.Component{ID: "hook", LibraryID: "rcl:hook", Slug: "hook", AssetKind: components.AssetKindHook}
	root := components.Component{ID: "root", LibraryID: "rcl:button", Slug: "button", AssetKind: components.AssetKindComponent, Dependencies: []components.AssetDependency{{LibraryID: hook.LibraryID, Version: "1.0.0"}}}
	reader := assets{root: root, dependency: hook, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Button = () => null", ContentSHA256: "root"}, "hook@1.0.0": {ComponentID: "hook", Version: "1.0.0", Content: "export const useHook = () => null", ContentSHA256: "hook"}}}
	runner := Runner{Assets: reader, Examples: examples{"root@1.0.0": {{Name: "idle", ExpectJSON: `[{"kind":"text","value":"Ready"}]`}}}, Contracts: contracts{"root@1.0.0": {SchemaVersion: "1", Examples: []ExampleTrace{{Example: "idle", Assertions: []Assertion{{Kind: "text", Value: "Ready"}}}}}, "hook@1.0.0": {SchemaVersion: "1", Fixtures: []HookFixture{{Name: "start", Assertions: []Assertion{{Kind: "state", Target: "status", Value: "recording"}}}}}}, Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
	_, err := runner.Run(context.Background(), Request{ComponentID: "root"})
	var validation ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "version_required", validation.Code)
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0", IncludeClosure: true})
	require.NoError(t, err)
	require.Equal(t, VerdictPassed, report.Verdict)
	require.Len(t, report.Results, 8)
	require.Equal(t, "rcl:hook", report.Results[0].AssetLibraryID)
	require.Equal(t, StageStatic, report.Results[1].Stage)
}

func TestRunnerDoesNotSuppressSiblingContractFailure(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}}}, Examples: examples{}, Contracts: contracts{}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Len(t, report.Results, 3)
	require.Equal(t, StageContract, report.Results[2].Stage)
}

func TestRunnerReportsAbsentContractAsUncoveredWithoutClaimingFailure(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}, Examples: examples{}, Contracts: FSContractReader{Root: t.TempDir()}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictBlocked, report.Verdict)
	require.Equal(t, VerdictBlocked, report.Results[2].Verdict)
	require.Contains(t, report.Results[2].Message, "no versioned test contract")
}

func TestRunnerRejectsAnEmptyVersionedSourceBeforeBehaviorEvaluation(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}}}, Examples: examples{"root@1.0.0": {{Name: "idle", ExpectJSON: `[{"kind":"text","value":"Ready"}]`}}}, Contracts: contracts{"root@1.0.0": {SchemaVersion: "1", Examples: []ExampleTrace{{Example: "idle", Assertions: []Assertion{{Kind: "text", Value: "Ready"}}}}}}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Equal(t, StageStatic, report.Results[1].Stage)
	require.Equal(t, VerdictFailed, report.Results[1].Verdict)
}

func TestRunnerReportsMissingDeclaredExampleWithoutHidingOtherTraces(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}}}, Examples: examples{"root@1.0.0": {{Name: "present", ExpectJSON: `[{"kind":"text","value":"Ready"}]`}}}, Contracts: contracts{"root@1.0.0": {SchemaVersion: "1", Examples: []ExampleTrace{{Example: "missing", Assertions: []Assertion{{Kind: "text", Value: "Ready"}}}, {Example: "present", Assertions: []Assertion{{Kind: "text", Value: "Ready"}}}}}}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Len(t, report.Results, 5)
	require.Equal(t, VerdictFailed, report.Results[3].Verdict)
	require.Equal(t, VerdictPassed, report.Results[4].Verdict)
}

func TestRunnerLinksClaimOnlyWhenThePublishedExperienceContractDeclaresIt(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "VoiceInputButton", AssetKind: components.AssetKindComponent}
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "experience", "components"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "experience", "components", "voice-input-button.json"), []byte(`{"claims":[{"id":"voice-input-action"}]}`), 0o644))
	runner := Runner{
		Assets:   assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const VoiceInputButton = () => null", ContentSHA256: "voice"}}},
		Examples: examples{"root@1.0.0": {{Name: "idle", ExpectJSON: `[{"kind":"text","value":"Ready"}]`}}},
		Contracts: contracts{"root@1.0.0": {
			SchemaVersion: "1",
			Examples:      []ExampleTrace{{Example: "idle", Assertions: []Assertion{{Kind: "text", Value: "Ready"}}, Claims: []string{"voice-input-action"}}},
		}},
		Claims: FSClaimReader{Root: dir},
	}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictPassed, report.Verdict)
	require.Equal(t, StageEvidence, report.Results[4].Stage)
	require.Equal(t, VerdictPassed, report.Results[4].Verdict)
}
