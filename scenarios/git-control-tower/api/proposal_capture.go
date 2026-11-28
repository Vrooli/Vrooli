package main

import (
	"bufio"
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
	baseIDs := make([]string, 0, len(result))
	if base != "" {
		entries, err := g.baseTreeEntries(ctx, base, result)
		if err != nil {
			return nil, nil, err
		}
		for i := range result {
			f := &result[i]
			entry, ok := entries[f.Path]
			if !ok {
				continue
			}
			if entry.kind != "blob" {
				return nil, nil, fmt.Errorf("%w: unsupported base entry %s", proposals.ErrInvalid, f.Path)
			}
			f.BaseMode, f.BaseBlobID = entry.mode, entry.oid
			baseIDs = append(baseIDs, entry.oid)
		}
	}
	baseIDs = uniqueObjectIDs(baseIDs)
	baseBlobs, err := g.batchBlobs(ctx, baseIDs, nil, total, fmt.Errorf("%w: base file exceeds 8 MiB", proposals.ErrInvalid))
	if err != nil {
		return nil, nil, err
	}
	for id, data := range baseBlobs {
		if err := keep(id, data); err != nil {
			return nil, nil, err
		}
	}
	workingIDs := make([]string, 0, len(result))
	workingPaths := make(map[string]string, len(result))
	for i := range result {
		f := &result[i]
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
			f.Mode = mode
			workingIDs = append(workingIDs, id)
			workingPaths[id] = f.Path
		}
		f.Captured = true
	}
	// The temporary object directory contains filtered working-tree blobs. A
	// single cat-file process retrieves those canonical bytes after hashing.
	// Base objects are read separately from the repository object database.
	workingIDs = uniqueObjectIDs(workingIDs)
	newWorkingIDs := workingIDs[:0]
	for _, id := range workingIDs {
		if _, alreadyCaptured := blobs[id]; !alreadyCaptured {
			newWorkingIDs = append(newWorkingIDs, id)
		}
	}
	workingBlobs, err := g.batchBlobs(ctx, newWorkingIDs, []string{"GIT_OBJECT_DIRECTORY=" + objects}, total, fmt.Errorf("working file exceeds capture limits"))
	if err != nil {
		path := "working file"
		if len(newWorkingIDs) > 0 {
			path = workingPaths[newWorkingIDs[0]]
		}
		return nil, nil, fmt.Errorf("capture %s exceeds limits or failed: %w", path, err)
	}
	for id, data := range workingBlobs {
		if err := keep(id, data); err != nil {
			return nil, nil, err
		}
	}
	return result, blobs, nil
}

type proposalTreeEntry struct {
	mode string
	kind string
	oid  string
}

func (g *proposalGit) baseTreeEntries(ctx context.Context, base string, files []proposals.File) (map[string]proposalTreeEntry, error) {
	if len(files) == 0 {
		return map[string]proposalTreeEntry{}, nil
	}
	args := []string{"ls-tree", "-z", base, "--"}
	for _, f := range files {
		args = append(args, f.Path)
	}
	out, err := g.read(ctx, nil, args...)
	if err != nil {
		return nil, err
	}
	entries := make(map[string]proposalTreeEntry, len(files))
	for _, record := range bytes.Split(out, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, path, ok := bytes.Cut(record, []byte{'\t'})
		if !ok {
			return nil, fmt.Errorf("malformed git ls-tree response")
		}
		fields := strings.Fields(string(header))
		if len(fields) != 3 {
			return nil, fmt.Errorf("malformed git ls-tree entry")
		}
		entries[string(path)] = proposalTreeEntry{mode: fields[0], kind: fields[1], oid: fields[2]}
	}
	return entries, nil
}

func uniqueObjectIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := ids[:0]
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// batchBlobs reads each unique object through one `git cat-file --batch`
// process. The size checks happen before allocating each payload, so even a
// malformed or unexpectedly large object cannot exceed the capture budget.
func (g *proposalGit) batchBlobs(ctx context.Context, ids, env []string, alreadyCaptured int, tooLarge error) (map[string][]byte, error) {
	blobs := make(map[string][]byte, len(ids))
	if len(ids) == 0 {
		return blobs, nil
	}
	var input strings.Builder
	for _, id := range ids {
		input.WriteString(id)
		input.WriteByte('\n')
	}
	cmd := exec.CommandContext(ctx, g.gitPath, "--no-optional-locks", "--literal-pathspecs", "-C", g.repoDir, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(input.String())
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(stdout)
	total := alreadyCaptured
	stop := func(e error) (map[string][]byte, error) {
		_ = stdout.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, e
	}
	for _, requested := range ids {
		header, readErr := reader.ReadString('\n')
		if readErr != nil {
			return stop(fmt.Errorf("read git cat-file batch header: %w", readErr))
		}
		fields := strings.Fields(header)
		if len(fields) == 2 && fields[1] == "missing" {
			return stop(fmt.Errorf("git cat-file could not read object %s", requested))
		}
		if len(fields) != 3 || fields[0] != requested || fields[1] != "blob" {
			return stop(fmt.Errorf("unexpected git cat-file batch response for %s", requested))
		}
		size, parseErr := strconv.ParseInt(fields[2], 10, 64)
		if parseErr != nil || size < 0 || size > maxProposalFileBytes {
			return stop(tooLarge)
		}
		if size > int64(maxProposalCaptureBytes-total) {
			return stop(fmt.Errorf("%w: proposal capture exceeds 64 MiB; split the package", proposals.ErrInvalid))
		}
		data := make([]byte, int(size))
		if _, err = io.ReadFull(reader, data); err != nil {
			return stop(fmt.Errorf("read git cat-file batch payload: %w", err))
		}
		terminator, readErr := reader.ReadByte()
		if readErr != nil || terminator != '\n' {
			return stop(fmt.Errorf("malformed git cat-file batch payload terminator"))
		}
		blobs[requested] = data
		total += len(data)
	}
	if err = cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git cat-file --batch failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return blobs, nil
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
