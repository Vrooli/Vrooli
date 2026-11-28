package hub_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/database"

	hub "notification-hub/internal/hub"
)

// attachmentEnv is a Service with a real filesystem attachment store.
type attachmentEnv struct {
	service *hub.Service
	db      *database.RoutedDB
	store   blobstore.BlobStore
	root    string
	now     time.Time
}

func newAttachmentEnv(t *testing.T) *attachmentEnv {
	t.Helper()
	service, db := testService(t)
	root := t.TempDir()
	store := blobstore.NewFilesystemBlobStore(root)
	service.SetAttachmentStore(hub.StaticBlobStore(store))
	env := &attachmentEnv{service: service, db: db, store: store, root: root, now: time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)}
	at(service, env.now)
	return env
}

func (e *attachmentEnv) advance(t *testing.T, d time.Duration) {
	t.Helper()
	e.now = e.now.Add(d)
	at(e.service, e.now)
}

func sha(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// put uploads content and returns its hash.
func (e *attachmentEnv) put(t *testing.T, content string) string {
	t.Helper()
	hash := sha(content)
	_, err := e.service.PutAttachmentBlob(context.Background(), hash, "", []byte(content))
	require.NoError(t, err)
	return hash
}

// file uploads content and returns the attachment referencing it.
func (e *attachmentEnv) file(t *testing.T, name, content string) hub.AskAttachment {
	t.Helper()
	return hub.AskAttachment{Name: name, Kind: hub.AttachmentFile, SHA256: e.put(t, content)}
}

func (e *attachmentEnv) folder(t *testing.T, name string, files map[string]string) hub.AskAttachment {
	t.Helper()
	folder := hub.AskAttachment{Name: name, Kind: hub.AttachmentFolder}
	for path, content := range files {
		folder.Entries = append(folder.Entries, hub.AskAttachmentEntry{Path: path, SHA256: e.put(t, content)})
	}
	return folder
}

func (e *attachmentEnv) ask(t *testing.T, key string, attachments ...hub.AskAttachment) hub.AskResult {
	t.Helper()
	spec := subjectSpec(e.now, key)
	spec.Attachments = attachments
	result, err := e.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	return result
}

func (e *attachmentEnv) attachments(t *testing.T, askID string) []hub.AskAttachment {
	t.Helper()
	list, err := e.service.AskAttachments(context.Background(), askID)
	require.NoError(t, err)
	return list
}

func (e *attachmentEnv) blobCount(t *testing.T) int {
	t.Helper()
	var count int
	require.NoError(t, e.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM attachment_blobs`).Scan(&count))
	return count
}

func names(list []hub.AskAttachment) []string {
	out := make([]string, 0, len(list))
	for _, a := range list {
		out = append(out, a.Name)
	}
	return out
}

func warningCodes(warnings []hub.PacketWarning) []string {
	out := make([]string, 0, len(warnings))
	for _, w := range warnings {
		out = append(out, w.Code)
	}
	return out
}

// attachmentWarnings keeps the warnings about attachments; the packet check
// adds its own about the cover text.
func attachmentWarnings(warnings []hub.PacketWarning) []hub.PacketWarning {
	var out []hub.PacketWarning
	for _, w := range warnings {
		if strings.HasPrefix(w.Code, "attachment_") {
			out = append(out, w)
		}
	}
	return out
}

func findWarning(t *testing.T, warnings []hub.PacketWarning, code string) hub.PacketWarning {
	t.Helper()
	for _, w := range warnings {
		if w.Code == code {
			return w
		}
	}
	require.Failf(t, "warning missing", "no %q warning in %v", code, warningCodes(warnings))
	return hub.PacketWarning{}
}

// ---- upload ----------------------------------------------------------------

func TestPutAttachmentBlob_StoresContentAddressedBytesAndReportsExisting(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	content := []byte("# Plan\nStore retention for 30 days.\n")
	hash := sha(string(content))

	first, err := env.service.PutAttachmentBlob(ctx, hash, "text/markdown", content)
	require.NoError(t, err)
	require.Equal(t, hub.BlobPutResult{SHA256: hash, SizeBytes: int64(len(content)), Existed: false}, first)
	second, err := env.service.PutAttachmentBlob(ctx, hash, "text/markdown", content)
	require.NoError(t, err)
	require.True(t, second.Existed, "the same bytes are not stored twice")
	require.Equal(t, 1, env.blobCount(t))

	require.FileExists(t, filepath.Join(env.root, "sha256", hash[:2], hash), "blobs are keyed sha256/<first byte>/<hash>")
	got, err := env.service.ReadAttachmentBlob(ctx, hash, 1<<20)
	require.NoError(t, err)
	require.Equal(t, content, got)
}

func TestPutAttachmentBlob_RefusesWhatItCannotTrust(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	tests := map[string]struct {
		hash    string
		content []byte
	}{
		"hash does not match the bytes": {hash: sha("other"), content: []byte("content")},
		"hash is not hex":               {hash: strings.Repeat("z", 64), content: []byte("content")},
		"hash is the wrong length":      {hash: "abc", content: []byte("content")},
		"hash is empty":                 {hash: "", content: []byte("content")},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := env.service.PutAttachmentBlob(ctx, tc.hash, "", tc.content)
			require.ErrorIs(t, err, hub.ErrInvalidArgument)
		})
	}
	require.Zero(t, env.blobCount(t), "a refused upload leaves nothing behind")
	entries, err := os.ReadDir(env.root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestPutAttachmentBlob_RefusesAFileOverSixtyFourMiB(t *testing.T) {
	env := newAttachmentEnv(t)
	content := make([]byte, hub.MaxAttachmentBlobBytes+1)
	sum := sha256.Sum256(content)
	_, err := env.service.PutAttachmentBlob(context.Background(), hex.EncodeToString(sum[:]), "", content)
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	require.Contains(t, err.Error(), "64 MiB")

	exact := content[:hub.MaxAttachmentBlobBytes]
	sum = sha256.Sum256(exact)
	_, err = env.service.PutAttachmentBlob(context.Background(), hex.EncodeToString(sum[:]), "", exact)
	require.NoError(t, err, "64 MiB itself is allowed")
}

func TestPutAttachmentBlob_CapsTheBytesWaitingForAnAsk(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	hub.SetUnreferencedBlobLimit(env.service, 10)
	put := func(content string) error {
		_, err := env.service.PutAttachmentBlob(ctx, sha(content), "", []byte(content))
		return err
	}

	require.NoError(t, put("123456"))
	require.ErrorIs(t, put("abcdef"), hub.ErrBlobQuota, "6 + 6 bytes would pass the 10 byte limit")
	require.Equal(t, 1, env.blobCount(t), "the refused upload stored nothing")
	require.NoError(t, put("123456"), "bytes the Hub already holds add nothing")
	require.NoError(t, put("abcd"), "6 + 4 is exactly the limit")

	// An ask that references a blob frees its share of the limit.
	env.ask(t, "evt-1", hub.AskAttachment{Name: "a.md", Kind: hub.AttachmentFile, SHA256: sha("123456")})
	require.NoError(t, put("ABCDEF"))
	require.ErrorIs(t, put("zzzzzzz"), hub.ErrBlobQuota)

	// Unused uploads expire after a day and free theirs.
	env.advance(t, 25*time.Hour)
	_, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.NoError(t, put("zzzzzzz"))
}

func TestPutAttachmentBlob_WithoutAStoreSaysSo(t *testing.T) {
	service, _ := testService(t)
	_, err := service.PutAttachmentBlob(context.Background(), sha("x"), "", []byte("x"))
	require.ErrorIs(t, err, hub.ErrAttachmentsUnavailable)
}

func TestStatAttachmentBlobs_ReportsOnlyWhatTheHubLacks(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	held := env.put(t, "held")
	absent := sha("absent")

	missing, err := env.service.StatAttachmentBlobs(ctx, []string{held, absent, absent})
	require.NoError(t, err)
	require.Equal(t, []string{absent}, missing, "a repeated hash is reported once, in request order")

	missing, err = env.service.StatAttachmentBlobs(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, missing)

	_, err = env.service.StatAttachmentBlobs(ctx, []string{"not-a-hash"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
}

func TestStatAttachmentBlobs_ALostFileCountsAsMissingSoItIsUploadedAgain(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	hash := env.put(t, "will be lost")
	require.NoError(t, os.Remove(filepath.Join(env.root, "sha256", hash[:2], hash)))

	missing, err := env.service.StatAttachmentBlobs(ctx, []string{hash})
	require.NoError(t, err)
	require.Equal(t, []string{hash}, missing)

	result, err := env.service.PutAttachmentBlob(ctx, hash, "", []byte("will be lost"))
	require.NoError(t, err)
	require.False(t, result.Existed, "the upload repairs the lost bytes")
	got, err := env.service.ReadAttachmentBlob(ctx, hash, 1<<10)
	require.NoError(t, err)
	require.Equal(t, "will be lost", string(got))
}

func TestReadAttachmentBlob_EnforcesTheCallersLimitAndReportsAbsence(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	hash := env.put(t, strings.Repeat("a", 100))

	_, err := env.service.ReadAttachmentBlob(ctx, hash, 99)
	require.ErrorIs(t, err, hub.ErrBlobTooLarge)
	got, err := env.service.ReadAttachmentBlob(ctx, hash, 100)
	require.NoError(t, err)
	require.Len(t, got, 100)
	_, err = env.service.ReadAttachmentBlob(ctx, sha("never uploaded"), 100)
	require.ErrorIs(t, err, hub.ErrNotFound)
}

// ---- normalization ---------------------------------------------------------

func TestAsk_AttachmentsAreStoredAndReadBackWithTheirPreviewKind(t *testing.T) {
	env := newAttachmentEnv(t)
	plan := env.file(t, "plan.md", "# Plan\n")
	plan.Label = "The plan"
	plan.SourcePath = "/work/plan.md"
	plan.SourceHost = "builder-1"
	shot := env.file(t, "shot.png", "not really a png")
	clip := env.file(t, "clip.mp4", "not really a video")
	noext := env.file(t, "notes", "plain notes with no extension")
	report := env.folder(t, "report", map[string]string{"index.md": "# Index", "data/rows.csv": "a,b\n1,2\n", "img/chart.svg": "<svg/>"})

	result := env.ask(t, "evt-1", plan, shot, clip, noext, report)

	require.Empty(t, attachmentWarnings(result.Warnings))
	require.Equal(t, 5, result.Ask.AttachmentCount)
	list := env.attachments(t, result.Ask.ID)
	require.Equal(t, []string{"plan.md", "shot.png", "clip.mp4", "notes", "report"}, names(list), "producer order is kept")

	byName := map[string]hub.AskAttachment{}
	for _, a := range list {
		byName[a.Name] = a
	}
	require.Equal(t, "markdown", byName["plan.md"].PreviewKind)
	require.Equal(t, "text/markdown; charset=utf-8", byName["plan.md"].MediaType)
	require.Equal(t, "The plan", byName["plan.md"].Label)
	require.Equal(t, "shot.png", byName["shot.png"].Label, "the label defaults to the name")
	require.Equal(t, int64(len("# Plan\n")), byName["plan.md"].SizeBytes, "sizes come from the Hub's own copy")
	require.Equal(t, "/work/plan.md", byName["plan.md"].SourcePath)
	require.Equal(t, "image", byName["shot.png"].PreviewKind)
	require.Equal(t, "video", byName["clip.mp4"].PreviewKind)
	require.Equal(t, "text", byName["notes"].PreviewKind, "an unknown extension is sniffed")
	require.Equal(t, hub.AttachmentRoleEvidence, byName["plan.md"].Role)

	folder := byName["report"]
	require.Equal(t, "directory", folder.PreviewKind)
	require.False(t, folder.Truncated)
	entryKinds := map[string]string{}
	for _, e := range folder.Entries {
		entryKinds[e.Path] = e.PreviewKind
		require.NotZero(t, e.SizeBytes)
	}
	require.Equal(t, map[string]string{"index.md": "markdown", "data/rows.csv": "csv", "img/chart.svg": "svg"}, entryKinds)
}

func TestAsk_ADroppedAttachmentNeverFailsTheAsk(t *testing.T) {
	env := newAttachmentEnv(t)
	kept := env.file(t, "kept.md", "kept")
	missing := hub.AskAttachment{Name: "lost.pdf", Kind: hub.AttachmentFile, SHA256: sha("never uploaded")}

	result := env.ask(t, "evt-1", missing, kept)

	require.Equal(t, hub.AskPending, result.Ask.State, "the ask exists")
	require.Equal(t, []string{"kept.md"}, names(env.attachments(t, result.Ask.ID)))
	warning := findWarning(t, result.Warnings, hub.WarnAttachmentBlobMissing)
	require.Equal(t, "attachments[0]", warning.Field)
	require.Equal(t, hub.WarningDropped, warning.Severity)
	require.Contains(t, warning.Message, "lost.pdf")
	require.Contains(t, warning.Message, "PutAttachmentBlob", "the warning names the fix")
}

func TestAsk_AnAskWithNoStoreDropsEveryAttachmentAndStillSends(t *testing.T) {
	service, _ := testService(t)
	spec := subjectSpec(time.Now(), "evt-1")
	spec.Attachments = []hub.AskAttachment{{Name: "a.md", Kind: hub.AttachmentFile, SHA256: sha("a")}}
	result, err := service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, result.Ask.State)
	findWarning(t, result.Warnings, hub.WarnAttachmentBlobMissing)
}

func TestAsk_InvalidAttachmentsAreDroppedWithAWarningEach(t *testing.T) {
	env := newAttachmentEnv(t)
	good := env.file(t, "good.md", "good")
	tests := map[string]hub.AskAttachment{
		"unknown kind":         {Name: "x", Kind: "symlink", SHA256: good.SHA256},
		"file without a hash":  {Name: "x", Kind: hub.AttachmentFile},
		"file with a bad hash": {Name: "x", Kind: hub.AttachmentFile, SHA256: "ABC"},
		"folder without files": {Name: "x", Kind: hub.AttachmentFolder},
		"folder of only unsafe entries": {Name: "x", Kind: hub.AttachmentFolder, Entries: []hub.AskAttachmentEntry{
			{Path: "../escape.md", SHA256: good.SHA256}, {Path: "/etc/passwd", SHA256: good.SHA256}, {Path: "a//b", SHA256: good.SHA256}, {Path: "", SHA256: good.SHA256},
		}},
	}
	for name, bad := range tests {
		t.Run(name, func(t *testing.T) {
			result := env.ask(t, "evt-"+strings.ReplaceAll(name, " ", "-"), bad, good)
			require.Equal(t, []string{"good.md"}, names(env.attachments(t, result.Ask.ID)))
			require.Contains(t, warningCodes(result.Warnings), hub.WarnAttachmentInvalid)
			for _, w := range attachmentWarnings(result.Warnings) {
				require.Equal(t, hub.WarningDropped, w.Severity)
				require.NotEmpty(t, w.Field)
				require.NotEmpty(t, w.Message)
			}
		})
	}
}

func TestAsk_NamesAreSanitizedAndKeptUnique(t *testing.T) {
	env := newAttachmentEnv(t)
	hash := env.put(t, "same bytes")
	attach := func(name string) hub.AskAttachment {
		return hub.AskAttachment{Name: name, Kind: hub.AttachmentFile, SHA256: hash}
	}

	result := env.ask(t, "evt-1", attach("My Report (final).md"), attach("notes.md"), attach("notes.md"), attach(""), attach(".hidden"), attach(strings.Repeat("n", 120)+".md"))

	got := names(env.attachments(t, result.Ask.ID))
	require.Equal(t, "My-Report--final-.md", got[0])
	require.Equal(t, "notes.md", got[1])
	require.Equal(t, "notes-2.md", got[2], "a repeated name is numbered before its extension, as the CLI does")
	require.Equal(t, "attachment-4", got[3], "an empty name with no source path is numbered")
	require.Equal(t, "hidden", got[4], "a name must start with a letter or digit")
	require.LessOrEqual(t, len(got[5]), 96)
	seen := map[string]bool{}
	for _, name := range got {
		require.Regexp(t, `^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`, name)
		require.False(t, seen[name], "duplicate %q", name)
		seen[name] = true
	}
	renamed := 0
	for _, w := range result.Warnings {
		if w.Code == hub.WarnAttachmentRenamed {
			renamed++
			require.Equal(t, hub.WarningAdvice, w.Severity, "a rename keeps the attachment")
		}
	}
	require.Equal(t, 5, renamed, "every changed name is reported; notes.md (first) is untouched")
}

func TestAsk_RepeatedNamesAreNumberedBeforeTheExtensionAndWithinTheLimit(t *testing.T) {
	env := newAttachmentEnv(t)
	hash := env.put(t, "same bytes")
	file := func(name string) hub.AskAttachment {
		return hub.AskAttachment{Name: name, Kind: hub.AttachmentFile, SHA256: hash}
	}
	folder := func(name string) hub.AskAttachment {
		return hub.AskAttachment{Name: name, Kind: hub.AttachmentFolder, Entries: []hub.AskAttachmentEntry{{Path: "a.md", SHA256: hash}}}
	}
	long := strings.Repeat("n", 100) + ".md"

	result := env.ask(t, "evt-1",
		file("plan.md"), file("plan.md"), file("plan.md"), file("plan-2.md"),
		file("archive.tar.gz"), file("archive.tar.gz"),
		file("README"), file("README"),
		folder("release-v1.2"), folder("release-v1.2"),
		file(long), file(long))

	got := names(env.attachments(t, result.Ask.ID))
	require.Equal(t, []string{"plan.md", "plan-2.md", "plan-3.md", "plan-2-2.md", "archive.tar.gz", "archive.tar-2.gz", "README", "README-2", "release-v1.2", "release-v1.2-2"}, got[:10],
		"a name the sender made unique is kept; folders have no extension")
	require.Len(t, got, 12)
	for _, name := range got[10:] {
		require.LessOrEqual(t, len(name), 96)
		require.True(t, strings.HasSuffix(name, ".md"), "%q keeps its extension when the stem is cut", name)
	}
	require.NotEqual(t, got[10], got[11])
}

func TestAsk_AnEmptyNameTakesTheSourceFileName(t *testing.T) {
	env := newAttachmentEnv(t)
	attachment := env.file(t, "", "report body")
	attachment.SourcePath = "/home/me/work/report.md"
	result := env.ask(t, "evt-1", attachment)
	require.Equal(t, []string{"report.md"}, names(env.attachments(t, result.Ask.ID)))
}

func TestAsk_AtMostTwentyFourAttachmentsAreKept(t *testing.T) {
	env := newAttachmentEnv(t)
	var list []hub.AskAttachment
	for i := 0; i < 27; i++ {
		list = append(list, env.file(t, fmt.Sprintf("f%02d.txt", i), fmt.Sprintf("content %d", i)))
	}

	result := env.ask(t, "evt-1", list...)

	got := names(env.attachments(t, result.Ask.ID))
	require.Len(t, got, 24)
	require.Equal(t, "f23.txt", got[23], "the first 24 are kept")
	warning := findWarning(t, result.Warnings, hub.WarnAttachmentLimit)
	require.Equal(t, hub.WarningDropped, warning.Severity)
	require.Contains(t, warning.Message, "3 attachments")
}

func TestAsk_InvalidAttachmentsAreDroppedBeforeTheCountLimitNotInsteadOfValidOnes(t *testing.T) {
	env := newAttachmentEnv(t)
	var list []hub.AskAttachment
	for i := 0; i < 6; i++ {
		list = append(list, hub.AskAttachment{Name: fmt.Sprintf("bad%d.txt", i), Kind: hub.AttachmentFile, SHA256: "not-a-hash"})
	}
	for i := 0; i < 26; i++ {
		list = append(list, env.file(t, fmt.Sprintf("f%02d.txt", i), fmt.Sprintf("content %d", i)))
	}

	result := env.ask(t, "evt-1", list...)

	got := names(env.attachments(t, result.Ask.ID))
	require.Len(t, got, 24, "the six invalid ones took no slot")
	require.Equal(t, "f00.txt", got[0])
	require.Equal(t, "f23.txt", got[23])
	invalid := 0
	for _, w := range result.Warnings {
		if w.Code == hub.WarnAttachmentInvalid {
			invalid++
		}
	}
	require.Equal(t, 6, invalid, "each invalid attachment is reported")
	require.Contains(t, findWarning(t, result.Warnings, hub.WarnAttachmentLimit).Message, "2 attachments", "only the valid ones past 24 are over the limit")
}

func TestAsk_WithoutAVerifiedProducerKeepsBlobsAndDropsHostPathProvenance(t *testing.T) {
	env := newAttachmentEnv(t)
	plan := env.file(t, "plan.md", "plan")
	plan.SourcePath, plan.SourceHost = "/etc/hostname", "any-host"
	spec := subjectSpec(env.now, "evt-anon")
	spec.Attachments = []hub.AskAttachment{plan, env.folder(t, "report", map[string]string{"a.md": "a"})}
	spec.VerifiedProducer = false

	result, err := env.service.AskDecision(context.Background(), spec, "")

	require.NoError(t, err, "a packet problem never fails the ask")
	require.Equal(t, hub.AskPending, result.Ask.State)
	require.Equal(t, 2, result.Ask.AttachmentCount)
	got := env.attachments(t, result.Ask.ID)
	require.Len(t, got, 2)
	require.Empty(t, got[0].SourcePath, "anonymous source paths must never be probed")
	require.Empty(t, got[0].SourceHost)
	require.NotContains(t, warningCodes(result.Warnings), "attachment_requires_identity")
}

func TestAsk_AttachmentBytesPastTwoHundredFiftySixMiBAreDropped(t *testing.T) {
	env := newAttachmentEnv(t)
	// The inventory is what the limit reads, so large blobs are recorded
	// without writing hundreds of megabytes.
	record := func(content string, size int64) hub.AskAttachment {
		hash := sha(content)
		_, err := env.db.ExecContext(context.Background(), `INSERT INTO attachment_blobs (sha256, size_bytes, media_type, stored_at, last_touched_at) VALUES (?, ?, '', ?, ?)`, hash, size, env.now.Format(time.RFC3339Nano), env.now.Format(time.RFC3339Nano))
		require.NoError(t, err)
		return hub.AskAttachment{Name: content + ".bin", Kind: hub.AttachmentFile, SHA256: hash}
	}
	result := env.ask(t, "evt-1", record("a", 100<<20), record("b", 100<<20), record("c", 100<<20), record("d", 50<<20))

	require.Equal(t, []string{"a.bin", "b.bin", "d.bin"}, names(env.attachments(t, result.Ask.ID)), "c is the one that crosses 256 MiB; d still fits")
	warning := findWarning(t, result.Warnings, hub.WarnAttachmentLimit)
	require.Contains(t, warning.Message, "256 MiB")
}

func TestAsk_AFolderKeepsTheFirstFiveHundredFilesAndSaysSo(t *testing.T) {
	env := newAttachmentEnv(t)
	hash := env.put(t, "x")
	folder := hub.AskAttachment{Name: "many", Kind: hub.AttachmentFolder}
	for i := 0; i < 520; i++ {
		folder.Entries = append(folder.Entries, hub.AskAttachmentEntry{Path: fmt.Sprintf("dir/file-%03d.txt", i), SHA256: hash})
	}

	result := env.ask(t, "evt-1", folder)

	list := env.attachments(t, result.Ask.ID)
	require.Len(t, list, 1)
	require.True(t, list[0].Truncated)
	require.Len(t, list[0].Entries, 500)
	require.Equal(t, "dir/file-000.txt", list[0].Entries[0].Path)
	require.Equal(t, "dir/file-499.txt", list[0].Entries[499].Path)
	warning := findWarning(t, result.Warnings, hub.WarnFolderTruncated)
	require.Equal(t, hub.WarningAdvice, warning.Severity, "the folder is kept")
}

func TestAsk_AFolderDropsOnlyTheEntriesTheHubLacks(t *testing.T) {
	env := newAttachmentEnv(t)
	folder := env.folder(t, "report", map[string]string{"a.md": "a", "b.md": "b"})
	folder.Entries = append(folder.Entries,
		hub.AskAttachmentEntry{Path: "lost.md", SHA256: sha("never uploaded")},
		hub.AskAttachmentEntry{Path: "a.md", SHA256: folder.Entries[0].SHA256}, // repeated path
	)

	result := env.ask(t, "evt-1", folder)

	list := env.attachments(t, result.Ask.ID)
	require.Len(t, list, 1)
	paths := []string{}
	for _, e := range list[0].Entries {
		paths = append(paths, e.Path)
	}
	require.ElementsMatch(t, []string{"a.md", "b.md"}, paths)
	warning := findWarning(t, result.Warnings, hub.WarnAttachmentBlobMissing)
	require.Contains(t, warning.Message, "1 files of the folder")
}

func TestAsk_OnlyOneExplainerIsKept(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.file(t, "first.html", "<html>one</html>")
	first.Role = hub.AttachmentRoleExplainer
	second := env.file(t, "second.html", "<html>two</html>")
	second.Role = hub.AttachmentRoleExplainer
	folder := env.folder(t, "folder", map[string]string{"index.html": "<html/>"})
	folder.Role = hub.AttachmentRoleExplainer

	result := env.ask(t, "evt-1", first, second, folder)

	roles := map[string]string{}
	for _, a := range env.attachments(t, result.Ask.ID) {
		roles[a.Name] = a.Role
	}
	require.Equal(t, map[string]string{"first.html": hub.AttachmentRoleExplainer, "second.html": hub.AttachmentRoleEvidence, "folder": hub.AttachmentRoleEvidence}, roles)
	explainerWarnings := 0
	for _, w := range result.Warnings {
		if w.Code == hub.WarnExplainerLimit {
			explainerWarnings++
			require.Equal(t, hub.WarningAdvice, w.Severity)
		}
	}
	require.Equal(t, 2, explainerWarnings, "the extra explainer and the folder explainer each say why they were demoted")
}

func TestAsk_ADroppedExplainerDoesNotUseUpTheSingleSlot(t *testing.T) {
	env := newAttachmentEnv(t)
	lost := hub.AskAttachment{Name: "lost.html", Kind: hub.AttachmentFile, Role: hub.AttachmentRoleExplainer, SHA256: sha("never uploaded")}
	real := env.file(t, "real.html", "<html/>")
	real.Role = hub.AttachmentRoleExplainer

	result := env.ask(t, "evt-1", lost, real)

	list := env.attachments(t, result.Ask.ID)
	require.Len(t, list, 1)
	require.Equal(t, hub.AttachmentRoleExplainer, list[0].Role)
}

func TestAsk_ASourcePathThatIsNotAbsoluteIsIgnoredWithAdvice(t *testing.T) {
	env := newAttachmentEnv(t)
	attachment := env.file(t, "a.md", "a")
	attachment.SourcePath = "relative/a.md"

	result := env.ask(t, "evt-1", attachment)

	require.Empty(t, env.attachments(t, result.Ask.ID)[0].SourcePath)
	require.Equal(t, hub.WarningAdvice, findWarning(t, result.Warnings, hub.WarnAttachmentSourceIgnore).Severity)
}

func TestAsk_TheHubServesTheClassifiersTypeNeverTheSendersMediaType(t *testing.T) {
	env := newAttachmentEnv(t)
	page := env.file(t, "page.html", "<html/>")
	page.MediaType = "application/x-evil"
	// A type the classifier cannot place is a download, whatever the sender says.
	blob := env.file(t, "data.unknownext", "\x00\x01\x02\x03")
	blob.MediaType = "text/html"
	// HTML in a legacy encoding is not valid UTF-8, and a sniff alone must not
	// make it an HTML page of the Hub's origin.
	legacy := env.file(t, "legacy", "<!DOCTYPE html><html>caf\xe9<script>alert(1)</script></html>")
	legacy.MediaType = "text/html"
	svg := env.file(t, "diagram.svg", "<svg xmlns='http://www.w3.org/2000/svg'/>")
	svg.MediaType = "text/html"
	folder := env.folder(t, "site", map[string]string{"index.unknownext": "\x00\x01"})
	folder.Entries[0].MediaType = "text/html"

	result := env.ask(t, "evt-1", page, blob, legacy, svg, folder)

	byName := map[string]hub.AskAttachment{}
	for _, a := range env.attachments(t, result.Ask.ID) {
		byName[a.Name] = a
	}
	require.Equal(t, "text/plain; charset=utf-8", byName["page.html"].MediaType)
	require.Equal(t, "unsupported", byName["data.unknownext"].PreviewKind)
	require.Equal(t, "application/octet-stream", byName["data.unknownext"].MediaType)
	require.Equal(t, "text/plain; charset=utf-8", byName["legacy"].MediaType)
	require.Equal(t, "image/svg+xml", byName["diagram.svg"].MediaType, "the classifier's own type for a known extension")
	require.Equal(t, "application/octet-stream", byName["site"].Entries[0].MediaType)
}

func TestAsk_PlainAsksKeepTheirAttachmentsToo(t *testing.T) {
	env := newAttachmentEnv(t)
	spec := decisionSpec(env.now, "plain-1")
	spec.Attachments = []hub.AskAttachment{env.file(t, "a.md", "a")}

	result, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	require.Equal(t, 1, result.Ask.AttachmentCount)
	require.Equal(t, []string{"a.md"}, names(env.attachments(t, result.Ask.ID)))

	// A retry of the same event is idempotent and changes nothing.
	again, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)
	require.Equal(t, result.Ask.ID, again.Ask.ID)
	require.Equal(t, []string{"a.md"}, names(env.attachments(t, again.Ask.ID)))
}

// ---- subject keys ------------------------------------------------------------

func TestSubject_AnIdenticalResendWithTheSameAttachmentsIsReasserted(t *testing.T) {
	env := newAttachmentEnv(t)
	sender := &recordingSender{}
	env.service.SetPushSender(sender)
	registerSubscription(t, env.service, "alice")
	plan := env.file(t, "plan.md", "plan")
	first := env.ask(t, "evt-1", plan)
	waitDelivered(t, env.service, first.Ask)

	env.advance(t, 10*time.Minute)
	repeat := env.ask(t, "evt-2", plan)

	require.Equal(t, hub.AskReasserted, repeat.Outcome)
	require.Equal(t, first.Ask.ID, repeat.Ask.ID)
	require.Zero(t, repeat.Ask.RevisionCount)
	require.Equal(t, 1, pushCount(sender))
}

func TestSubject_AResendWithChangedAttachmentsUpdatesInPlaceWithoutANewPush(t *testing.T) {
	env := newAttachmentEnv(t)
	sender := &recordingSender{}
	env.service.SetPushSender(sender)
	registerSubscription(t, env.service, "alice")
	first := env.ask(t, "evt-1", env.file(t, "plan.md", "plan v1"))
	waitDelivered(t, env.service, first.Ask)
	oldHash := sha("plan v1")

	env.advance(t, time.Hour)
	updated := env.ask(t, "evt-2", env.file(t, "plan.md", "plan v2"), env.file(t, "extra.png", "png"))

	require.Equal(t, hub.AskUpdated, updated.Outcome, "attachments are part of what the ask says")
	require.Equal(t, first.Ask.ID, updated.Ask.ID)
	require.Equal(t, 1, updated.Ask.RevisionCount)
	require.Equal(t, 2, updated.Ask.AttachmentCount)
	list := env.attachments(t, updated.Ask.ID)
	require.Equal(t, []string{"plan.md", "extra.png"}, names(list), "the revision replaces the attachment rows")
	require.Equal(t, sha("plan v2"), list[0].SHA256)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 1, pushCount(sender), "an in-place update is not pushed again")

	var references int
	require.NoError(t, env.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM ask_attachments WHERE sha256 = ?`, oldHash).Scan(&references))
	require.Zero(t, references, "nothing references the replaced content")
}

