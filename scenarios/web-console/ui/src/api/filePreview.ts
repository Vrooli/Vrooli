import { createClient } from "@connectrpc/connect";
import {
  DirectorySort as ProtoDirectorySort,
  EntryType as ProtoEntryType,
  FilePreviewService,
  PreviewKind as ProtoPreviewKind,
  SourceContext as ProtoSourceContext,
  type ResolveResponse,
} from "@vrooli/proto-types/web-console/v1/file_preview/file_preview_pb";
import type {
  DirectoryEntryType,
  DirectorySort,
  PreviewCandidate,
  PreviewKind,
  PreviewModel,
  PreviewSource,
  PreviewSourceContext,
  ResolveOutcome,
} from "@vrooli/file-viewer";

import { transport, API_BASE } from "./client";

// filePreviewClient is the Connect-Web client for FilePreviewService. Everything
// else in this module decodes its proto shapes into the PreviewSource contract
// the shared file viewer (@vrooli/file-viewer) consumes.
export const filePreviewClient = createClient(FilePreviewService, transport);

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
  [ProtoPreviewKind.DOCUMENT]: "document",
  [ProtoPreviewKind.PRESENTATION]: "presentation",
  [ProtoPreviewKind.EBOOK]: "ebook",
  [ProtoPreviewKind.MODEL_3D]: "model3d",
  [ProtoPreviewKind.FONT]: "font",
  [ProtoPreviewKind.DATABASE]: "database",
  [ProtoPreviewKind.ARCHIVE]: "archive",
  [ProtoPreviewKind.SPREADSHEET]: "spreadsheet",
  [ProtoPreviewKind.DIFF]: "diff",
  [ProtoPreviewKind.DIRECTORY]: "directory",
  [ProtoPreviewKind.UNSUPPORTED]: "unsupported",
};

function decodeKind(k: ProtoPreviewKind): PreviewKind {
  return Object.prototype.hasOwnProperty.call(PREVIEW_KIND_BY_PROTO, k) ? PREVIEW_KIND_BY_PROTO[k] : "unsupported";
}

// decodeEntryKind is decodeKind for listing entries, which may legitimately
// carry no kind: listings classify by extension alone, so an unmapped
// extension stays undetermined until the entry is opened and sniffed.
function decodeEntryKind(k: ProtoPreviewKind): PreviewKind | null {
  return k === ProtoPreviewKind.UNSPECIFIED ? null : decodeKind(k);
}

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

const ENTRY_TYPE_BY_PROTO: Record<ProtoEntryType, DirectoryEntryType> = {
  [ProtoEntryType.UNSPECIFIED]: "other",
  [ProtoEntryType.FILE]: "file",
  [ProtoEntryType.DIRECTORY]: "directory",
  [ProtoEntryType.SYMLINK]: "symlink",
  [ProtoEntryType.OTHER]: "other",
};

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

// Extensions the host renders thumbnails for (api: filepreview.Thumbnail).
// Other images (WebP, HEIC, …) would only answer 415, so rows never ask.
const THUMBNAIL_EXTENSIONS = new Set(["png", "jpg", "jpeg", "gif"]);

function canThumbnail(name: string): boolean {
  const ext = name.slice(name.lastIndexOf(".") + 1).toLowerCase();
  return name.includes(".") && THUMBNAIL_EXTENSIONS.has(ext);
}

function previewRuntimeHref(blobUrl: string): string {
  if (!blobUrl) return "";
  return previewBlobHref(blobUrl.replace(/\/blob$/, "/runtime/"));
}

function decodeCandidate(c: ResolveResponse["candidates"][number]): PreviewCandidate {
  return {
    resolvedPath: c.resolvedPath,
    resolutionBasis: c.resolutionBasis,
    kind: decodeKind(c.previewKind),
    canPreview: c.canPreview,
    basename: c.basename,
  };
}

function decodeTarget(resp: ResolveResponse): PreviewModel {
  return {
    archiveMember: resp.archiveMember,
    previewId: resp.previewId,
    inputPath: resp.inputPath,
    resolvedPath: resp.resolvedPath,
    basename: resp.basename,
    line: resp.hasLine ? resp.line : undefined,
    kind: decodeKind(resp.previewKind),
    mimeType: resp.mimeType,
    sizeBytes: Number(resp.sizeBytes),
    canPreview: resp.canPreview,
    canDownload: resp.canDownload,
    textContentAvailable: resp.textContentAvailable,
    listingAvailable: resp.listingAvailable,
    blobHref: previewBlobHref(resp.blobUrl),
    runtimeHref: previewRuntimeHref(resp.blobUrl),
    expiresMs: Number(resp.expiresUnixNano / 1_000_000n),
  };
}

// decodeResolve maps one proto resolve response to the viewer's outcome: an
// unresolved relative path (with folders to browse), an ambiguous one (several
// matching bases), or the single target.
function decodeResolve(resp: ResolveResponse): ResolveOutcome {
  if (resp.unresolved) return { outcome: "unresolved", inputPath: resp.inputPath, searchBases: resp.searchBases.map(decodeCandidate) };
  if (resp.candidates.length > 1) return { outcome: "ambiguous", inputPath: resp.inputPath, candidates: resp.candidates.map(decodeCandidate) };
  return { outcome: "resolved", model: decodeTarget(resp) };
}

