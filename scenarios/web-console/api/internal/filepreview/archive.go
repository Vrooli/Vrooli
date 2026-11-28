package filepreview

// Archive inspection never extracts to disk or turns an entry name into a host
// path. All work (including skipped TAR bodies) goes through bounded readers.
import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	ArchiveInputLimit    int64 = 128 << 20
	ArchiveWorkLimit     int64 = 128 << 20
	ArchiveMemberLimit   int64 = 16 << 20
	archiveMetadataLimit       = 4 << 20
	archiveEntryLimit          = 5000
	archiveDepthLimit          = 3
	virtualByteLimit           = 64 << 20
	virtualEntryLimit          = 256
)

type ArchiveEntry struct {
	ID, Name, Status string
	SizeBytes        int64
}
type ArchiveListing struct {
	Format, Status string
	Entries        []ArchiveEntry
}
type archiveBudget struct {
	mu        sync.Mutex
	remaining int64
}

func (b *archiveBudget) charge(n int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if int64(n) > b.remaining {
		b.remaining = 0
		return archiveFailure("limit")
	}
	b.remaining -= int64(n)
	return nil
}

type archiveFailure string

func (e archiveFailure) Error() string { return string(e) }
func ArchiveFailureCode(err error) string {
	if err == nil {
		return "ready"
	}
	var a archiveFailure
	if errors.As(err, &a) {
		return string(a)
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, tar.ErrFieldTooLong) {
		return "limit"
	}
	if errors.Is(err, ErrPreviewNotFound) {
		return "expired"
	}
	var p *Error
	if errors.As(err, &p) && p.Code == CodeStale {
		return "stale"
	}
	return "corrupt"
}

// Fresh checks the original source even for a nested virtual member. No virtual
// preview can outlive, change session, or keep serving after its source changes.
func (s *Store) Fresh(e *Entry) error {
	root := e
	if e.RootID != "" {
		var err error
		root, err = s.Lookup(e.SessionID, e.RootID)
		if err != nil {
			return err
		}
	}
	info, err := os.Stat(root.ResolvedPath)
	if err != nil {
		return ErrPreviewNotFound
	}
	if (root.SourceInfo != nil && !os.SameFile(root.SourceInfo, info)) || !info.Mode().IsRegular() || info.Size() != root.SizeBytes || info.ModTime().UnixNano() != root.ModTimeUnixNano {
		return newError(CodeStale, "Archive changed; reopen the original file")
	}
	return nil
}

type archiveReader struct {
	ctx    context.Context
	r      io.Reader
	budget *archiveBudget
}

func (r *archiveReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	// Bound individual decompressor work and cancellation latency.
	if len(p) > 32768 {
		p = p[:32768]
	}
	n, err := r.r.Read(p)
	if e := r.budget.charge(n); e != nil {
		return 0, e
	}
	return n, err
}

type archiveReaderAt struct {
	ctx       context.Context
	r         io.ReaderAt
	remaining int64
}

func (r *archiveReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if int64(len(p)) > r.remaining {
		return 0, archiveFailure("limit")
	}
	r.remaining -= int64(len(p))
	return r.r.ReadAt(p, off)
}

