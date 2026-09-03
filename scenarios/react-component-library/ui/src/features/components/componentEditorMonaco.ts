import { type Monaco } from "@monaco-editor/react";
import type { editor } from "monaco-editor";

import { componentsClient, resolveLibraryImport } from "../../api/components";
import {
  createLibraryImportDefinitionProvider,
  createLibraryImportHoverProvider,
  isLibraryImportSpecifier,
  libraryImportsInModel,
} from "./libraryImportProviders";

export function configureEditorBeforeMount(monaco: Monaco) {
  const diagnosticsOptions = { noSemanticValidation: true, noSyntaxValidation: false };
  monaco.languages.typescript.typescriptDefaults.setDiagnosticsOptions(diagnosticsOptions);
  monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions(diagnosticsOptions);
}

export function configureEditorMount(
  monacoEditor: editor.IStandaloneCodeEditor,
  monaco: Monaco,
  onSave: () => void,
) {
  monacoEditor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, onSave);
  const languages = ["typescript", "typescriptreact", "javascript", "javascriptreact"];
  const resolveImport = (specifier: string, fromPath: string) =>
    resolveLibraryImport({ specifier, fromPath });
  const model = monacoEditor.getModel();
  const diagnosticsOwner = "rcl-library-imports";
  let diagnosticsRun = 0;
  const refreshDiagnostics = async () => {
    if (!model) return;
    const run = ++diagnosticsRun;
    const imports = libraryImportsInModel(model).filter((item) =>
      isLibraryImportSpecifier(item.specifier),
    );
    const resolutions = await Promise.all(
      imports.map(async (item) => ({
        item,
        resolution: await resolveImport(item.specifier, model.uri.path),
      })),
    );
    if (run !== diagnosticsRun || model.isDisposed()) return;
    monaco.editor.setModelMarkers(
      model,
      diagnosticsOwner,
      resolutions
        .filter(({ resolution }) => !resolution.resolved)
        .map(({ item, resolution }) => ({
          severity: monaco.MarkerSeverity.Error,
          message: resolution.diagnostic || "The library import does not resolve.",
          ...item.range,
        })),
    );
  };
  const contentListener = model?.onDidChangeContent(() => void refreshDiagnostics());
  void refreshDiagnostics();
  monaco.languages.registerHoverProvider(
    languages,
    createLibraryImportHoverProvider(resolveImport),
  );
  monaco.languages.registerDefinitionProvider(
    languages,
    createLibraryImportDefinitionProvider(
      monaco,
      resolveImport,
      async (resolution, uri) => {
        if (monaco.editor.getModel(uri)) return;
        const source = await componentsClient.getComponentVersionContent({
          componentId: resolution.libraryId,
          version: resolution.version,
        });
        monaco.editor.createModel(source.content, "typescript", uri);
      },
      (_resolution, _range) => void refreshDiagnostics(),
    ),
  );
  monacoEditor.onDidDispose(() => {
    contentListener?.dispose();
    if (model && !model.isDisposed()) monaco.editor.setModelMarkers(model, diagnosticsOwner, []);
  });
}
