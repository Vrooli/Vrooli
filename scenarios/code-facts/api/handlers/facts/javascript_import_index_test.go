package facts

import (
	"strings"
	"testing"

	"code-facts/internal/catalog"
)

func TestJavaScriptImportDocumentsExtractLiteralModuleEdgesWithProvenance(t *testing.T) {
	source := catalog.SourceFile{ID: "ui-1", Path: "scenarios/demo/ui/src/App.tsx", Language: "typescript", Scope: "scenario:demo", Authority: "authoritative", Hash: "sha256:test"}
	payload := []byte("// import fake from 'bad';\nimport React from 'react';\nimport './local.css';\nexport { Button } from '../components/Button';\nconst lazy = import(\"./lazy\");\nconst cjs = require('zod');\nconst message = `import x from 'fake'`;\nconst pattern = /import from 'also-fake'/;\nfunction inner() { import x from 'nested-fake'; }\n")
	docs, incomplete, err := javascriptImportDocuments(source, payload)
	if err != nil || incomplete {
		t.Fatalf("docs=%+v partial=%v err=%v", docs, incomplete, err)
	}
	want := []string{"react", "./local.css", "../components/Button", "./lazy", "zod"}
	if len(docs) != len(want) {
		t.Fatalf("docs=%+v", docs)
	}
	for i, doc := range docs {
		if doc.ExactText != want[i] || doc.Path != source.Path || doc.Scope != source.Scope || doc.SourceHash != source.Hash || doc.StartLine == 0 || doc.Kind != "imports" {
			t.Fatalf("doc[%d]=%+v", i, doc)
		}
	}
	if docs[2].Title != source.Path+" re-exports ../components/Button" || !strings.Contains(docs[3].Body, "dynamically imports") {
		t.Fatalf("edge kinds lost: %+v", docs)
	}
	duplicates, partial, err := javascriptImportDocuments(source, []byte("import A from 'same'; import B from 'same';\n"))
	if err != nil || partial || len(duplicates) != 2 || duplicates[0].ID == duplicates[1].ID {
		t.Fatalf("same-line duplicate module edges lost identity: docs=%+v partial=%v err=%v", duplicates, partial, err)
	}
}

func TestJavaScriptImportDocumentsIgnoresCommentsAndMarksUnsupportedSpecifierPartial(t *testing.T) {
	source := catalog.SourceFile{ID: "ui-1", Path: "scenarios/demo/ui/index.js", Language: "javascript", Scope: "scenario:demo", Hash: "sha256:test"}
	docs, partial, err := javascriptImportDocuments(source, []byte("import(`./${name}`);\n// require('fake')\nimport(`./static`);\n"))
	if err != nil || !partial || len(docs) != 1 || docs[0].ExactText != "./static" {
		t.Fatalf("docs=%+v partial=%v err=%v", docs, partial, err)
	}
}

func TestPolyglotImportDocumentsDoesNotClaimNonCodeOrUnsupportedExtensions(t *testing.T) {
	source := catalog.SourceFile{Path: "scenarios/demo/ui/package.json", Language: "json"}
	docs, partial, err := importDocuments(source, []byte(`{"dependencies":{"fake":"1"}}`))
	if err != nil || partial || len(docs) != 0 {
		t.Fatalf("docs=%+v partial=%v err=%v", docs, partial, err)
	}
}
