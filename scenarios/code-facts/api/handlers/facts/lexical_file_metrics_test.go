package facts

import (
	"context"
	"strings"
	"testing"
)

func TestInspectLexicalFilePreservesLanguageMetricSemantics(t *testing.T) {
	tests := []struct {
		name, language, source string
		want                   lexicalFileMetrics
	}{
		{
			name: "Go markers and comments", language: "go",
			source: "package demo\n// TODO FIXME HACK\nfunc Work() {}\n\n",
			want:   lexicalFileMetrics{NonEmptyLines: 3, CodeLines: 2, CommentLines: 1, TodoCount: 1, FixmeCount: 1, HackCount: 1},
		},
		{
			name: "TypeScript assertions ignore comments", language: "typescript",
			source: "// as any and @ts-ignore\nconst result = value as any;\nconst item = value as Widget;\n// @ts-expect-error\nconst field = value!.field;\n",
			want:   lexicalFileMetrics{NonEmptyLines: 5, CodeLines: 3, CommentLines: 2, AsAnyCount: 1, AsTypeAssertionCount: 1, TsIgnoreCount: 2, NonNullAssertionCount: 1},
		},
		{
			name: "Python docstrings", language: "python",
			source: "import os\ndef work():\n    \"\"\"TODO documentation\n    details\n    \"\"\"\n    pass\n",
			want:   lexicalFileMetrics{NonEmptyLines: 6, CodeLines: 3, CommentLines: 3, TodoCount: 1, FallbackImportCount: 1, FallbackFunctionCount: 1},
		},
		{
			name: "Rust syntax fallbacks", language: "rust",
			source: "use std::fmt;\npub fn work() {}\n",
			want:   lexicalFileMetrics{NonEmptyLines: 2, CodeLines: 2, FallbackImportCount: 1, FallbackFunctionCount: 1},
		},
		{
			name: "JavaScript syntax fallbacks", language: "javascript",
			source: "import { value } from './value';\nconst work = () => {};\nif (ready) { return; }\n",
			want:   lexicalFileMetrics{NonEmptyLines: 3, CodeLines: 3, FallbackImportCount: 1, FallbackFunctionCount: 1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := inspectLexicalFile(context.Background(), strings.NewReader(test.source), test.language)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("metrics = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestInspectLexicalFileRejectsInvalidOrCanceledInput(t *testing.T) {
	if _, err := inspectLexicalFile(context.Background(), strings.NewReader(string([]byte{0xff})), "go"); err != errInvalidUTF8Source {
		t.Fatalf("invalid UTF-8 error = %v, want %v", err, errInvalidUTF8Source)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspectLexicalFile(ctx, strings.NewReader("package demo\n"), "go"); err != context.Canceled {
		t.Fatalf("canceled input error = %v, want context.Canceled", err)
	}
}

func TestInspectLexicalFileHandlesLongMinifiedSourceLine(t *testing.T) {
	content := "const payload = '" + strings.Repeat("x", 128*1024) + "';\n"
	metrics, err := inspectLexicalFile(context.Background(), strings.NewReader(content), "javascript")
	if err != nil {
		t.Fatalf("inspect long source line: %v", err)
	}
	if metrics.NonEmptyLines != 1 || metrics.CodeLines != 1 {
		t.Fatalf("long source line metrics = non-empty %d, code %d; want 1, 1", metrics.NonEmptyLines, metrics.CodeLines)
	}
}
