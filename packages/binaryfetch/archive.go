package binaryfetch

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ErrNoBinaryInArchive is returned when the requested binPath (or, when binPath
// is empty, any single regular file) cannot be located in the archive.
var ErrNoBinaryInArchive = errors.New("binaryfetch: binary not found in archive")

// maxArchiveEntryBytes bounds a single extracted file to guard against a
// decompression bomb in a fetched archive. 2 GiB comfortably covers large
// model-adjacent binaries while refusing absurd entries.
const maxArchiveEntryBytes int64 = 2 << 30

// ArchiveDecompressor opens a stream for a format that is not available in
// the standard library. The caller owns the optional dependency and registers
// it at process startup; binaryfetch itself stays standard-library-only.
type ArchiveDecompressor func(io.Reader) (io.ReadCloser, error)

var archiveDecompressors = struct {
	sync.RWMutex
	items map[string]ArchiveDecompressor
}{items: map[string]ArchiveDecompressor{}}

// RegisterArchiveDecompressor registers a decompressor by archive name and
// returns an unregister function for scoped embeddings and tests.
func RegisterArchiveDecompressor(format string, decompressor ArchiveDecompressor) func() {
	key := normalizeArchive(format)
	archiveDecompressors.Lock()
	previous, hadPrevious := archiveDecompressors.items[key]
	if decompressor == nil {
		delete(archiveDecompressors.items, key)
	} else {
		archiveDecompressors.items[key] = decompressor
	}
	archiveDecompressors.Unlock()
	return func() {
		archiveDecompressors.Lock()
		defer archiveDecompressors.Unlock()
		if hadPrevious {
			archiveDecompressors.items[key] = previous
		} else {
			delete(archiveDecompressors.items, key)
		}
	}
}

func registeredArchiveDecompressor(format string) (ArchiveDecompressor, bool) {
	archiveDecompressors.RLock()
	decompressor, ok := archiveDecompressors.items[normalizeArchive(format)]
	archiveDecompressors.RUnlock()
	return decompressor, ok
}

// extractBinary pulls the binary identified by binPath out of the archive at
// archivePath and writes it to destFile. When binPath is empty it selects the
// archive's sole regular file (erroring if there are zero or many). format is
// "tar.gz", "tar.bz2", or "zip".
func extractBinary(archivePath, format, binPath, destFile string) error {
	switch format {
	case "tar.gz", "tgz":
		return extractFromTarGz(archivePath, binPath, destFile)
	case "tar.bz2", "tbz2", "tbz":
		return extractFromTarBzip2(archivePath, binPath, destFile)
	case "tar.zst":
		return extractFromTarZstd(archivePath, binPath, destFile)
	case "zip":
		return extractFromZip(archivePath, binPath, destFile)
	default:
		return fmt.Errorf("binaryfetch: unsupported archive format %q", format)
	}
}

func extractFromTarZstd(archivePath, binPath, destFile string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	decompressor, ok := registeredArchiveDecompressor("tar.zst")
	if !ok {
		f.Close()
		return fmt.Errorf("binaryfetch: no decompressor registered for tar.zst")
	}
	stream, err := decompressor(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("binaryfetch: open tar.zst: %w", err)
	}
	defer func() { _ = stream.Close(); _ = f.Close() }()
	return extractFromTarReader(tar.NewReader(stream), binPath, destFile, archivePath, "tar.zst")
}

func extractFromTarReader(tr *tar.Reader, binPath, destFile, archivePath, format string) error {
	var candidates []string
	matchName := filepath.Clean(binPath)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("binaryfetch: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Clean(hdr.Name)
		if binPath != "" {
			if name == matchName || filepath.Base(name) == filepath.Base(matchName) {
				return writeReader(tr, destFile)
			}
			continue
		}
		candidates = append(candidates, name)
	}
	if binPath != "" {
		return fmt.Errorf("%w: %q not in archive", ErrNoBinaryInArchive, binPath)
	}
	if len(candidates) != 1 {
		return fmt.Errorf("%w: archive has %d regular files; set binPath to disambiguate", ErrNoBinaryInArchive, len(candidates))
	}
	// The stream is exhausted; reopen and select the sole file deterministically.
	if format == "tar.zst" {
		return extractFromTarZstd(archivePath, candidates[0], destFile)
	}
	return nil
}

func extractFromTarGz(archivePath, binPath, destFile string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("binaryfetch: open gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var (
		candidates []string
		matchName  = filepath.Clean(binPath)
	)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("binaryfetch: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Clean(hdr.Name)
		if binPath != "" {
			if name == matchName || filepath.Base(name) == filepath.Base(matchName) {
				return writeReader(tr, destFile)
			}
			continue
		}
		candidates = append(candidates, name)
		// Defer the actual copy until we know it's the sole candidate by
		// rewinding is not possible with a stream; instead, when binPath is
		// empty we require exactly one regular file, so copy the first and
		// validate the count afterwards by continuing to scan.
	}
	if binPath != "" {
		return fmt.Errorf("%w: %q not in archive", ErrNoBinaryInArchive, binPath)
	}
	// binPath empty: re-scan to enforce exactly one regular file, then extract it.
	if len(candidates) != 1 {
		return fmt.Errorf("%w: archive has %d regular files; set binPath to disambiguate", ErrNoBinaryInArchive, len(candidates))
	}
	return extractFromTarGz(archivePath, candidates[0], destFile)
}

func extractFromTarBzip2(archivePath, binPath, destFile string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	decompressed := bzip2.NewReader(f)
	tr := tar.NewReader(decompressed)
	var (
		candidates []string
		matchName  = filepath.Clean(binPath)
	)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("binaryfetch: read tar.bz2: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Clean(hdr.Name)
		if binPath != "" {
			if name == matchName || filepath.Base(name) == filepath.Base(matchName) {
				err := writeReader(tr, destFile)
				_ = f.Close()
				return err
			}
			continue
		}
		candidates = append(candidates, name)
	}
	_ = f.Close()
	if binPath != "" {
		return fmt.Errorf("%w: %q not in archive", ErrNoBinaryInArchive, binPath)
	}
	if len(candidates) != 1 {
		return fmt.Errorf("%w: archive has %d regular files; set binPath to disambiguate", ErrNoBinaryInArchive, len(candidates))
	}
	return extractFromTarBzip2(archivePath, candidates[0], destFile)
}

func extractFromZip(archivePath, binPath, destFile string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("binaryfetch: open zip: %w", err)
	}
	defer zr.Close()

	var candidates []*zip.File
	matchName := filepath.Clean(binPath)
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		name := filepath.Clean(zf.Name)
		if binPath != "" {
			if name == matchName || filepath.Base(name) == filepath.Base(matchName) {
				return copyZipEntry(zf, destFile)
			}
			continue
		}
		candidates = append(candidates, zf)
	}
	if binPath != "" {
		return fmt.Errorf("%w: %q not in archive", ErrNoBinaryInArchive, binPath)
	}
	if len(candidates) != 1 {
		return fmt.Errorf("%w: archive has %d regular files; set binPath to disambiguate", ErrNoBinaryInArchive, len(candidates))
	}
	return copyZipEntry(candidates[0], destFile)
}

