import type { ArchiveSource, PreviewArchive, PreviewCandidate, PreviewListing, PreviewModel, PreviewTextContent } from "../source/types.js";
import type { ArchiveView } from "../preview/types.js";

/**
 * An active "browse to locate a relative path" session. Set when a relative
 * path matched nowhere: the viewer shows a folder browser rooted at a base
 * directory, and `foundPath` is the resolved file once the user reaches a
 * directory that contains the relative tail.
 */
export interface PreviewSearch {
  /** The relative path the user is trying to open. */
  relPath: string;
  /** Resolved absolute path once the current directory contains it, else null. */
  foundPath: string | null;
}

/**
 * The viewer's state machine. Every open walks
 * idle -> resolving -> (loadingText | loadingListing ->) ready | unsupported |
 * error, and any renderer load failure (e.g. a stale blob) transitions to error.
 */
export type PreviewStatus =
  | "idle"
  | "resolving"
  | "loadingText"
  | "loadingListing"
  | "ready"
  | "unsupported"
  /** A relative path matched more than one location; the user picks which to open. */
  | "choosing"
  | "error";

/**
 * One entry in the navigation history. Walking into a directory pushes the
 * frame being left; Back pops it. Frames hold the fully loaded state, so
 * stepping back is instant while the preview is still live and falls back to a
 * re-resolve once it has expired.
 */
export interface PreviewFrame {
  archive?: PreviewArchive | null;
  archiveView?: ArchiveView;
  archiveSource?: ArchiveSource;
  requestedPath: string;
  model: PreviewModel;
  text: PreviewTextContent | null;
  listing: PreviewListing | null;
}

export interface PreviewState {
  archive?: PreviewArchive | null;
  archiveView?: ArchiveView;
  archiveSource?: ArchiveSource;
  status: PreviewStatus;
  open: boolean;
  requestedPath: string | null;
  model: PreviewModel | null;
  text: PreviewTextContent | null;
  listing: PreviewListing | null;
  error: string | null;
  /** True while an additional page is being appended to `listing`. */
  loadingMore: boolean;
  /** Frames below the current one; empty when this is the first target. */
  stack: PreviewFrame[];
  /** Ambiguous relative-path matches, present only in the "choosing" status. */
  candidates?: PreviewCandidate[];
  /** Active folder-browse-to-locate session for an unresolved relative path. */
  search?: PreviewSearch | null;
  /**
   * A heading to scroll to in the open document (`plan-1`, GitHub-style), set
   * by `openPreview(path, context, { anchor })`. Cleared by navigating away.
   */
  anchor?: string;
}

export const IDLE_PREVIEW_STATE: PreviewState = {
  status: "idle",
  open: false,
  requestedPath: null,
  model: null,
  text: null,
  listing: null,
  error: null,
  loadingMore: false,
  stack: [],
};
