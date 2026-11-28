package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"git-control-tower/internal/repo"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	"google.golang.org/protobuf/proto"
)

// BenchmarkRepoStatusRealPagedPair is an opt-in, read-only comparison against
// one existing checkout. It records only counts and cryptographic state
// digests, never paths or file contents. Each measured pair is fenced by a
// digest of porcelain, index, changed-path contents, author config, response
// semantics and worktree identity. If any other owner changes the checkout,
// the sample fails closed instead of producing a mixed-state comparison.
func BenchmarkRepoStatusRealPagedPair(b *testing.B) {
	repoDir := os.Getenv("GCT_BENCH_REAL_REPO")
	if repoDir == "" {
		b.Skip("set GCT_BENCH_REAL_REPO to a read-only checkout")
	}
	git := &ExecGitRunner{GitPath: "git"}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	status, fingerprint, err := fingerprintRepoStatus(ctx, repoDir, git)
	if err != nil {
		b.Fatalf("read-only initial worktree fingerprint: %v", err)
	}
	encoded, err := json.Marshal(status)
	if err != nil {
		b.Fatal(err)
	}
	entries := statusEntryCount(repoFilesStatusProto(status.Files))
	expectedPaths := map[string][]string{
		"staged": status.Files.Staged, "unstaged": status.Files.Unstaged,
		"untracked": status.Files.Untracked, "conflicts": status.Files.Conflicts,
		"ignored": status.Files.Ignored,
	}
	b.Logf("read-only worktree state: fingerprint-sha256=%s tracked=%d staged=%d unstaged=%d untracked=%d conflicts=%d ignored=%d status-json-B=%d entries=%d",
		fingerprint, status.Summary.Staged+status.Summary.Unstaged,
		status.Summary.Staged, status.Summary.Unstaged, status.Summary.Untracked,
		status.Summary.Conflicts, status.Summary.Ignored, len(encoded), entries)
	cache := NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)
	var compareOrder uint64
	var freshNS, cachedNS, freshPages, cachedPages, freshBytes, cachedBytes int64
	var freshMax, cachedMax int64
	var residentMax int64
	for i := 0; i < b.N; i++ {
		before, beforeFingerprint, err := fingerprintRepoStatus(ctx, repoDir, git)
		if err != nil {
			b.Fatalf("before sample %d: %v", i+1, err)
		}
		beforeJSON, err := json.Marshal(before)
		if err != nil {
			b.Fatal(err)
		}
		if beforeFingerprint != fingerprint || len(beforeJSON) != len(encoded) || statusEntryCount(repoFilesStatusProto(before.Files)) != entries {
			b.Fatalf("worktree fingerprint or counts changed before sample %d; refusing mixed-state comparison", i+1)
		}

		var fresh, cached realStatusPageWalk
		compareOrder++
		if compareOrder%2 == 0 {
			cached = walkRealStatusPages(ctx, repoDir, git, cache, entries)
			fresh = walkRealStatusPages(ctx, repoDir, git, nil, entries)
		} else {
			fresh = walkRealStatusPages(ctx, repoDir, git, nil, entries)
			cached = walkRealStatusPages(ctx, repoDir, git, cache, entries)
		}
		if !reflect.DeepEqual(fresh.paths, cached.paths) {
			b.Fatalf("fresh and cached pages returned different category membership in sample %d", i+1)
		}
		if !reflect.DeepEqual(fresh.paths, expectedPaths) {
			b.Fatalf("paged traversal did not return every expected category path exactly once in sample %d", i+1)
		}
		if fresh.pages != cached.pages || fresh.aggregateBytes == 0 || cached.aggregateBytes == 0 {
			b.Fatalf("fresh/cached page totals differ in sample %d: pages=%d/%d bytes=%d/%d", i+1, fresh.pages, cached.pages, fresh.aggregateBytes, cached.aggregateBytes)
		}
		after, afterFingerprint, err := fingerprintRepoStatus(ctx, repoDir, git)
		if err != nil {
			b.Fatalf("after sample %d: %v", i+1, err)
		}
		afterJSON, err := json.Marshal(after)
		if err != nil {
			b.Fatal(err)
		}
		if beforeFingerprint != afterFingerprint || len(beforeJSON) != len(afterJSON) || statusEntryCount(repoFilesStatusProto(after.Files)) != entries {
			b.Fatalf("worktree changed during sample %d; refusing mixed-state comparison", i+1)
		}
		freshNS += fresh.durationNS
		cachedNS += cached.durationNS
		freshPages += int64(fresh.pages)
		cachedPages += int64(cached.pages)
		freshBytes += int64(fresh.aggregateBytes)
		cachedBytes += int64(cached.aggregateBytes)
		freshMax = max(freshMax, int64(fresh.maxPageBytes))
		cachedMax = max(cachedMax, int64(cached.maxPageBytes))
		_, resident := cache.Usage()
		residentMax = max(residentMax, int64(resident))
		b.Logf("stable sample %d/%d fingerprint-sha256=%s pages=%d fresh/cached-assemblies=%d/1 bytes=%d/%d max-page=%d/%d B cache-resident=%d B",
			i+1, b.N, beforeFingerprint, fresh.pages, fresh.assemblies,
			fresh.aggregateBytes, cached.aggregateBytes, fresh.maxPageBytes, cached.maxPageBytes, resident)
	}
	b.ReportMetric(float64(freshNS)/float64(b.N)/1e6, "fresh-traversal-ms/op")
	b.ReportMetric(float64(cachedNS)/float64(b.N)/1e6, "cached-traversal-ms/op")
	b.ReportMetric(float64(freshPages)/float64(b.N), "fresh-pages/op")
	b.ReportMetric(float64(cachedPages)/float64(b.N), "cached-pages/op")
	b.ReportMetric(float64(freshBytes)/float64(b.N), "fresh-aggregate-B/op")
	b.ReportMetric(float64(cachedBytes)/float64(b.N), "cached-aggregate-B/op")
	b.ReportMetric(float64(freshMax), "fresh-max-page-B/op")
	b.ReportMetric(float64(cachedMax), "cached-max-page-B/op")
	b.ReportMetric(float64(residentMax), "cached-resident-B")
}