// extractAll extracts every regular file in the archive at archivePath into
// destDir, preserving relative paths. Directory entries are created as needed.
// Path-traversal entries (those that resolve outside destDir, e.g. via "..") are
// rejected. format is "tar.gz", "tar.bz2", or "zip".
func extractAll(archivePath, format, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("binaryfetch: create extract dir: %w", err)
	}
	switch format {
	case "tar.gz", "tgz":
		return extractAllTarGz(archivePath, destDir)
	case "tar.bz2", "tbz2", "tbz":
		return extractAllTarBzip2(archivePath, destDir)
	case "tar.zst":
		return extractAllTarZstd(archivePath, destDir)
	case "zip":
		return extractAllZip(archivePath, destDir)
	default:
		return fmt.Errorf("binaryfetch: unsupported archive format %q", format)
	}
}

func extractAllTarZstd(archivePath, destDir string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	decompressor, ok := registeredArchiveDecompressor("tar.zst")
	if !ok {
		f.Close()
		return fmt.Errorf("binaryfetch: no decompressor registered for tar.zst")
	}
	stream, err := decompressor(f)
	if err != nil {
		f.Close()
		return fmt.Errorf("binaryfetch: open tar.zst: %w", err)
	}
	defer func() { _ = stream.Close(); _ = f.Close() }()
	return extractAllTarReader(tar.NewReader(stream), destDir)
}

