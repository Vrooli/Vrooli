/**
 * The preview contract between the viewer and whatever holds the files.
 *
 * The viewer never talks to a server, a session or a filesystem. It talks to a
 * {@link PreviewSource}: an object that can resolve a path to a
 * {@link PreviewModel}, hand over text, list a directory and (optionally) open
 * archives. Web Console implements it over its session-bound Connect service;
 * Notification Hub implements it over a manifest of attachment URLs.
 */

/** What the viewer renders a file as. Kind detection belongs to the source. */
export type PreviewKind =
  | "markdown"
  | "code"
  | "text"
  | "svg"
  | "image"
  | "pdf"
  | "audio"
  | "video"
  | "archive"
  | "document"
  | "presentation"
  | "ebook"
  | "model3d"
  | "font"
  | "database"
  | "spreadsheet"
  | "csv"
  | "diff"
  | "directory"
  | "unsupported";

/** Every {@link PreviewKind}, for sources that validate kinds from the wire. */
export const PREVIEW_KINDS: readonly PreviewKind[] = [
  "markdown", "code", "text", "svg", "image", "pdf", "audio", "video", "archive", "document",
  "presentation", "ebook", "model3d", "font", "database", "spreadsheet", "csv", "diff", "directory", "unsupported",
];

export type DirectorySort = "dirs_first_name" | "name" | "size_desc" | "mtime_desc";
export type DirectoryEntryType = "file" | "directory" | "symlink" | "other";

/** Why a path is being opened; sources may use it for telemetry or policy. */
export type PreviewSourceContext = "message_link" | "inline_code" | "cli";

/**
 * The normalized resolve result every renderer reads from.
 *
 * `previewId` is the source's handle for this target: the viewer passes the
 * whole model back to {@link PreviewSource.readText} and friends, so a source
 * may keep whatever it needs there.
 */
export interface PreviewModel {
  previewId: string;
  /** The path as the user asked for it. */
  inputPath: string;
  /** The canonical path the viewer shows, breadcrumbs and navigates by. */
  resolvedPath: string;
  basename: string;
  /** 1-based line to reveal in text kinds. */
  line?: number;
  kind: PreviewKind;
  mimeType: string;
  sizeBytes: number;
  /** False when no renderer can show the bytes (the viewer offers download). */
  canPreview: boolean;
  canDownload: boolean;
  /** True when {@link PreviewSource.readText} serves this target's content. */
  textContentAvailable: boolean;
  /** True for directories, whose contents come from {@link PreviewSource.listDirectory}. */
  listingAvailable: boolean;
  /** True for a member opened inside an archive (it has no host parent folder). */
  archiveMember?: boolean;
  /**
   * Absolute or same-origin URL the bytes can be fetched from with
   * `credentials: "same-origin"`. Native `<img>`, `<video>`, `<audio>`,
   * `<iframe>` elements and the worker-based readers load from it.
   */
  blobHref: string;
  /** Root URL for an HTML page's own sub-resources; omit for an inert page. */
  runtimeHref?: string;
  /** Epoch ms after which this model's URLs stop working; 0 means never. */
  expiresMs: number;
}

/** One place a relative path matched, or one folder worth browsing. */
export interface PreviewCandidate {
  resolvedPath: string;
  /** Source-defined label key such as `session_cwd`, `project_root`, `absolute`. */
  resolutionBasis: string;
  kind: PreviewKind;
  canPreview: boolean;
  basename: string;
}

/**
 * What resolving a path produced.
 *
 * - `resolved`: exactly one target.
 * - `ambiguous`: a relative path matched several locations; the user picks.
 * - `unresolved`: it matched nowhere; `searchBases` are folders to browse so
 *   the user can locate it. Sources without folders return an empty list and
 *   the viewer reports the file as unavailable.
 */
export type ResolveOutcome =
  | { outcome: "resolved"; model: PreviewModel }
  | { outcome: "ambiguous"; inputPath: string; candidates: PreviewCandidate[] }
  | { outcome: "unresolved"; inputPath: string; searchBases: PreviewCandidate[] };

export interface PreviewTextContent {
  resolvedPath: string;
  kind: PreviewKind;
  mimeType: string;
  content: string;
  /** True when the source cut the content at its inline limit. */
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
  /** Epoch ms; 0 when unknown. */
  mtimeMs: number;
  canPreview: boolean;
  symlinkTarget: string;
  symlinkBroken: boolean;
  mode: string;
  /** null when the count was not determined. */
  childCount: number | null;
}

