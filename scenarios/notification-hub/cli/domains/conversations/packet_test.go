package conversations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func TestParseAttachSpec(t *testing.T) {
	dir := t.TempDir()
	odd := writeFile(t, dir, "a=b.txt", "x")

	spec, err := parseAttachSpec("report.pdf=Final report")
	require.NoError(t, err)
	require.Equal(t, attachSpec{Path: "report.pdf", Label: "Final report"}, spec)

	spec, err = parseAttachSpec("report.pdf")
	require.NoError(t, err)
	require.Equal(t, attachSpec{Path: "report.pdf"}, spec)

	spec, err = parseAttachSpec(odd)
	require.NoError(t, err)
	require.Equal(t, attachSpec{Path: odd}, spec, "an existing path wins over label splitting")

	for _, bad := range []string{"", "  ", "=label"} {
		_, err := parseAttachSpec(bad)
		require.Error(t, err, "%q", bad)
	}
}

func TestSanitizeAttachmentNameFollowsTheHubNameRule(t *testing.T) {
	valid := func(name string) bool {
		if name == "" || len(name) > maxAttachmentNameLength || !(name[0] >= 'a' && name[0] <= 'z' || name[0] >= 'A' && name[0] <= 'Z' || name[0] >= '0' && name[0] <= '9') {
			return false
		}
		for _, r := range name {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
				return false
			}
		}
		return true
	}
	for base, want := range map[string]string{
		"shot.png":            "shot.png",
		"My Shot (final).PNG": "My-Shot-final.PNG",
		".bashrc":             "bashrc",
		"café.png":            "caf.png",
		"日本語":                 "attachment",
		"a   b":               "a-b",
		"v1.2.tar.gz":         "v1.2.tar.gz",
	} {
		stem, ext := sanitizeAttachmentName(base, false)
		require.Equal(t, want, stem+ext, base)
		require.True(t, valid(stem+ext), base)
	}
	stem, ext := sanitizeAttachmentName("release v1.2", true)
	require.Equal(t, "release-v1.2", stem+ext, "folders have no extension")

	long := strings.Repeat("x", 300) + ".png"
	stem, ext = sanitizeAttachmentName(long, false)
	require.LessOrEqual(t, len(stem+ext), maxAttachmentNameLength)
	require.Equal(t, ".png", ext)
}

func TestUniqueNameNumbersDuplicatesBeforeTheExtension(t *testing.T) {
	p := newPacket()
	got := []string{
		p.uniqueName("a.png", false), p.uniqueName("a.png", false), p.uniqueName("a.png", false),
		p.uniqueName("docs", true), p.uniqueName("docs", true),
	}
	require.Equal(t, []string{"a.png", "a-2.png", "a-3.png", "docs", "docs-2"}, got)

	long := strings.Repeat("y", 200) + ".png"
	first, second := p.uniqueName(long, false), p.uniqueName(long, false)
	require.NotEqual(t, first, second)
	require.LessOrEqual(t, len(second), maxAttachmentNameLength)
	require.True(t, strings.HasSuffix(second, "-2.png"))
}

func TestAddAttachmentNamesSameNamedFilesFromDifferentFolders(t *testing.T) {
	dir := t.TempDir()
	one := writeFile(t, dir, "one/a.png", "1")
	two := writeFile(t, dir, "two/a.png", "2")
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: one}, v1.AskAttachment_ROLE_EVIDENCE))
	require.NoError(t, p.addAttachment(attachSpec{Path: two}, v1.AskAttachment_ROLE_EVIDENCE))
	require.Equal(t, "a.png", p.attachments[0].Name)
	require.Equal(t, "a-2.png", p.attachments[1].Name)
	require.Equal(t, one, p.attachments[0].SourcePath)
}