// ListArchive scans a bounded metadata index. No member is inflated for a ZIP
// listing. TAR necessarily streams all bytes and validates gzip's checksum.
func (s *Store) ListArchive(ctx context.Context, sessionID, id string) ArchiveListing {
	ctx, finish, err := s.beginArchiveJob(ctx)
	if err != nil {
		return ArchiveListing{Status: ArchiveFailureCode(err)}
	}
	defer finish()
	list, _, err := s.inspectArchive(ctx, sessionID, id, "")
	if err != nil {
		return ArchiveListing{Status: ArchiveFailureCode(err)}
	}
	return list
}
func (s *Store) OpenArchiveEntry(ctx context.Context, sessionID, id, memberID string) (*Entry, string) {
	ctx, finish, err := s.beginArchiveJob(ctx)
	if err != nil {
		return nil, ArchiveFailureCode(err)
	}
	defer finish()
	list, data, err := s.inspectArchive(ctx, sessionID, id, memberID)
	if err != nil {
		return nil, ArchiveFailureCode(err)
	}
	var member *ArchiveEntry
	for i := range list.Entries {
		if list.Entries[i].ID == memberID {
			member = &list.Entries[i]
			break
		}
	}
	if member == nil {
		return nil, "missing"
	}
	if member.Status != "ready" {
		return nil, "blocked"
	}
	source, err := s.Lookup(sessionID, id)
	if err != nil {
		return nil, "expired"
	}
	cls := classify(member.Name, func() ([]byte, error) { return data[:min(len(data), 512)], nil })
	// Active documents stay inert and cannot resolve relative host resources.
	if cls.kind == KindMarkdown {
		cls.kind = KindCode
		cls.mimeType = "text/plain; charset=utf-8"
	}
	if cls.kind == KindArchive && source.Depth+1 >= archiveDepthLimit {
		return nil, "depth"
	}
	previewID, err := s.newID()
	if err != nil {
		return nil, "busy"
	}
	// Construct the complete immutable record before publishing it. A lookup
	// must never observe an intermediate filesystem-shaped member capability.
	rootID := source.RootID
	if rootID == "" {
		rootID = source.ID
	}
	e := &Entry{
		ID: previewID, SessionID: sessionID, ResolvedPath: source.ResolvedPath + "!/" + member.Name,
		Basename: path.Base(member.Name), Kind: cls.kind, MIMEType: cls.mimeType,
		SizeBytes: int64(len(data)), CanDownload: true, CanPreview: cls.kind.CanPreview(),
		TextContentAvailable: cls.kind.TextContentAvailable(), Data: bytes.Clone(data),
		RootID: rootID, Depth: source.Depth + 1, ExpiresAt: source.ExpiresAt, Budget: source.Budget,
	}
	s.mu.Lock()
	s.sequence++
	e.Sequence = s.sequence
	s.byID[previewID] = e

	s.virtualBytes += int64(len(data))
	// Only virtual handles are evicted, oldest first. Callers already holding an
	// immutable entry can finish their bounded response without a data race.
	for s.virtualBytes > virtualByteLimit || s.virtualCountLocked() > virtualEntryLimit {
		var oldest *Entry
		for _, v := range s.byID {
			if v.RootID != "" && v.ID != e.ID && v.activeReaders == 0 && (oldest == nil || v.Sequence < oldest.Sequence) {
				oldest = v
			}
		}
		if oldest == nil {
			s.deleteLocked(e.ID)
			s.mu.Unlock()
			return nil, "busy"
		}
		s.deleteLocked(oldest.ID)
	}
	s.mu.Unlock()
	return e, "ready"
}
func (s *Store) virtualCountLocked() int {
	n := 0
	for _, e := range s.byID {
		if e.RootID != "" {
			n++
		}
	}
	return n
}
func (s *Store) deleteLocked(id string) {
	if e := s.byID[id]; e != nil && e.activeReaders > 0 {
		return
	}
	if e := s.byID[id]; e != nil && e.RootID != "" {
		s.virtualBytes -= int64(len(e.Data))
	}
	delete(s.byID, id)
}

func (s *Store) inspectArchive(ctx context.Context, sessionID, id, selected string) (ArchiveListing, []byte, error) {
	e, err := s.Lookup(sessionID, id)
	if err != nil {
		return ArchiveListing{}, nil, err
	}
	if e.Kind != KindArchive {
		return ArchiveListing{}, nil, archiveFailure("unsupported")
	}
	if err = s.Fresh(e); err != nil {
		return ArchiveListing{}, nil, err
	}
	if e.SizeBytes > ArchiveInputLimit {
		return ArchiveListing{}, nil, archiveFailure("limit")
	}
	var source io.ReaderAt
	if e.RootID != "" {
		source = bytes.NewReader(e.Data)
	} else {
		f, openErr := os.Open(e.ResolvedPath)
		if openErr != nil {
			return ArchiveListing{}, nil, openErr
		}
		defer f.Close()
		info, statErr := f.Stat()
		if statErr != nil {
			return ArchiveListing{}, nil, statErr
		}
		if (e.SourceInfo != nil && !os.SameFile(e.SourceInfo, info)) || info.Size() != e.SizeBytes || info.ModTime().UnixNano() != e.ModTimeUnixNano || !info.Mode().IsRegular() {
			return ArchiveListing{}, nil, archiveFailure("stale")
		}
		source = f
	}
	bounded := &archiveReaderAt{ctx: ctx, r: source, remaining: ArchiveInputLimit * 2}
	var magic [4]byte
	_, _ = bounded.ReadAt(magic[:], 0)
	var list ArchiveListing
	var data []byte
	lower := strings.ToLower(e.Basename)
	switch {
	case bytes.Equal(magic[:2], []byte("PK")) || strings.HasSuffix(lower, ".zip"):
		list, data, err = inspectZIP(ctx, bounded, e.SizeBytes, e.Budget, selected)
	case magic[0] == 0x1f && magic[1] == 0x8b:
		gz, gerr := gzip.NewReader(io.NewSectionReader(bounded, 0, e.SizeBytes))
		if gerr != nil {
			return list, nil, gerr
		}
		defer gz.Close()
		list, data, err = inspectTAR(ctx, gz, e.Budget, selected)
		list.Format = "TAR.GZ"
	case strings.HasSuffix(lower, ".tar"):
		list, data, err = inspectTAR(ctx, io.NewSectionReader(bounded, 0, e.SizeBytes), e.Budget, selected)
		list.Format = "TAR"
	default:
		err = archiveFailure("unsupported")
	}
	if err == nil {
		err = s.Fresh(e)
	}
	return list, data, err
}

