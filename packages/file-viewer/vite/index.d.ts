import type { Plugin } from "vite";

/**
 * Serves and emits the PDF.js resources and builds the slide runtime the
 * viewer loads by URL. Add it to the host's `plugins`, and add
 * `@vrooli/file-viewer` to `optimizeDeps.exclude`.
 */
export declare function fileViewer(): Plugin[];
export default fileViewer;
