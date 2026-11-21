import type {
  DirectoryEntry,
  DirectoryListOptions,
  DirectorySort,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewSource,
  PreviewTextContent,
  ResolveOutcome,
} from "./types.js";
import { NO_CAPABILITIES } from "./types.js";

/** One file a manifest source can open. */
export interface ManifestFile {
  /** Where the file sits in the manifest's virtual tree, e.g. `plan.md` or `screens/home.png`. */
  path: string;
  /** Where its bytes come from; same-origin, or CORS-readable if text is wanted. */
  url: string;
  kind: PreviewKind;
  mimeType: string;
  sizeBytes: number;
  /** Epoch ms; shown in folder listings when known. */
  mtimeMs?: number;
  /** Default: true unless `kind` is `unsupported`. */
  canPreview?: boolean;
  /** Default: true. */
  canDownload?: boolean;
  /** Small rendition for folder rows; enables thumbnails when any file has one. */
  thumbnailUrl?: string;
}

export interface ManifestSourceOptions {
  /** Identity of the manifest (an ask id, a bundle id …); a new scope resets the viewer. */
  scope: string;
  files: readonly ManifestFile[];
  /** Used to fetch text kinds. Default: the global `fetch`. */
  fetch?: typeof fetch;
  /** Text longer than this many characters is cut and reported as truncated. Default 1 MiB. */
  maxTextChars?: number;
}

const DEFAULT_MAX_TEXT_CHARS = 1 << 20;
const TEXT_KINDS: ReadonlySet<PreviewKind> = new Set<PreviewKind>(["markdown", "code", "text", "csv", "diff"]);

/**
 * Collapse `.`/`..`/empty segments and force one leading slash, so every
 * manifest path has a single canonical spelling: `"a/./b/../c"` → `"/a/c"`.
 */
export function normalizeManifestPath(path: string): string {
  const parts: string[] = [];
  for (const part of path.replaceAll("\\", "/").split("/")) {
    if (part === "" || part === ".") continue;
    if (part === "..") parts.pop();
    else parts.push(part);
  }
  return `/${parts.join("/")}`;
}

function baseName(path: string): string {
  const index = path.lastIndexOf("/");
  return index < 0 ? path : path.slice(index + 1);
}

function dirName(path: string): string {
  const index = path.lastIndexOf("/");
  return index <= 0 ? "/" : path.slice(0, index);
}

function compareEntries(sort: DirectorySort): (a: DirectoryEntry, b: DirectoryEntry) => number {
  const byName = (a: DirectoryEntry, b: DirectoryEntry) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" });
  switch (sort) {
    case "name": return byName;
    case "size_desc": return (a, b) => b.sizeBytes - a.sizeBytes || byName(a, b);
    case "mtime_desc": return (a, b) => b.mtimeMs - a.mtimeMs || byName(a, b);
    default: return (a, b) => Number(b.entryType === "directory") - Number(a.entryType === "directory") || byName(a, b);
  }
}

/**
 * A {@link PreviewSource} over files that are known up front.
 *
 * The manifest is a virtual read-only tree: every file carries its kind, size
 * and a URL, folders are implied by the paths, and bytes are fetched from the
 * URLs only when a renderer needs them. Nothing here touches a filesystem or a
 * session, which makes it the right source for attachments, bundles and tests.
 *
 * It declares no capabilities except thumbnails (when any file supplies one):
 * paths either match exactly or not at all, and there are no host parent
 * folders or archives to open.
 */
