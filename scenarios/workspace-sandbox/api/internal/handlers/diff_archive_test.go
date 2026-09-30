package handlers_test

// Transport tests retain the service's evidence source and errors. The service
// tests own live-versus-archive selection; both REST and Connect use GetDiff.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"

	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

func TestDiffTransportsRetainEvidenceAndRefuseUnavailableArchive(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "retained", true: "unavailable"}[unavailable], func(t *testing.T) {
			id := uuid.New()
			svc := &sandboxiface.FakeService{GetDiffFn: func(ctx context.Context, gotID uuid.UUID) (*types.DiffResult, error) {
				if gotID != id {
					t.Fatal("diff request changed sandbox identity")
				}
				if unavailable {
					return nil, errors.New("retained diff blob is unavailable")
				}
				return &types.DiffResult{SandboxID: id, Files: []*types.FileChange{{FilePath: "review.txt"}}, UnifiedDiff: "retained diff", PatchSHA256: strings.Repeat("a", 64), ArchiveState: types.ArchiveStateComplete}, nil
			}}
			live := newLive(t, svc)
			resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff"), "")
			wire, err := handlers.NewConnectHandler(svc).GetSandboxDiff(t.Context(), connect.NewRequest(&workspacev1.GetSandboxDiffRequest{SandboxId: id.String()}))
			if unavailable {
				if resp.StatusCode != http.StatusInternalServerError || err == nil || wire != nil || connect.CodeOf(err) != connect.CodeInternal {
					t.Fatalf("transport concealed lost evidence: HTTP %d %s; Connect %+v, %v", resp.StatusCode, body, wire, err)
				}
				return
			}
			var rest types.DiffResult
			if err := json.Unmarshal(body, &rest); err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusOK || err != nil || wire == nil || wire.Msg.UnifiedDiff != rest.UnifiedDiff || wire.Msg.ArchiveState != string(rest.ArchiveState) || wire.Msg.ArchiveState != "complete" || wire.Msg.PatchSha256 != rest.PatchSHA256 || rest.PatchSHA256 != strings.Repeat("a", 64) {
				t.Fatalf("transports disagree on retained evidence: %+v; %+v, %v", rest, wire, err)
			}
		})
	}
}

func TestApprovalTransportsPreserveReviewedPatchIdentity(t *testing.T) {
	id := uuid.New()
	digest := strings.Repeat("a", 64)
	calls := 0
	svc := &sandboxiface.FakeService{ApproveFn: func(ctx context.Context, req *types.ApprovalRequest) (*types.ApprovalResult, error) {
		calls++
		if req.SandboxID != id || req.ExpectedPatchSHA256 != digest || req.Mode != "all" || !req.Force {
			t.Fatalf("transport dropped or changed approval preconditions: %+v", req)
		}
		return &types.ApprovalResult{Success: true, AppliedPatchSHA256: digest}, nil
	}}
	live := newLive(t, svc)
	response, body := live.DoJSON(t, "POST", sandboxesPath(id, "/approve"), `{"force":true,"expectedPatchSha256":"`+digest+`"}`)
	var rest types.ApprovalResult
	if err := json.Unmarshal(body, &rest); err != nil {
		t.Fatal(err)
	}
	wire, err := handlers.NewConnectHandler(svc).PromoteSandbox(t.Context(), connect.NewRequest(&workspacev1.PromoteSandboxRequest{
		SandboxId: id.String(), Confirm: true, Force: true, ExpectedPatchSha256: digest,
	}))
	if response.StatusCode != http.StatusOK || rest.AppliedPatchSHA256 != digest || err != nil || wire.Msg.AppliedPatchSha256 != digest || calls != 2 {
		t.Fatalf("approval receipt lost identity: HTTP %d %+v; Connect %+v, %v; calls %d", response.StatusCode, rest, wire, err, calls)
	}
}

