package hub

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func sumOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func writeSource(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// "fresh" means the bytes were compared and match. A file that was only
// measured, because the hash budget of this read ran out, is "unknown": a size
// match proves nothing about the content.
func TestCompareSource_ASizeMatchIsNeverReportedFresh(t *testing.T) {
	dir := t.TempDir()
	path := writeSource(t, dir, "plan.md", "abcd")
	s := &Service{}

	budget := int64(driftHashBudget)
	require.Equal(t, SourceFresh, s.compareSource(path, sumOf("abcd"), 4, &budget), "hashed and equal")
	require.Equal(t, SourceChanged, s.compareSource(path, sumOf("abce"), 4, &budget), "hashed and different")
	spent := budget
	require.Equal(t, SourceFresh, s.compareSource(path, sumOf("abcd"), 4, &budget))
	require.Equal(t, spent, budget, "a repeat of an unchanged file is remembered and costs no budget")

	exhausted := int64(0)
	other := writeSource(t, dir, "other.md", "wxyz")
	require.Equal(t, SourceUnknown, s.compareSource(other, sumOf("wxyz"), 4, &exhausted), "same size, not hashed: nothing was compared")
	require.Equal(t, SourceChanged, s.compareSource(other, sumOf("wxyz"), 5, &exhausted), "a different size is decided without hashing")
	require.Equal(t, SourceMissing, s.compareSource(filepath.Join(dir, "gone.md"), sumOf("x"), 1, &exhausted))
}

func TestSourceStatus_AUnhashedFileIsUnknownWithoutACheckTimeAndAChangeDecidesAFolder(t *testing.T) {
	dir := t.TempDir()
	writeSource(t, dir, "report/a.md", "aaaa")
	writeSource(t, dir, "report/b.md", "bbbb")
	file := writeSource(t, dir, "one.md", "oooo")
	s := &Service{}
	host := "this-host"
	const checkedAt = "2026-11-13T09:00:00Z"
	folder := AskAttachment{Kind: AttachmentFolder, SourcePath: filepath.Join(dir, "report"), SourceHost: host, Entries: []AskAttachmentEntry{
		{Path: "a.md", SHA256: sumOf("aaaa"), SizeBytes: 4}, {Path: "b.md", SHA256: sumOf("bbbb"), SizeBytes: 4},
	}}
	one := AskAttachment{Kind: AttachmentFile, SourcePath: file, SourceHost: host, SHA256: sumOf("oooo"), SizeBytes: 4}

	budget := int64(0)
	status, at := s.sourceStatus(one, host, checkedAt, &budget)
	require.Equal(t, SourceUnknown, status)
	require.Empty(t, at, "an unchecked source has no check time")
	status, at = s.sourceStatus(folder, host, checkedAt, &budget)
	require.Equal(t, SourceUnknown, status, "no entry could be compared")
	require.Empty(t, at)

	// One entry that differs in size settles the folder, though the other could not be hashed.
	folder.Entries[1].SizeBytes = 5
	status, at = s.sourceStatus(folder, host, checkedAt, &budget)
	require.Equal(t, SourceChanged, status)
	require.Equal(t, checkedAt, at)

	folder.Entries[1].SizeBytes = 4
	budget = driftHashBudget
	status, at = s.sourceStatus(folder, host, checkedAt, &budget)
	require.Equal(t, SourceFresh, status)
	require.Equal(t, checkedAt, at)
}

func TestHashFile_ReadsAtMostItsLimit(t *testing.T) {
	path := writeSource(t, t.TempDir(), "grown.bin", strings.Repeat("x", 100))

	sum, err := hashFile(path, 100)
	require.NoError(t, err)
	require.Equal(t, sumOf(strings.Repeat("x", 100)), sum)

	_, err = hashFile(path, 99)
	require.ErrorIs(t, err, errHashLimit, "a file that grew past the limit is refused, not read to the end")
}
