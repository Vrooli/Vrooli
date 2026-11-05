package filepreview

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

// listRoot builds a directory tree and returns its path plus a resolver.
func listRoot(t *testing.T) (string, *Resolver) {
	t.Helper()
	root := t.TempDir()
	return root, newResolver(root)
}

// mustList lists dir and fails the test on error.
func mustList(t *testing.T, r *Resolver, dir string, opts ListOptions) *ListResult {
	t.Helper()
	res, err := r.ListDirectory(dir, opts)
	if err != nil {
		t.Fatalf("list %q: %v", dir, err)
	}
	return res
}

// names extracts entry names in page order.
func names(res *ListResult) []string {
	out := make([]string, 0, len(res.Entries))
	for _, e := range res.Entries {
		out = append(out, e.Name)
	}
	return out
}

// entryByName finds a named entry, or fails.
func entryByName(t *testing.T, res *ListResult, name string) DirEntryInfo {
	t.Helper()
	for _, e := range res.Entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("entry %q not found in %v", name, names(res))
	return DirEntryInfo{}
}

func TestListDirectoryOrdersDirectoriesFirst(t *testing.T) {
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "beta.md"), "b")
	mustWrite(t, filepath.Join(root, "Alpha.md"), "a")
	mustMkdir(t, filepath.Join(root, "zeta"))
	mustMkdir(t, filepath.Join(root, "Mid"))

	res := mustList(t, r, root, ListOptions{})

	want := []string{"Mid", "zeta", "Alpha.md", "beta.md"}
	got := names(res)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if res.EffectiveSort != SortDirsFirstName {
		t.Fatalf("effectiveSort = %q", res.EffectiveSort)
	}
	if res.TotalEntries != 4 {
		t.Fatalf("totalEntries = %d, want 4", res.TotalEntries)
	}
}

func TestListDirectoryNameSortIsCaseInsensitiveButTotal(t *testing.T) {
	root, r := listRoot(t)
	for _, n := range []string{"b.txt", "A.txt", "a.txt", "B.txt"} {
		mustWrite(t, filepath.Join(root, n), "x")
	}

	res := mustList(t, r, root, ListOptions{Sort: SortName})
	got := names(res)

	// Case-insensitive groups first (a's before b's); within a group the
	// byte-wise order breaks the tie deterministically.
	want := []string{"A.txt", "a.txt", "B.txt", "b.txt"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListDirectoryHidesDotEntriesByDefault(t *testing.T) {
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "visible.txt"), "v")
	mustWrite(t, filepath.Join(root, ".env"), "SECRET=1")
	mustMkdir(t, filepath.Join(root, ".git"))

	res := mustList(t, r, root, ListOptions{})
	if got := names(res); len(got) != 1 || got[0] != "visible.txt" {
		t.Fatalf("default listing = %v, want [visible.txt]", got)
	}
	if res.TotalEntries != 1 {
		t.Fatalf("totalEntries = %d, want 1 (hidden entries are filtered before paging)", res.TotalEntries)
	}

	res = mustList(t, r, root, ListOptions{ShowHidden: true})
	if len(res.Entries) != 3 {
		t.Fatalf("show-hidden listing = %v, want 3 entries", names(res))
	}
}

func TestListDirectoryEmptyIsNotAnError(t *testing.T) {
	root, r := listRoot(t)
	res := mustList(t, r, root, ListOptions{})
	if len(res.Entries) != 0 || res.TotalEntries != 0 {
		t.Fatalf("expected an empty listing, got %v", names(res))
	}
	if res.NextPageToken != "" {
		t.Fatalf("empty listing must not offer a next page")
	}
	if res.Entries == nil {
		t.Fatalf("entries must be an empty slice, never nil, so clients need no null check")
	}
}