func TestSubject_ARevisionWithoutAttachmentsKeepsThemUnlessToldToClear(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-1", env.file(t, "plan.md", "plan"))
	env.advance(t, time.Hour)

	kept := env.ask(t, "evt-2")
	require.Equal(t, hub.AskReasserted, kept.Outcome, "nothing sent, nothing removed")
	require.Equal(t, 1, kept.Ask.AttachmentCount)

	spec := subjectSpec(env.now, "evt-3")
	spec.ClearAttachments = true
	updated, err := env.service.AskDecision(context.Background(), spec, "")
	require.NoError(t, err)

	require.Equal(t, hub.AskUpdated, updated.Outcome)
	require.Zero(t, updated.Ask.AttachmentCount)
	require.Empty(t, env.attachments(t, first.Ask.ID))
}

// Revisions that run at the same time each add one part to the packet. Every part
// must survive: a revision merges against the packet the one before it stored,
// not against the packet it saw when it arrived.
func TestSubject_ConcurrentRevisionsEachKeepWhatTheOthersAdded(t *testing.T) {
	env := newAttachmentEnv(t)
	first := env.ask(t, "evt-0", env.file(t, "plan.md", "plan"))
	const revisers = 10
	specs := make([]hub.AskSpec, revisers)
	for i := range specs {
		name := fmt.Sprintf("extra-%02d.md", i)
		spec := subjectSpec(env.now, fmt.Sprintf("evt-%02d", i+1))
		spec.Attachments = []hub.AskAttachment{env.file(t, name, "content of "+name)}
		spec.Questions = []hub.AskQuestion{{ID: fmt.Sprintf("q-%02d", i), Type: "text", Prompt: fmt.Sprintf("Question %d?", i)}}
		specs[i] = spec
	}

	start := make(chan struct{})
	errs := make(chan error, revisers)
	var wg sync.WaitGroup
	for _, spec := range specs {
		wg.Add(1)
		go func(spec hub.AskSpec) {
			defer wg.Done()
			<-start
			_, err := env.service.AskDecision(context.Background(), spec, "")
			errs <- err
		}(spec)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	kept, err := env.service.GetAsk(context.Background(), first.Ask.ID)
	require.NoError(t, err)
	var names []string
	for _, attachment := range env.attachments(t, first.Ask.ID) {
		names = append(names, attachment.Name)
	}
	var prompts []string
	for _, question := range kept.Questions {
		prompts = append(prompts, question.ID)
	}
	want, wantQuestions := []string{"plan.md"}, []string{}
	for i := 0; i < revisers; i++ {
		want = append(want, fmt.Sprintf("extra-%02d.md", i))
		wantQuestions = append(wantQuestions, fmt.Sprintf("q-%02d", i))
	}
	require.ElementsMatch(t, want, names, "no revision lost an attachment another added")
	require.ElementsMatch(t, wantQuestions, prompts, "no revision lost a question another added")
	require.Equal(t, revisers, kept.RevisionCount)
}

func TestSubject_ChangedSourceOrLabelCountsAsAChange(t *testing.T) {
	env := newAttachmentEnv(t)
	plan := env.file(t, "plan.md", "plan")
	env.ask(t, "evt-1", plan)
	env.advance(t, time.Minute)

	relabelled := plan
	relabelled.Label = "The revised plan"
	require.Equal(t, hub.AskUpdated, env.ask(t, "evt-2", relabelled).Outcome)
	env.advance(t, time.Minute)
	require.Equal(t, hub.AskReasserted, env.ask(t, "evt-3", relabelled).Outcome)
}

func TestSubject_AResendWhoseAttachmentWasDroppedIsNotAnUpdateWhenTheOutcomeIsTheSame(t *testing.T) {
	env := newAttachmentEnv(t)
	lost := hub.AskAttachment{Name: "lost.pdf", Kind: hub.AttachmentFile, SHA256: sha("never uploaded")}
	first := env.ask(t, "evt-1", lost)
	require.Zero(t, first.Ask.AttachmentCount)
	env.advance(t, time.Minute)

	repeat := env.ask(t, "evt-2", lost)

	require.Equal(t, hub.AskReasserted, repeat.Outcome, "the stored packet is unchanged")
	findWarning(t, repeat.Warnings, hub.WarnAttachmentBlobMissing)
}

func TestListAsks_ReportsTheAttachmentCount(t *testing.T) {
	env := newAttachmentEnv(t)
	env.ask(t, "evt-1", env.file(t, "a.md", "a"), env.file(t, "b.md", "b"))
	page, err := env.service.ListAsksPage(context.Background(), "alice", hub.AskListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Asks, 1)
	require.Equal(t, 2, page.Asks[0].AttachmentCount)
}

// ---- drift -----------------------------------------------------------------

func TestAskAttachments_ReportsWhetherTheSourceStillMatchesTheStoredCopy(t *testing.T) {
	env := newAttachmentEnv(t)
	host, err := os.Hostname()
	require.NoError(t, err)
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
		return p
	}
	attach := func(name, content, source, sourceHost string) hub.AskAttachment {
		a := env.file(t, name, content)
		a.SourcePath, a.SourceHost = source, sourceHost
		return a
	}
	fresh := attach("fresh.md", "same", write("fresh.md", "same"), host)
	sameSize := attach("samesize.md", "abcd", write("samesize.md", "abcd"), host)
	shorter := attach("shorter.md", "long content", write("shorter.md", "long content"), host)
	gone := attach("gone.md", "gone", write("gone.md", "gone"), host)
	elsewhere := attach("elsewhere.md", "x", write("elsewhere.md", "x"), "another-host")
	nosource := env.file(t, "nosource.md", "n")
	report := env.folder(t, "report", map[string]string{"a.md": "a", "sub/b.md": "b"})
	report.SourcePath, report.SourceHost = filepath.Join(dir, "report"), host
	write("report/a.md", "a")
	write("report/sub/b.md", "b")

	result := env.ask(t, "evt-1", fresh, sameSize, shorter, gone, elsewhere, nosource, report)
	require.Empty(t, attachmentWarnings(result.Warnings))

	// The sources change after the ask was sent.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "samesize.md"), []byte("abce"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shorter.md"), []byte("short"), 0o600))
	require.NoError(t, os.Remove(filepath.Join(dir, "gone.md")))

	status := map[string]string{}
	for _, a := range env.attachments(t, result.Ask.ID) {
		status[a.Name] = a.SourceStatus
		if a.SourceStatus == hub.SourceUnknown {
			require.Empty(t, a.SourceCheckedAt, "an unchecked source has no check time")
		} else {
			require.Equal(t, env.now.Format(time.RFC3339), a.SourceCheckedAt)
		}
	}
	require.Equal(t, map[string]string{
		"fresh.md":     hub.SourceFresh,
		"samesize.md":  hub.SourceChanged, // same size, different bytes: only the hash tells
		"shorter.md":   hub.SourceChanged,
		"gone.md":      hub.SourceMissing,
		"elsewhere.md": hub.SourceUnknown, // another host's paths mean nothing here
		"nosource.md":  hub.SourceUnknown,
		"report":       hub.SourceFresh,
	}, status)

	// A folder is changed when one of its files is, and missing when it is gone.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "report", "sub", "b.md"), []byte("B"), 0o600))
	require.Equal(t, hub.SourceChanged, findAttachment(t, env.attachments(t, result.Ask.ID), "report").SourceStatus)
	require.NoError(t, os.RemoveAll(filepath.Join(dir, "report")))
	require.Equal(t, hub.SourceMissing, findAttachment(t, env.attachments(t, result.Ask.ID), "report").SourceStatus)
}