// extractTarLayer extracts an uncompressed OCI tar layer into destDir. OCI
// layers are handled separately from declared archive formats because the
// layer digest is the image reference's content address.
func extractTarLayer(archivePath, destDir string) error {
	file, err := os.Open(archivePath) //nolint:gosec // fetched into a controlled temp dir
	if err != nil {
		return err
	}
	defer file.Close()
	return extractAllTarReader(tar.NewReader(file), destDir)
}

func extractAllTarReader(tr *tar.Reader, destDir string) error {
	_, err := walkTar(tr, destDir, ExtractOptions{}, true)
	return err
}

// safeJoin joins destDir with a (possibly attacker-controlled) relative entry
// name, rejecting any name that is absolute or contains a ".." component (which
// could escape destDir) and any result that lands outside destDir.
func safeJoin(destDir, name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", &ArchiveError{Violation: ViolationTraversal, Entry: name, Detail: "escapes extract dir"}
	}
	target := filepath.Join(destDir, cleaned)
	if target != destDir && !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", &ArchiveError{Violation: ViolationTraversal, Entry: name, Detail: "escapes extract dir"}
	}
	return target, nil
}

func extractAllTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("binaryfetch: open gzip: %w", err)
	}
	defer gz.Close()

	return extractAllTarReader(tar.NewReader(gz), destDir)
}

func extractAllTarBzip2(archivePath, destDir string) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is fetched into a controlled temp dir
	if err != nil {
		return err
	}
	decompressed := bzip2.NewReader(f)
	err = extractAllTarReader(tar.NewReader(decompressed), destDir)
	_ = f.Close()
	if err != nil {
		return fmt.Errorf("binaryfetch: read tar.bz2: %w", err)
	}
	return nil
}

// resolveSafeSymlink returns where a symlink entry would point once placed
// under root, rejecting absolute targets and any target that resolves outside
// root. It performs no filesystem access, so the dry pass and the real
// extraction share exactly one escape rule.
func resolveSafeSymlink(root, name, linkname string) (string, error) {
	if filepath.IsAbs(filepath.FromSlash(linkname)) {
		return "", &ArchiveError{Violation: ViolationSymlinkEscape, Entry: name, Detail: "symlink has an absolute target"}
	}
	resolved, err := safeJoin(root, filepath.ToSlash(filepath.Join(filepath.Dir(name), filepath.FromSlash(linkname))))
	if err != nil {
		return "", &ArchiveError{Violation: ViolationSymlinkEscape, Entry: name, Detail: "symlink target escapes extract dir"}
	}
	return resolved, nil
}

func createSafeSymlink(root, name, linkname, target string) error {
	resolved, err := resolveSafeSymlink(root, name, linkname)
	if err != nil {
		return err
	}
	if _, err := os.Stat(resolved); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("binaryfetch: inspect symlink %q target: %w", name, err)
	}
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("binaryfetch: replace symlink %q: %w", name, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("binaryfetch: mkdir symlink parent %q: %w", name, err)
	}
	if err := os.Symlink(linkname, target); err != nil {
		return fmt.Errorf("binaryfetch: create symlink %q: %w", name, err)
	}
	return nil
}

func extractAllZip(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("binaryfetch: open zip: %w", err)
	}
	defer zr.Close()
	_, err = walkZip(zr.File, destDir, ExtractOptions{}, true)
	return err
}

func copyZipEntry(zf *zip.File, destFile string) error {
	rc, err := zf.Open()
	if err != nil {
		return fmt.Errorf("binaryfetch: open zip entry: %w", err)
	}
	defer rc.Close()
	return writeReader(rc, destFile)
}

func writeReader(r io.Reader, destFile string) error {
	return writeReaderMode(r, destFile, 0o600)
}

func writeReaderMode(r io.Reader, destFile string, mode os.FileMode) error {
	return writeReaderModeLimit(r, destFile, mode, maxArchiveEntryBytes)
}

func writeReaderModeLimit(r io.Reader, destFile string, mode os.FileMode, limit int64) error {
	out, err := os.OpenFile(destFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) //nolint:gosec // destFile is validated by safeJoin under a controlled dir
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(r, limit)); err != nil {
		out.Close()
		return fmt.Errorf("binaryfetch: extract entry: %w", err)
	}
	if err := out.Close(); err != nil {
		return err
	}
	// OpenFile mode is masked by umask; chmod so the entry's executable bit (e.g.
	// the entry binary in a dir-layout tree) survives.
	if err := os.Chmod(destFile, mode); err != nil {
		return fmt.Errorf("binaryfetch: chmod entry: %w", err)
	}
	return nil
}

