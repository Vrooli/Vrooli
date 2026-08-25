package file_preview

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	filepreviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/file_preview"
)

func TestValidationAndFlags(t *testing.T) {
	h := &handlers{}
	empty := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "session"}, {Name: "path"}, {Name: "preview-id"}, {Name: "sort"}, {Name: "page-size"}, {Name: "page-token"}, {Name: "show-hidden", Bool: true},
	}}})
	for _, call := range []func(cliapp.RunContext) error{h.resolve, h.text, h.list} {
		if err := call(empty); err == nil {
			t.Fatal("missing required flags unexpectedly succeeded")
		}
	}
	for _, raw := range []string{"", "dirs_first_name", "name", "size_desc", "mtime_desc"} {
		if _, err := parseSortFlag(raw); err != nil {
			t.Fatalf("parseSortFlag(%q): %v", raw, err)
		}
	}
	if _, err := parseSortFlag("bad"); err == nil {
		t.Fatal("invalid sort unexpectedly succeeded")
	}
	if _, err := parsePageSizeFlag("12"); err != nil {
		t.Fatal(err)
	}
	if _, err := parsePageSizeFlag("-1"); err == nil {
		t.Fatal("negative page size unexpectedly succeeded")
	}
	if _, err := parsePageSizeFlag("bad"); err == nil {
		t.Fatal("invalid page size unexpectedly succeeded")
	}
	if got := formatEntry(&filepreviewv1.DirectoryEntry{Name: "dir", EntryType: filepreviewv1.EntryType_ENTRY_TYPE_DIRECTORY, ChildCount: 3}); got == "" {
		t.Fatal("directory entry was empty")
	}
	if got := formatEntry(&filepreviewv1.DirectoryEntry{Name: "link", EntryType: filepreviewv1.EntryType_ENTRY_TYPE_SYMLINK, SymlinkTarget: "/tmp", SymlinkBroken: true}); got == "" {
		t.Fatal("symlink entry was empty")
	}
	if kindLabel(filepreviewv1.PreviewKind_PREVIEW_KIND_MARKDOWN) != "markdown" {
		t.Fatal("kind label mismatch")
	}
	if sortLabel(filepreviewv1.DirectorySort_DIRECTORY_SORT_NAME) != "name" {
		t.Fatal("sort label mismatch")
	}
}
