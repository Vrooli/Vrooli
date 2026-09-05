import type { Monaco } from "@monaco-editor/react";
import type { editor, IRange, languages, Position, Uri } from "monaco-editor";

import type { LibraryImportResolution } from "../../api/components";

const LIBRARY_IMPORT_PREFIX = "@vrooli/react-component-library/";

export interface ImportAtPosition {
  specifier: string;
  range: IRange;
}

export interface LibraryImportInModel extends ImportAtPosition {
  lineNumber: number;
}

export type ResolveImport = (
  specifier: string,
  fromPath: string,
) => Promise<LibraryImportResolution>;

export function importAtPosition(
  model: editor.ITextModel,
  position: Position,
): ImportAtPosition | undefined {
  const line = model.getLineContent(position.lineNumber);
  const importPattern = /(?:\bfrom\s*|\bimport\s*(?:\(\s*)?|\bexport\s+from\s*)(["'])([^"']+)\1/g;
  let match: RegExpExecArray | null;
  while ((match = importPattern.exec(line)) !== null) {
    const quoteOffset = match[0].indexOf(match[1] ?? "");
    const startColumn = match.index + quoteOffset + 2;
    const specifier = match[2];
    if (!specifier) continue;
    const endColumn = startColumn + specifier.length;
    if (position.column < startColumn || position.column > endColumn) continue;
    return {
      specifier,
      range: {
        startLineNumber: position.lineNumber,
        startColumn,
        endLineNumber: position.lineNumber,
        endColumn,
      },
    };
  }
  return undefined;
}

/** Return every import-like library specifier in a model, including repeated uses. */
export function libraryImportsInModel(model: editor.ITextModel): LibraryImportInModel[] {
  const imports: LibraryImportInModel[] = [];
  for (let lineNumber = 1; lineNumber <= model.getLineCount(); lineNumber += 1) {
    const line = model.getLineContent(lineNumber);
    const importPattern = /(?:\bfrom\s*|\bimport\s*(?:\(\s*)?|\bexport\s+from\s*)(["'])([^"']+)\1/g;
    let match: RegExpExecArray | null;
    while ((match = importPattern.exec(line)) !== null) {
      const quoteOffset = match[0].indexOf(match[1] ?? "");
      const startColumn = match.index + quoteOffset + 2;
      const specifier = match[2];
      if (!specifier) continue;
      imports.push({
        lineNumber,
        specifier,
        range: {
          startLineNumber: lineNumber,
          startColumn,
          endLineNumber: lineNumber,
          endColumn: startColumn + specifier.length,
        },
      });
    }
  }
  return imports;
}

function markdownResolution(resolution: LibraryImportResolution): string {
  if (!resolution.resolved) {
    return `**Unresolved library import**\n\n\`${resolution.specifier}\`\n\n${resolution.diagnostic || "The import does not resolve."}`;
  }
  return [
    `**${resolution.assetId || resolution.libraryId}**`,
    "",
    `- Version: \`${resolution.version}\``,
    `- Export kind: \`${resolution.exportKind}\``,
    `- Description: ${resolution.description || "No description recorded."}`,
  ].join("\n");
}

function definitionUri(monaco: Monaco, resolution: LibraryImportResolution) {
  const sourcePath = resolution.sourcePath.replace(/^\/+/, "");
  return monaco.Uri.parse(
    `rcl://library/${encodeURIComponent(resolution.libraryId)}/${encodeURIComponent(resolution.version)}/${sourcePath}`,
  );
}

export function createLibraryImportHoverProvider(resolve: ResolveImport): languages.HoverProvider {
  return {
    async provideHover(model, position) {
      const target = importAtPosition(model, position);
      if (!target) return undefined;
      const resolution = await resolve(target.specifier, model.uri.path);
      return { contents: [{ value: markdownResolution(resolution) }], range: target.range };
    },
  };
}

export function createLibraryImportDefinitionProvider(
  monaco: Monaco,
  resolve: ResolveImport,
  loadDefinition?: (resolution: LibraryImportResolution, uri: Uri) => Promise<void>,
  onUnresolved?: (resolution: LibraryImportResolution, range: IRange) => void,
): languages.DefinitionProvider {
  return {
    async provideDefinition(model, position) {
      const target = importAtPosition(model, position);
      if (!target) return undefined;
      const resolution = await resolve(target.specifier, model.uri.path);
      if (!resolution.resolved) {
        onUnresolved?.(resolution, target.range);
        return undefined;
      }
      const uri = definitionUri(monaco, resolution);
      await loadDefinition?.(resolution, uri);
      return {
        uri,
        range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
      };
    },
  };
}

export function isLibraryImportSpecifier(value: string): boolean {
  return value.startsWith(LIBRARY_IMPORT_PREFIX);
}
