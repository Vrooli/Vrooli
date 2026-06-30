import type { PreviewKind, PreviewRenderer } from "../types";
import { CodePreview, MarkdownPreview } from "./TextRenderers";
import { AudioPreview, ImagePreview, PdfPreview, VideoPreview } from "./MediaRenderers";
import { CsvPreview, DiffPreview } from "./DataRenderers";
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
  diff: DiffPreview,
  unsupported: UnsupportedPreview,
};

export function rendererForKind(kind: PreviewKind): PreviewRenderer {
  return renderers[kind] ?? UnsupportedPreview;
}
