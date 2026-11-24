package versions

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
	"react-component-library/internal/components"
)

type materializeComponents struct {
	components.Service
	assets     []components.Component
	versions   map[string][]components.ComponentVersion
	byID       map[string]components.Component
	getVersion map[string]components.ComponentVersion
	errAt      string
}

func (f *materializeComponents) List(_ context.Context, q components.SearchQuery) ([]components.Component, error) {
	if f.errAt == "list" {
		return nil, errors.New("catalog unavailable")
	}
	if q.Limit != 2000 {
		return nil, errors.New("unexpected catalog limit")
	}
	return f.assets, nil
}

func (f *materializeComponents) ListVersions(_ context.Context, id string, limit int) ([]components.ComponentVersion, error) {
	if f.errAt == "list-versions" {
		return nil, errors.New("versions unavailable")
	}
	if limit <= 0 {
		return nil, errors.New("unexpected version limit")
	}
	return f.versions[id], nil
}

func (f *materializeComponents) Get(_ context.Context, id string) (components.Component, error) {
	if f.errAt == "get" {
		return components.Component{}, errors.New("component unavailable")
	}
	asset, ok := f.byID[id]
	if !ok {
		return components.Component{}, errors.New("missing component")
	}
	return asset, nil
}

func (f *materializeComponents) GetVersion(_ context.Context, id, version string) (components.ComponentVersion, error) {
	if f.errAt == "get-version" {
		return components.ComponentVersion{}, errors.New("version unavailable")
	}
	row, ok := f.getVersion[id+"@"+version]
	if !ok {
		return components.ComponentVersion{}, errors.New("missing version")
	}
	return row, nil
}

type materializeRecorder struct {
	calls  []string
	result components.MaterializeResult
	err    error
}

func (f *materializeRecorder) EnsureMaterialized(_ context.Context, id, version, into string) (components.MaterializeResult, error) {
	f.calls = append(f.calls, id+"@"+version+"->"+into)
	return f.result, f.err
}

