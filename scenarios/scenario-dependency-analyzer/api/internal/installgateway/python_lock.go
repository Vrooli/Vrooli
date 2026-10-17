package installgateway

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var pinnedRequirement = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9_.-]*)==([^\\\s]+)(?:\s*\\)?\s*$`)

type pythonIndexSet struct {
	Lockfile string
	Primary  string
	Extras   []string
}

func pythonLockfilesFor(surfaceRoot, manager string) []string {
	if manager != "uv-lock" {
		return nil
	}
	paths, _ := filepath.Glob(filepath.Join(surfaceRoot, "requirements*.lock"))
	sort.Strings(paths)
	return paths
}

func pythonIndexSetsFor(surfaceRoot, manager string) map[string]pythonIndexSet {
	if manager != "uv-lock" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(surfaceRoot, "resource.json"))
	if err != nil {
		return nil
	}
	return pythonIndexSetsFromJSON(data)
}

func pythonIndexSetsFromJSON(data []byte) map[string]pythonIndexSet {
	var raw any
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	indexes := map[string]pythonIndexSet{}
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case map[string]any:
			if kind, _ := typed["kind"].(string); kind == "python-wheels" {
				lockfile, _ := typed["lockfile"].(string)
				indexSet := pythonIndexSet{}
				indexSet.Lockfile = lockfile
				indexSet.Primary, _ = typed["index_url"].(string)
				if extra, ok := typed["extra_index_url"].([]any); ok {
					for _, item := range extra {
						if index, ok := item.(string); ok && strings.TrimSpace(index) != "" {
							indexSet.Extras = append(indexSet.Extras, strings.TrimSpace(index))
						}
					}
				}
				if strings.TrimSpace(lockfile) != "" {
					indexes[filepath.Base(lockfile)] = indexSet
				}
			}
			for _, item := range typed {
				walk(item)
			}
		case []any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(raw)
	return indexes
}

func updatePythonLocks(ctx context.Context, resolution Resolution) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	if resolution.PythonBootstrap != nil {
		return bootstrapPythonLock(ctx, resolution)
	}
	if len(resolution.PythonLockfiles) == 0 {
		return "", fmt.Errorf("resource Python lock update has no requirements lockfiles")
	}
	if strings.TrimSpace(resolution.PackageName) == "" || !strings.Contains(strings.Join(resolution.Argv, " "), resolution.PackageName+"==") {
		return "", fmt.Errorf("resource Python lock update requires an explicit package version")
	}
	version := ""
	for _, arg := range resolution.Argv {
		if strings.HasPrefix(arg, resolution.PackageName+"==") {
			version = strings.TrimPrefix(arg, resolution.PackageName+"==")
		}
	}
	if version == "" {
		return "", fmt.Errorf("resource Python lock update requires an explicit package version")
	}
	work, err := os.MkdirTemp(resolution.SurfaceRoot, ".sda-python-lock-")
	if err != nil {
		return "", fmt.Errorf("create temporary Python lock workspace: %w", err)
	}
	defer os.RemoveAll(work)

	type pendingLock struct{ source, output string }
	pending := make([]pendingLock, 0, len(resolution.PythonLockfiles))
	for index, lockPath := range resolution.PythonLockfiles {
		original, err := os.ReadFile(lockPath)
		if err != nil {
			return "", fmt.Errorf("read Python lock %s: %w", filepath.Base(lockPath), err)
		}
		inputPath := filepath.Join(work, fmt.Sprintf("requirements-%d.in", index))
		outputPath := filepath.Join(work, fmt.Sprintf("requirements-%d.generated.lock", index))
		updatedInput, err := replacePythonRequirement(string(original), resolution.PackageName, version)
		if err != nil {
			return "", fmt.Errorf("prepare Python lock %s: %w", filepath.Base(lockPath), err)
		}
		if err := os.WriteFile(inputPath, []byte(updatedInput), 0o600); err != nil {
			return "", err
		}
		indexSet := resolution.PythonIndexSets[filepath.Base(lockPath)]
		if strings.TrimSpace(indexSet.Primary) == "" {
			indexSet.Primary = "https://pypi.org/simple"
		}
		output, err := compilePythonLock(ctx, resolution.SurfaceRoot, inputPath, outputPath, indexSet)
		if err != nil {
			return string(output), fmt.Errorf("uv wheel-only lock resolution failed for %s: %w", filepath.Base(lockPath), err)
		}
		generated, err := os.ReadFile(outputPath)
		if err != nil {
			return string(output), fmt.Errorf("read generated Python lock %s: %w", filepath.Base(lockPath), err)
		}
		relativeLock, err := filepath.Rel(resolution.RepositoryRoot, lockPath)
		if err != nil {
			return string(output), fmt.Errorf("resolve resource lock path: %w", err)
		}
		generatedText := canonicalizePythonLockOutput(generated, inputPath, filepath.ToSlash(relativeLock))
		if err := os.WriteFile(outputPath, []byte(generatedText), 0o600); err != nil {
			return string(output), err
		}
		pending = append(pending, pendingLock{source: outputPath, output: lockPath})
	}
	for _, lock := range pending {
		if err := os.Rename(lock.source, lock.output); err != nil {
			return "", fmt.Errorf("publish generated Python lock %s: %w", filepath.Base(lock.output), err)
		}
	}
	return "wheel-only Python locks updated through Scenario Dependency Analyzer", nil
}

func compilePythonLock(ctx context.Context, root, input, output string, indexes pythonIndexSet) (string, error) {
	args := []string{"pip", "compile", "--generate-hashes", "--only-binary", ":all:", "--no-build", "--no-config", "--python-version", "3.12", "--python-platform", "x86_64-unknown-linux-gnu", "--index-url", indexes.Primary}
	for _, index := range indexes.Extras {
		args = append(args, "--extra-index-url", index)
	}
	if len(indexes.Extras) > 0 {
		args = append(args, "--index-strategy", "unsafe-best-match")
	}
	args = append(args, "--output-file", output, input)
	command := exec.CommandContext(ctx, "uv", args...)
	command.Dir = root
	command.WaitDelay = 5 * time.Second
	var diagnostic hostToolOutput
	command.Stdout, command.Stderr = &diagnostic, &diagnostic
	err := command.Run()
	return diagnostic.String(), err
}

func canonicalizePythonLockOutput(generated []byte, inputPath, relativeLock string) string {
	lines := strings.Split(string(generated), "\n")
	start := 0
	for start < len(lines) && (strings.TrimSpace(lines[start]) == "" || strings.HasPrefix(strings.TrimSpace(lines[start]), "#")) {
		start++
	}
	body := strings.Join(lines[start:], "\n")
	body = strings.ReplaceAll(body, inputPath, relativeLock)
	// uv may render command input paths relative to its working directory even
	// when the caller passed an absolute path. Remove the temporary workspace
	// component in that form too, so retained locks never refer to deleted files.
	if marker := filepath.Base(filepath.Dir(inputPath)) + string(filepath.Separator) + filepath.Base(inputPath); marker != "" {
		body = strings.ReplaceAll(body, filepath.ToSlash(marker), relativeLock)
	}
	return "# Generated by Scenario Dependency Analyzer using uv; wheel-only policy.\n\n" + body
}

func replacePythonRequirement(contents, packageName, version string) (string, error) {
	target := strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(packageName))
	lines := strings.Split(contents, "\n")
	result := make([]string, 0, len(lines)+1)
	found := false
	for i := 0; i < len(lines); i++ {
		match := pinnedRequirement.FindStringSubmatch(lines[i])
		if len(match) == 3 && strings.NewReplacer("-", "", "_", "", ".", "").Replace(strings.ToLower(match[1])) == target {
			if found {
				return "", fmt.Errorf("package %q appears more than once in the lock", packageName)
			}
			found = true
			result = append(result, packageName+"=="+version)
			for i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
				i++
			}
			continue
		}
		result = append(result, lines[i])
	}
	if !found {
		result = append(result, packageName+"=="+version)
	}
	return strings.Join(result, "\n"), nil
}

func baseNames(paths []string) []string {
	result := make([]string, len(paths))
	for i, path := range paths {
		result[i] = filepath.Base(path)
	}
	return result
}