// normalizeArchive maps a declared archive value to the canonical format string,
// returning "" for a raw (non-archived) binary.
func normalizeArchive(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return ""
	case "tar.gz", "tgz":
		return "tar.gz"
	case "tar.bz2", "tbz2", "tbz":
		return "tar.bz2"
	case "zip":
		return "zip"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

// ArchiveViolation classifies why a bounded walk refused an archive. Callers
// map these onto their own reason codes; the strings are stable.
type ArchiveViolation string

const (
	ViolationTraversal        ArchiveViolation = "traversal"
	ViolationSymlinkEscape    ArchiveViolation = "symlink_escape"
	ViolationTooLarge         ArchiveViolation = "too_large"
	ViolationEntryLimit       ArchiveViolation = "entry_limit"
	ViolationUnsupportedEntry ArchiveViolation = "unsupported_entry"
	ViolationDeadline         ArchiveViolation = "deadline"
)

// ArchiveError is the typed refusal returned by the bounded walkers and by the
// shared path/symlink rules every extractor in this package uses.
type ArchiveError struct {
	Violation ArchiveViolation
	Entry     string
	Detail    string
}

func (e *ArchiveError) Error() string {
	if e.Entry == "" {
		return "binaryfetch: archive " + string(e.Violation) + ": " + e.Detail
	}
	return fmt.Sprintf("binaryfetch: archive entry %q %s (%s)", e.Entry, e.Detail, e.Violation)
}

// ExtractOptions bounds an archive walk. A zero MaxEntries or MaxExpandedBytes
// means unbounded; a zero MaxEntryBytes keeps the package's historical
// per-entry cap; a zero Deadline means no time budget.
type ExtractOptions struct {
	MaxEntries       int64
	MaxExpandedBytes int64
	MaxEntryBytes    int64
	Deadline         time.Time
}

func (o ExtractOptions) entryLimit() int64 {
	if o.MaxEntryBytes > 0 {
		return o.MaxEntryBytes
	}
	return maxArchiveEntryBytes
}

// ArchiveSummary reports what a bounded walk observed. Inspect and Extract
// return the same shape so a dry pass can be compared with the real one.
type ArchiveSummary struct {
	Entries       int64 `json:"entries"`
	ExpandedBytes int64 `json:"expanded_bytes"`
	Files         int64 `json:"files"`
	Directories   int64 `json:"directories"`
	Symlinks      int64 `json:"symlinks"`
	Hardlinks     int64 `json:"hardlinks"`
}

// InspectArchiveBounded walks every entry of the archive enforcing the same
// path, symlink, entry-type, count, size and time rules as
// ExtractArchiveBounded, without writing anything. It is the refuse-before-
// write pass a target performs on a release bundle. Supported formats: tar,
// tar.gz, tar.bz2, tar.zst (when a decompressor is registered) and zip.
func InspectArchiveBounded(archivePath, format string, opts ExtractOptions) (ArchiveSummary, error) {
	return walkArchive(archivePath, format, inspectRoot, opts, false)
}

// ExtractArchiveBounded extracts the archive into destDir under the declared
// bounds. It refuses on the first violation; callers own cleanup of the
// partially written destDir, which is why they extract into a staging root.
func ExtractArchiveBounded(archivePath, format, destDir string, opts ExtractOptions) (ArchiveSummary, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ArchiveSummary{}, fmt.Errorf("binaryfetch: create extract dir: %w", err)
	}
	return walkArchive(archivePath, format, destDir, opts, true)
}

// inspectRoot is a placeholder root for the dry pass: safeJoin and
// resolveSafeSymlink only reason about the relative shape of an entry, so no
// filesystem access happens beneath it.
const inspectRoot = string(os.PathSeparator) + "binaryfetch-inspect-root"

