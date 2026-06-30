import { createClient } from "@connectrpc/connect";
import {
  FilePreviewService,
  PreviewKind as ProtoPreviewKind,
  SourceContext as ProtoSourceContext,
} from "@vrooli/proto-types/web-console/v1/file_preview/file_preview_pb";

import { transport, API_BASE } from "./client";

// filePreviewClient is the Connect-Web client for FilePreviewService.
// Consumers should prefer the typed wrappers below, which decode the proto
// enum + bigint shapes into the PreviewModel the viewer/renderers consume.
export const filePreviewClient = createClient(FilePreviewService, transport);

// PreviewKind is the UI-side string union mirroring the proto PreviewKind enum.
// The renderer registry is keyed on these values.
export type PreviewKind =
  | "markdown"
  | "code"
  | "text"
  | "svg"
  | "image"
  | "pdf"
  | "audio"
  | "video"
  | "csv"
  | "diff"
  | "unsupported";

const PREVIEW_KIND_BY_PROTO: Record<ProtoPreviewKind, PreviewKind> = {
  [ProtoPreviewKind.UNSPECIFIED]: "unsupported",
  [ProtoPreviewKind.MARKDOWN]: "markdown",
  [ProtoPreviewKind.CODE]: "code",
  [ProtoPreviewKind.TEXT]: "text",
  [ProtoPreviewKind.SVG]: "svg",
  [ProtoPreviewKind.IMAGE]: "image",
  [ProtoPreviewKind.PDF]: "pdf",
  [ProtoPreviewKind.AUDIO]: "audio",
  [ProtoPreviewKind.VIDEO]: "video",
  [ProtoPreviewKind.CSV]: "csv",
  [ProtoPreviewKind.DIFF]: "diff",
  [ProtoPreviewKind.UNSUPPORTED]: "unsupported",
};

function decodeKind(k: ProtoPreviewKind): PreviewKind {
  return PREVIEW_KIND_BY_PROTO[k] ?? "unsupported";
}

export type PreviewSourceContext = "message_link" | "inline_code" | "cli";

function encodeSourceContext(src?: PreviewSourceContext): ProtoSourceContext {
  switch (src) {
    case "message_link":
      return ProtoSourceContext.MESSAGE_LINK;
    case "inline_code":
      return ProtoSourceContext.INLINE_CODE;
    case "cli":
      return ProtoSourceContext.CLI;
    default:
      return ProtoSourceContext.UNSPECIFIED;
  }
}

// PreviewModel is the normalized resolve result the viewer + renderers consume.
// It is the single shape every renderer reads from — never the raw proto.
export interface PreviewModel {
  previewId: string;
  inputPath: string;
  resolvedPath: string;
  basename: string;
  line?: number;
  resolutionBasis: string;
  kind: PreviewKind;
  mimeType: string;
  sizeBytes: number;
  canPreview: boolean;
  canDownload: boolean;
  supportsRange: boolean;
  textContentAvailable: boolean;
  /** Same-origin relative blob path issued by the server (e.g. /api/v1/...). */
  blobUrl: string;
  /** Absolute href for native media elements (API_BASE + blobUrl). */
  blobHref: string;
  warnings: string[];
}

export interface PreviewTextContent {
  resolvedPath: string;
  kind: PreviewKind;
  mimeType: string;
  content: string;
  truncated: boolean;
  line?: number;
}

// previewBlobHref joins the API base with a same-origin relative blob path so
// native <img>/<video>/<audio>/<iframe> elements fetch from the right origin
// even when the UI and API run on separate ports (desktop bundles).
export function previewBlobHref(blobUrl: string): string {
  if (!blobUrl) return "";
  if (/^https?:\/\//i.test(blobUrl)) return blobUrl;
  const base = (API_BASE || "").replace(/\/+$/, "");
  const path = blobUrl.startsWith("/") ? blobUrl : `/${blobUrl}`;
  return `${base}${path}`;
}

export async function resolveFilePreview(
  sessionId: string,
  path: string,
  sourceContext?: PreviewSourceContext,
): Promise<PreviewModel> {
  const resp = await filePreviewClient.resolve({
    sessionId,
    path,
    sourceContext: encodeSourceContext(sourceContext),
  });
  return {
    previewId: resp.previewId,
    inputPath: resp.inputPath,
    resolvedPath: resp.resolvedPath,
    basename: resp.basename,
    line: resp.hasLine ? resp.line : undefined,
    resolutionBasis: resp.resolutionBasis,
    kind: decodeKind(resp.previewKind),
    mimeType: resp.mimeType,
    sizeBytes: Number(resp.sizeBytes),
    canPreview: resp.canPreview,
    canDownload: resp.canDownload,
    supportsRange: resp.supportsRange,
    textContentAvailable: resp.textContentAvailable,
    blobUrl: resp.blobUrl,
    blobHref: previewBlobHref(resp.blobUrl),
    warnings: resp.warnings ?? [],
  };
}

export async function getFilePreviewText(
  sessionId: string,
  previewId: string,
): Promise<PreviewTextContent> {
  const resp = await filePreviewClient.getTextContent({ sessionId, previewId });
  return {
    resolvedPath: resp.resolvedPath,
    kind: decodeKind(resp.previewKind),
    mimeType: resp.mimeType,
    content: resp.content,
    truncated: resp.truncated,
    line: resp.hasLine ? resp.line : undefined,
  };
}