func TestMalformedApprovalCannotDropReviewPrecondition(t *testing.T) {
	for _, body := range []string{
		`{"expectedPatchSha256":123}`,
		`{"expectedPatchSha256":"truncated`,
		`{"expectedPatchSha25":"misspelled"}`,
		`{"mode":"all"} {"expectedPatchSha256":"second-document"}`,
		`null`,
	} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			svc := &sandboxiface.FakeService{ApproveFn: func(context.Context, *types.ApprovalRequest) (*types.ApprovalResult, error) {
				calls++
				return &types.ApprovalResult{Success: true}, nil
			}}
			live := newLive(t, svc)
			response, result := live.DoJSON(t, "POST", sandboxesPath(uuid.New(), "/approve"), body)
			if response.StatusCode != http.StatusBadRequest || calls != 0 {
				t.Fatalf("malformed review approval reached apply: HTTP %d, calls %d, %s", response.StatusCode, calls, result)
			}
		})
	}
}

// TestGetDiff_LiveStatus_HitsLivePath: Active sandbox dispatches to
// the live GetDiff (not the archive seam).
func TestGetDiff_LiveStatus_HitsLivePath(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	liveCalls, archiveCalls := 0, 0
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: id, Status: types.StatusActive, UpdatedAt: now, CreatedAt: now}, nil
		},
		GetDiffFn: func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
			liveCalls++
			return &types.DiffResult{
				SandboxID:   id,
				Files:       []*types.FileChange{{FilePath: "live.txt", ChangeType: types.ChangeTypeAdded}},
				UnifiedDiff: "live diff content",
				Generated:   now,
			}, nil
		},
		GetArchiveFn: func(ctx context.Context, id uuid.UUID) (*types.DiffResult, error) {
			archiveCalls++
			return nil, nil
		},
	}

	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body=%s", resp.StatusCode, body)
	}

	var got types.DiffResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if liveCalls != 1 {
		t.Errorf("live GetDiff called %d times, want 1", liveCalls)
	}
	if archiveCalls != 0 {
		t.Errorf("archive seam called for live status, calls=%d", archiveCalls)
	}
	if got.UnifiedDiff != "live diff content" {
		t.Errorf("unified_diff = %q, want live content", got.UnifiedDiff)
	}
	if got.ArchiveState != "" {
		t.Errorf("archive_state = %q, want empty (live response)", got.ArchiveState)
	}
}

func TestGetDiffRetainsServiceArchiveResult(t *testing.T) {
	for _, status := range []types.Status{types.StatusApproved, types.StatusRejected, types.StatusDeleted} {
		t.Run(string(status), func(t *testing.T) {
			id := uuid.New()
			now := time.Now()
			diffCalls := 0
			svc := &sandboxiface.FakeService{
				GetFn: func(ctx context.Context, gid uuid.UUID) (*types.Sandbox, error) {
					return &types.Sandbox{ID: gid, Status: status, UpdatedAt: now}, nil
				},
				GetDiffFn: func(ctx context.Context, gid uuid.UUID) (*types.DiffResult, error) {
					diffCalls++
					return &types.DiffResult{
						SandboxID:    gid,
						Files:        []*types.FileChange{{FilePath: "archived.txt"}},
						UnifiedDiff:  "archived content for " + string(status),
						Generated:    now,
						ArchiveState: types.ArchiveStateComplete,
					}, nil
				},
			}

			live := newLive(t, svc)
			resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff"), "")
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d body=%s", resp.StatusCode, body)
			}
			if diffCalls != 1 {
				t.Errorf("service GetDiff called %d times, want one", diffCalls)
			}
			var got types.DiffResult
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.UnifiedDiff != "archived content for "+string(status) {
				t.Errorf("unified_diff = %q, want archive content", got.UnifiedDiff)
			}
			if got.ArchiveState != types.ArchiveStateComplete {
				t.Errorf("archive_state = %q, want complete", got.ArchiveState)
			}
		})
	}
}

func TestGetDiffRetainsServiceNotCapturedResult(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, gid uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: gid, Status: types.StatusApproved, UpdatedAt: now}, nil
		},
		GetDiffFn: func(ctx context.Context, gid uuid.UUID) (*types.DiffResult, error) {
			return &types.DiffResult{SandboxID: gid, Files: []*types.FileChange{}, ArchiveState: types.ArchiveStateNotCaptured}, nil
		},
	}

	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	var got types.DiffResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ArchiveState != types.ArchiveStateNotCaptured {
		t.Errorf("archive_state = %q, want not_captured", got.ArchiveState)
	}
	if len(got.Files) != 0 {
		t.Errorf("files = %d, want 0 for not_captured", len(got.Files))
	}
}

