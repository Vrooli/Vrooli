package resources

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Extract archives offline without ever invoking the downloaded executable.
// Every entry stays inside one declared bundle root; links are created last,
// so extraction can never traverse an archive-provided symlink ancestor.
func extractOpenRGBZip(ctx context.Context, archive *os.File, destination string, p openRGBReleaseProfile) error {
	return extractOpenRGBZipWithIO(ctx, archive, destination, p, openRGBPrivateTreeIO())
}

func extractOpenRGBZipWithIO(ctx context.Context, archive *os.File, destination string, p openRGBReleaseProfile, owner openRGBTreeIO) error {
	info, err := archive.Stat()
	if err != nil {
		return err
	}
	reader, err := zip.NewReader(archive, info.Size())
	if err != nil {
		return err
	}
	if len(reader.File) > 1024 {
		return errors.New("OpenRGB ZIP exceeds entry bound")
	}
	type link struct{ name, target string }
	var links []link
	seen := map[string]bool{}
	var total uint64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := entry.Name
		if strings.ContainsAny(name, "\\\x00:\r\n") || strings.HasPrefix(name, "/") || path.Clean(strings.TrimSuffix(name, "/")) != strings.TrimSuffix(name, "/") {
			return errors.New("unsafe OpenRGB archive path")
		}
		if name == p.ArchiveRoot+"/" {
			continue
		}
		if !strings.HasPrefix(name, p.ArchiveRoot+"/") {
			return errors.New("OpenRGB archive contains undeclared bundle root")
		}
		relative := strings.TrimSuffix(strings.TrimPrefix(name, p.ArchiveRoot+"/"), "/")
		key := strings.ToLower(relative)
		if relative == "" || seen[key] {
			return errors.New("duplicate OpenRGB archive path")
		}
		seen[key] = true
		if entry.UncompressedSize64 > 256<<20-total {
			return errors.New("OpenRGB archive exceeds unpacked size bound")
		}
		total += entry.UncompressedSize64
		target := filepath.Join(destination, filepath.FromSlash(relative))
		mode := entry.Mode()
		if mode.IsDir() {
			if err := owner.mkdir(ctx, target); err != nil {
				return err
			}
			continue
		}
		if mode&os.ModeSymlink != 0 {
			if entry.UncompressedSize64 > 4096 {
				return errors.New("oversized OpenRGB archive link")
			}
			r, err := entry.Open()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(r, 4097))
			r.Close()
			if err != nil {
				return err
			}
			value := string(data)
			resolved := path.Clean(path.Join(path.Dir(relative), value))
			if value == "" || strings.ContainsAny(value, "\\:\x00\r\n") || path.IsAbs(value) || resolved == ".." || strings.HasPrefix(resolved, "../") {
				return errors.New("OpenRGB archive link escapes bundle")
			}
			links = append(links, link{target, value})
			continue
		}
		if !mode.IsRegular() || mode&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
			return errors.New("unsafe OpenRGB archive entry type")
		}
		if err := owner.mkdir(ctx, filepath.Dir(target)); err != nil {
			return err
		}
		r, err := entry.Open()
		if err != nil {
			return err
		}
		perm := os.FileMode(0o600)
		if mode.Perm()&0o111 != 0 || relative == p.Payload {
			perm = 0o700
		}
		var n int64
		copyErr := owner.write(ctx, target, perm, func(w io.Writer) error {
			var err error
			n, err = io.Copy(w, io.LimitReader(&managedUpdateContextReader{ctx: ctx, reader: r}, int64(entry.UncompressedSize64)+1))
			if err != nil {
				return err
			}
			if n != int64(entry.UncompressedSize64) {
				return errors.New("OpenRGB ZIP entry has another exact size")
			}
			return nil
		})
		closeErr := r.Close()
		if copyErr != nil || closeErr != nil || n != int64(entry.UncompressedSize64) {
			return errors.Join(fmt.Errorf("OpenRGB ZIP entry size/content mismatch: %s", name), copyErr, closeErr)
		}
	}
	for _, l := range links {
		if err := verifyOpenRGBExtractionLinkParent(destination, l.name, true); err != nil {
			return err
		}
		if err := owner.mkdir(ctx, filepath.Dir(l.name)); err != nil {
			return err
		}
	}
	for _, l := range links {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := verifyOpenRGBExtractionLinkParent(destination, l.name, false); err != nil {
			return err
		}
		if err := os.Symlink(l.target, l.name); err != nil {
			return err
		}
	}
	return nil
}

// Link parents must be actual directories, including during the delayed link
// phase: delaying only regular writes does not prevent one queued link from
// redirecting creation of another queued link. Framework aliases remain valid
// as leaves, but an alias may never be an extraction destination ancestor.
func verifyOpenRGBExtractionLinkParent(root, leaf string, allowMissing bool) error {
	parent := filepath.Dir(leaf)
	relative, err := filepath.Rel(root, parent)
	if err != nil || relative != "." && !filepath.IsLocal(relative) {
		return errors.New("OpenRGB extraction link parent escapes bundle")
	}
	current := root
	components := []string{}
	if relative != "." {
		components = strings.Split(relative, string(filepath.Separator))
	}
	for index := 0; ; index++ {
		info, err := os.Lstat(current)
		if os.IsNotExist(err) && allowMissing {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("OpenRGB extraction refuses a symlink or nondirectory link parent")
		}
		if index == len(components) {
			return nil
		}
		current = filepath.Join(current, components[index])
	}
}
