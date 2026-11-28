import type { PreviewModel, PreviewRendererProps } from "../preview/types.js";

export function previewModel(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "preview-1", inputPath: "/tmp/example", resolvedPath: "/tmp/example", basename: "example",
    kind: "image", mimeType: "image/png", sizeBytes: 1024, canPreview: true, canDownload: true,
    textContentAvailable: false, listingAvailable: false, blobHref: "/preview/blob", expiresMs: 0,
    ...overrides,
  };
}

export function previewProps(overrides: Partial<PreviewRendererProps> = {}): PreviewRendererProps {
  return {
    model: previewModel(), text: null, listing: null, loadingMore: false,
    onError: () => {}, onNavigate: () => {}, onLoadMore: () => {}, onListOptionsChange: () => {},
    ...overrides,
  };
}
