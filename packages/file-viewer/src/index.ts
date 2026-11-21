// @vrooli/file-viewer public API. Everything a host app needs is exported from
// here; see README.md for the PreviewSource contract and consumer setup.

// The contract between the viewer and whatever holds the files.
export * from "./source/types.js";
export {
  createManifestPreviewSource,
  normalizeManifestPath,
  type ManifestFile,
  type ManifestSourceOptions,
} from "./source/manifestSource.js";

// The viewer: controller (state machine) and shell.
export { useFilePreviewController, type FilePreviewController, type OpenPreviewOptions } from "./viewer/useFilePreviewController.js";
export { FileViewer, PreviewBreadcrumbs, type FileViewerProps, type ViewerActionContext } from "./viewer/FileViewer.js";
export {
  IDLE_PREVIEW_STATE,
  type PreviewFrame,
  type PreviewSearch,
  type PreviewState,
  type PreviewStatus,
} from "./viewer/state.js";

// Layout constants for hosts that dock the viewer beside their own workspace.
export { ARTIFACT_VIEWER_MIN_WIDTH, ARTIFACT_WORKSPACE_MIN_WIDTH, shouldFocusArtifactViewer } from "./preview/artifactViewerLayout.js";

// Renderers, for hosts that compose their own chrome (e.g. a file picker).
export { DirectoryPreview, renderers, rendererForKind } from "./preview/renderers/index.js";
export type { ArchiveView, DirectorySelection, PreviewRenderer, PreviewRendererProps } from "./preview/types.js";
export { DEFAULT_ARCHIVE_VIEW } from "./preview/types.js";
export { formatBytes } from "./preview/format.js";

// Markdown, the same renderer the viewer uses, for chat bodies and approvals.
export { MarkdownRenderer, CodeBlock, InlineCode, MermaidDiagram } from "./markdown/index.js";
export { useMermaidSvg } from "./markdown/hooks/useMermaidSvg.js";

// Engines the host supplies for content that needs an optional peer dependency.
export type { DiagramEngine, FileViewerLoaders, SpreadsheetEngine } from "./loaders.js";

// Copy, locale and host preferences.
export {
  FileViewerProvider,
  useFileViewerText,
  type FileViewerHost,
  type ReaderFontSizePreference,
} from "./i18n/context.js";
export { strings } from "./i18n/strings.js";
export type { MessageOverrides, StringKey } from "./i18n/catalog.js";
export { createTranslator, directionOf, KEYS_LOCALE, type Translate, type TranslateAdapter, type TranslateParams } from "./i18n/translate.js";

// Reading text size, shared with the host's own text-size controls.
export {
  clampReaderFontSize,
  READER_FONT_DEFAULT,
  READER_FONT_MAX,
  READER_FONT_MIN,
  useTypographyPinch,
} from "./lib/typography.js";