// TestGetDiff_CreatingStatus_LiveEmpty: a Creating sandbox has no
// upper dir yet — return an empty result with no ArchiveState (the
// response is from the live path; there's just no data yet).
func TestGetDiff_CreatingStatus_LiveEmpty(t *testing.T) {
	id := uuid.New()
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, gid uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: gid, Status: types.StatusCreating, CreatedAt: time.Now()}, nil
		},
		GetDiffFn: func(ctx context.Context, gid uuid.UUID) (*types.DiffResult, error) {
			return &types.DiffResult{SandboxID: gid, Files: []*types.FileChange{}}, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	var got types.DiffResult
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ArchiveState != "" {
		t.Errorf("archive_state = %q, want empty (Creating is live, just no data yet)", got.ArchiveState)
	}
	if len(got.Files) != 0 {
		t.Errorf("files = %d, want 0 for Creating", len(got.Files))
	}
}

// TestGetDiff_ViewModeOnArchived_Rejected: full_diff/source require a
// live overlay; archived sandboxes 400 on those modes.
func TestGetDiff_ViewModeOnArchived_Rejected(t *testing.T) {
	id := uuid.New()
	svc := &sandboxiface.FakeService{
		GetFn: func(ctx context.Context, gid uuid.UUID) (*types.Sandbox, error) {
			return &types.Sandbox{ID: gid, Status: types.StatusApproved}, nil
		},
	}
	live := newLive(t, svc)
	for _, mode := range []string{"full_diff", "source"} {
		t.Run(mode, func(t *testing.T) {
			resp, body := live.DoJSON(t, "GET", sandboxesPath(id, "/diff?mode="+mode), "")
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status for mode=%s = %d, want 400; body=%s", mode, resp.StatusCode, body)
			}
		})
	}
}

