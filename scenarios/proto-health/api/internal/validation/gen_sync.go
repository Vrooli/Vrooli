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
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	genSyncDefaultTimeout = 2 * time.Minute
	envSkipGenSync        = "PROTO_HEALTH_SKIP_GEN_SYNC"
	envGenSyncTimeout     = "PROTO_HEALTH_GEN_SYNC_TIMEOUT"
)

var bufPathRE = regexp.MustCompile(`(?:^|\s)([A-Za-z0-9_./-]+\.proto):\d+:\d+`)

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
	if truthy(os.Getenv(envSkipGenSync)) {
		return GenSyncStatus{InSync: true, Skipped: true, SkipMessage: envSkipGenSync + " is set"}, nil
	}
	protoRoot := filepath.Join(c.repoRoot, "packages", "proto")
	tempRoot, err := os.MkdirTemp("", "proto-health-gen-sync-*")
	if err != nil {
		return GenSyncStatus{}, fmt.Errorf("create temp workspace: %w", err)
	}
	defer os.RemoveAll(tempRoot)

	tempProtoRoot := filepath.Join(tempRoot, "proto")
	if err := copyGenSyncInputs(protoRoot, tempProtoRoot); err != nil {
		return GenSyncStatus{}, fmt.Errorf("copy packages/proto inputs: %w", err)
	}

	timeout := genSyncTimeout()
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "buf", "generate", "--path", filepath.ToSlash(filepath.Join("schemas", scenario)))
	cmd.Dir = tempProtoRoot
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return GenSyncStatus{
				Blocked: true,
				Detail:  fmt.Sprintf("generated-artifact sync could not be verified: buf generate exceeded %s", timeout),
			}, nil
		}
		files := offendingProtoFiles(stderr.String())
		if hasTargetProtoFile(files, scenario) {
			return GenSyncStatus{}, fmt.Errorf("run buf generate for %s: %w: %s", scenario, err, strings.TrimSpace(stderr.String()))
		}
		detail := "generated-artifact sync could not be verified: proto compilation failed outside the target scenario"
		if len(files) > 0 {
			detail = fmt.Sprintf("generated-artifact sync could not be verified: upstream proto %s failed to compile", strings.Join(files, ", "))
		}
		return GenSyncStatus{Blocked: true, BlockedBy: files, Detail: detail}, nil
	}
	if err := markPythonTyped(filepath.Join(tempProtoRoot, "gen", "python")); err != nil {
		return GenSyncStatus{}, fmt.Errorf("mark generated python packages typed: %w", err)
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

func genSyncTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envGenSyncTimeout))
	if raw == "" {
		return genSyncDefaultTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return genSyncDefaultTimeout
	}
	return d
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

func offendingProtoFiles(stderr string) []string {
	seen := map[string]bool{}
	var out []string
	for _, match := range bufPathRE.FindAllStringSubmatch(stderr, -1) {
		if len(match) < 2 {
			continue
		}
		path := filepath.ToSlash(strings.TrimSpace(match[1]))
		path = strings.TrimPrefix(path, "packages/proto/")
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func hasTargetProtoFile(files []string, scenario string) bool {
	prefix := filepath.ToSlash(filepath.Join("schemas", scenario)) + "/"
	for _, file := range files {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}

func copyGenSyncInputs(protoRoot, tempProtoRoot string) error {
	for _, name := range []string{"buf.yaml", "buf.gen.yaml", "buf.lock"} {
		src := filepath.Join(protoRoot, name)
		info, err := os.Stat(src)
		if err != nil {
			if os.IsNotExist(err) && name == "buf.lock" {
				continue
			}
			return err
		}
		if err := copyFile(src, filepath.Join(tempProtoRoot, name), info.Mode().Perm()); err != nil {
			return err
		}
	}
	if err := copyDir(filepath.Join(protoRoot, "schemas"), filepath.Join(tempProtoRoot, "schemas")); err != nil {
		return err
	}
	vendorRoot := filepath.Join(protoRoot, "vendor")
	if _, err := os.Stat(vendorRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return copyDir(vendorRoot, filepath.Join(tempProtoRoot, "vendor"))
}

func markPythonTyped(root string) error {
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		file := filepath.Join(path, "py.typed")
		f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		return f.Close()
	})
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
