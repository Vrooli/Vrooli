package main

import (
	"context"
	"encoding/json"
	"fmt"
	repodomain "git-control-tower/internal/repo"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkRepoStatusScale exercises the production status assembler and diff
// service against real disposable repositories. Run with -count=10 so the
// ten reported samples support a simple nearest-rank p95 (maximum for n=10).
func BenchmarkRepoStatusScale(b *testing.B) {
	for _, count := range []int{256, 4096} {
		b.Run(fmt.Sprintf("files_%d", count), func(b *testing.B) {
			repo := b.TempDir()
			mustGitBench(b, repo, "init", "-q")
			mustGitBench(b, repo, "config", "user.name", "GCT scale benchmark")
			mustGitBench(b, repo, "config", "user.email", "gct-scale@example.invalid")
			for i := 0; i < count; i++ {
				name := filepath.Join(repo, fmt.Sprintf("tracked-%05d.txt", i))
				if err := os.WriteFile(name, []byte("baseline line\n"), 0600); err != nil {
					b.Fatal(err)
				}
			}
			mustGitBench(b, repo, "add", ".")
			mustGitBench(b, repo, "commit", "-qm", "benchmark baseline")
			for i := 0; i < count; i += 4 {
				name := filepath.Join(repo, fmt.Sprintf("tracked-%05d.txt", i))
				if err := os.WriteFile(name, []byte("baseline line\nchanged line\n"), 0600); err != nil {
					b.Fatal(err)
				}
			}
			for i := 0; i < count/4; i++ {
				name := filepath.Join(repo, fmt.Sprintf("untracked-%05d.txt", i))
				if err := os.WriteFile(name, []byte("untracked line\n"), 0600); err != nil {
					b.Fatal(err)
				}
			}
			git := &ExecGitRunner{}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			b.Run("status", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo})
					if err != nil {
						b.Fatal(err)
					}
					encoded, err := json.Marshal(status)
					if err != nil {
						b.Fatal(err)
					}
					if count == 4096 && len(encoded) > 350_000 {
						b.Fatalf("4096-file reference workload status JSON = %d bytes; budget is 350000", len(encoded))
					}
					b.ReportMetric(float64(len(encoded)), "status-json-B")
				}
			})
			if count == 4096 {
				seed, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo})
				if err != nil {
					b.Fatal(err)
				}
				full := &repov1.GetRepoStatusResponse{RepoDir: seed.RepoDir, Files: repoFilesStatusProto(seed.Files), FileStats: repoFileStatsProto(seed.FileStats), FileHotspots: intMapProto(seed.FileHotspots), Scopes: repoScopesProto(seed.Scopes), Summary: &repov1.StatusSummary{Staged: int32(seed.Summary.Staged), Unstaged: int32(seed.Summary.Unstaged), Untracked: int32(seed.Summary.Untracked), Conflicts: int32(seed.Summary.Conflicts), Ignored: int32(seed.Summary.Ignored)}}
				b.Run("status_page_client_aggregate", func(b *testing.B) {
					var totalBytes, totalPages int
					for i := 0; i < b.N; i++ {
						token := ""
						pages := 0
						bytes := 0
						for {
							page, e := boundRepoStatusResponse(full, repo, 512, token)
							if e != nil {
								b.Fatal(e)
							}
							pages++
							bytes += int(page.GetPage().GetResponseBytes())
							token = page.GetPage().GetNextPageToken()
							if token == "" {
								break
							}
						}
						totalBytes += bytes
						totalPages += pages
					}
					b.ReportMetric(float64(totalBytes)/float64(b.N), "aggregate-page-bounded-response-B")
					b.ReportMetric(float64(totalPages)/float64(b.N), "pages/op")
				})
				b.Run("status_page_fresh_git_roundtrips", func(b *testing.B) {
					var totalBytes, totalPages int
					for i := 0; i < b.N; i++ {
						token := ""
						pages := 0
						bytes := 0
						for {
							fresh, e := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)})
							if e != nil {
								b.Fatal(e)
							}
							response := benchmarkStatusResponse(fresh)
							page, e := boundRepoStatusResponse(response, repo, 512, token)
							if e != nil {
								b.Fatal(e)
							}
							pages++
							bytes += int(page.GetPage().GetResponseBytes())
							token = page.GetPage().GetNextPageToken()
							if token == "" {
								break
							}
						}
						totalBytes += bytes
						totalPages += pages
					}
					b.ReportMetric(float64(totalBytes)/float64(b.N), "aggregate-page-bounded-response-B")
					b.ReportMetric(float64(totalPages)/float64(b.N), "pages/op")
				})
				b.Run("status_page_comparable_pair", func(b *testing.B) {
					var baseNS, optNS, baseBytes, optBytes, basePages, optPages, baseAssemblies, optAssemblies, baseMaxPage, optMaxPage int64
					cache := NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)
					for i := 0; i < b.N; i++ {
						if repoStatusPageCompareOrder.Add(1)%2 == 0 {
							d, p, by, a, maxPage := benchmarkFullStatusPages(ctx, repo, git)
							baseNS += d
							basePages += int64(p)
							baseBytes += int64(by)
							baseAssemblies += int64(a)
							if int64(maxPage) > baseMaxPage {
								baseMaxPage = int64(maxPage)
							}
							d, p, by, a, maxPage = benchmarkCachedStatusPages(ctx, repo, git, cache)
							optNS += d
							optPages += int64(p)
							optBytes += int64(by)
							optAssemblies += int64(a)
							if int64(maxPage) > optMaxPage {
								optMaxPage = int64(maxPage)
							}
						} else {
							d, p, by, a, maxPage := benchmarkCachedStatusPages(ctx, repo, git, cache)
							optNS += d
							optPages += int64(p)
							optBytes += int64(by)
							optAssemblies += int64(a)
							if int64(maxPage) > optMaxPage {
								optMaxPage = int64(maxPage)
							}
							d, p, by, a, maxPage = benchmarkFullStatusPages(ctx, repo, git)
							baseNS += d
							basePages += int64(p)
							baseBytes += int64(by)
							baseAssemblies += int64(a)
							if int64(maxPage) > baseMaxPage {
								baseMaxPage = int64(maxPage)
							}
						}
					}
					b.ReportMetric(float64(baseNS)/float64(b.N)/1e6, "full-per-page-ms/op")
					b.ReportMetric(float64(optNS)/float64(b.N)/1e6, "cached-guard-ms/op")
					b.ReportMetric(float64(basePages)/float64(b.N), "full-pages/op")
					b.ReportMetric(float64(optPages)/float64(b.N), "cached-pages/op")
					b.ReportMetric(float64(baseBytes)/float64(b.N), "full-bounded-response-B/op")
					b.ReportMetric(float64(optBytes)/float64(b.N), "cached-bounded-response-B/op")
					b.ReportMetric(float64(baseAssemblies)/float64(b.N), "full-status-assemblies/op")
					b.ReportMetric(float64(optAssemblies)/float64(b.N), "cached-status-assemblies/op")
					b.ReportMetric(float64(baseMaxPage), "full-max-page-response-B/op")
					b.ReportMetric(float64(optMaxPage), "cached-max-page-response-B/op")
				})
			}
			b.Run("single_file_diff", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					diff, err := GetDiff(ctx, DiffDeps{Git: git, RepoDir: repo}, DiffRequest{Path: "tracked-00000.txt", Mode: ViewModeFullDiff})
					if err != nil {
						b.Fatal(err)
					}
					encoded, err := json.Marshal(diff)
					if err != nil {
						b.Fatal(err)
					}
					b.ReportMetric(float64(len(encoded)), "diff-json-B")
				}
			})
			b.Log("status workload:", count, "tracked;", count/4, "modified;", count/4, "untracked")
		})
	}
}

