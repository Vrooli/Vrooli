package components

import (
	"context"
	"fmt"
	"testing"
)

type ingestMapReader map[string]string

func (r ingestMapReader) Read(_ context.Context, scenario, sourcePath string) ([]byte, error) {
	if scenario != "agent-buyer-range" {
		return nil, fmt.Errorf("unexpected scenario %q", scenario)
	}
	if source, ok := r[sourcePath]; ok {
		return []byte(source), nil
	}
	return nil, fmt.Errorf("source %q not found", sourcePath)
}

func TestReadIngestUnitResolvesMultiDotTypeScriptCompanion(t *testing.T) {
	reader := ingestMapReader{
		"ui/src/features/notes/AttachmentUpload.tsx": `import { selectors } from "../../consts/selectors"; export const AttachmentUpload = () => selectors.notes;`,
		"ui/src/consts/selectors.ts":                 `import { librarySelectors } from "./selectors.library"; export const selectors = { notes: librarySelectors.notes };`,
		"ui/src/consts/selectors.library.ts":         `export const librarySelectors = { notes: "notes" } as const;`,
	}

	unit, err := readIngestUnit(context.Background(), reader, "agent-buyer-range", "ui/src/features/notes/AttachmentUpload.tsx", []string{"ui/src/consts/selectors.library.ts"})
	if err != nil {
		t.Fatalf("read ingest closure: %v", err)
	}
	if len(unit) != 3 {
		t.Fatalf("expected entry plus two companions, got %#v", ingestFilePaths(unit))
	}
	if _, err := resolveRelativeIngestImport(context.Background(), reader, "agent-buyer-range", "ui/src/consts", "./selectors.library"); err != nil {
		t.Fatalf("resolve multi-dot extensionless import: %v", err)
	}
}
