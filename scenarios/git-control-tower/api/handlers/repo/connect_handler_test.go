package repo_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	hrepo "git-control-tower/handlers/repo"
	drepo "git-control-tower/internal/repo"
	"git-control-tower/internal/worktree"
	"git-control-tower/internal/worktree/mocks"

	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

func newTestRepoClient(t *testing.T, insp worktree.Inspector) (repoconnect.RepoServiceClient, func()) {
	t.Helper()
	svc := drepo.NewService(insp)
	path, handler := hrepo.NewHandler(hrepo.Deps{Service: svc})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	return repoconnect.NewRepoServiceClient(srv.Client(), srv.URL), srv.Close
}

func TestRepoHandler_GetRepoStatus_LinkedWorktree(t *testing.T) {
	insp := &mocks.FakeInspector{
		IdentifyResult: worktree.Identity{
			IsLinkedWorktree:    true,
			CommonRepoRoot:      "/home/user/repo",
			WorktreeName:        "feature",
			WorktreeHead:        "abc",
			LinkedWorktreeCount: 3,
			Branch:              "feature",
		},
	}
	client, cleanup := newTestRepoClient(t, insp)
	defer cleanup()

	resp, err := client.GetRepoStatus(context.Background(), connect.NewRequest(&repov1.GetRepoStatusRequest{RepoPath: "/home/user/wt/feature"}))
	if err != nil {
		t.Fatalf("GetRepoStatus: %v", err)
	}
	if !resp.Msg.Worktree.IsLinkedWorktree {
		t.Fatalf("expected IsLinkedWorktree=true")
	}
	if resp.Msg.Worktree.WorktreeName != "feature" || resp.Msg.Branch != "feature" {
		t.Fatalf("unexpected: %+v", resp.Msg)
	}
	if resp.Msg.Worktree.LinkedWorktreeCount != 3 {
		t.Fatalf("expected count=3, got %d", resp.Msg.Worktree.LinkedWorktreeCount)
	}
}

