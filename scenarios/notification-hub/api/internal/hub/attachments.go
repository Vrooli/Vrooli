package hub

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filekind"
	"notification-hub/internal/dbvalue"
)

// Decision-packet attachments: files and folders the operator opens inside the
// Hub. The Hub keeps its own copy of every byte (see blobs.go) and serves only
// those copies. The rules are in docs/concepts/DECISION_PACKETS.md
// ("Robustness rules" and "Attachments"): an attachment the Hub cannot accept is
// left out and reported as a PacketWarning; it never fails the ask.

// Attachment kinds and roles.
const (
	AttachmentFile   = "file"
	AttachmentFolder = "folder"

	AttachmentRoleEvidence  = "evidence"
	AttachmentRoleExplainer = "explainer"
)

// Source drift states reported on the read model.
const (
	SourceFresh   = "fresh"
	SourceChanged = "changed"
	SourceMissing = "missing"
	SourceUnknown = "unknown"
)

// Warning codes of attachments. They extend the packet check.
const (
	WarnAttachmentLimit         = "attachment_limit"
	WarnAttachmentBlobMissing   = "attachment_blob_missing"
	WarnAttachmentInvalid       = "attachment_invalid"
	WarnAttachmentRenamed       = "attachment_name_changed"
	WarnAttachmentSourceIgnore  = "attachment_source_ignored"
	WarnAttachmentRemoveUnknown = "attachment_remove_unknown"
	WarnFolderTruncated         = "attachment_folder_truncated"
	WarnExplainerLimit          = "attachment_explainer_limit"
)

const (
	maxAttachmentsPerAsk = 24
	// MaxAttachmentBlobBytes is the largest single file the Hub stores.
	MaxAttachmentBlobBytes = 64 << 20
	// maxAskAttachmentBytes bounds every referenced byte of one ask.
	maxAskAttachmentBytes = 256 << 20
	maxFolderEntries      = 500
	maxAttachmentName     = 96
	maxAttachmentLabel    = 200
	maxEntryPathBytes     = 512
	maxSourcePathBytes    = 4096
	maxSourceHostBytes    = 255
	maxMediaTypeBytes     = 255
	sniffBlobBytes        = filekind.SniffBytes

	// driftHashLimit is the largest source file the drift check hashes: a file
	// over the largest stored blob cannot match its copy. driftHashBudget bounds
	// the bytes hashed by one read of one ask, however many files it names; a file
	// left unhashed by it is reported as unknown.
	driftHashLimit  = MaxAttachmentBlobBytes
	driftHashBudget = 64 << 20
	// driftMemoLimit bounds the remembered hash comparisons.
	driftMemoLimit = 4096
)

// AskAttachmentEntry is one file of a folder attachment.
type AskAttachmentEntry struct {
	// Path is relative to the folder, "/"-separated, never absolute or ".."-ed.
	Path      string
	SHA256    string
	SizeBytes int64
	MediaType string
	// PreviewKind is set by the Hub (read model); ignored on a request.
	PreviewKind string
}

// AskAttachment is one file or folder of a decision packet. On a request the
// producer sets Name, Label, Kind, Role, the source fields and the content
// hashes; the Hub fills sizes, media types and preview kinds from its own
// store, and the drift fields on read.
type AskAttachment struct {
	Name  string
	Label string
	// Kind is AttachmentFile or AttachmentFolder.
	Kind string
	// Role is AttachmentRoleEvidence or AttachmentRoleExplainer.
	Role string
	// SourcePath and SourceHost say where the producer found the content. They
	// are shown to the operator and used only for the drift check; the Hub never
	// serves a host path.
	SourcePath string
	SourceHost string
	// SHA256 names a file's content. SizeBytes and MediaType are the Hub's own
	// (a request's values are ignored): the stored size and the Content-Type the
	// shared classifier gives the file.
	SHA256    string
	SizeBytes int64
	MediaType string
	// Entries are a folder's files.
	Entries []AskAttachmentEntry
	// Truncated reports that the folder had more files than the Hub keeps.
	Truncated bool

	// Read model, set by the Hub.
	PreviewKind     string
	SourceStatus    string
	SourceCheckedAt string
}

// attachmentState is the Service's attachment machinery. The zero value is
// usable: without a blob store, attachments are unavailable.
type attachmentState struct {
	cfg atomic.Pointer[attachmentConfig]
	// writeMu serializes storing a blob and deleting one, so the sweep can never
	// remove the bytes of a blob an upload has just confirmed.
	writeMu sync.Mutex

	driftMu sync.Mutex
	drift   map[driftKey]bool
}