var repoStatusPageCompareOrder atomic.Uint64

// BenchmarkRepoStatusPageMatrix is the reusable multi-size continuation proof.
// Each operation is a paired, full page traversal of the same disposable
// fixture; `-count=10 -benchtime=1x` yields ten samples per size for p50/p95.
func BenchmarkRepoStatusPageMatrix(b *testing.B) {
	for _, tracked := range []int{4096, 16384, 32768} {
		b.Run(fmt.Sprintf("tracked_%d", tracked), func(b *testing.B) {
			repo := makeRepoStatusScaleFixture(b, tracked)
			git := &ExecGitRunner{}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			cache := NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)
			var baseNS, cachedNS, basePages, cachedPages, baseBytes, cachedBytes int64
			var baseMaxPage, cachedMaxPage, cachedResident int64
			var baseAssemblies, cachedAssemblies int64
			for i := 0; i < b.N; i++ {
				var full, optimized repoStatusPageMeasurement
				wantEntries := 2 * (tracked / 4)
				if repoStatusPageCompareOrder.Add(1)%2 == 1 {
					full = measureRepoStatusPageTraversal(ctx, repo, git, wantEntries, nil)
					optimized = measureRepoStatusPageTraversal(ctx, repo, git, wantEntries, cache)
				} else {
					optimized = measureRepoStatusPageTraversal(ctx, repo, git, wantEntries, cache)
					full = measureRepoStatusPageTraversal(ctx, repo, git, wantEntries, nil)
				}
				if !reflect.DeepEqual(full.paths, optimized.paths) {
					b.Fatal("fresh and cached page traversals returned different category membership")
				}
				if !sameCategoryPathSets(full.paths, expectedStatusMatrixPaths(tracked)) {
					b.Fatal("page traversal did not return the complete expected category membership")
				}
				if full.returnedEntries != wantEntries || optimized.returnedEntries != wantEntries {
					b.Fatalf("category entries: fresh=%d cached=%d want=%d", full.returnedEntries, optimized.returnedEntries, wantEntries)
				}
				if full.maxPageBytes > repoStatusResponseBudget || optimized.maxPageBytes > repoStatusResponseBudget {
					b.Fatalf("page exceeds response cap: fresh=%d cached=%d", full.maxPageBytes, optimized.maxPageBytes)
				}
				if full.pages != optimized.pages {
					b.Fatalf("page count differs: fresh=%d cached=%d", full.pages, optimized.pages)
				}
				entries, resident := cache.Usage()
				if resident > repoStatusPageCacheMaxBytes || entries > repoStatusPageCacheMaxEntries {
					b.Fatalf("snapshot cache exceeded capacity: entries=%d bytes=%d", entries, resident)
				}
				baseNS += full.durationNS
				cachedNS += optimized.durationNS
				basePages += int64(full.pages)
				cachedPages += int64(optimized.pages)
				baseBytes += int64(full.aggregateBytes)
				cachedBytes += int64(optimized.aggregateBytes)
				baseMaxPage = max(baseMaxPage, int64(full.maxPageBytes))
				cachedMaxPage = max(cachedMaxPage, int64(optimized.maxPageBytes))
				baseAssemblies += int64(full.assemblies)
				cachedAssemblies += int64(optimized.assemblies)
				cachedResident = max(cachedResident, int64(resident))
			}
			b.ReportMetric(float64(baseNS)/float64(b.N)/1e6, "fresh-traversal-ms/op")
			b.ReportMetric(float64(cachedNS)/float64(b.N)/1e6, "cached-traversal-ms/op")
			b.ReportMetric(float64(basePages)/float64(b.N), "fresh-pages/op")
			b.ReportMetric(float64(cachedPages)/float64(b.N), "cached-pages/op")
			b.ReportMetric(float64(baseBytes)/float64(b.N), "fresh-aggregate-B/op")
			b.ReportMetric(float64(cachedBytes)/float64(b.N), "cached-aggregate-B/op")
			b.ReportMetric(float64(baseMaxPage), "fresh-max-page-B/op")
			b.ReportMetric(float64(cachedMaxPage), "cached-max-page-B/op")
			b.ReportMetric(float64(baseAssemblies)/float64(b.N), "fresh-assemblies/op")
			b.ReportMetric(float64(cachedAssemblies)/float64(b.N), "cached-assemblies/op")
			b.ReportMetric(float64(cachedResident), "cached-resident-B")
			b.ReportMetric(float64(cache.EntryCount()), "cache-entries")
			b.ReportAllocs()
			b.Logf("fixture: %d tracked, %d modified, %d untracked; %d changed category entries", tracked, tracked/4, tracked/4, 2*(tracked/4))
		})
	}
}

