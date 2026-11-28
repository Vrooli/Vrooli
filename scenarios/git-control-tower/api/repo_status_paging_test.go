package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	repodomain "git-control-tower/internal/repo"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func TestBoundRepoStatusPagesLargeAndDenseOutputs(t *testing.T) {
	files := &repov1.FilesStatus{}
	for i := 0; i < 4096; i++ {
		path := fmt.Sprintf("%s/tracked-%05d.txt", strings.Repeat("segment/", 240), i)
		switch i % 4 {
		case 0:
			files.Staged = append(files.Staged, path)
		case 1:
			files.Unstaged = append(files.Unstaged, path)
		case 2:
			files.Untracked = append(files.Untracked, path)
		case 3:
			files.Conflicts = append(files.Conflicts, path)
		}
	}
	full := &repov1.GetRepoStatusResponse{RepoDir: "/repo", Files: files, Summary: &repov1.StatusSummary{Staged: 1024, Unstaged: 1024, Untracked: 1024, Conflicts: 1024}}
	seen := map[string]int{}
	token := ""
	var snapshot string
	for {
		page, err := boundRepoStatusResponse(full, "/repo", 256, token)
		if err != nil {
			t.Fatal(err)
		}
		if int(page.GetPage().GetResponseBytes()) > repoStatusResponseBudget {
			t.Fatalf("page bytes %d exceed budget", page.GetPage().GetResponseBytes())
		}
		if page.GetPage().GetReturnedEntries() >= 256 {
			t.Fatal("dense payload did not reduce the requested page size")
		}
		if snapshot == "" {
			snapshot = page.GetPage().GetSnapshotId()
		} else if page.GetPage().GetSnapshotId() != snapshot {
			t.Fatal("snapshot changed during stable pagination")
		}
		for _, group := range [][]string{page.Files.Staged, page.Files.Unstaged, page.Files.Untracked, page.Files.Conflicts} {
			for _, p := range group {
				seen[p]++
			}
		}
		token = page.GetPage().GetNextPageToken()
		if token == "" {
			break
		}
	}
	if len(seen) != 4096 {
		t.Fatalf("got %d unique category paths, want 4096", len(seen))
	}
	for p, n := range seen {
		if n != 1 {
			t.Fatalf("path %q appeared %d times", p, n)
		}
	}
	if int(full.GetSummary().GetStaged()) != len(full.GetFiles().GetStaged()) {
		t.Fatal("global summary was not preserved")
	}
}

func TestRepoStatusPageCacheContinuesCapturedSnapshotDuringWorktreeChanges(t *testing.T) {
	for _, mutation := range []string{"tracked-content", "index", "untracked"} {
		t.Run(mutation, func(t *testing.T) {
			repo, git := newPagingGitRepo(t)
			mustGitPaging(t, repo, "config", "user.name", "Status Pages")
			mustGitPaging(t, repo, "config", "user.email", "status@example.invalid")
			writePagingFile(t, repo, "tracked.txt", "base\n")
			mustGitPaging(t, repo, "add", "tracked.txt")
			mustGitPaging(t, repo, "commit", "-qm", "base")
			writePagingFile(t, repo, "tracked.txt", "base\nfirst change\n")
			writePagingFile(t, repo, "untracked-a.txt", "one\n")
			writePagingFile(t, repo, "untracked-b.txt", "two\n")
			if mutation == "index" {
				mustGitPaging(t, repo, "add", "tracked.txt")
			}
			cache := NewRepoStatusPageCache(1<<20, 8)
			_, full, _ := readPagingStatus(t, repo, git)
			first, err := cache.Store(context.Background(), repo, full, 1)
			if err != nil {
				t.Fatal(err)
			}
			token := first.GetPage().GetNextPageToken()
			if token == "" {
				t.Fatal("expected continuation token")
			}
			replay, err := cache.Continue(context.Background(), repo, token, 1)
			if err != nil {
				t.Fatalf("same snapshot replay: %v", err)
			}
			if replay.GetPage().GetSnapshotId() != first.GetPage().GetSnapshotId() {
				t.Fatal("replay changed snapshot id")
			}
			switch mutation {
			case "tracked-content":
				writePagingFile(t, repo, "tracked.txt", "base\nsecond change, same category\n")
			case "index":
				writePagingFile(t, repo, "tracked.txt", "base\nfirst change\nstaged again\n")
				mustGitPaging(t, repo, "add", "tracked.txt")
			case "untracked":
				writePagingFile(t, repo, "untracked-c.txt", "three\n")
			}
			_, err = cache.Continue(context.Background(), repo, token, 1)
			if err != nil {
				t.Fatalf("worktree %s change invalidated captured snapshot: %v", mutation, err)
			}
		})
	}
}