type driftKey struct {
	path  string
	size  int64
	mtime int64
	sha   string
}

// localHostname is the host name the drift check compares sources against.
func localHostname() string {
	host, err := os.Hostname()
	if err != nil {
		return ""
	}
	return host
}

// attachmentDigest identifies what a producer attached: names, labels, roles,
// sources and content hashes, never the values the Hub derives. Two requests
// with the same digest attach the same thing.
func attachmentDigest(list []AskAttachment) string {
	if len(list) == 0 {
		return ""
	}
	type entry struct {
		Path   string `json:"p"`
		SHA256 string `json:"h"`
	}
	type item struct {
		Name       string  `json:"n"`
		Label      string  `json:"l"`
		Kind       string  `json:"k"`
		Role       string  `json:"r"`
		SourcePath string  `json:"sp,omitempty"`
		SourceHost string  `json:"sh,omitempty"`
		SHA256     string  `json:"h,omitempty"`
		Entries    []entry `json:"e,omitempty"`
		Truncated  bool    `json:"t,omitempty"`
	}
	items := make([]item, 0, len(list))
	for _, a := range list {
		it := item{Name: a.Name, Label: a.Label, Kind: a.Kind, Role: a.Role, SourcePath: a.SourcePath, SourceHost: a.SourceHost, SHA256: a.SHA256, Truncated: a.Truncated}
		for _, e := range a.Entries {
			it.Entries = append(it.Entries, entry{Path: e.Path, SHA256: e.SHA256})
		}
		items = append(items, it)
	}
	encoded, _ := json.Marshal(items)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// ---- normalization -------------------------------------------------------

// prepareAttachments turns a request's attachments into what the Hub stores:
// structure and limits first, then the Hub's own blob inventory. Anything it
// cannot accept is dropped and reported; it never returns an error for a
// packet problem, only for a failing store.
func (s *Service) prepareAttachments(ctx context.Context, spec AskSpec, check bool) (AskSpec, []PacketWarning, error) {
	if len(spec.Attachments) == 0 {
		spec.Attachments = nil
		return spec, nil, nil
	}
	if !spec.VerifiedProducer {
		// The Hub may retain bytes an anonymous caller uploads, but it must never
		// inspect a caller-selected host path while preparing or later reading the
		// ask. Drop source provenance and keep only the content-addressed copy.
		for i := range spec.Attachments {
			spec.Attachments[i].SourcePath = ""
			spec.Attachments[i].SourceHost = ""
		}
	}
	shaped, warnings := shapeAttachments(spec.Attachments)
	resolved, more, err := s.resolveAttachments(ctx, shaped, check)
	if err != nil {
		return spec, nil, err
	}
	spec.Attachments = resolved
	return spec, append(warnings, more...), nil
}

func attachmentWarning(code, field, message, severity string) PacketWarning {
	return PacketWarning{Code: code, Field: field, Message: message, Severity: severity}
}

// shapeAttachments applies every rule that needs no blob lookup: names, kinds,
// roles, sources, hashes and folder entries, then the count limit. An invalid
// attachment is dropped before the limit is counted, so it never takes the place
// of a valid one.
func shapeAttachments(in []AskAttachment) ([]AskAttachment, []PacketWarning) {
	var warnings []PacketWarning
	out := make([]AskAttachment, 0, min(len(in), maxAttachmentsPerAsk))
	taken := make(map[string]bool, len(in))
	for index, raw := range in {
		if len(out) == maxAttachmentsPerAsk {
			warnings = append(warnings, attachmentWarning(WarnAttachmentLimit, "attachments",
				fmt.Sprintf("%d attachments were over the limit of %d and were left out; attach fewer files or put them in a folder.", len(in)-index, maxAttachmentsPerAsk), WarningDropped))
			break
		}
		field := fmt.Sprintf("attachments[%d]", index)
		item, itemWarnings, ok := shapeAttachment(raw, field, index, taken)
		warnings = append(warnings, itemWarnings...)
		if ok {
			out = append(out, item)
		}
	}
	return out, warnings
}

func shapeAttachment(raw AskAttachment, field string, index int, taken map[string]bool) (AskAttachment, []PacketWarning, bool) {
	var warnings []PacketWarning
	drop := func(message string) (AskAttachment, []PacketWarning, bool) {
		return AskAttachment{}, append(warnings, attachmentWarning(WarnAttachmentInvalid, field, message, WarningDropped)), false
	}
	item := AskAttachment{Kind: strings.ToLower(strings.TrimSpace(raw.Kind))}
	if item.Kind == "" {
		// A sender that names entries means a folder.
		item.Kind = AttachmentFile
		if len(raw.Entries) > 0 {
			item.Kind = AttachmentFolder
		}
	}
	if item.Kind != AttachmentFile && item.Kind != AttachmentFolder {
		return drop(fmt.Sprintf("the attachment kind %q is not file or folder, so it was left out.", raw.Kind))
	}
	item.Role = strings.ToLower(strings.TrimSpace(raw.Role))
	switch item.Role {
	case "", AttachmentRoleEvidence:
		item.Role = AttachmentRoleEvidence
	case AttachmentRoleExplainer:
	default:
		item.Role = AttachmentRoleEvidence
		warnings = append(warnings, attachmentWarning(WarnAttachmentInvalid, field, fmt.Sprintf("the attachment role %q is unknown; it is listed as evidence.", raw.Role), WarningAdvice))
	}

	name, renamed := attachmentName(raw.Name, raw.SourcePath, index, item.Kind == AttachmentFolder)
	unique := uniqueAttachmentName(name, item.Kind == AttachmentFolder, taken)
	renamed = renamed || unique != name
	taken[unique] = true
	item.Name = unique
	if renamed {
		warnings = append(warnings, attachmentWarning(WarnAttachmentRenamed, field+".name",
			fmt.Sprintf("the attachment name %q is not allowed or already used, so it is shown as %q; names are 1-%d of letters, digits, '.', '_' and '-', unique within the ask.", strings.TrimSpace(raw.Name), unique, maxAttachmentName), WarningAdvice))
	}

	item.Label = strings.TrimSpace(raw.Label)
	if !utf8.ValidString(item.Label) || item.Label == "" {
		item.Label = unique
	}
	if utf8.RuneCountInString(item.Label) > maxAttachmentLabel {
		item.Label = string([]rune(item.Label)[:maxAttachmentLabel])
	}

	item.SourcePath = strings.TrimSpace(raw.SourcePath)
	if item.SourcePath != "" && (!filepath.IsAbs(item.SourcePath) || !utf8.ValidString(item.SourcePath) || len(item.SourcePath) > maxSourcePathBytes || strings.ContainsRune(item.SourcePath, 0)) {
		warnings = append(warnings, attachmentWarning(WarnAttachmentSourceIgnore, field+".source_path", "source_path must be an absolute path on the sending host, so it was ignored; the operator sees no origin for this attachment.", WarningAdvice))
		item.SourcePath = ""
	}
	item.SourceHost = strings.TrimSpace(raw.SourceHost)
	if !utf8.ValidString(item.SourceHost) || len(item.SourceHost) > maxSourceHostBytes {
		item.SourceHost = ""
	}

	if item.Kind == AttachmentFile {
		hash := normalizeBlobHash(raw.SHA256)
		if hash == "" {
			return drop("a file attachment needs the sha256 of its uploaded content, so it was left out.")
		}
		item.SHA256 = hash
		return item, warnings, true
	}

	entries, entryWarnings, truncated := shapeEntries(raw.Entries, field)
	warnings = append(warnings, entryWarnings...)
	if len(entries) == 0 {
		return drop("a folder attachment needs at least one valid file entry, so it was left out.")
	}
	item.Entries = entries
	item.Truncated = raw.Truncated || truncated
	if truncated {
		warnings = append(warnings, attachmentWarning(WarnFolderTruncated, field,
			fmt.Sprintf("the folder has more than %d files; only the first %d are kept.", maxFolderEntries, maxFolderEntries), WarningAdvice))
	}
	return item, warnings, true
}

// shapeEntries keeps a folder's valid, distinct entries, at most maxFolderEntries.
func shapeEntries(in []AskAttachmentEntry, field string) (entries []AskAttachmentEntry, warnings []PacketWarning, truncated bool) {
	seen := make(map[string]bool, len(in))
	invalid := 0
	for _, raw := range in {
		entryPath, ok := cleanEntryPath(raw.Path)
		hash := normalizeBlobHash(raw.SHA256)
		if !ok || hash == "" {
			invalid++
			continue
		}
		if seen[entryPath] {
			continue
		}
		if len(entries) == maxFolderEntries {
			truncated = true
			break
		}
		seen[entryPath] = true
		entries = append(entries, AskAttachmentEntry{Path: entryPath, SHA256: hash})
	}
	if invalid > 0 {
		warnings = append(warnings, attachmentWarning(WarnAttachmentInvalid, field+".entries",
			fmt.Sprintf("%d folder entries had an unsafe path or no sha256 and were left out; entry paths are relative, use '/', and never contain '..'.", invalid), WarningDropped))
	}
	return entries, warnings, truncated
}

// attachmentName turns a producer's name into a valid one: 1-96 of A-Z a-z 0-9
// . _ -, starting with a letter or digit. renamed reports that it changed.
func attachmentName(name, sourcePath string, index int, folder bool) (string, bool) {
	trimmed := strings.TrimSpace(name)
	candidate := trimmed
	if candidate == "" {
		candidate = path.Base(filepath.ToSlash(sourcePath))
	}
	var b strings.Builder
	for _, r := range candidate {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	cleaned := strings.TrimLeft(b.String(), "._-")
	cleaned = cutName(cleaned, maxAttachmentName, "", folder)
	if cleaned == "" {
		cleaned = fmt.Sprintf("attachment-%d", index+1)
	}
	return cleaned, cleaned != trimmed
}

// maxNameExtension is the longest suffix treated as a file extension when a
// name is cut or numbered; anything longer is part of the stem.
const maxNameExtension = 16

// splitName separates a file name's extension (folders have none).
func splitName(name string, folder bool) (stem, ext string) {
	if !folder {
		if ext = path.Ext(name); len(ext) > maxNameExtension || ext == name {
			ext = ""
		}
	}
	return strings.TrimSuffix(name, ext), ext
}

// cutName fits name plus suffix (placed before the extension) within limit
// bytes by cutting the stem, so the extension survives.
func cutName(name string, limit int, suffix string, folder bool) string {
	stem, ext := splitName(name, folder)
	if room := limit - len(suffix) - len(ext); len(stem) > room {
		stem = strings.TrimRight(trimTo(stem, max(room, 0)), "-.")
	}
	return stem + suffix + ext
}

// uniqueAttachmentName returns name, or name numbered before its extension
// (notes-2.md) when taken already holds it. The CLI numbers its duplicates the
// same way, so a name the sender already made unique is never renamed by the Hub.
func uniqueAttachmentName(name string, folder bool, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	for n := 2; ; n++ {
		if candidate := cutName(name, maxAttachmentName, fmt.Sprintf("-%d", n), folder); !taken[candidate] {
			return candidate
		}
	}
}

func trimTo(value string, max int) string {
	if len(value) > max {
		return value[:max]
	}
	return value
}

// cleanEntryPath accepts a relative, canonical, "/"-separated path of valid
// UTF-8 inside the folder.
func cleanEntryPath(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > maxEntryPathBytes || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\\") {
		return "", false
	}
	if strings.HasPrefix(value, "/") || path.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return "", false
	}
	return value, true
}