func TestListDirectoryPagesWithoutGapsOrOverlap(t *testing.T) {
	root, r := listRoot(t)
	const total = 25
	for i := 0; i < total; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%02d.txt", i)), "x")
	}

	seen := make([]string, 0, total)
	opts := ListOptions{PageSize: 7}
	pages := 0
	for {
		res := mustList(t, r, root, opts)
		seen = append(seen, names(res)...)
		pages++
		if res.NextPageToken == "" {
			break
		}
		if pages > 10 {
			t.Fatalf("pagination did not terminate")
		}
		opts.PageToken = res.NextPageToken
	}

	if pages != 4 {
		t.Fatalf("pages = %d, want 4 (25 entries at 7 per page)", pages)
	}
	if len(seen) != total {
		t.Fatalf("saw %d entries across pages, want %d", len(seen), total)
	}
	unique := map[string]bool{}
	for _, n := range seen {
		if unique[n] {
			t.Fatalf("entry %q appeared on more than one page", n)
		}
		unique[n] = true
	}
	for i := 0; i < total; i++ {
		if !unique[fmt.Sprintf("f%02d.txt", i)] {
			t.Fatalf("entry f%02d.txt was skipped across pages", i)
		}
	}
}

func TestListDirectoryStaleTokenIsRejected(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 10; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%02d.txt", i)), "x")
	}

	first := mustList(t, r, root, ListOptions{PageSize: 4})
	if first.NextPageToken == "" {
		t.Fatalf("expected a continuation token")
	}

	// Mutating the directory moves its mtime, which the token pins.
	waitForDistinctDirMTime(t, root, func() {
		mustWrite(t, filepath.Join(root, "late-arrival.txt"), "x")
	})

	_, err := r.ListDirectory(root, ListOptions{PageSize: 4, PageToken: first.NextPageToken})
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeStale {
		t.Fatalf("expected a stale-token error, got %v", err)
	}
}

func TestListDirectoryTokenRejectsMismatchedSortOrFilter(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 6; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%d.txt", i)), "x")
	}
	first := mustList(t, r, root, ListOptions{PageSize: 2})

	for _, tc := range []struct {
		name string
		opts ListOptions
	}{
		{"different sort", ListOptions{PageSize: 2, Sort: SortName, PageToken: first.NextPageToken}},
		{"different filter", ListOptions{PageSize: 2, ShowHidden: true, PageToken: first.NextPageToken}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.ListDirectory(root, tc.opts)
			var fe *Error
			if !errors.As(err, &fe) || fe.Code != CodeInvalid {
				t.Fatalf("expected an invalid-token error, got %v", err)
			}
		})
	}
}

func TestListDirectoryRejectsMalformedToken(t *testing.T) {
	root, r := listRoot(t)
	_, err := r.ListDirectory(root, ListOptions{PageToken: "not-a-real-token!!"})
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeInvalid {
		t.Fatalf("expected an invalid-token error, got %v", err)
	}
}

func TestListDirectoryTokenIsBoundToItsDirectory(t *testing.T) {
	root, r := listRoot(t)
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")
	mustMkdir(t, a)
	mustMkdir(t, b)
	for i := 0; i < 6; i++ {
		mustWrite(t, filepath.Join(a, fmt.Sprintf("f%d.txt", i)), "x")
		mustWrite(t, filepath.Join(b, fmt.Sprintf("f%d.txt", i)), "x")
	}

	first := mustList(t, r, a, ListOptions{PageSize: 2})
	_, err := r.ListDirectory(b, ListOptions{PageSize: 2, PageToken: first.NextPageToken})
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeInvalid {
		t.Fatalf("a token from one directory must not continue another, got %v", err)
	}
}

func TestListDirectoryClampsPageSize(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 3; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%d.txt", i)), "x")
	}
	// Oversized requests are clamped, not rejected.
	res := mustList(t, r, root, ListOptions{PageSize: MaxListPageSize * 10})
	if len(res.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(res.Entries))
	}
}