func TestRepoStatusPageCacheTraversesEveryCategoryExactlyOnceWithinByteBudget(t *testing.T) {
	repo, git := newPagingGitRepo(t)
	mustGitPaging(t, repo, "config", "user.name", "Status Pages")
	mustGitPaging(t, repo, "config", "user.email", "status@example.invalid")
	writePagingFile(t, repo, "tracked.txt", "base\n")
	mustGitPaging(t, repo, "add", "tracked.txt")
	mustGitPaging(t, repo, "commit", "-qm", "base")
	writePagingFile(t, repo, "tracked.txt", "base\nmodified\n")
	writePagingFile(t, repo, "new-a.txt", "a\n")
	writePagingFile(t, repo, "new-b.txt", "b\n")
	_, full, _ := readPagingStatus(t, repo, git)
	cache := NewRepoStatusPageCache(1<<20, 8)
	page, err := cache.Store(context.Background(), repo, full, 1)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	pages := 0
	token := ""
	for {
		pages++
		if page.GetPage().GetResponseBytes() > repoStatusResponseBudget {
			t.Fatalf("page %d bytes=%d", pages, page.GetPage().GetResponseBytes())
		}
		for _, group := range [][]string{page.Files.Staged, page.Files.Unstaged, page.Files.Untracked, page.Files.Conflicts, page.Files.Ignored} {
			for _, path := range group {
				seen[path]++
			}
		}
		token = page.GetPage().GetNextPageToken()
		if token == "" {
			break
		}
		page, err = cache.Continue(context.Background(), repo, token, 1)
		if err != nil {
			t.Fatal(err)
		}
	}
	if pages != 3 || len(seen) != 3 {
		t.Fatalf("pages=%d paths=%d, want 3 each", pages, len(seen))
	}
	for path, count := range seen {
		if count != 1 {
			t.Fatalf("%q returned %d times", path, count)
		}
	}
}