func TestFolderWalkListsRelativeSlashPathsAndSkipsJunk(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "pkg")
	writeFile(t, root, "b.txt", "b")
	writeFile(t, root, "a/c.txt", "c")
	writeFile(t, root, ".git/config", "ignored")
	writeFile(t, root, "node_modules/x/index.js", "ignored")
	writeFile(t, root, "a/__pycache__/m.pyc", "ignored")
	writeFile(t, root, ".DS_Store", "ignored")
	writeFile(t, root, ".env.example", "kept: dotfiles are not junk")
	outside := writeFile(t, dir, "secret.txt", "secret")
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link.txt")))
	require.NoError(t, os.Symlink(filepath.Dir(outside), filepath.Join(root, "linkdir")))

	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: root}, v1.AskAttachment_ROLE_EVIDENCE))
	att := p.attachments[0]
	var rels []string
	for _, entry := range att.Entries {
		rels = append(rels, entry.Rel)
		require.NotEmpty(t, entry.SHA256)
		require.NotEmpty(t, entry.MediaType)
	}
	require.Equal(t, []string{".env.example", "a/c.txt", "b.txt"}, rels)
	require.False(t, att.Truncated)
	require.Empty(t, p.Warnings)
}

func TestFolderWalkFollowsAFolderSymlinkNamedOnTheCommandLine(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real/a.txt", "a")
	require.NoError(t, os.Symlink(filepath.Join(dir, "real"), filepath.Join(dir, "alias")))
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: filepath.Join(dir, "alias")}, v1.AskAttachment_ROLE_EVIDENCE))
	require.Len(t, p.attachments[0].Entries, 1)
	require.Equal(t, filepath.Join(dir, "alias"), p.attachments[0].SourcePath)
}

func TestFolderWalkCapsAtFiveHundredFilesAndSaysSo(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < maxFolderFiles+1; i++ {
		writeFile(t, dir, "f/"+strings.Repeat("0", 4-len(itoa(i)))+itoa(i), "x")
	}
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: filepath.Join(dir, "f")}, v1.AskAttachment_ROLE_EVIDENCE))
	require.Len(t, p.attachments[0].Entries, maxFolderFiles)
	require.True(t, p.attachments[0].Truncated)
	require.Len(t, p.Warnings, 1)

	exact := t.TempDir()
	for i := 0; i < maxFolderFiles; i++ {
		writeFile(t, exact, strings.Repeat("0", 4-len(itoa(i)))+itoa(i), "x")
	}
	p = newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: exact}, v1.AskAttachment_ROLE_EVIDENCE))
	require.False(t, p.attachments[0].Truncated, "exactly 500 files is not truncated")
}

func TestAddAttachmentRefusesWhatCannotBeAttached(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	require.NoError(t, os.Mkdir(empty, 0o755))
	writeFile(t, empty, ".git/HEAD", "junk only")
	for name, spec := range map[string]string{
		"missing":   filepath.Join(dir, "nope"),
		"junk-only": empty,
	} {
		err := newPacket().addAttachment(attachSpec{Path: spec}, v1.AskAttachment_ROLE_EVIDENCE)
		require.Error(t, err, name)
	}
	err := newPacket().addAttachment(attachSpec{Path: dir}, v1.AskAttachment_ROLE_EXPLAINER)
	require.ErrorContains(t, err, "give one HTML file")
}

func TestMediaTypeUsesExtensionThenSniffsContent(t *testing.T) {
	for path, want := range map[string]string{
		"plan.md": "text/markdown", "x.PNG": "image/png", "m.glb": "model/gltf-binary", "page.html": "text/html",
	} {
		require.Equal(t, want, mediaTypeFor(path, nil), path)
	}
	require.Equal(t, "image/png", mediaTypeFor("noext", []byte("\x89PNG\r\n\x1a\n....")))
	require.Equal(t, "text/plain", mediaTypeFor("noext", []byte("hello")), "charset parameters are dropped")
	require.Equal(t, "application/octet-stream", mediaTypeFor("noext", []byte{0, 1, 2, 3}))
}

// sparseFile makes a file of the given size without writing its bytes.
func sparseFile(t *testing.T, dir, name string, size int64) string {
	t.Helper()
	path := writeFile(t, dir, name, "")
	require.NoError(t, os.Truncate(path, size))
	return path
}