// Preflight counts actual central-directory records before archive/zip can
// allocate an index. ZIP64 and multipart need a separate bounded implementation.
func zipDirectory(r io.ReaderAt, size int64) error {
	if size < 22 {
		return zip.ErrFormat
	}
	tail := make([]byte, min(size, 65557))
	if _, err := r.ReadAt(tail, size-int64(len(tail))); err != nil {
		return err
	}
	offset := -1
	for i := len(tail) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(tail[i:]) == 0x06054b50 {
			if i+22+int(binary.LittleEndian.Uint16(tail[i+20:])) != len(tail) {
				return zip.ErrFormat
			}
			offset = i
			break
		}
	}
	if offset < 0 {
		return zip.ErrFormat
	}
	end := tail[offset:]
	count := int(binary.LittleEndian.Uint16(end[10:]))
	n := int64(binary.LittleEndian.Uint32(end[12:]))
	start := int64(binary.LittleEndian.Uint32(end[16:]))
	endPos := size - int64(len(tail)) + int64(offset)
	if binary.LittleEndian.Uint16(end[4:]) != 0 || binary.LittleEndian.Uint16(end[6:]) != 0 || binary.LittleEndian.Uint16(end[8:]) != uint16(count) || count == 65535 || n == 65535 || n == 0xffffffff || start == 0xffffffff {
		return archiveFailure("unsupported")
	}
	if n > archiveMetadataLimit || count > archiveEntryLimit {
		return archiveFailure("limit")
	}
	if start < 0 || start+n != endPos {
		return zip.ErrFormat
	}
	data := make([]byte, n)
	if _, err := r.ReadAt(data, start); err != nil {
		return err
	}
	actual := 0
	for pos := 0; pos < len(data); {
		if len(data)-pos < 46 || binary.LittleEndian.Uint32(data[pos:]) != 0x02014b50 {
			return zip.ErrFormat
		}
		name := int(binary.LittleEndian.Uint16(data[pos+28:]))
		extra := int(binary.LittleEndian.Uint16(data[pos+30:]))
		comment := int(binary.LittleEndian.Uint16(data[pos+32:]))
		next := pos + 46 + name + extra + comment
		if next > len(data) {
			return zip.ErrFormat
		}
		if binary.LittleEndian.Uint16(data[pos+34:]) != 0 {
			return archiveFailure("unsupported")
		}
		pos = next
		actual++
		if actual > archiveEntryLimit {
			return archiveFailure("limit")
		}
	}
	if actual != count {
		return zip.ErrFormat
	}
	return nil
}
func inspectZIP(ctx context.Context, r io.ReaderAt, size int64, budget *archiveBudget, selected string) (ArchiveListing, []byte, error) {
	list := ArchiveListing{Format: "ZIP", Status: "ready"}
	if err := zipDirectory(r, size); err != nil {
		return list, nil, err
	}
	z, err := zip.NewReader(r, size)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return list, nil, err
	}
	var data []byte
	metadata := 0
	for i, f := range z.File {
		if err = ctx.Err(); err != nil {
			return list, nil, err
		}
		status := "ready"
		mode := f.Mode()
		switch {
		case mode.IsDir():
			status = "directory"
		case mode&os.ModeSymlink != 0:
			status = "link"
		case !mode.IsRegular():
			status = "special"
		case f.Flags&1 != 0:
			status = "encrypted"
		case f.Method != zip.Store && f.Method != zip.Deflate:
			status = "unsupported"
		case f.UncompressedSize64 > uint64(ArchiveMemberLimit):
			status = "too_large"
		}
		name := portableArchiveName(f.Name)
		if mode.IsDir() && (name == "" || name == ".") {
			continue
		}
		if !safeArchiveName(name) {
			status = "unsafe"
		}
		name = strings.ToValidUTF8(name, "�")
		metadata += len(name) + 128
		if metadata > archiveMetadataLimit {
			return list, nil, archiveFailure("limit")
		}
		e := ArchiveEntry{ID: strconv.Itoa(i), Name: strings.TrimSuffix(name, "/"), Status: status, SizeBytes: int64(min(f.UncompressedSize64, uint64(1<<63-1)))}
		list.Entries = append(list.Entries, e)
		if e.ID == selected && status == "ready" {
			body, err := f.Open()
			if err != nil {
				return list, nil, err
			}
			data, err = io.ReadAll(io.LimitReader(&archiveReader{ctx: ctx, r: body, budget: budget}, ArchiveMemberLimit+1))
			closeErr := body.Close()
			if err != nil {
				return list, nil, err
			}
			if closeErr != nil {
				return list, nil, closeErr
			}
			if int64(len(data)) > ArchiveMemberLimit {
				return list, nil, archiveFailure("limit")
			}
		}
	}
	return list, data, nil
}
func inspectTAR(ctx context.Context, source io.Reader, budget *archiveBudget, selected string) (ArchiveListing, []byte, error) {
	list := ArchiveListing{Status: "ready"}
	raw := &archiveReader{ctx: ctx, r: source, budget: budget}
	tr := tar.NewReader(raw)
	metadata := 0
	physical := 0
	var data []byte
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return list, nil, err
		}
		if physical >= archiveEntryLimit {
			return list, nil, archiveFailure("limit")
		}
		ordinal := physical
		physical++
		if h.Typeflag == tar.TypeDir && (portableArchiveName(h.Name) == "" || h.Name == ".") {
			continue
		}
		metadata += len(h.Name) + len(h.Linkname) + len(h.Uname) + len(h.Gname) + 128
		for k, v := range h.PAXRecords {
			metadata += len(k) + len(v)
			if strings.HasPrefix(k, "GNU.sparse") {
				return list, nil, archiveFailure("unsupported")
			}
		}
		if metadata > archiveMetadataLimit {
			return list, nil, archiveFailure("limit")
		}
		status := "ready"
		switch h.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
		case tar.TypeDir:
			status = "directory"
		case tar.TypeLink, tar.TypeSymlink:
			status = "link"
		case tar.TypeGNUSparse:
			return list, nil, archiveFailure("unsupported")
		default:
			status = "special"
		}
		if h.Size > ArchiveMemberLimit && status == "ready" {
			status = "too_large"
		}
		name := portableArchiveName(h.Name)
		if !safeArchiveName(name) {
			status = "unsafe"
		}
		displayName := strings.ToValidUTF8(name, "�")
		metadata += len(displayName) - len(name)
		if metadata > archiveMetadataLimit {
			return list, nil, archiveFailure("limit")
		}
		e := ArchiveEntry{ID: strconv.Itoa(ordinal), Name: strings.TrimSuffix(displayName, "/"), Status: status, SizeBytes: h.Size}
		list.Entries = append(list.Entries, e)
		if e.ID == selected && status == "ready" {
			data, err = io.ReadAll(io.LimitReader(tr, ArchiveMemberLimit+1))
			if int64(len(data)) > ArchiveMemberLimit {
				return list, nil, archiveFailure("limit")
			}
		} else {
			_, err = io.Copy(io.Discard, io.LimitReader(tr, ArchiveWorkLimit+1))
		}
		if err != nil {
			return list, nil, err
		}
	}
	// TAR end markers precede gzip's trailer. Drain it under the same budget to
	// verify checksums/truncation; only zero padding is accepted after the TAR.
	buf := make([]byte, 32768)
	for {
		n, err := raw.Read(buf)
		for _, v := range buf[:n] {
			if v != 0 {
				return list, nil, tar.ErrHeader
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return list, nil, err
		}
	}
	return list, data, nil
}
func safeArchiveName(name string) bool {
	if len(name) == 0 || len(name) > 4096 || !utf8.ValidString(name) || strings.HasPrefix(name, "/") || strings.ContainsAny(name, "\\:<>\"|?*") {
		return false
	}
	for _, c := range name {
		if unicode.IsControl(c) || unicode.In(c, unicode.Cf) {
			return false
		}
	}
	parts := strings.Split(strings.TrimSuffix(name, "/"), "/")
	if len(parts) > 32 {
		return false
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || strings.HasSuffix(p, ".") || strings.HasSuffix(p, " ") {
			return false
		}
		base := strings.ToUpper(strings.SplitN(p, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return false
		}
	}
	return true
}

// Leading ./ is conventional in TARs created from a working directory. Removing
// that prefix changes no location; interior dot/traversal components stay blocked.
func portableArchiveName(name string) string {
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	return name
}

// Admission spans parsing, capability publication and eviction, keeping parsed
// buffers within the same two-job bound while they wait for the store mutex.
func (s *Store) beginArchiveJob(ctx context.Context) (context.Context, func(), error) {
	select {
	case s.archiveJobs <- struct{}{}:
	default:
		return ctx, nil, archiveFailure("busy")
	}
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	return bounded, func() { cancel(); <-s.archiveJobs }, nil
}