func TestListDirectoryClassifiesEntryKinds(t *testing.T) {
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "notes.md"), "# hi")
	mustWrite(t, filepath.Join(root, "main.go"), "package main")
	mustWrite(t, filepath.Join(root, "photo.png"), "\x89PNG")
	mustWrite(t, filepath.Join(root, "mystery.zzz"), "?")
	mustMkdir(t, filepath.Join(root, "sub"))

	res := mustList(t, r, root, ListOptions{})

	for name, want := range map[string]Kind{
		"notes.md":  KindMarkdown,
		"main.go":   KindCode,
		"photo.png": KindImage,
		"sub":       KindDirectory,
	} {
		if got := entryByName(t, res, name).Kind; got != want {
			t.Fatalf("%s kind = %q, want %q", name, got, want)
		}
	}

	// An unmapped extension stays undetermined rather than being asserted as
	// unsupported: the resolve that follows a click is what sniffs content.
	if got := entryByName(t, res, "mystery.zzz").Kind; got != "" {
		t.Fatalf("unmapped extension kind = %q, want empty (determined on open)", got)
	}
	if !entryByName(t, res, "mystery.zzz").CanPreview {
		t.Fatalf("an unclassified regular file must still be openable")
	}
}

func TestListDirectoryReportsEntryTypesAndMetadata(t *testing.T) {
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "data.txt"), "hello world")
	mustMkdir(t, filepath.Join(root, "sub"))

	res := mustList(t, r, root, ListOptions{})

	file := entryByName(t, res, "data.txt")
	if file.Type != EntryTypeFile {
		t.Fatalf("file type = %q", file.Type)
	}
	if file.SizeBytes != int64(len("hello world")) {
		t.Fatalf("file size = %d, want %d", file.SizeBytes, len("hello world"))
	}
	if file.ModTimeUnixNano == 0 {
		t.Fatalf("file mtime must be populated")
	}
	if file.Mode == "" {
		t.Fatalf("file mode must be populated")
	}

	dir := entryByName(t, res, "sub")
	if dir.Type != EntryTypeDirectory {
		t.Fatalf("dir type = %q", dir.Type)
	}
}

func TestListDirectoryCountsChildren(t *testing.T) {
	root, r := listRoot(t)
	sub := filepath.Join(root, "sub")
	mustMkdir(t, sub)
	for i := 0; i < 3; i++ {
		mustWrite(t, filepath.Join(sub, fmt.Sprintf("c%d.txt", i)), "x")
	}
	mustWrite(t, filepath.Join(sub, ".hidden"), "x")
	mustMkdir(t, filepath.Join(root, "empty"))

	res := mustList(t, r, root, ListOptions{})
	if got := entryByName(t, res, "sub").ChildCount; got != 3 {
		t.Fatalf("child count = %d, want 3 (the hidden child is filtered like the parent listing)", got)
	}
	if got := entryByName(t, res, "empty").ChildCount; got != 0 {
		t.Fatalf("empty dir child count = %d, want 0", got)
	}

	res = mustList(t, r, root, ListOptions{ShowHidden: true})
	if got := entryByName(t, res, "sub").ChildCount; got != 4 {
		t.Fatalf("child count with hidden = %d, want 4", got)
	}
}

func TestListDirectoryTruncatesPastScanCeiling(t *testing.T) {
	// Exercising MaxEntriesScanned directly would mean creating 50k files.
	// Assert the invariant that matters instead: the scan never returns more
	// than the ceiling, and a truncated result says so.
	root, r := listRoot(t)
	for i := 0; i < 12; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%02d.txt", i)), "x")
	}
	res := mustList(t, r, root, ListOptions{})
	if res.Truncated {
		t.Fatalf("a small directory must not report truncation")
	}
	if res.TotalEntries > MaxEntriesScanned {
		t.Fatalf("scan exceeded the ceiling")
	}

	scanned, truncated, err := scanDirectory(root, false, MaxEntriesScanned)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if truncated || len(scanned) != 12 {
		t.Fatalf("scan returned %d entries, truncated=%t", len(scanned), truncated)
	}

	// With a low ceiling the scan stops and says so.
	scanned, truncated, err = scanDirectory(root, false, 5)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !truncated || len(scanned) != 5 {
		t.Fatalf("bounded scan returned %d entries, truncated=%t; want 5/true", len(scanned), truncated)
	}
}