// TestGetDiffFile_Found: the per-file blob endpoint streams the
// archived bytes verbatim.
func TestGetDiffFile_Found(t *testing.T) {
	id := uuid.New()
	wantContent := []byte("archived file body\n")
	svc := &sandboxiface.FakeService{
		FetchArchiveFileFn: func(ctx context.Context, gid uuid.UUID, path string) ([]byte, error) {
			if path != "src/x.txt" {
				return nil, blobstore.ErrNotFound
			}
			return wantContent, nil
		},
	}
	live := newLive(t, svc)
	resp, gotBody := live.Do(t, "GET", sandboxesPath(id, "/diff/file?path=src/x.txt"), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if string(gotBody) != string(wantContent) {
		t.Errorf("body = %q, want %q", gotBody, wantContent)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want octet-stream", ct)
	}
}

// TestGetDiffFile_NotFound: missing entry → 404.
func TestGetDiffFile_NotFound(t *testing.T) {
	svc := &sandboxiface.FakeService{
		FetchArchiveFileFn: func(ctx context.Context, gid uuid.UUID, path string) ([]byte, error) {
			return nil, blobstore.ErrNotFound
		},
	}
	live := newLive(t, svc)
	resp, _ := live.DoJSON(t, "GET",
		sandboxesPath(uuid.New(), "/diff/file?path=missing.txt"), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// TestGetDiffFile_MissingPath_400: empty path → 400.
func TestGetDiffFile_MissingPath_400(t *testing.T) {
	live := newLive(t, &sandboxiface.FakeService{})
	resp, _ := live.DoJSON(t, "GET", sandboxesPath(uuid.New(), "/diff/file"), "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for missing path", resp.StatusCode)
	}
}

// TestListHistory_PassesFilters: every documented query parameter is
// forwarded into the ArchiveListFilter.
func TestListHistory_PassesFilters(t *testing.T) {
	var captured types.ArchiveListFilter
	svc := &sandboxiface.FakeService{
		ListHistoryFn: func(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error) {
			captured = filter
			return []*types.DiffArchive{
				{SandboxID: uuid.New(), SandboxStatus: types.StatusApproved, ArchiveState: types.ArchiveStateComplete},
			}, 1, nil
		},
	}
	live := newLive(t, svc)

	url := "/api/v1/sandboxes/history?status=approved&status=rejected" +
		"&projectRoot=%2Ftmp%2Fp&owner=alice&agentManagerRunId=run-1" +
		"&search=needle&snapshotAtFrom=2026-01-01T00:00:00Z" +
		"&snapshotAtTo=2026-12-31T23:59:59Z" +
		"&sortBy=total_blob_bytes&sortDesc=true&limit=25&offset=50"

	resp, body := live.DoJSON(t, "GET", url, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}

	if len(captured.Statuses) != 2 ||
		captured.Statuses[0] != types.StatusApproved ||
		captured.Statuses[1] != types.StatusRejected {
		t.Errorf("Statuses = %v, want [approved rejected]", captured.Statuses)
	}
	if captured.ProjectRoot != "/tmp/p" {
		t.Errorf("ProjectRoot = %q", captured.ProjectRoot)
	}
	if captured.Owner != "alice" {
		t.Errorf("Owner = %q", captured.Owner)
	}
	if captured.AgentManagerRunID != "run-1" {
		t.Errorf("RunID = %q", captured.AgentManagerRunID)
	}
	if captured.Search != "needle" {
		t.Errorf("Search = %q", captured.Search)
	}
	if captured.SortBy != "total_blob_bytes" {
		t.Errorf("SortBy = %q", captured.SortBy)
	}
	if !captured.SortDesc {
		t.Errorf("SortDesc = false, want true")
	}
	if captured.Limit != 25 {
		t.Errorf("Limit = %d, want 25", captured.Limit)
	}
	if captured.Offset != 50 {
		t.Errorf("Offset = %d, want 50", captured.Offset)
	}
	if captured.SnapshotAtFrom.IsZero() || captured.SnapshotAtTo.IsZero() {
		t.Errorf("date bounds not parsed: from=%v to=%v", captured.SnapshotAtFrom, captured.SnapshotAtTo)
	}

	var got handlers.ListHistoryResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TotalCount != 1 || len(got.Archives) != 1 {
		t.Errorf("response = %+v, want 1 row", got)
	}
}

// TestListHistory_BadDate_400: malformed date returns 400 with a
// caller-friendly error.
func TestListHistory_BadDate_400(t *testing.T) {
	svc := &sandboxiface.FakeService{}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "GET", "/api/v1/sandboxes/history?snapshotAtFrom=not-a-date", "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if !strings.Contains(string(body), "RFC3339") {
		t.Errorf("body = %s, want RFC3339 hint", body)
	}
}

// TestListHistory_DefaultsEmpty: with no archives, returns empty array
// (not null) for predictable client decoding.
func TestListHistory_DefaultsEmpty(t *testing.T) {
	svc := &sandboxiface.FakeService{
		ListHistoryFn: func(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error) {
			return nil, 0, nil
		},
	}
	live := newLive(t, svc)
	resp, body := live.DoJSON(t, "GET", "/api/v1/sandboxes/history", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.StatusCode, body)
	}
	var got handlers.ListHistoryResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Archives == nil {
		t.Errorf("Archives is nil; want empty array")
	}
	if got.TotalCount != 0 {
		t.Errorf("TotalCount = %d, want 0", got.TotalCount)
	}
}

// TestListHistory_RouteOrderDoesNotShadowID: /sandboxes/history must
// be matched before /sandboxes/{id}, so the literal path wins. If
// route registration regresses, GetSandbox would parse "history" as a
// UUID and 400 on it instead of returning history listing.
func TestListHistory_RouteOrderDoesNotShadowID(t *testing.T) {
	called := false
	svc := &sandboxiface.FakeService{
		ListHistoryFn: func(ctx context.Context, filter types.ArchiveListFilter) ([]*types.DiffArchive, int, error) {
			called = true
			return []*types.DiffArchive{}, 0, nil
		},
		GetFn: func(ctx context.Context, id uuid.UUID) (*types.Sandbox, error) {
			t.Fatal("GetSandbox should NOT be called for /sandboxes/history")
			return nil, nil
		},
	}
	live := newLive(t, svc)
	resp, _ := live.DoJSON(t, "GET", "/api/v1/sandboxes/history", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (history must take precedence over /{id})", resp.StatusCode)
	}
	if !called {
		t.Error("ListHistory was not invoked; route may have been shadowed by /sandboxes/{id}")
	}
}
