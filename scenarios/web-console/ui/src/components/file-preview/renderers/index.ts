import type { PreviewKind, PreviewRenderer } from "../types";
import { CodePreview, MarkdownPreview } from "./TextRenderers";
import { AudioPreview, ImagePreview, PdfPreview, VideoPreview } from "./MediaRenderers";
import { CsvPreview, DiffPreview } from "./DataRenderers";
import { ArchivePreview } from "./ArchivePreview";
import { DirectoryPreview } from "./DirectoryPreview";
import { SpreadsheetPreview } from "./SpreadsheetPreview";
import { DocumentPreview } from "./DocumentPreview";
import { PresentationPreview } from "./PresentationPreview";
import { DatabasePreview } from "./DatabasePreview";
import { FontPreview } from "./FontPreview";
import { ModelPreview } from "./ModelPreview";
import { EbookPreview } from "./EbookPreview";
import { UnsupportedPreview } from "./UnsupportedPreview";

// renderers maps every PreviewKind to its dedicated renderer. The viewer routes
// purely on model.kind, so adding a kind is: add the proto enum value, the
// PreviewKind union member, and one entry here. SVG and raster image share the
// blob-backed ImagePreview.
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
