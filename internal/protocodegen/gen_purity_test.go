package protocodegen

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenTreeContainsOnlyBufOutput(t *testing.T) {
	genRoot := protoGenPath(t)
	var violations []string

	err := filepath.WalkDir(genRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(genRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !isAllowedGeneratedPath(rel) {
			violations = append(violations, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", genRoot, err)
	}

	if len(violations) > 0 {
		t.Fatalf(
			"packages/proto/gen is buf-output only; move non-buf artifacts into the owning scenario:\n  %s",
			strings.Join(violations, "\n  "),
		)
	}
}

func isAllowedGeneratedPath(path string) bool {
	switch {
	case path == "typescript/package.json":
		return true
	case path == "descriptor/image.binpb":
		return true
	case strings.HasPrefix(path, "manifests/") && strings.HasSuffix(path, ".lock.json"):
		return true
	case strings.HasPrefix(path, "go/"):
		return strings.HasSuffix(path, ".pb.go") || strings.HasSuffix(path, ".connect.go")
	case strings.HasPrefix(path, "typescript/"):
		return strings.HasSuffix(path, "_pb.ts") || strings.HasSuffix(path, "_pb.js")
	case strings.HasPrefix(path, "python/"):
		return path == "python/py.typed" ||
			strings.HasSuffix(path, "/py.typed") ||
			strings.HasSuffix(path, "_pb2.py") ||
			strings.HasSuffix(path, "_pb2.pyi")
	default:
		return false
	}
}

func protoGenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRootPath(t), "packages", "proto", "gen")
}
