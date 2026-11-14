package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"git-control-tower/internal/proposals"
)

const maxProposalFileBytes = 8 << 20
const maxProposalCaptureBytes = proposals.MaxCaptureBytes

// Capture retains Git's canonical (clean-filtered) bytes in scenario storage.
// Temporary object storage lets us obtain those bytes without adding objects
// to the operator's repository. No index, ref or working-tree write occurs.
func (g *proposalGit) Capture(ctx context.Context, base string, files []proposals.File) ([]proposals.File, map[string][]byte, error) {
	objects, err := os.MkdirTemp("", "gct-capture-objects-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(objects)
	result := append([]proposals.File(nil), files...)
	blobs := map[string][]byte{}
	total := 0
	keep := func(id string, data []byte) error {
		if _, ok := blobs[id]; ok {
			return nil
		}
		total += len(data)
		if total > maxProposalCaptureBytes {
			return fmt.Errorf("%w: proposal capture exceeds 64 MiB; split the package", proposals.ErrInvalid)
		}
		blobs[id] = data
		return nil
	}
	for i := range result {
		f := &result[i]
		if base != "" {
			entry, err := g.read(ctx, nil, "ls-tree", "-z", base, "--", f.Path)
			if err != nil {
				return nil, nil, err
			}
			if header, _, ok := strings.Cut(string(entry), "\t"); ok {
				fields := strings.Fields(header)
				if len(fields) != 3 || fields[1] != "blob" {
					return nil, nil, fmt.Errorf("%w: unsupported base entry %s", proposals.ErrInvalid, f.Path)
				}
				f.BaseMode, f.BaseBlobID = fields[0], fields[2]
				data, err := g.boundedBlob(ctx, f.BaseBlobID)
				if err != nil {
					return nil, nil, err
				}
				if err = keep(f.BaseBlobID, data); err != nil {
					return nil, nil, err
				}
			}
		}
		if !f.Deleted {
			data, mode, err := g.captureWorkingFile(f.Path)
			if err != nil {
				return nil, nil, err
			}
			if mode != "120000" && f.Mode != "" {
				mode = f.Mode
			}
			if sha256Hex(data) != f.SHA256 {
				return nil, nil, fmt.Errorf("%w: %s changed during capture", proposals.ErrConflict, f.Path)
			}
			args := []string{"--literal-pathspecs", "-C", g.repoDir, "hash-object", "-w", "--stdin"}
			if mode != "120000" {
				args = append(args, "--path="+f.Path)
			}
			cmd := exec.CommandContext(ctx, g.gitPath, args...)
			cmd.Env = append(os.Environ(), "GIT_OBJECT_DIRECTORY="+objects)
			cmd.Stdin = bytes.NewReader(data)
			out, err := cmd.Output()
			if err != nil {
				return nil, nil, fmt.Errorf("capture %s: %w", f.Path, err)
			}
			id := strings.TrimSpace(string(out))
			if id != f.BlobID {
				return nil, nil, fmt.Errorf("%w: %s filters or content changed during capture", proposals.ErrConflict, f.Path)
			}
			cmd = exec.CommandContext(ctx, g.gitPath, "-C", g.repoDir, "cat-file", "blob", id)
			cmd.Env = append(os.Environ(), "GIT_OBJECT_DIRECTORY="+objects)
			var limited bytes.Buffer
			cmd.Stdout = &boundedCaptureWriter{target: &limited, remaining: maxProposalFileBytes}
			if err = cmd.Run(); err != nil {
				return nil, nil, fmt.Errorf("capture %s exceeds limits or failed: %w", f.Path, err)
			}
			f.Mode = mode
			if err = keep(id, limited.Bytes()); err != nil {
				return nil, nil, err
			}
		}
		f.Captured = true
	}
	return result, blobs, nil
}

func (g *proposalGit) boundedBlob(ctx context.Context, id string) ([]byte, error) {
	info, err := g.read(ctx, nil, "cat-file", "-s", id)
	if err != nil {
		return nil, err
	}
	size, err := strconv.ParseInt(strings.TrimSpace(string(info)), 10, 64)
	if err != nil || size > maxProposalFileBytes {
		return nil, fmt.Errorf("%w: base file exceeds 8 MiB", proposals.ErrInvalid)
	}
	return g.read(ctx, nil, "cat-file", "blob", id)
}

func (g *proposalGit) captureWorkingFile(path string) ([]byte, string, error) {
	full := filepath.Join(g.repoDir, filepath.FromSlash(path))
	root, err := filepath.EvalSymlinks(g.repoDir)
	if err != nil {
		return nil, "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(full))
	if err != nil {
		return nil, "", err
	}
	rel, err := filepath.Rel(root, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("%w: path escapes repository through a symlink", proposals.ErrInvalid)
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(full)
		return []byte(target), "120000", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxProposalFileBytes {
		return nil, "", fmt.Errorf("%w: %s must be a regular file under 8 MiB or a symlink", proposals.ErrInvalid, path)
	}
	file, err := os.Open(full)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxProposalFileBytes+1))
	if len(data) > maxProposalFileBytes {
		return nil, "", fmt.Errorf("%w: file grew beyond capture limit", proposals.ErrConflict)
	}
	mode := "100644"
	if info.Mode()&0111 != 0 {
		mode = "100755"
	}
	return data, mode, err
}

type boundedCaptureWriter struct {
	target    io.Writer
	remaining int
}

func (w *boundedCaptureWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		return 0, fmt.Errorf("capture exceeds file limit")
	}
	n, err := w.target.Write(data)
	w.remaining -= n
	return n, err
}
