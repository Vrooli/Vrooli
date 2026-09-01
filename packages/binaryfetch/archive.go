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
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("binaryfetch: create extract dir: %w", err)
	}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("binaryfetch: read tar: %w", err)
		}
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("binaryfetch: mkdir %q: %w", hdr.Name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("binaryfetch: mkdir parent of %q: %w", hdr.Name, err)
			}
			mode := os.FileMode(hdr.Mode).Perm()
			if mode == 0 {
				mode = 0o644
			}
			if err := writeReaderMode(tr, target, mode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := createSafeSymlink(destDir, hdr.Name, hdr.Linkname, target); err != nil {
				return err
			}
		case tar.TypeLink:
			linkTarget, err := safeJoin(destDir, hdr.Linkname)
			if err != nil {
				return fmt.Errorf("binaryfetch: hardlink %q target: %w", hdr.Name, err)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("binaryfetch: mkdir hardlink parent of %q: %w", hdr.Name, err)
			}
			// OCI layers can contain a regular placeholder followed by its
			// canonical hardlink name. Replace that entry so extraction remains
			// faithful to the layer instead of failing on an existing path.
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("binaryfetch: replace hardlink target %q: %w", hdr.Name, err)
			}
			if err := os.Link(linkTarget, target); err != nil {
				return fmt.Errorf("binaryfetch: create hardlink %q -> %q: %w", hdr.Name, hdr.Linkname, err)
			}
		case tar.TypeXGlobalHeader, tar.TypeXHeader, tar.TypeGNULongName, tar.TypeGNULongLink:
			// Archive metadata, not filesystem content. Go's tar reader consumes
			// most of these itself but surfaces the pax global header, which
			// every git-produced tarball carries. Rejecting it as an unsupported
			// entry type made a legitimate upstream archive un-extractable —
			// the strictness added to stop silent drops has to distinguish
			// "carries no file" from "carries a file we cannot place".
			continue
		default:
			return fmt.Errorf("binaryfetch: unsupported tar entry type %d for %q", hdr.Typeflag, hdr.Name)
		}
	}
}

// safeJoin joins destDir with a (possibly attacker-controlled) relative entry
// name, rejecting any name that is absolute or contains a ".." component (which
// could escape destDir) and any result that lands outside destDir.
func safeJoin(destDir, name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("binaryfetch: archive entry %q escapes extract dir", name)
	}
	target := filepath.Join(destDir, cleaned)
	if target != destDir && !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("binaryfetch: archive entry %q escapes extract dir", name)
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

func createSafeSymlink(root, name, linkname, target string) error {
	if filepath.IsAbs(filepath.FromSlash(linkname)) {
		return fmt.Errorf("binaryfetch: symlink %q has an absolute target", name)
	}
	resolved, err := safeJoin(root, filepath.ToSlash(filepath.Join(filepath.Dir(name), filepath.FromSlash(linkname))))
	if err != nil {
		return fmt.Errorf("binaryfetch: symlink %q target escapes extract dir: %w", name, err)
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

	for _, zf := range zr.File {
		target, err := safeJoin(destDir, zf.Name)
		if err != nil {
			return err
		}
		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("binaryfetch: mkdir %q: %w", zf.Name, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("binaryfetch: mkdir parent of %q: %w", zf.Name, err)
		}
		mode := zf.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("binaryfetch: open zip entry %q: %w", zf.Name, err)
		}
		err = writeReaderMode(rc, target, mode)
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
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
	out, err := os.OpenFile(destFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) //nolint:gosec // destFile is validated by safeJoin under a controlled dir
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(r, maxArchiveEntryBytes)); err != nil {
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