func walkArchive(archivePath, format, destDir string, opts ExtractOptions, write bool) (ArchiveSummary, error) {
	f, err := os.Open(archivePath) //nolint:gosec // caller-owned archive path
	if err != nil {
		return ArchiveSummary{}, err
	}
	defer f.Close()
	switch normalizeArchive(format) {
	case "tar":
		return walkTar(tar.NewReader(f), destDir, opts, write)
	case "tar.gz":
		gz, err := gzip.NewReader(f)
		if err != nil {
			return ArchiveSummary{}, fmt.Errorf("binaryfetch: open gzip: %w", err)
		}
		defer gz.Close()
		return walkTar(tar.NewReader(gz), destDir, opts, write)
	case "tar.bz2":
		return walkTar(tar.NewReader(bzip2.NewReader(f)), destDir, opts, write)
	case "tar.zst":
		decompressor, ok := registeredArchiveDecompressor("tar.zst")
		if !ok {
			return ArchiveSummary{}, fmt.Errorf("binaryfetch: no decompressor registered for tar.zst")
		}
		stream, err := decompressor(f)
		if err != nil {
			return ArchiveSummary{}, fmt.Errorf("binaryfetch: open tar.zst: %w", err)
		}
		defer stream.Close()
		return walkTar(tar.NewReader(stream), destDir, opts, write)
	case "zip":
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return ArchiveSummary{}, fmt.Errorf("binaryfetch: open zip: %w", err)
		}
		defer zr.Close()
		return walkZip(zr.File, destDir, opts, write)
	default:
		return ArchiveSummary{}, fmt.Errorf("binaryfetch: unsupported archive format %q", format)
	}
}

// bounds accumulates the walk counters and enforces the declared limits before
// any byte of the offending entry is written.
type bounds struct {
	opts    ExtractOptions
	summary ArchiveSummary
}

func (b *bounds) admit(name string, size int64) error {
	if !b.opts.Deadline.IsZero() && time.Now().After(b.opts.Deadline) {
		return &ArchiveError{Violation: ViolationDeadline, Entry: name, Detail: "time budget exhausted"}
	}
	b.summary.Entries++
	if b.opts.MaxEntries > 0 && b.summary.Entries > b.opts.MaxEntries {
		return &ArchiveError{Violation: ViolationEntryLimit, Entry: name, Detail: fmt.Sprintf("exceeds %d entries", b.opts.MaxEntries)}
	}
	if size < 0 || size > b.opts.entryLimit() {
		return &ArchiveError{Violation: ViolationTooLarge, Entry: name, Detail: fmt.Sprintf("entry exceeds %d bytes", b.opts.entryLimit())}
	}
	b.summary.ExpandedBytes += size
	if b.opts.MaxExpandedBytes > 0 && b.summary.ExpandedBytes > b.opts.MaxExpandedBytes {
		return &ArchiveError{Violation: ViolationTooLarge, Entry: name, Detail: fmt.Sprintf("expanded size exceeds %d bytes", b.opts.MaxExpandedBytes)}
	}
	return nil
}

// deadlineReader aborts a long copy once the time budget is spent, so a slow
// decompression bomb cannot outlive the budget between entry checks.
type deadlineReader struct {
	r        io.Reader
	deadline time.Time
	entry    string
}

func (d deadlineReader) Read(p []byte) (int, error) {
	if !d.deadline.IsZero() && time.Now().After(d.deadline) {
		return 0, &ArchiveError{Violation: ViolationDeadline, Entry: d.entry, Detail: "time budget exhausted"}
	}
	return d.r.Read(p)
}

