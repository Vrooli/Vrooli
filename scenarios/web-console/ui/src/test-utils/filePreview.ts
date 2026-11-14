import type { PreviewModel, PreviewRendererProps } from "../components/file-preview/types";

export function previewModel(overrides: Partial<PreviewModel> = {}): PreviewModel {
  return {
    previewId: "preview-1", inputPath: "/tmp/example", resolvedPath: "/tmp/example", basename: "example", resolutionBasis: "absolute",
    kind: "image", mimeType: "image/png", sizeBytes: 1024, canPreview: true, canDownload: true, supportsRange: true,
    textContentAvailable: false, listingAvailable: false, blobUrl: "/preview/blob", blobHref: "/preview/blob", expiresMs: 0, warnings: [],
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