type repoStatusPageMeasurement struct {
	durationNS      int64
	pages           int
	aggregateBytes  int
	maxPageBytes    int
	assemblies      int
	returnedEntries int
	paths           map[string][]string
}

func measureRepoStatusPageTraversal(ctx context.Context, repo string, git *ExecGitRunner, expectedEntries int, cache *RepoStatusPageCache) repoStatusPageMeasurement {
	start := time.Now()
	out := repoStatusPageMeasurement{paths: map[string][]string{}}
	token := ""
	var snapshot string
	for {
		var status *RepoStatus
		var err error
		if cache == nil || out.pages == 0 {
			status, err = GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)})
			if err != nil {
				panic(err)
			}
			out.assemblies++
		}
		identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repo)
		if err != nil {
			panic(err)
		}
		var page *repov1.GetRepoStatusResponse
		if cache == nil {
			page, err = boundRepoStatusResponse(benchmarkStatusResponse(status, identity), repo, 512, token)
		} else if token == "" {
			page, err = cache.Store(ctx, repo, benchmarkStatusResponse(status, identity), 512)
		} else {
			page, err = cache.Continue(ctx, repo, token, 512)
		}
		if err != nil {
			panic(err)
		}
		if snapshot == "" {
			snapshot = page.GetPage().GetSnapshotId()
		} else if page.GetPage().GetSnapshotId() != snapshot {
			panic("snapshot changed during fixture traversal")
		}
		if int(page.GetPage().GetTotalEntries()) != expectedEntries {
			panic(fmt.Sprintf("page total %d != expected %d", page.GetPage().GetTotalEntries(), expectedEntries))
		}
		pageBytes := int(page.GetPage().GetResponseBytes())
		if pageBytes > repoStatusResponseBudget {
			panic(fmt.Sprintf("page size %d exceeds cap", pageBytes))
		}
		out.pages++
		out.aggregateBytes += pageBytes
		out.maxPageBytes = max(out.maxPageBytes, pageBytes)
		out.returnedEntries += int(page.GetPage().GetReturnedEntries())
		for category, paths := range map[string][]string{
			"staged": page.GetFiles().GetStaged(), "unstaged": page.GetFiles().GetUnstaged(),
			"untracked": page.GetFiles().GetUntracked(), "conflicts": page.GetFiles().GetConflicts(),
			"ignored": page.GetFiles().GetIgnored(),
		} {
			out.paths[category] = append(out.paths[category], paths...)
		}
		token = page.GetPage().GetNextPageToken()
		if token == "" {
			break
		}
	}
	for category, paths := range out.paths {
		seen := make(map[string]bool, len(paths))
		for _, path := range paths {
			if seen[path] {
				panic(fmt.Sprintf("duplicate %s path %q", category, path))
			}
			seen[path] = true
		}
	}
	if out.returnedEntries != statusEntryCount(&repov1.FilesStatus{Staged: out.paths["staged"], Unstaged: out.paths["unstaged"], Untracked: out.paths["untracked"], Conflicts: out.paths["conflicts"], Ignored: out.paths["ignored"]}) {
		panic("returned page-entry total differs from traversed paths")
	}
	out.durationNS = time.Since(start).Nanoseconds()
	return out
}