func findAttachment(t *testing.T, list []hub.AskAttachment, name string) hub.AskAttachment {
	t.Helper()
	for _, a := range list {
		if a.Name == name {
			return a
		}
	}
	require.Failf(t, "attachment missing", "no attachment %q in %v", name, names(list))
	return hub.AskAttachment{}
}

// ---- serving -----------------------------------------------------------------

func TestOpenAskAttachment_ServesStoredBytesOnlyToTheRecipient(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	plan := env.file(t, "plan.md", "# The plan")
	plan.SourcePath = "/work/notes/the-plan.md"
	report := env.folder(t, "report", map[string]string{"sub/index.md": "# Index", "a.csv": "x,y"})
	result := env.ask(t, "evt-1", plan, report)
	read := func(c hub.AttachmentContent) string {
		defer c.Content.Close()
		data, err := io.ReadAll(c.Content)
		require.NoError(t, err)
		return string(data)
	}

	file, err := env.service.OpenAskAttachment(ctx, result.Ask.ID, "alice", "plan.md", "")
	require.NoError(t, err)
	require.Equal(t, "the-plan.md", file.FileName, "the operator is offered the source's file name")
	require.Equal(t, "text/markdown; charset=utf-8", file.MediaType)
	require.Equal(t, sha("# The plan"), file.SHA256)
	require.Equal(t, "# The plan", read(file))

	entry, err := env.service.OpenAskAttachment(ctx, result.Ask.ID, "alice", "report", "sub/index.md")
	require.NoError(t, err)
	require.Equal(t, "index.md", entry.FileName)
	require.Equal(t, "# Index", read(entry))

	refusals := map[string]struct{ ask, recipient, name, path string }{
		"another recipient":         {result.Ask.ID, "bob", "plan.md", ""},
		"no recipient":              {result.Ask.ID, "", "plan.md", ""},
		"unknown ask":               {"no-such-ask", "alice", "plan.md", ""},
		"unknown attachment":        {result.Ask.ID, "alice", "other.md", ""},
		"a file with a path":        {result.Ask.ID, "alice", "plan.md", "x"},
		"a folder without a path":   {result.Ask.ID, "alice", "report", ""},
		"a path outside the folder": {result.Ask.ID, "alice", "report", "../plan.md"},
		"an unknown folder path":    {result.Ask.ID, "alice", "report", "missing.md"},
	}
	for name, r := range refusals {
		t.Run(name, func(t *testing.T) {
			_, err := env.service.OpenAskAttachment(ctx, r.ask, r.recipient, r.name, r.path)
			require.ErrorIs(t, err, hub.ErrNotFound)
		})
	}
}

