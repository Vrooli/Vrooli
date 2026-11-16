import type { WorkbookResult } from "./preview/workbookData.js";

/**
 * Engines the package does not bundle. Each one needs a third-party library a
 * host may not be able, or may not want, to install. The host supplies a
 * loader that imports it; the package never imports those libraries by bare
 * specifier, so a host without one installs and bundles nothing extra and the
 * affected content falls back to a translated "not available here" state.
 */

/** Parses a spreadsheet off the main thread. */
export interface SpreadsheetEngine {
  /**
   * Fetch and parse the workbook at `href`. Resolve with a result for every
   * failure the viewer words (`invalid`, `encrypted`, `limit`, `fetch`);
   * reject only for an unexpected fault. Abort `signal` to release the worker.
   */
  parse(href: string, signal: AbortSignal): Promise<WorkbookResult>;
}

/** The slice of the Mermaid API the viewer calls. Mermaid's default export satisfies it. */
export interface DiagramEngine {
  initialize(config: Record<string, unknown>): void;
  render(id: string, source: string): Promise<{ svg: string }>;
}

export interface FileViewerLoaders {
  /**
   * Enables spreadsheet previews. Use the engine this package ships:
   * `() => import("@vrooli/file-viewer/spreadsheet").then((m) => m.spreadsheetEngine)`.
   * It needs the optional peer `xlsx`.
   */
  spreadsheet?: () => Promise<SpreadsheetEngine>;
  /**
   * Enables Mermaid diagrams in Markdown:
   * `() => import("mermaid").then((m) => m.default)`.
   * It needs the optional peer `mermaid`.
   */
  diagram?: () => Promise<DiagramEngine>;
}