func expectedStatusMatrixPaths(tracked int) map[string][]string {
	out := map[string][]string{"staged": {}, "unstaged": {}, "untracked": {}, "conflicts": {}, "ignored": {}}
	for i := 0; i < tracked/4; i++ {
		out["unstaged"] = append(out["unstaged"], fmt.Sprintf("tracked-%06d.txt", i))
		out["untracked"] = append(out["untracked"], fmt.Sprintf("untracked-%06d.txt", i))
	}
	return out
}

func sameCategoryPathSets(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for category, left := range a {
		right, ok := b[category]
		if !ok || len(left) != len(right) {
			return false
		}
		seen := make(map[string]int, len(left))
		for _, path := range left {
			seen[path]++
		}
		for _, path := range right {
			seen[path]--
		}
		for _, count := range seen {
			if count != 0 {
				return false
			}
		}
	}
	return true
}

func makeRepoStatusScaleFixture(b *testing.B, tracked int) string {
	b.Helper()
	repo := b.TempDir()
	mustGitBench(b, repo, "init", "-q")
	mustGitBench(b, repo, "config", "user.name", "GCT scale matrix")
	mustGitBench(b, repo, "config", "user.email", "gct-matrix@example.invalid")
	for i := 0; i < tracked; i++ {
		name := filepath.Join(repo, fmt.Sprintf("tracked-%06d.txt", i))
		if err := os.WriteFile(name, []byte("baseline line\n"), 0600); err != nil {
			b.Fatal(err)
		}
	}
	mustGitBench(b, repo, "add", ".")
	mustGitBench(b, repo, "commit", "-qm", "status scale matrix baseline")
	for i := 0; i < tracked/4; i++ {
		name := filepath.Join(repo, fmt.Sprintf("tracked-%06d.txt", i))
		if err := os.WriteFile(name, []byte("baseline line\nchanged line\n"), 0600); err != nil {
			b.Fatal(err)
		}
	}
	for i := 0; i < tracked/4; i++ {
		name := filepath.Join(repo, fmt.Sprintf("untracked-%06d.txt", i))
		if err := os.WriteFile(name, []byte("untracked line\n"), 0600); err != nil {
			b.Fatal(err)
		}
	}
	return repo
}