func TestRepoHandler_GetRepoStatus_RequiresRepoPath(t *testing.T) {
	client, cleanup := newTestRepoClient(t, &mocks.FakeInspector{})
	defer cleanup()
	_, err := client.GetRepoStatus(context.Background(), connect.NewRequest(&repov1.GetRepoStatusRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", err)
	}
}

func TestRepoHandler_FileSurface_UsesTypedCallbacks(t *testing.T) {
	path, handler := hrepo.NewHandler(hrepo.Deps{
		GetFiles: func(context.Context, *repov1.GetFilesRequest) (*repov1.GetFilesResponse, error) {
			return &repov1.GetFilesResponse{Files: []*repov1.RepoFileInfo{{Path: "main.go", Language: "go", Status: "tracked"}}}, nil
		},
		GetDirectoryContents: func(context.Context, *repov1.GetDirectoryContentsRequest) (*repov1.GetDirectoryContentsResponse, error) {
			return &repov1.GetDirectoryContentsResponse{Entries: []*repov1.DirectoryEntry{{Name: "api", Path: "api", IsDir: true}}}, nil
		},
		GetRelatedFiles: func(context.Context, *repov1.GetRelatedFilesRequest) (*repov1.GetRelatedFilesResponse, error) {
			return &repov1.GetRelatedFilesResponse{Path: "main.go", Related: []*repov1.RelatedFile{{Path: "main_test.go", RelationType: "test"}}}, nil
		},
		SearchContent: func(context.Context, *repov1.SearchContentRequest) (*repov1.SearchContentResponse, error) {
			return &repov1.SearchContentResponse{Matches: []*repov1.ContentMatch{{Path: "main.go", LineNumber: 1, Content: "package main"}}}, nil
		},
		DeletePath: func(context.Context, *repov1.DeletePathRequest) (*repov1.DeletePathResponse, error) {
			return &repov1.DeletePathResponse{Success: true, Path: "old.go"}, nil
		},
		SaveFileContent: func(context.Context, *repov1.SaveFileContentRequest) (*repov1.SaveFileContentResponse, error) {
			return &repov1.SaveFileContentResponse{Success: true, Path: "main.go", ContentHash: "hash", BytesWritten: 12}, nil
		},
	})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := repoconnect.NewRepoServiceClient(srv.Client(), srv.URL)

	files, err := client.GetFiles(context.Background(), connect.NewRequest(&repov1.GetFilesRequest{}))
	if err != nil || len(files.Msg.Files) != 1 || files.Msg.Files[0].Path != "main.go" {
		t.Fatalf("GetFiles: response=%v err=%v", files.Msg, err)
	}
	directory, err := client.GetDirectoryContents(context.Background(), connect.NewRequest(&repov1.GetDirectoryContentsRequest{}))
	if err != nil || len(directory.Msg.Entries) != 1 || !directory.Msg.Entries[0].IsDir {
		t.Fatalf("GetDirectoryContents: response=%v err=%v", directory.Msg, err)
	}
	related, err := client.GetRelatedFiles(context.Background(), connect.NewRequest(&repov1.GetRelatedFilesRequest{}))
	if err != nil || len(related.Msg.Related) != 1 || related.Msg.Related[0].RelationType != "test" {
		t.Fatalf("GetRelatedFiles: response=%v err=%v", related.Msg, err)
	}
	search, err := client.SearchContent(context.Background(), connect.NewRequest(&repov1.SearchContentRequest{}))
	if err != nil || len(search.Msg.Matches) != 1 || search.Msg.Matches[0].LineNumber != 1 {
		t.Fatalf("SearchContent: response=%v err=%v", search.Msg, err)
	}
	deleted, err := client.DeletePath(context.Background(), connect.NewRequest(&repov1.DeletePathRequest{}))
	if err != nil || !deleted.Msg.Success {
		t.Fatalf("DeletePath: response=%v err=%v", deleted.Msg, err)
	}
	saved, err := client.SaveFileContent(context.Background(), connect.NewRequest(&repov1.SaveFileContentRequest{}))
	if err != nil || !saved.Msg.Success || saved.Msg.ContentHash != "hash" {
		t.Fatalf("SaveFileContent: response=%v err=%v", saved.Msg, err)
	}
}

func TestRepoHandler_SettingsSurface_UsesTypedCallbacks(t *testing.T) {
	path, handler := hrepo.NewHandler(hrepo.Deps{
		GetGroupingRules: func(context.Context, *repov1.GetGroupingRulesRequest) (*repov1.GroupingRulesResponse, error) {
			return &repov1.GroupingRulesResponse{Enabled: true, Rules: []*repov1.GroupingRule{{Id: "api", Label: "API"}}}, nil
		},
		SaveGroupingRules: func(context.Context, *repov1.SaveGroupingRulesRequest) (*repov1.GroupingRulesResponse, error) {
			return &repov1.GroupingRulesResponse{Enabled: true}, nil
		},
		GetGitignoreHealth: func(context.Context, *repov1.GetGitignoreHealthRequest) (*repov1.GitignoreHealthResponse, error) {
			return &repov1.GitignoreHealthResponse{RootEntryCount: 2, Suggestions: []*repov1.GitignoreSuggestion{{Line: 4, Pattern: "api/tmp/"}}}, nil
		},
		MoveGitignoreEntry: func(context.Context, *repov1.MoveGitignoreEntryRequest) (*repov1.MoveGitignoreEntryResponse, error) {
			return &repov1.MoveGitignoreEntryResponse{Success: true, RemovedFrom: ".gitignore"}, nil
		},
		GetTrackedBinaries: func(context.Context, *repov1.GetTrackedBinariesRequest) (*repov1.TrackedBinariesResponse, error) {
			return &repov1.TrackedBinariesResponse{TotalBytes: 42, Binaries: []*repov1.TrackedBinary{{Path: "bin/tool"}}}, nil
		},
		UntrackBinary: func(context.Context, *repov1.UntrackBinaryRequest) (*repov1.UntrackBinaryResponse, error) {
			return &repov1.UntrackBinaryResponse{Success: true, RemovedFromIndex: true}, nil
		},
	})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := repoconnect.NewRepoServiceClient(srv.Client(), srv.URL)

	grouping, err := client.GetGroupingRules(context.Background(), connect.NewRequest(&repov1.GetGroupingRulesRequest{}))
	if err != nil || !grouping.Msg.Enabled || grouping.Msg.Rules[0].Id != "api" {
		t.Fatalf("GetGroupingRules: response=%v err=%v", grouping.Msg, err)
	}
	if saved, err := client.SaveGroupingRules(context.Background(), connect.NewRequest(&repov1.SaveGroupingRulesRequest{})); err != nil || !saved.Msg.Enabled {
		t.Fatalf("SaveGroupingRules: response=%v err=%v", saved.Msg, err)
	}
	health, err := client.GetGitignoreHealth(context.Background(), connect.NewRequest(&repov1.GetGitignoreHealthRequest{}))
	if err != nil || health.Msg.RootEntryCount != 2 || health.Msg.Suggestions[0].Pattern != "api/tmp/" {
		t.Fatalf("GetGitignoreHealth: response=%v err=%v", health.Msg, err)
	}
	moved, err := client.MoveGitignoreEntry(context.Background(), connect.NewRequest(&repov1.MoveGitignoreEntryRequest{}))
	if err != nil || !moved.Msg.Success {
		t.Fatalf("MoveGitignoreEntry: response=%v err=%v", moved.Msg, err)
	}
	binaries, err := client.GetTrackedBinaries(context.Background(), connect.NewRequest(&repov1.GetTrackedBinariesRequest{}))
	if err != nil || binaries.Msg.TotalBytes != 42 || binaries.Msg.Binaries[0].Path != "bin/tool" {
		t.Fatalf("GetTrackedBinaries: response=%v err=%v", binaries.Msg, err)
	}
	untracked, err := client.UntrackBinary(context.Background(), connect.NewRequest(&repov1.UntrackBinaryRequest{}))
	if err != nil || !untracked.Msg.Success || !untracked.Msg.RemovedFromIndex {
		t.Fatalf("UntrackBinary: response=%v err=%v", untracked.Msg, err)
	}
}