func walkTar(tr *tar.Reader, destDir string, opts ExtractOptions, write bool) (ArchiveSummary, error) {
	if write {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return ArchiveSummary{}, fmt.Errorf("binaryfetch: create extract dir: %w", err)
		}
	}
	b := &bounds{opts: opts}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return b.summary, nil
		}
		if err != nil {
			return b.summary, fmt.Errorf("binaryfetch: read tar: %w", err)
		}
		switch hdr.Typeflag {
		case tar.TypeXGlobalHeader, tar.TypeXHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			// Archive metadata, not filesystem content. Go's tar reader consumes
			// most of these itself but surfaces the pax global header, which
			// every git-produced tarball carries. Rejecting it as an unsupported
			// entry type made a legitimate upstream archive un-extractable —
			// the strictness added to stop silent drops has to distinguish
			// "carries no file" from "carries a file we cannot place".
			continue
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return b.summary, err
		}
		size := int64(0)
		if hdr.Typeflag == tar.TypeReg {
			size = hdr.Size
		}
		if err := b.admit(hdr.Name, size); err != nil {
			return b.summary, err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			b.summary.Directories++
			if write {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return b.summary, fmt.Errorf("binaryfetch: mkdir %q: %w", hdr.Name, err)
				}
			}
		case tar.TypeReg:
			b.summary.Files++
			if !write {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return b.summary, fmt.Errorf("binaryfetch: mkdir parent of %q: %w", hdr.Name, err)
			}
			mode := os.FileMode(hdr.Mode).Perm()
			if mode == 0 {
				mode = 0o644
			}
			if err := writeReaderModeLimit(deadlineReader{r: tr, deadline: opts.Deadline, entry: hdr.Name}, target, mode, opts.entryLimit()); err != nil {
				return b.summary, err
			}
		case tar.TypeSymlink:
			b.summary.Symlinks++
			if _, err := resolveSafeSymlink(destDir, hdr.Name, hdr.Linkname); err != nil {
				return b.summary, err
			}
			if write {
				if err := createSafeSymlink(destDir, hdr.Name, hdr.Linkname, target); err != nil {
					return b.summary, err
				}
			}
		case tar.TypeLink:
			b.summary.Hardlinks++
			linkTarget, err := safeJoin(destDir, hdr.Linkname)
			if err != nil {
				return b.summary, &ArchiveError{Violation: ViolationTraversal, Entry: hdr.Name, Detail: "hardlink target escapes extract dir"}
			}
			if !write {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return b.summary, fmt.Errorf("binaryfetch: mkdir hardlink parent of %q: %w", hdr.Name, err)
			}
			// OCI layers can contain a regular placeholder followed by its
			// canonical hardlink name. Replace that entry so extraction remains
			// faithful to the layer instead of failing on an existing path.
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return b.summary, fmt.Errorf("binaryfetch: replace hardlink target %q: %w", hdr.Name, err)
			}
			if err := os.Link(linkTarget, target); err != nil {
				return b.summary, fmt.Errorf("binaryfetch: create hardlink %q -> %q: %w", hdr.Name, hdr.Linkname, err)
			}
		default:
			return b.summary, &ArchiveError{Violation: ViolationUnsupportedEntry, Entry: hdr.Name, Detail: fmt.Sprintf("unsupported tar entry type %d", hdr.Typeflag)}
		}
	}
}

func walkZip(files []*zip.File, destDir string, opts ExtractOptions, write bool) (ArchiveSummary, error) {
	b := &bounds{opts: opts}
	for _, zf := range files {
		target, err := safeJoin(destDir, zf.Name)
		if err != nil {
			return b.summary, err
		}
		if zf.Mode()&os.ModeSymlink != 0 || zf.Mode()&os.ModeType&^os.ModeDir != 0 {
			return b.summary, &ArchiveError{Violation: ViolationUnsupportedEntry, Entry: zf.Name, Detail: "unsupported zip entry type"}
		}
		if zf.FileInfo().IsDir() {
			if err := b.admit(zf.Name, 0); err != nil {
				return b.summary, err
			}
			b.summary.Directories++
			if write {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return b.summary, fmt.Errorf("binaryfetch: mkdir %q: %w", zf.Name, err)
				}
			}
			continue
		}
		if zf.UncompressedSize64 > uint64(maxArchiveEntryBytes) {
			return b.summary, &ArchiveError{Violation: ViolationTooLarge, Entry: zf.Name, Detail: "entry exceeds the per-entry cap"}
		}
		if err := b.admit(zf.Name, int64(zf.UncompressedSize64)); err != nil {
			return b.summary, err
		}
		b.summary.Files++
		if !write {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return b.summary, fmt.Errorf("binaryfetch: mkdir parent of %q: %w", zf.Name, err)
		}
		mode := zf.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		rc, err := zf.Open()
		if err != nil {
			return b.summary, fmt.Errorf("binaryfetch: open zip entry %q: %w", zf.Name, err)
		}
		err = writeReaderModeLimit(deadlineReader{r: rc, deadline: opts.Deadline, entry: zf.Name}, target, mode, opts.entryLimit())
		rc.Close()
		if err != nil {
			return b.summary, err
		}
	}
	return b.summary, nil
}