// BenchmarkRepoStatusGCTComposition is a disposable fallback matching the
// current read-only GCT status category counts without copying its paths or
// contents. It uses the same page size, baseline/cache pair, exact membership
// checks and response limits as the matrix benchmark.
func BenchmarkRepoStatusGCTComposition(b *testing.B) {
	const tracked = 75908
	const staged = 149
	const unstaged = 6006
	const untracked = 2215
	const stagedAndUnstaged = 5
	repoDir, expected := makeRepoStatusGCTCompositionFixture(b, repoStatusGCTComposition{tracked: tracked, staged: staged, unstaged: unstaged, untracked: untracked, stagedAndUnstaged: stagedAndUnstaged})
	fixtureBytes, err := directoryRegularFileBytes(repoDir)
	if err != nil {
		b.Fatal(err)
	}
	git := &ExecGitRunner{GitPath: "git"}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cache := NewRepoStatusPageCache(repoStatusPageCacheMaxBytes, repoStatusPageCacheMaxEntries)
	wantEntries := staged + unstaged + untracked
	var compareOrder uint64
	var freshNS, cachedNS, freshBytes, cachedBytes, freshPages, cachedPages, freshAssemblies, cachedAssemblies int64
	var freshMax, cachedMax, residentMax int64
	for i := 0; i < b.N; i++ {
		var fresh, cached repoStatusPageMeasurement
		compareOrder++
		if compareOrder%2 == 0 {
			cached = measureRepoStatusPageTraversal(ctx, repoDir, git, wantEntries, cache)
			fresh = measureRepoStatusPageTraversal(ctx, repoDir, git, wantEntries, nil)
		} else {
			fresh = measureRepoStatusPageTraversal(ctx, repoDir, git, wantEntries, nil)
			cached = measureRepoStatusPageTraversal(ctx, repoDir, git, wantEntries, cache)
		}
		if !reflect.DeepEqual(fresh.paths, cached.paths) || !sameCategoryPathSets(fresh.paths, expected) {
			b.Fatalf("fresh/cached traversal differs from exact staged/unstaged/untracked fixture membership: fresh=%s cached=%s expected=%s", categoryPathCounts(fresh.paths), categoryPathCounts(cached.paths), categoryPathCounts(expected))
		}
		if fresh.pages != cached.pages || fresh.returnedEntries != wantEntries || cached.returnedEntries != wantEntries {
			b.Fatalf("incomplete category traversal: pages=%d/%d entries=%d/%d want=%d", fresh.pages, cached.pages, fresh.returnedEntries, cached.returnedEntries, wantEntries)
		}
		if fresh.maxPageBytes > repoStatusResponseBudget || cached.maxPageBytes > repoStatusResponseBudget {
			b.Fatalf("page exceeded byte budget: fresh=%d cached=%d cap=%d", fresh.maxPageBytes, cached.maxPageBytes, repoStatusResponseBudget)
		}
		freshNS += fresh.durationNS
		cachedNS += cached.durationNS
		freshBytes += int64(fresh.aggregateBytes)
		cachedBytes += int64(cached.aggregateBytes)
		freshPages += int64(fresh.pages)
		cachedPages += int64(cached.pages)
		freshAssemblies += int64(fresh.assemblies)
		cachedAssemblies += int64(cached.assemblies)
		freshMax = max(freshMax, int64(fresh.maxPageBytes))
		cachedMax = max(cachedMax, int64(cached.maxPageBytes))
		_, resident := cache.Usage()
		residentMax = max(residentMax, int64(resident))
	}
	b.ReportMetric(float64(freshNS)/float64(b.N)/1e6, "fresh-traversal-ms/op")
	b.ReportMetric(float64(cachedNS)/float64(b.N)/1e6, "cached-traversal-ms/op")
	b.ReportMetric(float64(freshPages)/float64(b.N), "fresh-pages/op")
	b.ReportMetric(float64(cachedPages)/float64(b.N), "cached-pages/op")
	b.ReportMetric(float64(freshAssemblies)/float64(b.N), "fresh-assemblies/op")
	b.ReportMetric(float64(cachedAssemblies)/float64(b.N), "cached-assemblies/op")
	b.ReportMetric(float64(freshBytes)/float64(b.N), "fresh-aggregate-B/op")
	b.ReportMetric(float64(cachedBytes)/float64(b.N), "cached-aggregate-B/op")
	b.ReportMetric(float64(freshMax), "fresh-max-page-B/op")
	b.ReportMetric(float64(cachedMax), "cached-max-page-B/op")
	b.ReportMetric(float64(residentMax), "cached-resident-B")
	b.Logf("disposable GCT composition: tracked-base=%d staged=%d unstaged=%d staged-and-unstaged=%d untracked=%d fixture-regular-file-bytes=%d", tracked, staged, unstaged, stagedAndUnstaged, untracked, fixtureBytes)
}

func directoryRegularFileBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total, err
}

type repoStatusGCTComposition struct {
	tracked           int
	staged            int
	unstaged          int
	untracked         int
	stagedAndUnstaged int
}

func makeRepoStatusGCTCompositionFixture(b *testing.B, composition repoStatusGCTComposition) (string, map[string][]string) {
	b.Helper()
	tracked, staged, unstaged, untracked, overlap := composition.tracked, composition.staged, composition.unstaged, composition.untracked, composition.stagedAndUnstaged
	if tracked < staged+unstaged-overlap || overlap < 0 || overlap > min(staged, unstaged) {
		b.Fatalf("invalid tracked fixture base/category composition: base=%d staged=%d unstaged=%d overlap=%d", tracked, staged, unstaged, overlap)
	}
	repoDir := b.TempDir()
	mustGitBench(b, repoDir, "init", "-q")
	mustGitBench(b, repoDir, "config", "user.name", "GCT real composition fixture")
	mustGitBench(b, repoDir, "config", "user.email", "gct-real-composition@example.invalid")
	for i := 0; i < tracked; i++ {
		name := filepath.Join(repoDir, fmt.Sprintf("tracked-%06d.txt", i))
		if err := os.WriteFile(name, []byte("baseline\n"), 0600); err != nil {
			b.Fatal(err)
		}
	}
	mustGitBench(b, repoDir, "add", ".")
	mustGitBench(b, repoDir, "commit", "-qm", "GCT changed-category baseline")
	stageNames := make([]string, 0, staged)
	unstagedNames := make([]string, 0, unstaged)
	expected := map[string][]string{"staged": {}, "unstaged": {}, "untracked": {}, "conflicts": {}, "ignored": {}}
	for i := 0; i < staged; i++ {
		name := fmt.Sprintf("tracked-%06d.txt", i)
		writeGCTCompositionFile(b, repoDir, name, "baseline\nstaged change\n")
		stageNames = append(stageNames, name)
		expected["staged"] = append(expected["staged"], name)
	}
	if len(stageNames) > 0 {
		mustGitBench(b, repoDir, append([]string{"add", "--"}, stageNames...)...)
	}
	for i := staged - overlap; i < staged; i++ {
		name := fmt.Sprintf("tracked-%06d.txt", i)
		writeGCTCompositionFile(b, repoDir, name, "baseline\nstaged change\nunstaged change\n")
		unstagedNames = append(unstagedNames, name)
		expected["unstaged"] = append(expected["unstaged"], name)
	}
	for i := staged; i < staged+unstaged-overlap; i++ {
		name := fmt.Sprintf("tracked-%06d.txt", i)
		writeGCTCompositionFile(b, repoDir, name, "baseline\nunstaged change\n")
		unstagedNames = append(unstagedNames, name)
		expected["unstaged"] = append(expected["unstaged"], name)
	}
	for i := 0; i < untracked; i++ {
		name := fmt.Sprintf("untracked-%06d.txt", i)
		writeGCTCompositionFile(b, repoDir, name, "untracked\n")
		expected["untracked"] = append(expected["untracked"], name)
	}
	return repoDir, expected
}