// normalizeBlobHash returns the lowercase hex SHA-256 in value, or "".
func normalizeBlobHash(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != sha256.Size*2 {
		return ""
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return value
}

// resolveAttachments checks every referenced blob against the Hub's inventory,
// drops what is not stored, takes sizes from the store, classifies each file
// with the shared classifier, and applies the byte and explainer limits.
func (s *Service) resolveAttachments(ctx context.Context, in []AskAttachment, check bool) ([]AskAttachment, []PacketWarning, error) {
	hashes := make([]string, 0, len(in))
	for _, a := range in {
		if a.Kind == AttachmentFile {
			hashes = append(hashes, a.SHA256)
		}
		for _, e := range a.Entries {
			hashes = append(hashes, e.SHA256)
		}
	}
	// Referencing a blob keeps the sweep away from it for another 24 hours, so
	// the window between this check and the ask's insert can never lose bytes.
	var blobs map[string]blobInfo
	if check {
		// A dry run uploads nothing and touches nothing: every file is taken as
		// stored, with the size the request states.
		blobs = declaredBlobs(in)
	} else {
		var err error
		if blobs, err = s.touchBlobs(ctx, hashes); err != nil {
			return nil, nil, err
		}
	}

	var warnings []PacketWarning
	out := make([]AskAttachment, 0, len(in))
	var total int64
	overLimit := 0
	explainerSeen := false
	for index, a := range in {
		field := fmt.Sprintf("attachments[%d]", index)
		resolved, missing, ok := s.resolveAttachment(ctx, a, blobs)
		if missing > 0 {
			subject := "the file"
			if a.Kind == AttachmentFolder {
				subject = fmt.Sprintf("%d files of the folder", missing)
			}
			warnings = append(warnings, attachmentWarning(WarnAttachmentBlobMissing, field,
				fmt.Sprintf("the Hub does not hold the uploaded content of %s %q, so it was left out; upload each file with PutAttachmentBlob before the ask.", subject, a.Name), WarningDropped))
		}
		if !ok {
			continue
		}
		size := attachmentBytes(resolved)
		if total+size > maxAskAttachmentBytes {
			overLimit++
			continue
		}
		total += size
		if resolved.Role == AttachmentRoleExplainer {
			switch {
			case resolved.Kind != AttachmentFile:
				resolved.Role = AttachmentRoleEvidence
				warnings = append(warnings, attachmentWarning(WarnExplainerLimit, field+".role", "an explainer must be a single HTML file, not a folder; it is listed as evidence.", WarningAdvice))
			case explainerSeen:
				resolved.Role = AttachmentRoleEvidence
				warnings = append(warnings, attachmentWarning(WarnExplainerLimit, field+".role", "an ask shows at most one explainer; this one is listed as evidence.", WarningAdvice))
			default:
				explainerSeen = true
			}
		}
		out = append(out, resolved)
	}
	if overLimit > 0 {
		warnings = append(warnings, attachmentWarning(WarnAttachmentLimit, "attachments",
			fmt.Sprintf("%d attachments went past the %d MiB limit for one ask and were left out; attach less, or link a summary.", overLimit, maxAskAttachmentBytes>>20), WarningDropped))
	}
	return out, warnings, nil
}

// declaredBlobs is the inventory a dry run assumes: each file at the size its
// attachment states.
func declaredBlobs(in []AskAttachment) map[string]blobInfo {
	blobs := make(map[string]blobInfo)
	for _, a := range in {
		if a.Kind == AttachmentFile {
			blobs[a.SHA256] = blobInfo{size: a.SizeBytes}
		}
		for _, e := range a.Entries {
			blobs[e.SHA256] = blobInfo{size: e.SizeBytes}
		}
	}
	return blobs
}

func attachmentBytes(a AskAttachment) int64 {
	if a.Kind == AttachmentFile {
		return a.SizeBytes
	}
	var total int64
	for _, e := range a.Entries {
		total += e.SizeBytes
	}
	return total
}

// resolveAttachment fills one attachment from the blob inventory. missing counts
// the files that were not stored; ok is false when nothing of it is left.
func (s *Service) resolveAttachment(ctx context.Context, a AskAttachment, blobs map[string]blobInfo) (_ AskAttachment, missing int, ok bool) {
	if a.Kind == AttachmentFile {
		blob, found := blobs[a.SHA256]
		if !found {
			return a, 1, false
		}
		a.SizeBytes = blob.size
		classifyAs := a.Name
		if base := path.Base(filepath.ToSlash(a.SourcePath)); a.SourcePath != "" && path.Ext(base) != "" {
			classifyAs = base
		}
		a.PreviewKind, a.MediaType = s.classifyBlob(ctx, classifyAs, a.SHA256)
		return a, 0, true
	}
	entries := make([]AskAttachmentEntry, 0, len(a.Entries))
	for _, e := range a.Entries {
		blob, found := blobs[e.SHA256]
		if !found {
			missing++
			continue
		}
		e.SizeBytes = blob.size
		e.PreviewKind, e.MediaType = s.classifyBlob(ctx, e.Path, e.SHA256)
		entries = append(entries, e)
	}
	a.Entries = entries
	a.PreviewKind = string(filekind.KindDirectory)
	return a, missing, len(entries) > 0
}

// classifyBlob picks the preview kind and the Content-Type the Hub serves one
// stored file with, from the shared classifier alone. A media type the sender
// declared is never served: it would let a producer choose how the operator's
// browser treats the file (text/html would run it), and a file the classifier
// cannot place is application/octet-stream.
func (s *Service) classifyBlob(ctx context.Context, name, sha string) (previewKind, mediaType string) {
	result := filekind.Classify(name, func() ([]byte, error) { return s.sniffBlob(ctx, sha) })
	return string(result.Kind), result.MIME
}

// ---- persistence ---------------------------------------------------------

// insertAttachmentsTx stores the attachments of a new ask.
func insertAttachmentsTx(ctx context.Context, tx dbTx, askID string, list []AskAttachment) error {
	if len(list) == 0 {
		return nil
	}
	return writeAttachmentsTx(ctx, tx, askID, list)
}

// replaceAttachmentsTx makes list the ask's attachments, replacing the rows of
// the revision it supersedes.
func replaceAttachmentsTx(ctx context.Context, tx dbTx, askID string, list []AskAttachment) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM ask_attachment_entries WHERE ask_id = ?`, askID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM ask_attachments WHERE ask_id = ?`, askID); err != nil {
		return err
	}
	return writeAttachmentsTx(ctx, tx, askID, list)
}

