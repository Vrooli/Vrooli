import { createClient } from "@connectrpc/connect";
import {
  DirectorySort as ProtoDirectorySort,
  EntryType as ProtoEntryType,
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
  | "directory"
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
  [ProtoPreviewKind.DIRECTORY]: "directory",
  [ProtoPreviewKind.UNSUPPORTED]: "unsupported",
};

function decodeKind(k: ProtoPreviewKind): PreviewKind {
  return PREVIEW_KIND_BY_PROTO[k] ?? "unsupported";
}

// decodeEntryKind is decodeKind for listing entries, which may legitimately
// carry no kind: listings classify by extension alone, so an unmapped
// extension stays undetermined until the entry is opened and sniffed.
function decodeEntryKind(k: ProtoPreviewKind): PreviewKind | null {
  return k === ProtoPreviewKind.UNSPECIFIED ? null : decodeKind(k);
}

// DirectorySort is the UI-side union mirroring the proto sort enum.
export type DirectorySort = "dirs_first_name" | "name" | "size_desc" | "mtime_desc";

const PROTO_BY_SORT: Record<DirectorySort, ProtoDirectorySort> = {
  dirs_first_name: ProtoDirectorySort.DIRS_FIRST_NAME,
  name: ProtoDirectorySort.NAME,
  size_desc: ProtoDirectorySort.SIZE_DESC,
  mtime_desc: ProtoDirectorySort.MTIME_DESC,
};

const SORT_BY_PROTO: Record<ProtoDirectorySort, DirectorySort> = {
  [ProtoDirectorySort.UNSPECIFIED]: "dirs_first_name",
  [ProtoDirectorySort.DIRS_FIRST_NAME]: "dirs_first_name",
  [ProtoDirectorySort.NAME]: "name",
  [ProtoDirectorySort.SIZE_DESC]: "size_desc",
  [ProtoDirectorySort.MTIME_DESC]: "mtime_desc",
};

export type DirectoryEntryType = "file" | "directory" | "symlink" | "other";

const ENTRY_TYPE_BY_PROTO: Record<ProtoEntryType, DirectoryEntryType> = {
  [ProtoEntryType.UNSPECIFIED]: "other",
  [ProtoEntryType.FILE]: "file",
  [ProtoEntryType.DIRECTORY]: "directory",
  [ProtoEntryType.SYMLINK]: "symlink",
  [ProtoEntryType.OTHER]: "other",
};

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
  /** True for directories, whose contents come from listDirectory. */
  listingAvailable: boolean;
  /** Same-origin relative blob path issued by the server (e.g. /api/v1/...). */
  blobUrl: string;
  /** Absolute href for native media elements (API_BASE + blobUrl). */
  blobHref: string;
  /** Epoch ms at which previewId (and every handle it issued) stops working. */
  expiresMs: number;
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

/** One child of a listed directory. */
export interface DirectoryEntry {
  name: string;
  entryType: DirectoryEntryType;
  /** null when the kind is only determined once the entry is opened. */
  kind: PreviewKind | null;
  sizeBytes: number;
  mtimeMs: number;
  canPreview: boolean;
  symlinkTarget: string;
  symlinkBroken: boolean;
  mode: string;
  /** null when the count was not determined (unreadable or too large). */
  childCount: number | null;
}

/** The accumulated state of one open directory, across any loaded pages. */
export interface PreviewListing {
  resolvedPath: string;
  /** "" at a filesystem root. */
  parentPath: string;
  entries: DirectoryEntry[];
  totalEntries: number;
  truncated: boolean;
  nextPageToken: string;
  /** The ordering actually applied, which may differ from the one requested. */
  effectiveSort: DirectorySort;
  sort: DirectorySort;
  showHidden: boolean;
  warnings: string[];
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
    listingAvailable: resp.listingAvailable,
    blobUrl: resp.blobUrl,
    blobHref: previewBlobHref(resp.blobUrl),
    expiresMs: Number(resp.expiresUnixNano / 1_000_000n),
    warnings: resp.warnings ?? [],
  };
}

// listDirectory fetches one page of a resolved directory. Pass a pageToken to
// continue; the caller appends to the entries it already holds.
export async function listDirectory(
  sessionId: string,
  previewId: string,
  options: { sort: DirectorySort; showHidden: boolean; pageToken?: string },
): Promise<PreviewListing> {
  const resp = await filePreviewClient.listDirectory({
    sessionId,
    previewId,
    sort: PROTO_BY_SORT[options.sort],
    showHidden: options.showHidden,
    pageToken: options.pageToken ?? "",
  });
  return {
    resolvedPath: resp.resolvedPath,
    parentPath: resp.parentPath,
    entries: (resp.entries ?? []).map((e) => ({
      name: e.name,
      entryType: ENTRY_TYPE_BY_PROTO[e.entryType] ?? "other",
      kind: decodeEntryKind(e.previewKind),
      sizeBytes: Number(e.sizeBytes),
      mtimeMs: Number(e.mtimeUnixNano / 1_000_000n),
      canPreview: e.canPreview,
      symlinkTarget: e.symlinkTarget,
      symlinkBroken: e.symlinkBroken,
      mode: e.mode,
      childCount: e.childCount < 0n ? null : Number(e.childCount),
    })),
    totalEntries: resp.totalEntries,
    truncated: resp.truncated,
    nextPageToken: resp.nextPageToken,
    effectiveSort: SORT_BY_PROTO[resp.effectiveSort] ?? "dirs_first_name",
    sort: options.sort,
    showHidden: options.showHidden,
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