func benchmarkFullStatusPages(ctx context.Context, repo string, git *ExecGitRunner) (int64, int, int, int, int) {
	start := time.Now()
	token := ""
	pages, bytes, assemblies, maxPage := 0, 0, 0, 0
	for {
		status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)})
		if err != nil {
			panic(err)
		}
		identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repo)
		if err != nil {
			panic(err)
		}
		assemblies++
		page, err := boundRepoStatusResponse(benchmarkStatusResponse(status, identity), repo, 512, token)
		if err != nil {
			panic(err)
		}
		pages++
		bytes += int(page.GetPage().GetResponseBytes())
		if int(page.GetPage().GetResponseBytes()) > maxPage {
			maxPage = int(page.GetPage().GetResponseBytes())
		}
		token = page.GetPage().GetNextPageToken()
		if token == "" {
			break
		}
	}
	return time.Since(start).Nanoseconds(), pages, bytes, assemblies, maxPage
}

func benchmarkCachedStatusPages(ctx context.Context, repo string, git *ExecGitRunner, cache *RepoStatusPageCache) (int64, int, int, int, int) {
	start := time.Now()
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)})
	if err != nil {
		panic(err)
	}
	assemblies := 1
	identity, err := repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repo)
	if err != nil {
		panic(err)
	}
	page, err := cache.Store(ctx, repo, benchmarkStatusResponse(status, identity), 512)
	if err != nil {
		panic(err)
	}
	pages, bytes, maxPage := 1, int(page.GetPage().GetResponseBytes()), int(page.GetPage().GetResponseBytes())
	token := page.GetPage().GetNextPageToken()
	for token != "" {
		identity, err = repodomain.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repo)
		if err != nil {
			panic(err)
		}
		page, err = cache.Continue(ctx, repo, token, 512)
		if err != nil {
			panic(err)
		}
		pages++
		bytes += int(page.GetPage().GetResponseBytes())
		if int(page.GetPage().GetResponseBytes()) > maxPage {
			maxPage = int(page.GetPage().GetResponseBytes())
		}
		token = page.GetPage().GetNextPageToken()
	}
	return time.Since(start).Nanoseconds(), pages, bytes, assemblies, maxPage
}

