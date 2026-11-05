package hostreqkit

import (
	"io/fs"
	"os"
	"testing"
)

// TestCompareFileContentDistinguishesUnreadableFromDiffering is the regression
// guard for a live false blocker: /etc/sudoers.d drop-ins are 0440 root:root,
// so an unprivileged read returns a permission error. FileContentMatches
// reported that identically to a content mismatch, which made a required
// safeguard report "missing or stale" on a host where the grant was correct.
func TestCompareFileContentDistinguishesUnreadableFromDiffering(t *testing.T) {
	original := ReadFileFn
	t.Cleanup(func() { ReadFileFn = original })

	cases := []struct {
		name string
		read func(string) ([]byte, error)
		want FileComparison
	}{
		{"match", func(string) ([]byte, error) { return []byte("want"), nil }, FileComparisonMatch},
		{"differs", func(string) ([]byte, error) { return []byte("other"), nil }, FileComparisonDiffers},
		{"absent", func(string) ([]byte, error) { return nil, fs.ErrNotExist }, FileComparisonAbsent},
		{"unreadable", func(string) ([]byte, error) { return nil, fs.ErrPermission }, FileComparisonUnreadable},
		{
			"unreadable wrapped in PathError",
			func(path string) ([]byte, error) {
				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrPermission}
			},
			FileComparisonUnreadable,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ReadFileFn = testCase.read
			if got := CompareFileContent("/etc/sudoers.d/example", "want"); got != testCase.want {
				t.Fatalf("CompareFileContent = %q, want %q", got, testCase.want)
			}
		})
	}
}

// TestCompareFileContentOnRealUnreadableFile proves the permission branch fires
// against the filesystem rather than only against an injected error. Root reads
// every file regardless of mode, so the expected outcome is asserted per
// effective uid instead of skipping.
func TestCompareFileContentOnRealUnreadableFile(t *testing.T) {
	path := t.TempDir() + "/grant"
	if err := os.WriteFile(path, []byte("want"), 0o200); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	want := FileComparisonUnreadable
	if os.Geteuid() == 0 {
		want = FileComparisonMatch
	}
	if got := CompareFileContent(path, "want"); got != want {
		t.Fatalf("CompareFileContent on write-only file as uid %d = %q, want %q", os.Geteuid(), got, want)
	}
}