// The scan ceiling has to bound entries EXAMINED, not just entries kept.
// A directory full of dotfiles yields almost nothing after the hidden filter,
// so a ceiling applied only to the survivors would read the whole directory.
func TestListDirectoryCeilingBoundsHiddenEntriesToo(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 40; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf(".hidden%02d", i)), "x")
	}
	mustWrite(t, filepath.Join(root, "visible.txt"), "x")

	r.MaxEntriesScanned = 10
	res := mustList(t, r, root, ListOptions{})
	if !res.Truncated {
		t.Fatalf("a scan stopped by the ceiling must report truncation even when the hidden filter left few entries")
	}
	if len(res.Entries) > 10 {
		t.Fatalf("entries = %d, want at most the ceiling", len(res.Entries))
	}
}

// The same bound applies to counting a subdirectory's children.
func TestListDirectoryChildCountBoundsHiddenEntriesToo(t *testing.T) {
	root, r := listRoot(t)
	sub := filepath.Join(root, "sub")
	mustMkdir(t, sub)
	for i := 0; i < childCountCeiling+50; i++ {
		mustWrite(t, filepath.Join(sub, fmt.Sprintf(".h%04d", i)), "")
	}

	res := mustList(t, r, root, ListOptions{})
	if got := entryByName(t, res, "sub").ChildCount; got != ChildCountUnknown {
		t.Fatalf("child count = %d, want ChildCountUnknown once the examine budget is exceeded", got)
	}
}

// Paging must keep working when the server downgraded the sort. The client
// keeps sending the sort it asked for, so a token that recorded the *applied*
// sort would reject every page after the first.
func TestListDirectoryPagesThroughADowngradedSort(t *testing.T) {
	root, r := listRoot(t)
	const total = 12
	for i := 0; i < total; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%02d.txt", i)), "x")
	}
	// Force the downgrade path without building a 5,001-entry directory.
	r.StatSortLimit = 4

	first, err := r.ListDirectory(root, ListOptions{Sort: SortSizeDesc, PageSize: 5})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if first.EffectiveSort != SortDirsFirstName {
		t.Fatalf("effectiveSort = %q, want the downgrade", first.EffectiveSort)
	}
	if len(first.Warnings) == 0 {
		t.Fatalf("a downgrade must be reported, not silent")
	}
	if first.NextPageToken == "" {
		t.Fatalf("expected a continuation token")
	}

	// The client still says size_desc — exactly what the UI holds.
	seen := append([]string(nil), names(first)...)
	opts := ListOptions{Sort: SortSizeDesc, PageSize: 5, PageToken: first.NextPageToken}
	for {
		page, pErr := r.ListDirectory(root, opts)
		if pErr != nil {
			t.Fatalf("continuation page: %v", pErr)
		}
		seen = append(seen, names(page)...)
		if page.NextPageToken == "" {
			break
		}
		opts.PageToken = page.NextPageToken
	}
	if len(seen) != total {
		t.Fatalf("saw %d entries across pages, want %d", len(seen), total)
	}
}

func TestListDirectoryDowngradesExpensiveSortAboveThreshold(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 5; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%d.txt", i)), "x")
	}

	// Below the threshold the requested sort is honoured.
	res := mustList(t, r, root, ListOptions{Sort: SortSizeDesc})
	if res.EffectiveSort != SortSizeDesc {
		t.Fatalf("effectiveSort = %q, want %q", res.EffectiveSort, SortSizeDesc)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", res.Warnings)
	}
}

func TestListDirectorySortsBySizeAndTime(t *testing.T) {
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "small.txt"), "x")
	mustWrite(t, filepath.Join(root, "large.txt"), "xxxxxxxxxxxxxxxxxxxx")
	mustWrite(t, filepath.Join(root, "medium.txt"), "xxxxxxxxxx")

	res := mustList(t, r, root, ListOptions{Sort: SortSizeDesc})
	want := []string{"large.txt", "medium.txt", "small.txt"}
	got := names(res)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("size sort = %v, want %v", got, want)
		}
	}

	// Newest first. Set explicit times so the assertion does not depend on
	// filesystem timestamp granularity.
	base := time.Now().Add(-time.Hour)
	mustChtimes(t, filepath.Join(root, "small.txt"), base)
	mustChtimes(t, filepath.Join(root, "medium.txt"), base.Add(30*time.Minute))
	mustChtimes(t, filepath.Join(root, "large.txt"), base.Add(45*time.Minute))

	res = mustList(t, r, root, ListOptions{Sort: SortMTimeDesc})
	want = []string{"large.txt", "medium.txt", "small.txt"}
	got = names(res)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mtime sort = %v, want %v", got, want)
		}
	}
}

