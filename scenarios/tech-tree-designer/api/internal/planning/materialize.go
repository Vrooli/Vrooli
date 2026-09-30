package planning

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type FilesystemMaterializer struct {
	SchemasRoot string
	ProtoRoot   string
	Command     func(context.Context, string, ...string) error
}

func NewFilesystemMaterializer(schemasRoot string) *FilesystemMaterializer {
	return &FilesystemMaterializer{SchemasRoot: schemasRoot}
}

var _ Materializer = (*FilesystemMaterializer)(nil)

func (m *FilesystemMaterializer) Materialize(ctx context.Context, scenario Scenario) (MaterializeResult, error) {
	schemasRoot, err := resolveSchemasRoot(m.SchemasRoot)
	if err != nil {
		return MaterializeResult{}, err
	}
	protoRoot := strings.TrimSpace(m.ProtoRoot)
	if protoRoot == "" {
		protoRoot = filepath.Dir(schemasRoot)
	}
	destRoot := filepath.Join(schemasRoot, scenario.Slug)
	var written []string
	for _, file := range scenario.Files {
		path, err := NormalizeProtoPath(file.Path)
		if err != nil {
			return MaterializeResult{}, err
		}
		rel := strings.TrimPrefix(path, scenario.Slug+"/")
		dst := filepath.Join(destRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return MaterializeResult{}, fmt.Errorf("prepare materialized proto path: %w", err)
		}
		if err := os.WriteFile(dst, []byte(file.Text), 0o644); err != nil {
			return MaterializeResult{}, fmt.Errorf("write materialized proto file: %w", err)
		}
		written = append(written, filepath.ToSlash(dst))
	}
	sort.Strings(written)
	run := m.Command
	if run == nil {
		run = func(ctx context.Context, dir string, args ...string) error {
			cmd := exec.CommandContext(ctx, "make", args...)
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("make %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
			}
			return nil
		}
	}
	if err := run(ctx, protoRoot, "generate"); err != nil {
		return MaterializeResult{}, err
	}
	return MaterializeResult{Slug: scenario.Slug, WrittenPaths: written, Generated: true}, nil
}

func resolveSchemasRoot(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "packages", "proto", "schemas")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
	}
	return "", fmt.Errorf("could not locate packages/proto/schemas from %s", wd)
}
