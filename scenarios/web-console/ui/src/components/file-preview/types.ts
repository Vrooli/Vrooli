import type { PreviewArchive, ArchiveView, ArchiveSource } from "../../api/filePreview";
import type {
  DirectoryEntry,
  DirectorySort,
  PreviewCandidate,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
} from "../../api/filePreview";

export type {
  DirectoryEntry,
  DirectorySort,
  PreviewCandidate,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
};

// An active "browse to locate a relative path" session. Set when a relative
// path matched nowhere: the viewer shows a folder browser rooted at a base
// directory, and foundPath is the resolved file once the user reaches a
// directory that contains the relative tail.
export interface PreviewSearch {
  /** The relative path the user is trying to open. */
  relPath: string;
  /** Resolved absolute path once the current directory contains it, else null. */
  foundPath: string | null;
}

// PreviewStatus is the viewer state machine. Every open() walks
// idle → resolving → (loadingText | loadingListing →) ready | unsupported |
// error, and any renderer load failure (e.g. a 409 stale blob) transitions to
// error.
export type PreviewStatus =
  | "idle"
  | "resolving"
  | "loadingText"
  | "loadingListing"
  | "ready"
  | "unsupported"
  // A relative path matched more than one base; the user picks which to open.
  | "choosing"
  | "error";

// PreviewFrame is one entry in the navigation history. Walking into a
// directory pushes the frame being left; Back pops it. Frames hold the fully
// loaded state, so stepping back is instant while the preview id is still
// live and falls back to a re-resolve once it has expired.
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

// PreviewRendererProps is the uniform contract every renderer receives. `text`
// is non-null only for text-kind previews, `listing` only for directories;
// blob kinds read model.blobHref.
export interface PreviewRendererProps {
  archive?: PreviewArchive | null;
  archiveView?: ArchiveView;
  onArchiveFolder?: (folder: string) => void;
  onArchiveEntry?: (entry: { id: string; name: string; }) => void;
  onArchiveViewChange?: (view: Partial<ArchiveView>) => void;
  model: PreviewModel;
  text: PreviewTextContent | null;
  listing: PreviewListing | null;
  /** Renderers call this when native media/blob loading fails. */
  onError: (message: string) => void;
  /** Resolve again to renew an expired or changed blob handle. */
  onReopen?: () => void;
  /** Open another path in this viewer, pushing the current one onto history. */
  onNavigate: (path: string) => void;
  /** Append the next page of a directory listing. */
  onLoadMore: () => void;
  /** Re-list the current directory with a new ordering or hidden filter. */
  onListOptionsChange: (options: { sort?: DirectorySort; showHidden?: boolean; }) => void;
  loadingMore: boolean;
  /** Retain the folder and its options while refreshing the current listing. */
  listingRefreshing?: boolean;
  /**
   * Session whose local files can be resolved — lets a renderer (e.g. the
   * markdown reader) turn relative inline image paths into previewable blobs.
   */
  imageSessionId?: string;
  /** Pick mode: rows carry a checkbox and file rows toggle instead of opening. */
  selection?: DirectorySelection;
}

// DirectorySelection turns the folder browser into a picker (attach from the
// host). Folders still open on tap — their checkbox selects them — while a file
// row's only action is to toggle, since opening it would leave the picker.
export interface DirectorySelection {
  isSelected: (path: string) => boolean;
  onToggle: (path: string, entry: DirectoryEntry) => void;
  /** Accessible name for one row's checkbox. */
  toggleLabel: (name: string) => string;
  /** Open a file to check it before picking; adds a Preview button to file rows. */
  onPreview?: (path: string) => void;
  /** Accessible name for one row's Preview button. */
  previewLabel?: (name: string) => string;
}

export type PreviewRenderer = (props: PreviewRendererProps) => JSX.Element;