func TestListDirectoryDescribesSymlinks(t *testing.T) {
	requireSymlinks(t)
	root, r := listRoot(t)
	mustWrite(t, filepath.Join(root, "real.png"), "\x89PNG")
	mustMkdir(t, filepath.Join(root, "realdir"))
	mustSymlink(t, "real.png", filepath.Join(root, "link-to-file"))
	mustSymlink(t, "realdir", filepath.Join(root, "link-to-dir"))
	mustSymlink(t, "nowhere", filepath.Join(root, "link-broken"))

	res := mustList(t, r, root, ListOptions{})

	toFile := entryByName(t, res, "link-to-file")
	if toFile.Type != EntryTypeSymlink {
		t.Fatalf("type = %q, want symlink", toFile.Type)
	}
	if toFile.SymlinkTarget != "real.png" {
		t.Fatalf("target = %q, want real.png", toFile.SymlinkTarget)
	}
	if toFile.SymlinkBroken || !toFile.CanPreview {
		t.Fatalf("a resolving link must be previewable")
	}

	toDir := entryByName(t, res, "link-to-dir")
	if toDir.Kind != KindDirectory {
		t.Fatalf("a link to a directory must open as a directory, got kind %q", toDir.Kind)
	}

	broken := entryByName(t, res, "link-broken")
	if !broken.SymlinkBroken {
		t.Fatalf("broken link not flagged")
	}
	if broken.CanPreview {
		t.Fatalf("a broken link must not be openable")
	}
	if broken.SymlinkTarget != "nowhere" {
		t.Fatalf("a broken link must still report its target, got %q", broken.SymlinkTarget)
	}
}

// Building a page must never read file contents. This guards against a future
// refactor reintroducing content sniffing into the listing path, which would
// turn one page into hundreds of opens.
//
// The proof is observable rather than instrumented: these files hold valid
// UTF-8 under an unmapped extension, which is exactly what classify() would
// label KindText after a sniff. Seeing no kind at all is only possible if
// nothing opened them.
func TestListDirectoryReadsNoFileContents(t *testing.T) {
	root, r := listRoot(t)
	for i := 0; i < 20; i++ {
		mustWrite(t, filepath.Join(root, fmt.Sprintf("f%02d.zzz", i)), "plain utf-8 that a sniff would classify as text")
	}

	res := mustList(t, r, root, ListOptions{})
	if len(res.Entries) != 20 {
		t.Fatalf("entries = %d, want 20", len(res.Entries))
	}
	for _, e := range res.Entries {
		if e.Kind != "" {
			t.Fatalf("entry %q was classified as %q; listing must not sniff content", e.Name, e.Kind)
		}
	}
}

// A file the process cannot open must still list — with its name, size, and
// mode intact. Anything that opened entries to classify them would fail here.
func TestListDirectoryListsUnreadableFiles(t *testing.T) {
	requireUnreadableFiles(t)
	root, r := listRoot(t)
	locked := filepath.Join(root, "locked.zzz")
	mustWrite(t, locked, "secret")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o600) })

	res := mustList(t, r, root, ListOptions{})
	e := entryByName(t, res, "locked.zzz")
	if e.Type != EntryTypeFile {
		t.Fatalf("type = %q, want file", e.Type)
	}
	if e.SizeBytes != int64(len("secret")) {
		t.Fatalf("size = %d, want %d", e.SizeBytes, len("secret"))
	}
}

