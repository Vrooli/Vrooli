package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxCandidateSecurityBytes = 16 << 20

type indexBlob struct {
	oid string
}

// ReadStagedCandidate reads only the exact staged blobs for paths changed in
// the index. It never reads working-tree files. A candidate larger than the
// Security Health request budget, a conflicted entry, an unsupported gitlink,
// or a non-UTF-8 path is reported as incomplete instead of being skipped.
func (r *ExecGitRunner) ReadStagedCandidate(ctx context.Context, repoDir string) (map[string][]byte, error) {
	changedOutput, err := r.readCandidateGit(ctx, repoDir, "diff", "--cached", "--name-only", "-z", "--no-renames", "--")
	if err != nil {
		return nil, fmt.Errorf("read staged candidate paths: %w", err)
	}
	changed := make(map[string]struct{})
	for _, path := range bytes.Split(changedOutput, []byte{0}) {
		if len(path) != 0 {
			changed[string(path)] = struct{}{}
		}
	}
	if len(changed) == 0 {
		return map[string][]byte{}, nil
	}

	indexOutput, err := r.readCandidateGit(ctx, repoDir, "ls-files", "--stage", "-z")
	if err != nil {
		return nil, fmt.Errorf("read staged index entries: %w", err)
	}
	entries, err := parseStagedCandidateIndex(indexOutput, changed)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(entries))
	objectIDs := make([]string, 0, len(entries))
	seenObjects := make(map[string]struct{}, len(entries))
	for path, entry := range entries {
		paths = append(paths, path)
		if _, ok := seenObjects[entry.oid]; !ok {
			objectIDs = append(objectIDs, entry.oid)
			seenObjects[entry.oid] = struct{}{}
		}
	}
	sort.Strings(paths)
	sort.Strings(objectIDs)
	if len(paths) == 0 { // All changed entries are deletions.
		return map[string][]byte{}, nil
	}
	objects, err := r.readCandidateBlobs(ctx, repoDir, objectIDs)
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(paths))
	for path, entry := range entries {
		blob, ok := objects[entry.oid]
		if !ok {
			return nil, fmt.Errorf("staged candidate blob is unavailable")
		}
		files[path] = blob
	}
	return files, nil
}

func parseStagedCandidateIndex(indexOutput []byte, changed map[string]struct{}) (map[string]indexBlob, error) {
	entries := make(map[string]indexBlob, len(changed))
	for _, record := range bytes.Split(indexOutput, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		separator := bytes.IndexByte(record, '\t')
		if separator < 0 {
			return nil, fmt.Errorf("parse staged index entry")
		}
		fields := strings.Fields(string(record[:separator]))
		if len(fields) != 3 {
			return nil, fmt.Errorf("parse staged index metadata")
		}
		path := string(record[separator+1:])
		if _, needed := changed[path]; !needed {
			continue
		}
		if fields[2] != "0" {
			return nil, fmt.Errorf("staged candidate has unresolved conflicts")
		}
		if fields[0] == "160000" {
			return nil, fmt.Errorf("staged candidate contains an unsupported submodule")
		}
		if !utf8.ValidString(path) {
			return nil, fmt.Errorf("staged candidate contains a non-UTF-8 path")
		}
		entries[path] = indexBlob{oid: fields[1]}
	}
	return entries, nil
}

func (r *ExecGitRunner) readCandidateGit(ctx context.Context, repoDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.gitPath(), readArgs(repoDir, args...)...)
	var stdout limitedBuffer
	stdout.limit = maxCandidateSecurityBytes + (1 << 20)
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git candidate inspection failed")
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("staged candidate inventory exceeds its bounded size")
	}
	return stdout.Bytes(), nil
}

func (r *ExecGitRunner) readCandidateBlobs(ctx context.Context, repoDir string, oids []string) (map[string][]byte, error) {
	if len(oids) == 0 {
		return map[string][]byte{}, nil
	}
	var stdin bytes.Buffer
	for _, oid := range oids {
		stdin.WriteString(oid)
		stdin.WriteByte('\n')
	}
	cmd := exec.CommandContext(ctx, r.gitPath(), readArgs(repoDir, "cat-file", "--batch")...)
	cmd.Stdin = &stdin
	var stdout limitedBuffer
	stdout.limit = maxCandidateSecurityBytes + (1 << 20)
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("read staged candidate blobs failed")
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("staged candidate exceeds the scanner request limit")
	}

	return parseStagedCandidateBlobs(stdout.Bytes(), oids)
}

func parseStagedCandidateBlobs(output []byte, oids []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(oids))
	reader := bufio.NewReader(bytes.NewReader(output))
	total := 0
	for _, expectedOID := range oids {
		header, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("parse staged candidate blob header")
		}
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[0] != expectedOID || fields[1] != "blob" {
			return nil, fmt.Errorf("staged candidate contains a non-blob object")
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > maxCandidateSecurityBytes-total {
			return nil, fmt.Errorf("staged candidate exceeds the scanner request limit")
		}
		blob := make([]byte, size)
		if _, err := io.ReadFull(reader, blob); err != nil {
			return nil, fmt.Errorf("read staged candidate blob")
		}
		separator, err := reader.ReadByte()
		if err != nil || separator != '\n' {
			return nil, fmt.Errorf("parse staged candidate blob framing")
		}
		total += size
		result[fields[0]] = blob
	}
	if _, err := reader.ReadByte(); err != io.EOF {
		return nil, fmt.Errorf("staged candidate blob output contains trailing data")
	}
	return result, nil
}
