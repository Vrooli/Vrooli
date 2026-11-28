import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const bootstrap = vi.hoisted(() => ({
  setCompilerOptions: vi.fn(),
  setTypeScriptDiagnostics: vi.fn(),
  setJavaScriptDiagnostics: vi.fn(),
  defineTheme: vi.fn(),
  loaderConfig: vi.fn(),
  editorWorker: vi.fn(),
  typeScriptWorker: vi.fn(),
}));

const mockMonaco = {
  languages: {
    typescript: {
      typescriptDefaults: {
        setCompilerOptions: bootstrap.setCompilerOptions,
        setDiagnosticsOptions: bootstrap.setTypeScriptDiagnostics,
      },
      javascriptDefaults: { setDiagnosticsOptions: bootstrap.setJavaScriptDiagnostics },
      JsxEmit: { ReactJSX: "react-jsx" },
      ModuleKind: { ESNext: "esnext" },
      ModuleResolutionKind: { NodeJs: "node" },
      ScriptTarget: { ESNext: "latest" },
    },
  },
  editor: { defineTheme: bootstrap.defineTheme },
};

vi.mock("monaco-editor/esm/vs/editor/editor.api", () => mockMonaco);
vi.mock("monaco-editor/esm/vs/language/typescript/monaco.contribution", () => ({}));
vi.mock("@monaco-editor/react", () => ({ loader: { config: bootstrap.loaderConfig } }));
vi.mock("monaco-editor/esm/vs/editor/editor.worker?worker", () => ({
  default: function EditorWorker() {
    bootstrap.editorWorker();
  },
}));
vi.mock("monaco-editor/esm/vs/language/typescript/ts.worker?worker", () => ({
  default: function TypeScriptWorker() {
    bootstrap.typeScriptWorker();
  },
}));

describe("local Monaco bootstrap", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    Reflect.deleteProperty(self, "MonacoEnvironment");
    vi.clearAllMocks();
  });

  it("configures the bundled loader, language workers, compiler, diagnostics, and theme", async () => {
    const { monaco } = await import("./monaco");

    expect(bootstrap.loaderConfig).toHaveBeenCalledWith({ monaco });
    expect(bootstrap.setCompilerOptions).toHaveBeenCalledWith({
      allowNonTsExtensions: true,
      allowJs: true,
      jsx: "react-jsx",
      module: "esnext",
      moduleResolution: "node",
      noEmit: true,
      target: "latest",
      esModuleInterop: true,
    });
    expect(bootstrap.setTypeScriptDiagnostics).toHaveBeenCalledWith({
      noSemanticValidation: true,
      noSyntaxValidation: false,
    });
    expect(bootstrap.setJavaScriptDiagnostics).toHaveBeenCalledWith({
      noSemanticValidation: true,
      noSyntaxValidation: false,
    });
    expect(bootstrap.defineTheme).toHaveBeenCalledWith(
      "rcl-nocturne",
      expect.objectContaining({
        base: "vs-dark",
        inherit: true,
        rules: expect.arrayContaining([
          expect.objectContaining({ token: "keyword", foreground: "c8a8ff" }),
          expect.objectContaining({ token: "string", foreground: "9bd8b0" }),
        ]),
        colors: expect.objectContaining({
          "editor.background": "#0b0a0f",
          "editor.foreground": "#e7e4ef",
        }),
      }),
    );
  });

  it("routes JavaScript and TypeScript labels to their local worker", async () => {
    await import("./monaco");

    const environment = self.MonacoEnvironment;
    expect(environment).toBeDefined();
    expect(environment?.getWorker?.("editor", "typescript")).toBeDefined();
    expect(environment?.getWorker?.("editor", "javascript")).toBeDefined();
    expect(environment?.getWorker?.("editor", "json")).toBeDefined();
    expect(bootstrap.typeScriptWorker).toHaveBeenCalledTimes(2);
    expect(bootstrap.editorWorker).toHaveBeenCalledOnce();
  });
});
