package validation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const genSyncTimeout = 2 * time.Minute

type GeneratedArtifactChecker struct {
	repoRoot string
}

func NewGeneratedArtifactChecker(repoRoot string) *GeneratedArtifactChecker {
	return &GeneratedArtifactChecker{repoRoot: repoRoot}
}

func (c *GeneratedArtifactChecker) CheckScenario(ctx context.Context, scenario string) (GenSyncStatus, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return GenSyncStatus{}, fmt.Errorf("scenario name is required")
	}
	protoRoot := filepath.Join(c.repoRoot, "packages", "proto")
	tempRoot, err := os.MkdirTemp("", "proto-health-gen-sync-*")
	if err != nil {
		return GenSyncStatus{}, fmt.Errorf("create temp workspace: %w", err)
	}
	defer os.RemoveAll(tempRoot)

	tempProtoRoot := filepath.Join(tempRoot, "proto")
	if err := copyDir(protoRoot, tempProtoRoot); err != nil {
		return GenSyncStatus{}, fmt.Errorf("copy packages/proto: %w", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, genSyncTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "make", "generate")
	cmd.Dir = tempProtoRoot
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return GenSyncStatus{}, fmt.Errorf("run make generate: %w", runCtx.Err())
		}
		return GenSyncStatus{}, fmt.Errorf("run make generate: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	paths := scenarioGenDirs(scenario)
	var drift []string
	for _, rel := range paths {
		equal, err := dirsEqual(filepath.Join(protoRoot, rel), filepath.Join(tempProtoRoot, rel))
		if err != nil {
			return GenSyncStatus{}, err
		}
		if !equal {
			drift = append(drift, filepath.ToSlash(filepath.Join("packages", "proto", rel)))
		}
	}
	if len(drift) == 0 {
		return GenSyncStatus{InSync: true}, nil
	}
	return GenSyncStatus{
		InSync: false,
		Drift:  drift,
		Detail: fmt.Sprintf("%d generated slice(s) differ after regeneration", len(drift)),
	}, nil
}

func scenarioGenDirs(scenario string) []string {
	pythonScenario := strings.ReplaceAll(scenario, "-", "_")
	return []string{
		filepath.Join("gen", "go", scenario),
		filepath.Join("gen", "typescript", scenario),
		filepath.Join("gen", "typescript", "js", scenario),
		filepath.Join("gen", "python", pythonScenario),
	}
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func dirsEqual(a, b string) (bool, error) {
	left, err := fileDigests(a)
	if err != nil {
		return false, err
	}
	right, err := fileDigests(b)
	if err != nil {
		return false, err
	}
	if len(left) != len(right) {
		return false, nil
	}
	for path, l := range left {
		r, ok := right[path]
		if !ok || !bytes.Equal(l, r) {
			return false, nil
		}
	}
	return true, nil
}

func fileDigests(root string) (map[string][]byte, error) {
	files := map[string][]byte{}
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = b
		return nil
	})
	return files, err
}
