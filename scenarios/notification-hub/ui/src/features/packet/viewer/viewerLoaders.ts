import type { FileViewerLoaders } from "@vrooli/file-viewer";

/**
 * The two engines the shared viewer takes from its host. Both are imported on
 * first use, so a decision with no workbook or diagram never downloads them.
 * Defined once at module level: the provider's context is keyed by these functions.
 */
export const viewerLoaders: FileViewerLoaders = {
  spreadsheet: () => import("@vrooli/file-viewer/spreadsheet").then((module) => module.spreadsheetEngine),
  diagram: () => import("mermaid").then((module) => module.default),
};