func TestRepoStatusPageCacheRejectsForgedCursorAndInvalidationExpiry(t *testing.T) {
	repo, git := newPagingGitRepo(t)
	mustGitPaging(t, repo, "config", "user.name", "Status Pages")
	mustGitPaging(t, repo, "config", "user.email", "status@example.invalid")
	writePagingFile(t, repo, "tracked.txt", "base\n")
	mustGitPaging(t, repo, "add", "tracked.txt")
	mustGitPaging(t, repo, "commit", "-qm", "base")
	writePagingFile(t, repo, "tracked.txt", "base\nchange\n")
	writePagingFile(t, repo, "one.txt", "1\n")
	writePagingFile(t, repo, "two.txt", "2\n")
	cache := NewRepoStatusPageCache(1<<20, 1)
	_, full, _ := readPagingStatus(t, repo, git)
	first, err := cache.Store(context.Background(), repo, full, 1)
	if err != nil {
		t.Fatal(err)
	}
	token := first.GetPage().GetNextPageToken()
	raw, _ := base64.RawURLEncoding.DecodeString(token)
	var cursor signedRepoStatusCursor
	if err = json.Unmarshal(raw, &cursor); err != nil {
		t.Fatal(err)
	}
	cursor.Offset++
	forgedRaw, _ := json.Marshal(cursor)
	forged := base64.RawURLEncoding.EncodeToString(forgedRaw)
	if _, err = cache.Continue(context.Background(), repo, forged, 1); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("forged cursor error=%v", err)
	}
	// Replaying a valid token is idempotent while its snapshot remains resident.
	if _, err = cache.Continue(context.Background(), repo, token, 1); err != nil {
		t.Fatalf("valid replay: %v", err)
	}
	if _, err = cache.Continue(context.Background(), repo+"-other", token, 1); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("cross-repository cursor error=%v, want InvalidArgument", err)
	}
	cache.InvalidateRepo(repo)
	if _, err = cache.Continue(context.Background(), repo, token, 1); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("invalidated cursor error=%v", err)
	}
	// Cache expiry is capacity eviction, not a time-to-live. A content-only
	// edit to a path already listed as unstaged leaves the page projection
	// unchanged, so add a new untracked path to create a distinct status
	// snapshot and exercise actual capacity eviction.
	first, err = cache.Store(context.Background(), repo, full, 1)
	if err != nil {
		t.Fatal(err)
	}
	expired := first.GetPage().GetNextPageToken()
	writePagingFile(t, repo, "three.txt", "3\n")
	_, nextFull, _ := readPagingStatus(t, repo, git)
	second, err := cache.Store(context.Background(), repo, nextFull, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.GetPage().GetSnapshotId() == first.GetPage().GetSnapshotId() {
		t.Fatal("new untracked path did not create a distinct status snapshot")
	}
	if _, err = cache.Continue(context.Background(), repo, expired, 1); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("evicted cursor error=%v", err)
	}
	if token := second.GetPage().GetNextPageToken(); token == "" {
		t.Fatal("new snapshot has no cursor to verify eviction preserved the replacement")
	} else if _, err := cache.Continue(context.Background(), repo, token, 1); err != nil {
		t.Fatalf("new snapshot cursor: %v", err)
	}
}

func TestRepoStatusPageSnapshotStoresAreSafeForConcurrentFirstPageRequests(t *testing.T) {
	repo, git := newPagingGitRepo(t)
	mustGitPaging(t, repo, "config", "user.name", "Status Pages")
	mustGitPaging(t, repo, "config", "user.email", "status@example.invalid")
	writePagingFile(t, repo, "tracked.txt", "base\n")
	mustGitPaging(t, repo, "add", "tracked.txt")
	mustGitPaging(t, repo, "commit", "-qm", "base")
	writePagingFile(t, repo, "tracked.txt", "base\nchange\n")
	writePagingFile(t, repo, "one.txt", "1\n")
	writePagingFile(t, repo, "two.txt", "2\n")
	_, full, _ := readPagingStatus(t, repo, git)
	cache := NewRepoStatusPageCache(1<<20, 8)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cache.Store(context.Background(), repo, full, 1)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if cache.EntryCount() != 1 {
		t.Fatalf("concurrent same-snapshot starts stored %d entries, want 1", cache.EntryCount())
	}
}

func TestRepoStatusPageAssemblyCacheCoalescesConcurrentStarts(t *testing.T) {
	cache := NewRepoStatusCache(0)
	var calls atomic.Int32
	start := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := cache.Get(context.Background(), "repo", func(context.Context) (*RepoStatus, error) { calls.Add(1); <-release; return &RepoStatus{}, nil })
			if err != nil {
				t.Error(err)
			}
		}()
	}
	close(start)
	deadline := time.Now().Add(time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()
	if got := calls.Load(); got != 1 {
		t.Fatalf("concurrent first pages assembled status %d times, want 1", got)
	}
}