func writeGCTCompositionFile(b *testing.B, repoDir, name, content string) {
	b.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, filepath.FromSlash(name)), []byte(content), 0600); err != nil {
		b.Fatal(err)
	}
}

func categoryPathCounts(paths map[string][]string) string {
	categories := make([]string, 0, len(paths))
	for category := range paths {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	parts := make([]string, 0, len(categories))
	for _, category := range categories {
		parts = append(parts, fmt.Sprintf("%s=%d", category, len(paths[category])))
	}
	return fmt.Sprint(parts)
}

type realStatusPageWalk struct {
	durationNS     int64
	pages          int
	aggregateBytes int
	maxPageBytes   int
	assemblies     int
	paths          map[string][]string
}

func walkRealStatusPages(ctx context.Context, repoDir string, git *ExecGitRunner, cache *RepoStatusPageCache, expectedEntries int) realStatusPageWalk {
	start := time.Now()
	out := realStatusPageWalk{paths: map[string][]string{}}
	token, snapshot := "", ""
	for {
		var status *RepoStatus
		var err error
		if cache == nil || out.pages == 0 {
			status, err = GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repoDir, StatusCache: NewRepoStatusCache(0)})
			if err != nil {
				panic(err)
			}
			out.assemblies++
		}
		identity, err := repo.NewService(newWorktreeInspector()).GetRepoStatus(ctx, repoDir)
		if err != nil {
			panic(err)
		}
		var page *repov1.GetRepoStatusResponse
		if cache == nil {
			page, err = boundRepoStatusResponse(benchmarkStatusResponse(status, identity), repoDir, 512, token)
		} else if token == "" {
			page, err = cache.Store(ctx, repoDir, benchmarkStatusResponse(status, identity), 512)
		} else {
			page, err = cache.Continue(ctx, repoDir, token, 512)
		}
		if err != nil {
			panic(err)
		}
		metadata := page.GetPage()
		if metadata == nil || int(metadata.GetTotalEntries()) != expectedEntries || int(metadata.GetResponseBytes()) > repoStatusResponseBudget {
			panic(fmt.Sprintf("invalid page metadata: total=%d want=%d bytes=%d", metadata.GetTotalEntries(), expectedEntries, metadata.GetResponseBytes()))
		}
		if snapshot == "" {
			snapshot = metadata.GetSnapshotId()
		} else if metadata.GetSnapshotId() != snapshot {
			panic("snapshot changed during measured traversal")
		}
		out.pages++
		out.aggregateBytes += int(metadata.GetResponseBytes())
		out.maxPageBytes = max(out.maxPageBytes, int(metadata.GetResponseBytes()))
		for category, paths := range map[string][]string{
			"staged": page.GetFiles().GetStaged(), "unstaged": page.GetFiles().GetUnstaged(),
			"untracked": page.GetFiles().GetUntracked(), "conflicts": page.GetFiles().GetConflicts(),
			"ignored": page.GetFiles().GetIgnored(),
		} {
			out.paths[category] = append(out.paths[category], paths...)
		}
		token = metadata.GetNextPageToken()
		if metadata.GetComplete() {
			break
		}
		if token == "" {
			panic("incomplete status page lacks continuation token")
		}
	}
	for category, paths := range out.paths {
		seen := make(map[string]struct{}, len(paths))
		for _, path := range paths {
			if _, ok := seen[path]; ok {
				panic(fmt.Sprintf("duplicate %s path in measured pages", category))
			}
			seen[path] = struct{}{}
		}
	}
	if out.aggregateBytes == 0 || out.maxPageBytes > repoStatusResponseBudget {
		panic("empty or oversized status traversal")
	}
	out.durationNS = time.Since(start).Nanoseconds()
	return out
}

func fingerprintRepoStatus(ctx context.Context, repoDir string, git *ExecGitRunner) (*RepoStatus, string, error) {
	status, err := GetRepoStatus(ctx, RepoStatusDeps{Git: git, RepoDir: repoDir, StatusCache: NewRepoStatusCache(0)})
	if err != nil {
		return nil, "", err
	}
	semantic, err := proto.MarshalOptions{Deterministic: true}.Marshal(benchmarkStatusResponse(status))
	if err != nil {
		return nil, "", err
	}
	h := sha256.New()
	_, _ = h.Write(semantic)
	_, _ = h.Write(status.statusPorcelain)
	return status, hex.EncodeToString(h.Sum(nil)), nil
}