// ---- garbage collection ------------------------------------------------------

func TestSweepAttachmentBlobs_DeletesUnreferencedBlobsADayAfterTheirLastUse(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	referenced := env.file(t, "kept.md", "referenced")
	env.ask(t, "evt-1", referenced)
	orphan := env.put(t, "orphan")
	orphanPath := filepath.Join(env.root, "sha256", orphan[:2], orphan)
	require.FileExists(t, orphanPath)

	env.advance(t, 23*time.Hour)
	deleted, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Zero(t, deleted, "a blob a sender may still be about to reference is kept for 24 hours")
	require.FileExists(t, orphanPath)

	env.advance(t, 2*time.Hour)
	deleted, err = env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.NoFileExists(t, orphanPath, "the bytes go with the inventory row")
	require.NoFileExists(t, orphanPath+".meta")
	require.Equal(t, 1, env.blobCount(t))
	got, err := env.service.ReadAttachmentBlob(ctx, referenced.SHA256, 1<<10)
	require.NoError(t, err)
	require.Equal(t, "referenced", string(got), "a referenced blob is never swept, however old")
}

func TestSweepAttachmentBlobs_KeepsItemNoteFilesReferencedByAnswerChanges(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	hash := env.put(t, "note attachment")
	ask := env.ask(t, "item-note", env.file(t, "brief.md", "brief"))
	_, err := env.service.AnswerAsk(ctx, hub.AnswerInput{
		AskID: ask.Ask.ID, Answer: "retry", Actor: "alice",
		Changes: map[string]any{"notes": map[string]any{"measure.latency": map[string]any{"label": "Latency", "text": "See this plot", "attachments": []any{map[string]any{"name": "plot.csv", "sha256": hash, "mediaType": "text/csv", "sizeBytes": 14}}}}},
	})
	require.NoError(t, err)
	answered, err := env.service.GetAsk(ctx, ask.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, hash, answered.AnswerChanges["notes"].(map[string]any)["measure.latency"].(map[string]any)["attachments"].([]any)[0].(map[string]any)["sha256"])
	env.advance(t, 25*time.Hour)
	deleted, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Zero(t, deleted, "a note attachment referenced by the accepted answer remains available")
	content, err := env.service.ReadAttachmentBlob(ctx, hash, 1<<10)
	require.NoError(t, err)
	require.Equal(t, "note attachment", string(content))
}