/**
 * The file viewer's source for one terminal session: paths resolve against the
 * session's machine, and every handle the host issues is bound to the session.
 * Uncached; prefer sessionPreviewSource so equal sessions share one object.
 */
export function createSessionPreviewSource(sessionId: string): PreviewSource {
  return {
    scope: sessionId,
    capabilities: { thumbnails: true, archives: true, htmlRuntime: true, locate: true, parentFolders: true },

    async resolve(path, options) {
      const resp = await filePreviewClient.resolve(
        { sessionId, path, sourceContext: encodeSourceContext(options?.context) },
        { signal: options?.signal },
      );
      return decodeResolve(resp);
    },

    async readText(model, signal) {
      const resp = await filePreviewClient.getTextContent({ sessionId, previewId: model.previewId }, { signal });
      return {
        resolvedPath: resp.resolvedPath,
        kind: decodeKind(resp.previewKind),
        mimeType: resp.mimeType,
        content: resp.content,
        truncated: resp.truncated,
        line: resp.hasLine ? resp.line : undefined,
      };
    },

    async listDirectory(model, options) {
      const resp = await filePreviewClient.listDirectory({
        sessionId,
        previewId: model.previewId,
        sort: PROTO_BY_SORT[options.sort],
        showHidden: options.showHidden,
        pageToken: options.pageToken ?? "",
      }, { signal: options.signal });
      return {
        resolvedPath: resp.resolvedPath,
        parentPath: resp.parentPath,
        entries: (Array.isArray(resp.entries) ? resp.entries : []).map((e) => ({
          name: e.name,
          entryType: Object.prototype.hasOwnProperty.call(ENTRY_TYPE_BY_PROTO, e.entryType) ? ENTRY_TYPE_BY_PROTO[e.entryType] : "other",
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
        effectiveSort: Object.prototype.hasOwnProperty.call(SORT_BY_PROTO, resp.effectiveSort) ? SORT_BY_PROTO[resp.effectiveSort] : "dirs_first_name",
        sort: options.sort,
        showHidden: options.showHidden,
        warnings: Array.isArray(resp.warnings) ? resp.warnings : [],
      };
    },

    async listArchive(model, signal) {
      const resp = await filePreviewClient.listArchive({ sessionId, previewId: model.previewId }, { signal });
      return { format: resp.format, status: resp.status, entries: resp.entries.map((e) => ({ id: e.entryId, name: e.name, sizeBytes: Number(e.sizeBytes), status: e.status })) };
    },

    async openArchiveEntry(entry, signal) {
      const resp = await filePreviewClient.openArchiveEntry({ sessionId, previewId: entry.previewId, entryId: entry.entryId }, { signal });
      return { status: resp.status, target: resp.target ? decodeTarget(resp.target) : null };
    },

    // For a directory preview id, `entry` names one child; for a file preview
    // id it is omitted. `size` is the longest edge in device pixels.
    thumbnailHref(model, options = {}) {
      if (!canThumbnail(options.entry ?? model.basename)) return undefined;
      const query = new URLSearchParams();
      if (options.entry) query.set("entry", options.entry);
      if (options.size) query.set("size", String(Math.round(options.size)));
      const qs = query.toString();
      return previewBlobHref(`/api/v1/sessions/${encodeURIComponent(sessionId)}/file-previews/${encodeURIComponent(model.previewId)}/thumbnail${qs ? `?${qs}` : ""}`);
    },
  };
}

// Sources are cheap but components pass them as props and effect dependencies,
// and the Markdown renderer compares them by identity: equal sessions must get
// the same object. A bounded cache keeps long-lived tabs from accumulating one
// per session ever seen.
const MAX_CACHED_SOURCES = 32;
const sources = new Map<string, PreviewSource>();

export function sessionPreviewSource(sessionId: string): PreviewSource {
  const cached = sources.get(sessionId);
  if (cached) return cached;
  const created = createSessionPreviewSource(sessionId);
  if (sources.size >= MAX_CACHED_SOURCES) {
    const oldest = sources.keys().next();
    if (!oldest.done) sources.delete(oldest.value);
  }
  sources.set(sessionId, created);
  return created;
}

/**
 * The absolute path a session-relative path (usually ".") names on the
 * session's machine, or "" when it names nothing. When several bases match,
 * the session's working directory wins: it is the one an operator means by ".".
 */
export async function resolveSessionPath(sessionId: string, path: string): Promise<string> {
  const result = await sessionPreviewSource(sessionId).resolve(path);
  if (result.outcome === "resolved") return result.model.resolvedPath;
  if (result.outcome === "ambiguous") return (result.candidates.find((c) => c.resolutionBasis === "session_cwd") ?? result.candidates[0])?.resolvedPath ?? "";
  return "";
}