export function createManifestPreviewSource(options: ManifestSourceOptions): PreviewSource {
  const fetchImpl = options.fetch ?? ((input, init) => fetch(input, init));
  const maxTextChars = options.maxTextChars ?? DEFAULT_MAX_TEXT_CHARS;

  const files = new Map<string, ManifestFile>();
  const folders = new Map<string, Set<string>>([["/", new Set()]]);
  for (const file of options.files) {
    const path = normalizeManifestPath(file.path);
    if (path === "/" || files.has(path)) continue;
    files.set(path, file);
    for (let child = path; child !== "/"; ) {
      const parent = dirName(child);
      const siblings = folders.get(parent) ?? new Set<string>();
      siblings.add(child);
      folders.set(parent, siblings);
      child = parent;
    }
  }

  const fileModel = (inputPath: string, path: string, file: ManifestFile): PreviewModel => {
    const canPreview = file.canPreview ?? file.kind !== "unsupported";
    return {
      previewId: path,
      inputPath,
      resolvedPath: path,
      basename: baseName(path),
      kind: file.kind,
      mimeType: file.mimeType,
      sizeBytes: file.sizeBytes,
      canPreview,
      canDownload: file.canDownload ?? true,
      textContentAvailable: TEXT_KINDS.has(file.kind),
      listingAvailable: false,
      blobHref: file.url,
      expiresMs: 0,
    };
  };

  const folderModel = (inputPath: string, path: string): PreviewModel => ({
    previewId: path,
    inputPath,
    resolvedPath: path,
    basename: path === "/" ? "/" : baseName(path),
    kind: "directory",
    mimeType: "inode/directory",
    sizeBytes: 0,
    canPreview: true,
    canDownload: false,
    textContentAvailable: false,
    listingAvailable: true,
    blobHref: "",
    expiresMs: 0,
  });

  const hasThumbnails = options.files.some((file) => Boolean(file.thumbnailUrl));

  return {
    scope: options.scope,
    capabilities: { ...NO_CAPABILITIES, thumbnails: hasThumbnails },

    resolve(path): Promise<ResolveOutcome> {
      const normalized = normalizeManifestPath(path);
      const file = files.get(normalized);
      if (file) return Promise.resolve({ outcome: "resolved", model: fileModel(path, normalized, file) });
      if (folders.has(normalized)) return Promise.resolve({ outcome: "resolved", model: folderModel(path, normalized) });
      return Promise.resolve({ outcome: "unresolved", inputPath: path, searchBases: [] });
    },

    async readText(model, signal): Promise<PreviewTextContent> {
      const file = files.get(model.previewId);
      if (!file) throw new Error(`Not in this attachment set: ${model.resolvedPath}`);
      const response = await fetchImpl(file.url, { credentials: "same-origin", signal });
      if (!response.ok) throw new Error(`Could not read ${model.basename} (${String(response.status)})`);
      const text = await response.text();
      const truncated = text.length > maxTextChars;
      return {
        resolvedPath: model.resolvedPath,
        kind: model.kind,
        mimeType: model.mimeType,
        content: truncated ? text.slice(0, maxTextChars) : text,
        truncated,
        line: model.line,
      };
    },

    listDirectory(model, listOptions: DirectoryListOptions): Promise<PreviewListing> {
      const children = [...(folders.get(model.previewId) ?? [])];
      const entries: DirectoryEntry[] = children.flatMap((child): DirectoryEntry[] => {
        const name = baseName(child);
        if (!listOptions.showHidden && name.startsWith(".")) return [];
        const file = files.get(child);
        if (file) {
          return [{
            name, entryType: "file", kind: file.kind, sizeBytes: file.sizeBytes, mtimeMs: file.mtimeMs ?? 0,
            canPreview: file.canPreview ?? file.kind !== "unsupported", symlinkTarget: "", symlinkBroken: false, mode: "", childCount: null,
          }];
        }
        return [{
          name, entryType: "directory", kind: "directory", sizeBytes: 0, mtimeMs: 0, canPreview: true,
          symlinkTarget: "", symlinkBroken: false, mode: "", childCount: folders.get(child)?.size ?? 0,
        }];
      });
      entries.sort(compareEntries(listOptions.sort));
      return Promise.resolve({
        resolvedPath: model.resolvedPath,
        parentPath: model.resolvedPath === "/" ? "" : dirName(model.resolvedPath),
        entries,
        totalEntries: entries.length,
        truncated: false,
        nextPageToken: "",
        effectiveSort: listOptions.sort,
        sort: listOptions.sort,
        showHidden: listOptions.showHidden,
        warnings: [],
      });
    },

    thumbnailHref(model, thumbnailOptions) {
      const path = thumbnailOptions?.entry ? normalizeManifestPath(`${model.resolvedPath}/${thumbnailOptions.entry}`) : model.resolvedPath;
      return files.get(path)?.thumbnailUrl;
    },
  };
}
