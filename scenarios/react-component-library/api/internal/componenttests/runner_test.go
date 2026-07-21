package componenttests

import (
	"context"
	"encoding/json"
	"errors"
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

type stories map[string]components.StoryContract

func (s stories) ListStories(_ context.Context, query components.StoryQuery) ([]components.ComponentStory, error) {
	story, ok := s[query.ComponentID+"@"+query.Version]
	if !ok {
		return nil, nil
	}
	raw, err := json.Marshal(story)
	if err != nil {
		return nil, err
	}
	return []components.ComponentStory{{ComponentID: query.ComponentID, Version: query.Version, Kind: story.Kind, ContractJSON: string(raw)}}, nil
}

type executor struct {
	results map[string]StoryExecution
	err     error
}

func (e executor) ExecuteStory(_ context.Context, libraryID, version, storyID string) (StoryExecution, error) {
	if e.err != nil {
		return StoryExecution{}, e.err
	}
	if result, ok := e.results[libraryID+"@"+version+":"+storyID]; ok {
		return result, nil
	}
	return StoryExecution{Passed: true, Duration: time.Millisecond}, nil
}

func componentStory(id string) components.StoryContract {
	return components.StoryContract{SchemaVersion: 1, Kind: components.StoryKindComponent, Args: components.StoryArgsSchema{Fields: []components.StoryField{}}, Environment: components.StoryEnvironment{Fixtures: []components.StoryFixture{}}, Stories: []components.StoryDefinition{{ID: id, Name: id}}}
}

func TestRunnerRequiresPinAndReportsEachClosureAsset(t *testing.T) {
	hook := components.Component{ID: "hook", LibraryID: "rcl:hook", Slug: "hook", AssetKind: components.AssetKindHook}
	root := components.Component{ID: "root", LibraryID: "rcl:button", Slug: "button", AssetKind: components.AssetKindComponent, Dependencies: []components.AssetDependency{{LibraryID: hook.LibraryID, Version: "1.0.0"}}}
	reader := assets{root: root, dependency: hook, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Button = () => null", ContentSHA256: "root"}, "hook@1.0.0": {ComponentID: "hook", Version: "1.0.0", Content: "export const useHook = () => null", ContentSHA256: "hook"}}}
	runner := Runner{Assets: reader, Stories: stories{"root@1.0.0": componentStory("idle"), "hook@1.0.0": {SchemaVersion: 1, Kind: components.StoryKindHook, Args: components.StoryArgsSchema{Fields: []components.StoryField{}}, Environment: components.StoryEnvironment{Fixtures: []components.StoryFixture{}}, Stories: []components.StoryDefinition{{ID: "start", Name: "start"}}}}, Executor: executor{}, Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
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

func TestRunnerFailsWhenStoryIsMissing(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}, Stories: stories{}, Executor: executor{}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Equal(t, StageContract, report.Results[2].Stage)
	require.Contains(t, report.Results[2].Message, "exactly one indexed story")
}

func TestRunnerRejectsAnEmptyVersionedSourceBeforeBehaviorEvaluation(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0"}}}, Stories: stories{"root@1.0.0": componentStory("idle")}, Executor: executor{}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Equal(t, StageStatic, report.Results[1].Stage)
	require.Equal(t, VerdictFailed, report.Results[1].Verdict)
}

func TestRunnerRecordsRenderedStoryFailure(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}, Stories: stories{"root@1.0.0": componentStory("broken")}, Executor: executor{results: map[string]StoryExecution{"rcl:root@1.0.0:broken": {Passed: false, Failures: []string{"expect[0] text Missing not found"}, Duration: time.Millisecond}}}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	result := report.Results[len(report.Results)-1]
	require.Equal(t, VerdictFailed, result.Verdict)
	require.Equal(t, "broken", result.Subject)
	require.Contains(t, result.Message, "Missing")
}

func TestRunnerFailsWhenAStoryDoesNotMount(t *testing.T) {
	root := components.Component{ID: "root", LibraryID: "rcl:root", Slug: "root", AssetKind: components.AssetKindComponent}
	runner := Runner{Assets: assets{root: root, versions: map[string]components.ComponentVersion{"root@1.0.0": {ComponentID: "root", Version: "1.0.0", Content: "export const Root = () => null", ContentSHA256: "root"}}}, Stories: stories{"root@1.0.0": componentStory("missing")}, Executor: executor{err: errors.New("harness completed without a story result")}}
	report, err := runner.Run(context.Background(), Request{ComponentID: "root", Version: "1.0.0"})
	require.NoError(t, err)
	require.Equal(t, VerdictFailed, report.Verdict)
	require.Contains(t, report.Results[len(report.Results)-1].Message, "did not render")
}
