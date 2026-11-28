import { lazy, type ComponentType, type LazyExoticComponent } from "react";

import type { PreviewKind, PreviewRenderer, PreviewRendererProps } from "../types.js";
import { UnsupportedPreview } from "./UnsupportedPreview.js";

/**
 * A renderer loaded on first use, so a host's initial bundle carries only the
 * viewer shell and each kind's code arrives when a file of that kind opens.
 */
function lazyRenderer<Module>(load: () => Promise<Module>, pick: (module: Module) => ComponentType<PreviewRendererProps>): LazyExoticComponent<ComponentType<PreviewRendererProps>> {
  return lazy(async () => ({ default: pick(await load()) }));
}

const CodePreview = lazyRenderer(() => import("./TextRenderers.js"), (m) => m.CodePreview);
const MarkdownPreview = lazyRenderer(() => import("./MarkdownPreview.js"), (m) => m.MarkdownPreview);
const ImagePreview = lazyRenderer(() => import("./ImagePreview.js"), (m) => m.ImagePreview);
const PdfPreview = lazyRenderer(() => import("./PdfPreview.js"), (m) => m.PdfPreview);
const AudioPreview = lazyRenderer(() => import("./AudioPreview.js"), (m) => m.AudioPreview);
const VideoPreview = lazyRenderer(() => import("./VideoPreview.js"), (m) => m.VideoPreview);
const CsvPreview = lazyRenderer(() => import("./CsvPreview.js"), (m) => m.CsvPreview);
const DiffPreview = lazyRenderer(() => import("./DiffPreview.js"), (m) => m.DiffPreview);
const ArchivePreview = lazyRenderer(() => import("./ArchivePreview.js"), (m) => m.ArchivePreview);
/** The folder browser on its own, for hosts that build a picker around it. */
export const DirectoryPreview = lazyRenderer(() => import("./DirectoryPreview.js"), (m) => m.DirectoryPreview);
const SpreadsheetPreview = lazyRenderer(() => import("./SpreadsheetPreview.js"), (m) => m.SpreadsheetPreview);
const DocumentPreview = lazyRenderer(() => import("./DocumentPreview.js"), (m) => m.DocumentPreview);
const PresentationPreview = lazyRenderer(() => import("./PresentationPreview.js"), (m) => m.PresentationPreview);
const DatabasePreview = lazyRenderer(() => import("./DatabasePreview.js"), (m) => m.DatabasePreview);
const FontPreview = lazyRenderer(() => import("./FontPreview.js"), (m) => m.FontPreview);
const ModelPreview = lazyRenderer(() => import("./ModelPreview.js"), (m) => m.ModelPreview);
const EbookPreview = lazyRenderer(() => import("./EbookPreview.js"), (m) => m.EbookPreview);

/**
 * Maps every PreviewKind to its dedicated renderer. The viewer routes purely on
 * `model.kind`, so adding a kind is: add the PreviewKind union member and one
 * entry here. SVG and raster image share the blob-backed ImagePreview.
 *
 * Every entry except the fallback is lazy; render inside a Suspense boundary
 * (the FileViewer shell provides one).
 */
export const renderers: Record<PreviewKind, PreviewRenderer> = {
  markdown: MarkdownPreview,
  code: CodePreview,
  text: CodePreview,
  svg: ImagePreview,
  image: ImagePreview,
  pdf: PdfPreview,
  audio: AudioPreview,
  video: VideoPreview,
  csv: CsvPreview,
  spreadsheet: SpreadsheetPreview,
  document: DocumentPreview,
  presentation: PresentationPreview,
  ebook: EbookPreview,
  model3d: ModelPreview,
  font: FontPreview,
  database: DatabasePreview,
  diff: DiffPreview,
  directory: DirectoryPreview,
  archive: ArchivePreview,
  unsupported: UnsupportedPreview,
};

export function rendererForKind(kind: PreviewKind): PreviewRenderer {
  return Object.prototype.hasOwnProperty.call(renderers, kind) ? renderers[kind] : UnsupportedPreview;
}