func writeAttachmentsTx(ctx context.Context, tx dbTx, askID string, list []AskAttachment) error {
	for position, a := range list {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ask_attachments (ask_id, name, position, label, kind, role, source_path, source_host, sha256, size_bytes, media_type, preview_kind, truncated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			askID, a.Name, position, a.Label, a.Kind, a.Role, a.SourcePath, a.SourceHost, a.SHA256, a.SizeBytes, a.MediaType, a.PreviewKind, dbvalue.BoolInt(a.Truncated)); err != nil {
			return fmt.Errorf("persist attachment %q: %w", a.Name, err)
		}
		for entryPosition, e := range a.Entries {
			if _, err := tx.ExecContext(ctx, `INSERT INTO ask_attachment_entries (ask_id, attachment_name, position, path, sha256, size_bytes, media_type, preview_kind) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				askID, a.Name, entryPosition, e.Path, e.SHA256, e.SizeBytes, e.MediaType, e.PreviewKind); err != nil {
				return fmt.Errorf("persist attachment %q entry %q: %w", a.Name, e.Path, err)
			}
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE asks SET attachment_count = ?, attachments_digest = ? WHERE id = ?`, len(list), attachmentDigest(list), askID)
	return err
}

// ---- read model ----------------------------------------------------------

// AskAttachments returns an ask's attachments with their read model: preview
// kinds and, when the sender ran on this host, whether each source still
// matches the stored copy. It never changes anything.
func (s *Service) AskAttachments(ctx context.Context, askID string) ([]AskAttachment, error) {
	return s.askAttachments(ctx, askID, true)
}

// AskAttachmentsUnchecked is AskAttachments without the drift check: every
// source status is unknown and no sender path is read. A reader other than the
// owner gets this, so the Hub never reads a host file on behalf of a caller that
// could use the result to probe the host (see docs/internal/SECURITY.md).
func (s *Service) AskAttachmentsUnchecked(ctx context.Context, askID string) ([]AskAttachment, error) {
	return s.askAttachments(ctx, askID, false)
}

func (s *Service) askAttachments(ctx context.Context, askID string, checkSources bool) ([]AskAttachment, error) {
	list, err := s.loadAttachments(ctx, strings.TrimSpace(askID))
	if err != nil || len(list) == 0 {
		return list, err
	}
	if !checkSources {
		for index := range list {
			list[index].SourceStatus, list[index].SourceCheckedAt = SourceUnknown, ""
		}
		return list, nil
	}
	host := localHostname()
	checkedAt := s.clock.Now().UTC().Format(time.RFC3339)
	budget := int64(driftHashBudget)
	for index := range list {
		list[index].SourceStatus, list[index].SourceCheckedAt = s.sourceStatus(list[index], host, checkedAt, &budget)
	}
	return list, nil
}

func (s *Service) loadAttachments(ctx context.Context, askID string) ([]AskAttachment, error) {
	return loadAttachmentsFrom(ctx, s.db, askID)
}

// rowQuerier is the read side of a database or transaction.
type rowQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// loadAttachmentsFrom reads an ask's attachments through q, so a write
// transaction can read them without a second connection.
func loadAttachmentsFrom(ctx context.Context, q rowQuerier, askID string) ([]AskAttachment, error) {
	rows, err := q.QueryContext(ctx, `SELECT name, label, kind, role, source_path, source_host, sha256, size_bytes, media_type, preview_kind, truncated FROM ask_attachments WHERE ask_id = ? ORDER BY position`, askID)
	if err != nil {
		return nil, err
	}
	var list []AskAttachment
	for rows.Next() {
		var a AskAttachment
		var truncated int
		if err := rows.Scan(&a.Name, &a.Label, &a.Kind, &a.Role, &a.SourcePath, &a.SourceHost, &a.SHA256, &a.SizeBytes, &a.MediaType, &a.PreviewKind, &truncated); err != nil {
			_ = rows.Close()
			return nil, err
		}
		a.Truncated = truncated != 0
		list = append(list, a)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	entryRows, err := q.QueryContext(ctx, `SELECT attachment_name, path, sha256, size_bytes, media_type, preview_kind FROM ask_attachment_entries WHERE ask_id = ? ORDER BY attachment_name, position`, askID)
	if err != nil {
		return nil, err
	}
	defer entryRows.Close()
	byName := make(map[string][]AskAttachmentEntry)
	for entryRows.Next() {
		var name string
		var e AskAttachmentEntry
		if err := entryRows.Scan(&name, &e.Path, &e.SHA256, &e.SizeBytes, &e.MediaType, &e.PreviewKind); err != nil {
			return nil, err
		}
		byName[name] = append(byName[name], e)
	}
	if err := entryRows.Err(); err != nil {
		return nil, err
	}
	for index := range list {
		list[index].Entries = byName[list[index].Name]
	}
	return list, nil
}

// sourceStatus compares an attachment's source with the stored copy. It runs
// only when the sender was on this host: another host's paths mean nothing
// here. The source is read for comparison and never served. A status is
// "fresh" only when the content was compared and matched; "unknown" when it
// could not be (another host, an unreadable path, or the hash budget ran out).
func (s *Service) sourceStatus(a AskAttachment, host, checkedAt string, budget *int64) (status, at string) {
	if a.SourcePath == "" || a.SourceHost == "" || host == "" || a.SourceHost != host {
		return SourceUnknown, ""
	}
	if a.Kind == AttachmentFile {
		return s.checkedStatus(s.compareSource(a.SourcePath, a.SHA256, a.SizeBytes, budget), checkedAt)
	}
	info, err := os.Stat(a.SourcePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return SourceMissing, checkedAt
	case err != nil:
		return SourceUnknown, ""
	case !info.IsDir():
		return SourceChanged, checkedAt
	}
	// A difference found in any file settles the folder as changed, whatever
	// the others could or could not be compared to.
	unknown := false
	for _, e := range a.Entries {
		switch s.compareSource(filepath.Join(a.SourcePath, filepath.FromSlash(e.Path)), e.SHA256, e.SizeBytes, budget) {
		case SourceFresh:
		case SourceUnknown:
			unknown = true
		default:
			return SourceChanged, checkedAt
		}
	}
	if unknown {
		return SourceUnknown, ""
	}
	return SourceFresh, checkedAt
}

// checkedStatus pairs a status with the time it was determined; an unknown
// status was not determined.
func (s *Service) checkedStatus(status, checkedAt string) (string, string) {
	if status == SourceUnknown {
		return status, ""
	}
	return status, checkedAt
}

// compareSource reports whether the file at sourcePath still holds the stored
// content: missing, changed (a different size, kind or hash), fresh (the hash
// was compared and matches), or unknown (the file could not be read, or it was
// not hashed because the budget of this read ran out).
func (s *Service) compareSource(sourcePath, wantSHA string, wantSize int64, budget *int64) string {
	info, err := os.Stat(sourcePath)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return SourceMissing
	case err != nil:
		return SourceUnknown
	case !info.Mode().IsRegular() || info.Size() != wantSize:
		return SourceChanged
	}
	key := driftKey{path: sourcePath, size: info.Size(), mtime: info.ModTime().UnixNano(), sha: wantSHA}
	st := &s.attachments
	st.driftMu.Lock()
	cached, known := st.drift[key]
	st.driftMu.Unlock()
	if !known {
		if info.Size() > *budget {
			return SourceUnknown
		}
		sum, hashErr := hashFile(sourcePath, driftHashLimit)
		if hashErr != nil {
			if errors.Is(hashErr, fs.ErrNotExist) {
				return SourceMissing
			}
			return SourceUnknown
		}
		*budget -= info.Size()
		cached = sum == wantSHA
		st.driftMu.Lock()
		if st.drift == nil || len(st.drift) >= driftMemoLimit {
			st.drift = make(map[driftKey]bool)
		}
		st.drift[key] = cached
		st.driftMu.Unlock()
	}
	if cached {
		return SourceFresh
	}
	return SourceChanged
}

var errHashLimit = errors.New("file is larger than the hash limit")

// hashFile returns the SHA-256 of the file, reading at most limit bytes: a file
// that grew past it since it was measured is errHashLimit, not read to the end.
func hashFile(name string, limit int64) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, limit+1))
	if err != nil {
		return "", err
	}
	if n > limit {
		return "", errHashLimit
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---- serving -------------------------------------------------------------

// AttachmentContent is one stored file opened for serving.
type AttachmentContent struct {
	// FileName is the name to offer the operator.
	FileName  string
	SHA256    string
	SizeBytes int64
	MediaType string
	// Content must be closed by the caller.
	Content io.ReadSeekCloser
}

// OpenAskAttachment opens one file of an ask's attachments for its recipient:
// the attachment's own file, or entryPath inside a folder. Anything the
// recipient may not read, and anything that does not exist, is ErrNotFound.
func (s *Service) OpenAskAttachment(ctx context.Context, askID, recipient, name, entryPath string) (AttachmentContent, error) {
	ask, err := s.GetAskForRecipient(ctx, askID, recipient)
	if err != nil {
		return AttachmentContent{}, err
	}
	var kind, sha, sourcePath, mediaType string
	var size int64
	err = s.db.QueryRowContext(ctx, `SELECT kind, sha256, source_path, size_bytes, media_type FROM ask_attachments WHERE ask_id = ? AND name = ?`, ask.ID, name).Scan(&kind, &sha, &sourcePath, &size, &mediaType)
	if errors.Is(err, sql.ErrNoRows) && name == CoverPageName && entryPath == "" {
		return s.openCoverPage(ctx, ask)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return AttachmentContent{}, ErrNotFound
	}
	if err != nil {
		return AttachmentContent{}, err
	}
	fileName := name
	switch {
	case kind == AttachmentFile && entryPath == "":
		if base := path.Base(filepath.ToSlash(sourcePath)); sourcePath != "" && base != "" && base != "." && base != "/" {
			fileName = base
		}
	case kind == AttachmentFolder && entryPath != "":
		err = s.db.QueryRowContext(ctx, `SELECT sha256, size_bytes, media_type FROM ask_attachment_entries WHERE ask_id = ? AND attachment_name = ? AND path = ?`, ask.ID, name, entryPath).Scan(&sha, &size, &mediaType)
		if errors.Is(err, sql.ErrNoRows) {
			return AttachmentContent{}, ErrNotFound
		}
		if err != nil {
			return AttachmentContent{}, err
		}
		fileName = path.Base(entryPath)
	default:
		return AttachmentContent{}, ErrNotFound
	}
	store, err := s.blobStore(ctx)
	if err != nil {
		return AttachmentContent{}, err
	}
	reader, _, err := store.Get(ctx, blobKey(sha))
	if err != nil {
		return AttachmentContent{}, fmt.Errorf("%w: attachment content is not stored", ErrNotFound)
	}
	return AttachmentContent{FileName: fileName, SHA256: sha, SizeBytes: size, MediaType: mediaType, Content: seekable(reader)}, nil
}

// bytesSeeker serves an in-memory blob.
type bytesSeeker struct{ *bytes.Reader }

func (bytesSeeker) Close() error { return nil }

// seekable returns the reader as a ReadSeekCloser. Stores whose readers cannot
// seek (the in-memory test store) are read fully, bounded by the blob limit.
func seekable(rc io.ReadCloser) io.ReadSeekCloser {
	if seeker, ok := rc.(io.ReadSeekCloser); ok {
		return seeker
	}
	data, _ := io.ReadAll(io.LimitReader(rc, MaxAttachmentBlobBytes+1))
	_ = rc.Close()
	return bytesSeeker{bytes.NewReader(data)}
}

// blobStore resolves the attachment store for this request (test-mode requests
// use their own root).
func (s *Service) blobStore(ctx context.Context) (blobstore.BlobStore, error) {
	cfg := s.attachments.cfg.Load()
	if cfg == nil || cfg.storeFor == nil {
		return nil, ErrAttachmentsUnavailable
	}
	return cfg.storeFor(ctx)
}
