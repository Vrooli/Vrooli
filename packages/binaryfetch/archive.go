package binaryfetch

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoBinaryInArchive is returned when the requested binPath (or, when binPath
// is empty, any single regular file) cannot be located in the archive.
var ErrNoBinaryInArchive = errors.New("binaryfetch: binary not found in archive")

// maxArchiveEntryBytes bounds a single extracted file to guard against a
// decompression bomb in a fetched archive. 2 GiB comfortably covers large
// model-adjacent binaries while refusing absurd entries.
const maxArchiveEntryBytes int64 = 2 << 30

// extractBinary pulls the binary identified by binPath out of the archive at
// archivePath and writes it to destFile. When binPath is empty it selects the
// archive's sole regular file (erroring if there are zero or many). format is
// "tar.gz" or "zip".
func extractBinary(archivePath, format, binPath, destFile string) error {
	switch format {
	case "tar.gz", "tgz":
		return extractFromTarGz(archivePath, binPath, destFile)
	case "zip":
		return extractFromZip(archivePath, binPath, destFile)
	default:
		return fmt.Errorf("binaryfetch: unsupported archive format %q", format)
	}
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
// rejected. format is "tar.gz" or "zip".
func extractAll(archivePath, format, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("binaryfetch: create extract dir: %w", err)
	}
	switch format {
	case "tar.gz", "tgz":
		return extractAllTarGz(archivePath, destDir)
	case "zip":
		return extractAllZip(archivePath, destDir)
	default:
		return fmt.Errorf("binaryfetch: unsupported archive format %q", format)
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

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
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
			mode := os.FileMode(hdr.Mode).Perm() //nolint:gosec // mode is masked to permission bits
			if mode == 0 {
				mode = 0o644
			}
			if err := writeReaderMode(tr, target, mode); err != nil {
				return err
			}
		default:
			// Skip symlinks, devices, fifos: a fetched backend tree is plain files.
			continue
		}
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
	case "zip":
		return "zip"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