func TestRepoStatusPageCacheCapacityAcrossMatrixSizes(t *testing.T) {
	for _, count := range []int{2048, 8192, 16384} {
		t.Run(fmt.Sprintf("paths_%d", count), func(t *testing.T) {
			repo, git := newPagingGitRepo(t)
			paths := make([]string, count)
			for i := range paths {
				paths[i] = fmt.Sprintf("untracked-%06d.txt", i)
				writePagingFile(t, repo, paths[i], "")
			}
			porcelain, err := git.StatusPorcelainV2(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			name, _ := git.ConfigGet(context.Background(), repo, "user.name")
			email, _ := git.ConfigGet(context.Background(), repo, "user.email")
			identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(context.Background(), repo)
			if err != nil {
				t.Fatal(err)
			}
			status := &RepoStatus{RepoDir: repo, Files: RepoFilesStatus{Untracked: paths}, Author: RepoAuthorStatus{Name: name, Email: email}, statusPorcelain: porcelain}
			response := &repov1.GetRepoStatusResponse{RepoDir: repo, Files: repoFilesStatusProto(status.Files), Worktree: repoWorktreeIdentityProto(identity), BranchStatus: &repov1.BranchStatus{Head: "first"}}
			cache := NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, 1)
			first, err := cache.Store(context.Background(), repo, response, 512)
			if err != nil {
				t.Fatal(err)
			}
			firstToken := first.GetPage().GetNextPageToken()
			if _, err := cache.Continue(context.Background(), repo, firstToken, 512); err != nil {
				t.Fatalf("valid cursor replay at %d paths: %v", count, err)
			}
			secondResponse := &repov1.GetRepoStatusResponse{RepoDir: repo, Files: response.GetFiles(), Worktree: response.GetWorktree(), BranchStatus: &repov1.BranchStatus{Head: "second"}}
			second, err := cache.Store(context.Background(), repo, secondResponse, 512)
			if err != nil {
				t.Fatal(err)
			}
			secondToken := second.GetPage().GetNextPageToken()
			entries, bytes := cache.Usage()
			if entries != 1 || bytes <= 0 || bytes > repoStatusPageCacheMaxBytes {
				t.Fatalf("capacity at %d paths: entries=%d bytes=%d", count, entries, bytes)
			}
			if _, err := cache.Continue(context.Background(), repo, firstToken, 512); connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("evicted cursor at %d paths=%v, want FailedPrecondition", count, err)
			}
			statusCache := NewRepoStatusCache(0)
			statusCache.SetPageSnapshots(cache)
			statusCache.Invalidate(repo)
			if _, err := cache.Continue(context.Background(), repo, secondToken, 512); connect.CodeOf(err) != connect.CodeFailedPrecondition {
				t.Fatalf("invalidated cursor at %d paths=%v, want FailedPrecondition", count, err)
			}
		})
	}
}

func newPagingGitRepo(t *testing.T) (string, *ExecGitRunner) {
	t.Helper()
	repo := t.TempDir()
	mustGitPaging(t, repo, "init", "-q")
	return repo, &ExecGitRunner{GitPath: "git"}
}
func mustGitPaging(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
func writePagingFile(t *testing.T, repo, path, content string) {
	t.Helper()
	full := filepath.Join(repo, path)
	if err := os.WriteFile(full, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
func readPagingStatus(t *testing.T, repo string, git *ExecGitRunner) (*RepoStatus, *repov1.GetRepoStatusResponse, repodomain.Status) {
	t.Helper()
	ctx := context.Background()
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)})
	if err != nil {
		t.Fatal(err)
	}
	identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	return status, repoStatusResponseProto(status, identity), identity
}

func TestBoundRepoStatusRejectsChangedContinuation(t *testing.T) {
	full := &repov1.GetRepoStatusResponse{RepoDir: "/repo", Files: &repov1.FilesStatus{Untracked: []string{"a.txt", "b.txt"}}}
	first, err := boundRepoStatusResponse(full, "/repo", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	changed := &repov1.GetRepoStatusResponse{RepoDir: "/repo", Files: &repov1.FilesStatus{Untracked: []string{"a.txt", "b.txt", "c.txt"}}}
	_, err = boundRepoStatusResponse(changed, "/repo", 1, first.GetPage().GetNextPageToken())
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("changed snapshot error = %v, want FailedPrecondition", err)
	}
}

func TestBoundRepoStatusLegacyRequestCannotSilentlyTruncate(t *testing.T) {
	full := &repov1.GetRepoStatusResponse{RepoDir: "/repo", Files: &repov1.FilesStatus{Untracked: []string{strings.Repeat("x", repoStatusResponseBudget)}}}
	_, err := boundRepoStatusResponse(full, "/repo", 0, "")
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("legacy oversized result error=%v", err)
	}
}