func TestMaterializeVersionAllSkipsColdDraftsAndMapsRestoredRows(t *testing.T) {
	ctx := context.Background()
	asset := components.Component{ID: "cmp-1", LibraryID: "lib-1"}
	service := &materializeComponents{
		assets: []components.Component{asset},
		versions: map[string][]components.ComponentVersion{"cmp-1": {
			{Version: "0.1.0", Status: components.VersionStatusDraft},
			{Version: "1.0.0", Status: components.VersionStatusReleased, Presence: "evicted"},
			{Version: "1.1.0", Status: components.VersionStatusDeprecated},
			{Version: "1.2.0", Status: components.VersionStatusArchived},
			{Version: "2.0.0", Status: components.ComponentVersionStatus("retired")},
		}},
		getVersion: map[string]components.ComponentVersion{
			"cmp-1@1.0.0": {Presence: "materialized"},
			"cmp-1@1.1.0": {}, "cmp-1@1.2.0": {},
		},
	}
	materializer := &materializeRecorder{result: components.MaterializeResult{Directory: "versions/1.0.0", FilesWritten: 3}}
	h := NewConnectHandler(Deps{Components: service, Materializer: materializer})
	resp, err := h.MaterializeVersion(ctx, connect.NewRequest(&versionsv1.MaterializeVersionRequest{All: true}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Versions) != 3 {
		t.Fatalf("materialized rows=%d, want released, deprecated, and archived only", len(resp.Msg.Versions))
	}
	wantCalls := []string{"cmp-1@1.0.0->", "cmp-1@1.1.0->", "cmp-1@1.2.0->"}
	if len(materializer.calls) != len(wantCalls) {
		t.Fatalf("materializer calls=%v, want %v", materializer.calls, wantCalls)
	}
	for i, want := range wantCalls {
		if materializer.calls[i] != want {
			t.Fatalf("materializer calls=%v, want %v", materializer.calls, wantCalls)
		}
	}
	if got := resp.Msg.Versions[0]; got.ComponentId != "cmp-1" || got.LibraryId != "lib-1" || got.Version != "1.0.0" || got.Directory != "versions/1.0.0" || got.FilesWritten != 3 || !got.AlreadyPresent {
		t.Fatalf("restored version response lost catalog or materialization evidence: %#v", got)
	}
}

func TestMaterializeVersionValidationAndDependencyErrorsHaveUsefulCodes(t *testing.T) {
	ctx := context.Background()
	h := NewConnectHandler(Deps{})
	_, err := h.MaterializeVersion(ctx, connect.NewRequest(&versionsv1.MaterializeVersionRequest{ComponentId: "cmp-1", Version: "1.0.0"}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("missing catalog code=%v, want failed precondition", got)
	}

	h = NewConnectHandler(Deps{Components: &materializeComponents{}})
	_, err = h.MaterializeVersion(ctx, connect.NewRequest(&versionsv1.MaterializeVersionRequest{ComponentId: "cmp-1"}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("incomplete selection code=%v, want invalid argument", got)
	}

	service := &materializeComponents{byID: map[string]components.Component{"cmp-1": {ID: "cmp-1", LibraryID: "lib-1"}}}
	materializer := &materializeRecorder{err: errors.New("mirror unavailable")}
	h = NewConnectHandler(Deps{Components: service, Materializer: materializer})
	_, err = h.MaterializeVersion(ctx, connect.NewRequest(&versionsv1.MaterializeVersionRequest{ComponentId: "cmp-1", Version: "1.0.0"}))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("materialization failure code=%v, want failed precondition", got)
	}
	if len(materializer.calls) != 1 || materializer.calls[0] != "cmp-1@1.0.0->" {
		t.Fatalf("specific version selection was not materialized: %v", materializer.calls)
	}
}

func TestListVersionsAllMapsRowsAndStopsAtRequestedLimit(t *testing.T) {
	ctx := context.Background()
	first := components.Component{ID: "cmp-1", LibraryID: "lib-1"}
	second := components.Component{ID: "cmp-2", LibraryID: "lib-2"}
	service := &materializeComponents{
		assets: []components.Component{first, second},
		versions: map[string][]components.ComponentVersion{
			"cmp-1": {
				{ID: "v1", ComponentID: "cmp-1", LibraryID: "lib-1", Version: "1.0.0", Status: components.VersionStatusReleased, Presence: "materialized"},
				{ID: "v2", ComponentID: "cmp-1", LibraryID: "lib-1", Version: "1.1.0", Status: components.VersionStatusArchived, Presence: "evicted"},
			},
			"cmp-2": {{ID: "v3", ComponentID: "cmp-2", LibraryID: "lib-2", Version: "2.0.0", Status: components.VersionStatusReleased}},
		},
	}
	h := NewConnectHandler(Deps{Components: service})
	resp, err := h.ListVersions(ctx, connect.NewRequest(&versionsv1.ListVersionsRequest{All: true, Limit: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Versions) != 2 {
		t.Fatalf("listed %d versions, want requested limit of two", len(resp.Msg.Versions))
	}
	if got := resp.Msg.Versions[0]; got.Id != "v1" || got.ComponentId != "cmp-1" || got.LibraryId != "lib-1" || got.Presence != "materialized" {
		t.Fatalf("first version mapping lost catalog fields: %#v", got)
	}
	if got := resp.Msg.Versions[1]; got.Id != "v2" || got.Status != "archived" || got.Presence != "evicted" {
		t.Fatalf("second version mapping lost lifecycle or presence: %#v", got)
	}

	service.errAt = "list-versions"
	_, err = h.ListVersions(ctx, connect.NewRequest(&versionsv1.ListVersionsRequest{All: true, Limit: 10}))
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("catalog version query code=%v, want internal", got)
	}
}