// BenchmarkRepoStatusRealWorktree is opt-in and read-only. Set
// GCT_BENCH_REAL_REPO to an existing checkout; the benchmark picks one already
// modified path for its per-file diff sample and never changes the worktree.
func BenchmarkRepoStatusRealWorktree(b *testing.B) {
	repo := os.Getenv("GCT_BENCH_REAL_REPO")
	if repo == "" {
		b.Skip("set GCT_BENCH_REAL_REPO to a read-only checkout")
	}
	git := &ExecGitRunner{}
	ctx := context.Background()
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo})
	if err != nil {
		b.Fatal(err)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("real worktree: tracked=%d changed=%d staged=%d unstaged=%d untracked=%d status-json=%d B", status.Summary.Staged+status.Summary.Unstaged, status.Summary.Staged+status.Summary.Unstaged+status.Summary.Untracked, status.Summary.Staged, status.Summary.Unstaged, status.Summary.Untracked, len(encoded))
	path := ""
	if len(status.Files.Unstaged) > 0 {
		path = status.Files.Unstaged[0]
	} else if len(status.Files.Staged) > 0 {
		path = status.Files.Staged[0]
	}
	if path == "" {
		b.Fatal("real worktree needs at least one changed tracked file for diff sample")
	}
	b.Run("status", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repo})
			if err != nil {
				b.Fatal(err)
			}
			payload, err := json.Marshal(result)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(payload)), "status-json-B")
		}
	})
	b.Run("one_changed_file_diff", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result, err := GetDiff(ctx, DiffDeps{Git: git, RepoDir: repo}, DiffRequest{Path: path, Staged: len(status.Files.Staged) > 0 && path == status.Files.Staged[0], Mode: ViewModeFullDiff})
			if err != nil {
				b.Fatal(err)
			}
			payload, err := json.Marshal(result)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportMetric(float64(len(payload)), "diff-json-B")
		}
	})
}

func mustGitBench(b *testing.B, repo string, args ...string) {
	b.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		b.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func benchmarkStatusResponse(status *RepoStatus, identity ...repodomain.Status) *repov1.GetRepoStatusResponse {
	response := &repov1.GetRepoStatusResponse{Branch: status.Branch.Head, Detached: status.Branch.Head == "", BranchStatus: &repov1.BranchStatus{Head: status.Branch.Head, Upstream: status.Branch.Upstream, Ahead: int32(status.Branch.Ahead), Behind: int32(status.Branch.Behind), Oid: status.Branch.OID}, Files: repoFilesStatusProto(status.Files), FileStats: repoFileStatsProto(status.FileStats), FileHotspots: intMapProto(status.FileHotspots), Scopes: repoScopesProto(status.Scopes), Summary: &repov1.StatusSummary{Staged: int32(status.Summary.Staged), Unstaged: int32(status.Summary.Unstaged), Untracked: int32(status.Summary.Untracked), Conflicts: int32(status.Summary.Conflicts), Ignored: int32(status.Summary.Ignored)}, RepoDir: status.RepoDir}
	if len(identity) > 0 {
		response.Worktree = repoWorktreeIdentityProto(identity[0])
	}
	return response
}