func TestListDirectoryRejectsNonDirectory(t *testing.T) {
	root, r := listRoot(t)
	file := filepath.Join(root, "a.txt")
	mustWrite(t, file, "x")

	_, err := r.ListDirectory(file, ListOptions{})
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeNotPreviewable {
		t.Fatalf("expected not_previewable, got %v", err)
	}
}

func TestListDirectoryMissingDirectory(t *testing.T) {
	root, r := listRoot(t)
	_, err := r.ListDirectory(filepath.Join(root, "nope"), ListOptions{})
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeNotFound {
		t.Fatalf("expected not_found, got %v", err)
	}
}

func TestListDirectoryReportsParentPath(t *testing.T) {
	root, r := listRoot(t)
	sub := filepath.Join(root, "sub")
	mustMkdir(t, sub)

	res := mustList(t, r, sub, ListOptions{})
	if res.ParentPath != root {
		t.Fatalf("parentPath = %q, want %q", res.ParentPath, root)
	}

	// A filesystem root is its own parent, which the API reports as "".
	if got := parentPath(filepath.VolumeName(root) + string(filepath.Separator)); got != "" {
		t.Fatalf("parentPath at a filesystem root = %q, want empty", got)
	}
}

func TestNormalizeSortFallsBackToDefault(t *testing.T) {
	for _, in := range []Sort{"", "bogus", "SIZE_DESC"} {
		if got := NormalizeSort(in); got != SortDirsFirstName {
			t.Fatalf("NormalizeSort(%q) = %q, want %q", in, got, SortDirsFirstName)
		}
	}
	for _, in := range []Sort{SortName, SortSizeDesc, SortMTimeDesc, SortDirsFirstName} {
		if got := NormalizeSort(in); got != in {
			t.Fatalf("NormalizeSort(%q) = %q", in, got)
		}
	}
}

func TestPageTokenRoundTrip(t *testing.T) {
	in := pageToken{Sort: SortName, ShowHidden: true, Offset: 42, DirModTime: 1234, Version: pageTokenVersion, Path: "/tmp/x"}
	encoded, err := encodePageToken(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out, err := decodePageToken(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out != in {
		t.Fatalf("round trip = %+v, want %+v", out, in)
	}
}

func TestPageTokenRejectsUnknownVersion(t *testing.T) {
	encoded, err := encodePageToken(pageToken{Version: pageTokenVersion + 1})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, err = decodePageToken(encoded)
	var fe *Error
	if !errors.As(err, &fe) || fe.Code != CodeInvalid {
		t.Fatalf("expected an invalid-token error, got %v", err)
	}
}

// --------------------------------------------------------------------------
// helpers
// --------------------------------------------------------------------------

func mustChtimes(t *testing.T, path string, when time.Time) {
	t.Helper()
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatalf("chtimes %q: %v", path, err)
	}
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %q -> %q: %v", link, target, err)
	}
}

// requireSymlinks skips on platforms where creating a symlink needs elevated
// privileges (unprivileged Windows without developer mode).
func requireSymlinks(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		return
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "target"), filepath.Join(dir, "link")); err != nil {
		t.Skip("symlink creation is not permitted on this host")
	}
}

// requireUnreadableFiles skips where chmod cannot actually revoke read access:
// Windows ignores POSIX mode bits, and root bypasses them.
func requireUnreadableFiles(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "POSIX mode bits do not gate reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses file permissions")
	}
}

// waitForDistinctDirMTime runs mutate and ensures the directory's mtime
// actually advanced, so the stale-token assertion does not depend on the
// filesystem's timestamp granularity.
func waitForDistinctDirMTime(t *testing.T, dir string, mutate func()) {
	t.Helper()
	before, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	mutate()
	deadline := time.Now().Add(2 * time.Second)
	for {
		after, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if after.ModTime().UnixNano() != before.ModTime().UnixNano() {
			return
		}
		if time.Now().After(deadline) {
			// Coarse timestamps: force the change the guard relies on.
			bump := before.ModTime().Add(2 * time.Second)
			if err := os.Chtimes(dir, bump, bump); err != nil {
				t.Fatalf("chtimes: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