func TestSweepAttachmentBlobs_ConfirmingOrReferencingABlobRestartsItsGracePeriod(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	hash := env.put(t, "reused")

	env.advance(t, 20*time.Hour)
	missing, err := env.service.StatAttachmentBlobs(ctx, []string{hash})
	require.NoError(t, err)
	require.Empty(t, missing, "the sender is told the Hub has it ...")

	env.advance(t, 10*time.Hour) // 30h after the upload, 10h after the confirmation
	deleted, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Zero(t, deleted, "... so it must still be there when the ask arrives")
	result := env.ask(t, "evt-1", hub.AskAttachment{Name: "reused.txt", Kind: hub.AttachmentFile, SHA256: hash})
	require.Empty(t, attachmentWarnings(result.Warnings))
}

func TestSweepAttachmentBlobs_ABlobFreedByARevisionIsSweptLater(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	env.ask(t, "evt-1", env.file(t, "plan.md", "v1"))
	env.advance(t, time.Hour)
	env.ask(t, "evt-2", env.file(t, "plan.md", "v2"))

	env.advance(t, 25*time.Hour)
	deleted, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, deleted, "only the replaced revision's content")
	_, err = env.service.ReadAttachmentBlob(ctx, sha("v1"), 100)
	require.ErrorIs(t, err, hub.ErrNotFound)
	_, err = env.service.ReadAttachmentBlob(ctx, sha("v2"), 100)
	require.NoError(t, err)
}

func TestSweepAttachmentBlobs_KeepsABlobReferencedOnlyByAFolderEntry(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	env.ask(t, "evt-1", env.folder(t, "report", map[string]string{"a.md": "only in a folder"}))

	env.advance(t, 72*time.Hour)
	deleted, err := env.service.SweepAttachmentBlobs(ctx)
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func TestSweepAttachmentBlobs_WithoutAStoreDoesNothing(t *testing.T) {
	service, _ := testService(t)
	deleted, err := service.SweepAttachmentBlobs(context.Background())
	require.NoError(t, err)
	require.Zero(t, deleted)
}

func TestAsk_ASenderThatCappedAFolderCanSayItTruncatedIt(t *testing.T) {
	env := newAttachmentEnv(t)
	folder := env.folder(t, "report", map[string]string{"a.md": "a"})
	folder.Truncated = true

	result := env.ask(t, "evt-1", folder)

	list := env.attachments(t, result.Ask.ID)
	require.Len(t, list, 1)
	require.True(t, list[0].Truncated, "the sender's own cap is persisted and returned")
	require.Empty(t, attachmentWarnings(result.Warnings), "the Hub did not truncate anything itself")
}