/** The accumulated state of one open directory, across any loaded pages. */
export interface PreviewListing {
  resolvedPath: string;
  /** "" at a root. */
  parentPath: string;
  entries: DirectoryEntry[];
  totalEntries: number;
  truncated: boolean;
  /** Continuation token for the next page; "" when there is none. */
  nextPageToken: string;
  /** The ordering the source actually applied. */
  effectiveSort: DirectorySort;
  sort: DirectorySort;
  showHidden: boolean;
  warnings: string[];
}

export interface DirectoryListOptions {
  sort: DirectorySort;
  showHidden: boolean;
  /** Continue a previous page; omit for the first page. */
  pageToken?: string;
  signal?: AbortSignal;
}

export interface ArchiveEntry { id: string; name: string; sizeBytes: number; status: string }
export interface PreviewArchive { format: string; status: string; entries: ArchiveEntry[] }
/** Identifies one member of an archive previously resolved by the source. */
export interface ArchiveSource { previewId: string; entryId: string }
export interface ArchiveEntryResult { status: string; target: PreviewModel | null }

export interface ThumbnailOptions {
  /** For a directory model: the child to render. Omit for a file model. */
  entry?: string;
  /** Longest edge in device pixels. */
  size?: number;
}

export interface ResolveOptions {
  context?: PreviewSourceContext;
  signal?: AbortSignal;
}

/**
 * Affordances a source can offer. The viewer hides everything a source does
 * not declare, so a source never has to stub an operation it cannot serve.
 */
export interface PreviewCapabilities {
  /** {@link PreviewSource.thumbnailHref}: image thumbnails in directory rows. */
  thumbnails: boolean;
  /** {@link PreviewSource.listArchive} and `openArchiveEntry`: archives open as folders. */
  archives: boolean;
  /** `model.runtimeHref` is honoured: HTML pages load their own sub-resources. */
  htmlRuntime: boolean;
  /**
   * Relative paths can be ambiguous or unresolved, so the viewer offers the
   * candidate picker and the browse-to-locate banner. Without it both outcomes
   * become an ordinary "unavailable" error.
   */
  locate: boolean;
  /**
   * Files have a host parent folder the viewer can open ("open containing
   * folder") and directories are real, browsable locations.
   */
  parentFolders: boolean;
}

/** All capabilities off: a source that can only resolve, read and show bytes. */
export const NO_CAPABILITIES: PreviewCapabilities = Object.freeze({
  thumbnails: false, archives: false, htmlRuntime: false, locate: false, parentFolders: false,
});

export interface PreviewSource {
  /**
   * Identity of what this source can open (a session id, an ask id, …). The
   * viewer discards everything it holds when `scope` changes, so a previous
   * scope's preview can never flash in the next one. Equal scopes mean
   * interchangeable sources; a new object with the same scope is not a reset.
   */
  readonly scope: string;
  readonly capabilities: PreviewCapabilities;

  /** Resolve a path (absolute, relative, or an archive member path) to a target. */
  resolve(path: string, options?: ResolveOptions): Promise<ResolveOutcome>;

  /** The content of a target whose `textContentAvailable` is true. */
  readText(model: PreviewModel, signal?: AbortSignal): Promise<PreviewTextContent>;

  /** One page of a target whose `listingAvailable` is true. */
  listDirectory(model: PreviewModel, options: DirectoryListOptions): Promise<PreviewListing>;

  /** Required when `capabilities.archives` is true. */
  listArchive?(model: PreviewModel, signal?: AbortSignal): Promise<PreviewArchive>;
  /** Required when `capabilities.archives` is true. */
  openArchiveEntry?(source: ArchiveSource, signal?: AbortSignal): Promise<ArchiveEntryResult>;

  /** Required when `capabilities.thumbnails` is true; undefined when this file has none. */
  thumbnailHref?(model: PreviewModel, options?: ThumbnailOptions): string | undefined;
}

/** A source with the archive operations its capability promises. */
export type ArchiveCapableSource = PreviewSource & Required<Pick<PreviewSource, "listArchive" | "openArchiveEntry">>;

/** Narrowing guard: true only when the source declares archives and implements them. */
export function supportsArchives(source: PreviewSource | undefined | null): source is ArchiveCapableSource {
  return Boolean(source?.capabilities.archives && source.listArchive && source.openArchiveEntry);
}

/** True only when the source declares thumbnails and can build their URLs. */
export function supportsThumbnails(source: PreviewSource | undefined | null): boolean {
  return Boolean(source?.capabilities.thumbnails && source.thumbnailHref);
}