func TestOversizeFileIsRefusedFromItsSizeWithoutBeingRead(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read an unreadable file, so the test cannot prove it was never opened")
	}
	path := sparseFile(t, t.TempDir(), "big.bin", maxAttachmentFileBytes+1)
	require.NoError(t, os.Chmod(path, 0))
	_, err := describeFile(path, maxAttachmentTotalBytes)
	var tooLarge *fileTooLargeError
	require.ErrorAs(t, err, &tooLarge, "a size check must come before any open or hash")
	require.ErrorContains(t, err, "64.0 MiB per-file limit")
}

func TestPerFileLimitAllowsExactlySixtyFourMiB(t *testing.T) {
	path := sparseFile(t, t.TempDir(), "exact.bin", maxAttachmentFileBytes)
	file, err := describeFile(path, maxAttachmentTotalBytes)
	require.NoError(t, err)
	require.Equal(t, maxAttachmentFileBytes, file.Size)
}

func TestAttachmentsAfterTheTotalLimitAreRefusedAndEarlierOnesKept(t *testing.T) {
	dir := t.TempDir()
	p := newPacket()
	for i := 0; i < 4; i++ {
		path := sparseFile(t, dir, "part"+itoa(i)+".bin", maxAttachmentFileBytes)
		require.NoError(t, p.addAttachment(attachSpec{Path: path}, v1.AskAttachment_ROLE_EVIDENCE), "exactly 256 MiB in total is allowed")
	}
	err := p.addAttachment(attachSpec{Path: writeFile(t, dir, "one-more.txt", "x")}, v1.AskAttachment_ROLE_EVIDENCE)
	require.ErrorContains(t, err, "over the 256.0 MiB total limit")
	require.Len(t, p.attachments, 4)
	require.Equal(t, int64(4)*maxAttachmentFileBytes, p.total)
}

func TestFolderSkipsAnOversizeFileAndKeepsTheRest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "f/a.txt", "a")
	sparseFile(t, dir, "f/huge.bin", maxAttachmentFileBytes+1)
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: filepath.Join(dir, "f")}, v1.AskAttachment_ROLE_EVIDENCE))
	require.Len(t, p.attachments[0].Entries, 1)
	require.Len(t, p.Warnings, 1)
	require.Contains(t, p.Warnings[0], "Left out huge.bin")
	require.Contains(t, p.Warnings[0], "per-file limit")
}

func TestFolderThatDoesNotFitTheTotalIsLeftOutWhole(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		sparseFile(t, dir, "f/part"+itoa(i)+".bin", maxAttachmentFileBytes)
	}
	p := newPacket()
	err := p.addAttachment(attachSpec{Path: filepath.Join(dir, "f")}, v1.AskAttachment_ROLE_EVIDENCE)
	require.ErrorContains(t, err, "total limit")
	require.Empty(t, p.attachments)
	require.Zero(t, p.total, "a refused folder must not use up the budget")
}

func TestResolveRefAcceptsNamesFolderEntriesAndPaths(t *testing.T) {
	dir := t.TempDir()
	plan := writeFile(t, dir, "plan.md", "x")
	writeFile(t, dir, "shots/a.png", "a")
	p := newPacket()
	require.NoError(t, p.addAttachment(attachSpec{Path: plan}, v1.AskAttachment_ROLE_EVIDENCE))
	require.NoError(t, p.addAttachment(attachSpec{Path: filepath.Join(dir, "shots")}, v1.AskAttachment_ROLE_EVIDENCE))

	for ref, want := range map[string]string{
		"plan.md":                            "plan.md",
		plan:                                 "plan.md",
		"shots":                              "shots",
		"shots/a.png":                        "shots/a.png",
		filepath.Join(dir, "shots", "a.png"): "shots/a.png",
		filepath.Join(dir, "shots"):          "shots",
	} {
		got, ok := p.resolveRef(ref)
		require.True(t, ok, ref)
		require.Equal(t, want, got, ref)
	}
	for _, ref := range []string{"", "other.md", "shots/missing.png", filepath.Join(dir, "elsewhere.md")} {
		_, ok := p.resolveRef(ref)
		require.False(t, ok, ref)
	}
}
