package filerelations

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestServiceGetRelatedFilesFindsImportsAndConventions(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeRelationFixture(t, repoRoot, "src/components/Button.tsx", `
import React from "react";
import { formatLabel } from "../utils/format";
export { ButtonGroup } from "./ButtonGroup";
`)
	writeRelationFixture(t, repoRoot, "src/components/ButtonGroup.tsx", "export const ButtonGroup = null;")
	writeRelationFixture(t, repoRoot, "src/components/Button.test.tsx", "test('button', () => {});")
	writeRelationFixture(t, repoRoot, "src/components/Button.types.ts", "export type ButtonTone = 'primary';")
	writeRelationFixture(t, repoRoot, "src/components/index.ts", "export * from './Button';")
	writeRelationFixture(t, repoRoot, "src/utils/format.ts", "export const formatLabel = (v: string) => v;")

	related, err := NewService().GetRelatedFiles(context.Background(), "src/components/Button.tsx", repoRoot)
	if err != nil {
		t.Fatalf("GetRelatedFiles returned error: %v", err)
	}

	assertRelatedFiles(t, related, []RelatedFile{
		{Path: "src/utils/format.ts", RelationType: RelationImports},
		{Path: "src/components/ButtonGroup.tsx", RelationType: RelationImports},
		{Path: "src/components/Button.test.tsx", RelationType: RelationTest},
		{Path: "src/components/index.ts", RelationType: RelationIndex},
		{Path: "src/components/Button.types.ts", RelationType: RelationTypes},
	})
}

func TestServiceGetRelatedFilesMapsNestedTestBackToSource(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeRelationFixture(t, repoRoot, "src/lib/cache.ts", "export const cache = new Map();")
	writeRelationFixture(t, repoRoot, "src/lib/__tests__/cache.spec.ts", "import { cache } from '../cache';")

	related, err := NewService().GetRelatedFiles(context.Background(), "src/lib/__tests__/cache.spec.ts", repoRoot)
	if err != nil {
		t.Fatalf("GetRelatedFiles returned error: %v", err)
	}

	assertRelatedFiles(t, related, []RelatedFile{
		{Path: "src/lib/cache.ts", RelationType: RelationImports},
		{Path: "src/lib/cache.ts", RelationType: RelationTest},
	})
}

func TestServiceGetRelatedFilesReturnsEmptyForUnsupportedFile(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	writeRelationFixture(t, repoRoot, "README.md", "# Example\n")

	related, err := NewService().GetRelatedFiles(context.Background(), "README.md", repoRoot)
	if err != nil {
		t.Fatalf("GetRelatedFiles returned error: %v", err)
	}
	if len(related) != 0 {
		t.Fatalf("related = %#v, want empty", related)
	}
}

func writeRelationFixture(t *testing.T, repoRoot, relPath, content string) {
	t.Helper()

	absPath := filepath.Join(repoRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(absPath), err)
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relPath, err)
	}
}

func assertRelatedFiles(t *testing.T, got, want []RelatedFile) {
	t.Helper()

	sortRelatedFiles(got)
	sortRelatedFiles(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("related files = %#v, want %#v", got, want)
	}
}

func sortRelatedFiles(files []RelatedFile) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Path == files[j].Path {
			return files[i].RelationType < files[j].RelationType
		}
		return files[i].Path < files[j].Path
	})
}
