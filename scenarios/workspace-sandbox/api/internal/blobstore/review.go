package blobstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"workspace-sandbox/internal/types"
)

// MaterializeReview publishes before/, after/, changes.patch and snapshot.json
// atomically under the snapshot's existing evidence namespace. It is a derived
// view, never a new source of truth. Replays verify the whole tree and refuse
// tampering instead of silently repairing a workspace a reviewer might be using.
func (s *Store) MaterializeReview(ctx context.Context, snapshot *types.ReviewSnapshot) (string, error) {
	if snapshot == nil || snapshot.SHA256 != snapshot.ContentSHA256() || snapshot.ID != types.ReviewSnapshotID(snapshot.SandboxID, snapshot.RequestID) {
		return "", errors.New("invalid review identity")
	}
	files, err := reviewTreeFiles(snapshot)
	if err != nil {
		return "", err
	}
	anchor, err := s.blobPath(snapshot.ID.String(), snapshot.PatchSHA256)
	if err != nil {
		return "", err
	}
	parent := filepath.Dir(anchor)
	root := filepath.Join(parent, "review-tree")
	manifest, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	if len(manifest) > int(types.MaxReviewFileBytes) {
		return "", errors.New("review manifest exceeds byte limit")
	}
	var bodyBytes int64
	for _, file := range files {
		bodyBytes += file.Size
	}
	// Verify retained source bytes even on replay. A derived copy must not mask
	// missing authoritative evidence. The bounded bodies are loaded one at a time.
	read := func(path string) ([]byte, error) {
		if path == "snapshot.json" {
			return manifest, nil
		}
		entry := files[path]
		limit := types.MaxReviewFileBytes
		if path == "changes.patch" {
			limit = types.MaxReviewInputBytes
		}
		body, err := s.get(ctx, snapshot.ID.String(), entry.SHA256, limit)
		if err != nil {
			return nil, err
		}
		if path != "changes.patch" && int64(len(body)) != entry.Size {
			return nil, errors.New("retained review size mismatch")
		}
		if path == "changes.patch" && bodyBytes+int64(len(body)) > 2*types.MaxReviewInputBytes {
			return nil, errors.New("expanded review input exceeds byte limit")
		}
		return body, nil
	}
	if info, err := os.Lstat(root); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("review tree is not a directory")
		}
		if err := verifyReviewTree(root, files, read); err != nil {
			return "", err
		}
		return root, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	// One reserved staging path per snapshot bounds crash leftovers. Only this
	// derived unpublished tree is recoverable by deletion under LockReview.
	stage := filepath.Join(parent, "review-tree.pending")
	if info, err := os.Lstat(stage); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("review staging path is not a directory")
		}
		if err := os.RemoveAll(stage); err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := os.Mkdir(stage, 0700); err != nil {
		return "", err
	}
	defer os.RemoveAll(stage) // only the private unpublished directory above
	for _, side := range []string{"before", "after"} {
		if err := os.Mkdir(filepath.Join(stage, side), 0700); err != nil {
			return "", err
		}
	}
	for path, file := range files {
		body, err := read(path)
		if err != nil {
			return "", fmt.Errorf("materialize %s: %w", path, err)
		}
		dest := filepath.Join(stage, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return "", err
		}
		if file.Mode == 120000 {
			err = os.Symlink(string(body), dest)
		} else {
			mode := os.FileMode(0444)
			if file.Mode == 100755 {
				mode = 0555
			}
			err = os.WriteFile(dest, body, mode)
		}
		if err != nil {
			return "", err
		}
	}
	if err := os.Rename(stage, root); err != nil {
		return "", err
	}
	return root, nil
}

func reviewTreeFiles(snapshot *types.ReviewSnapshot) (map[string]types.ReviewFile, error) {
	if len(snapshot.Before) > types.MaxReviewFiles || len(snapshot.After) > types.MaxReviewFiles || snapshot.InputBytes < 0 || snapshot.InputBytes > types.MaxReviewInputBytes {
		return nil, errors.New("review manifest exceeds input limits")
	}
	files := map[string]types.ReviewFile{
		"snapshot.json": {FileFingerprint: types.FileFingerprint{Mode: 100644}},
		"changes.patch": {FileFingerprint: types.FileFingerprint{Mode: 100644, SHA256: snapshot.PatchSHA256}},
	}
	var total int64
	for side, entries := range map[string][]types.ReviewFile{"before": snapshot.Before, "after": snapshot.After} {
		for _, file := range entries {
			if !filepath.IsLocal(file.Path) || file.Path == "." || filepath.Clean(file.Path) != file.Path || !hashPattern.MatchString(file.SHA256) || file.Size < 0 || file.Size > types.MaxReviewFileBytes || (file.Mode != 100644 && file.Mode != 100755 && file.Mode != 120000) {
				return nil, errors.New("invalid retained review file")
			}
			path := filepath.Join(side, file.Path)
			if _, exists := files[path]; exists {
				return nil, errors.New("duplicate retained review path")
			}
			files[path] = file
			total += file.Size
			if total > 2*types.MaxReviewInputBytes {
				return nil, errors.New("review bodies exceed input limit")
			}
		}
	}
	// Reject a file/symlink that is also another entry's parent before any writes.
	for path := range files {
		for parent := filepath.Dir(path); parent != "."; parent = filepath.Dir(parent) {
			if _, exists := files[parent]; exists {
				return nil, errors.New("retained review paths overlap")
			}
		}
	}
	return files, nil
}

func verifyReviewTree(root string, files map[string]types.ReviewFile, read func(string) ([]byte, error)) error {
	opened, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer opened.Close()
	seen := 0
	dirs := map[string]bool{".": true, "before": true, "after": true}
	for path := range files {
		for parent := filepath.Dir(path); parent != "."; parent = filepath.Dir(parent) {
			dirs[parent] = true
		}
	}
	seenDirs := 0
	err = fs.WalkDir(opened.FS(), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.FromSlash(path)
		if entry.IsDir() {
			if !dirs[rel] {
				return errors.New("unexpected directory in review tree")
			}
			seenDirs++
			return nil
		}
		file, exists := files[rel]
		if !exists {
			return errors.New("unexpected file in review tree")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var body []byte
		if file.Mode == 120000 && info.Mode()&os.ModeSymlink != 0 {
			var link string
			link, err = opened.Readlink(rel)
			body = []byte(link)
		} else if file.Mode != 120000 && info.Mode().IsRegular() && info.Mode().Perm() == map[int]os.FileMode{100644: 0444, 100755: 0555}[file.Mode] {
			if info.Size() > types.MaxReviewInputBytes {
				return errors.New("review file exceeds read limit")
			}
			body, err = opened.ReadFile(rel)
		} else {
			return errors.New("review file type or mode changed")
		}
		if err != nil {
			return err
		}
		want, err := read(rel)
		if err != nil {
			return err
		}
		if !bytes.Equal(body, want) {
			return errors.New("review file content changed")
		}
		seen++
		return nil
	})
	if err == nil && (seen != len(files) || seenDirs != len(dirs)) {
		return errors.New("review tree is incomplete")
	}
	return err
}
