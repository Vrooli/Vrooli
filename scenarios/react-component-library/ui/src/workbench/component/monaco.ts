/**
 * Local Monaco bootstrap for the governed source editor.
 *
 * `@monaco-editor/react` defaults to downloading Monaco from a CDN. Vrooli runs
 * offline and behind tunnels, so a CDN loader would turn "open the editor" into
 * a network dependency. Importing the ESM editor here keeps the worker and
 * language service inside the scenario's own bundle.
 *
 * This module is only imported by the lazily loaded `CodeEditor`, so the
 * several-megabyte editor chunk is fetched the first time a person actually
 * edits source — never on a page or component visit.
 */
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
import "monaco-editor/esm/vs/language/typescript/monaco.contribution";
import { loader } from "@monaco-editor/react";
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";
import TypeScriptWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker";

self.MonacoEnvironment = {
  getWorker(_workerID, label) {
    if (label === "typescript" || label === "javascript") {
      return new TypeScriptWorker();
    }
    return new EditorWorker();
  },
};

monaco.languages.typescript.typescriptDefaults.setCompilerOptions({
  allowNonTsExtensions: true,
  allowJs: true,
  jsx: monaco.languages.typescript.JsxEmit.ReactJSX,
  module: monaco.languages.typescript.ModuleKind.ESNext,
  moduleResolution: monaco.languages.typescript.ModuleResolutionKind.NodeJs,
  noEmit: true,
  target: monaco.languages.typescript.ScriptTarget.ESNext,
  esModuleInterop: true,
});

// The library's imports resolve through the catalog, not through node_modules.
// Syntax validation still catches broken local edits; semantic validation would
// paint every `@vrooli/react-component-library/...` import red.
monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions({
  noSemanticValidation: true,
  noSyntaxValidation: false,
});
monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
  noSemanticValidation: true,
  noSyntaxValidation: false,
});

monaco.editor.defineTheme("rcl-nocturne", {
  base: "vs-dark",
  inherit: true,
  rules: [
    { token: "comment", foreground: "6f6a80", fontStyle: "italic" },
    { token: "keyword", foreground: "c8a8ff" },
    { token: "string", foreground: "9bd8b0" },
    { token: "number", foreground: "f0c987" },
    { token: "type", foreground: "8fb8ff" },
    { token: "type.identifier", foreground: "8fb8ff" },
    { token: "delimiter", foreground: "8b8698" },
  ],
  colors: {
    "editor.background": "#0b0a0f",
    "editor.foreground": "#e7e4ef",
    "editorLineNumber.foreground": "#48445a",
    "editorLineNumber.activeForeground": "#b7b1c8",
    "editor.selectionBackground": "#5b4b8a66",
    "editor.lineHighlightBackground": "#17151f",
    "editorCursor.foreground": "#c8a8ff",
    "editorIndentGuide.background1": "#211e2c",
    "editorWidget.background": "#14121c",
    "editorSuggestWidget.background": "#14121c",
    "editorSuggestWidget.selectedBackground": "#2a2438",
    "scrollbarSlider.background": "#2b273966",
  },
});

loader.config({ monaco });

export { monaco };
