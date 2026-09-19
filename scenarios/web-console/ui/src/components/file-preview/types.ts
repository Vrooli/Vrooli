import type {
  DirectoryEntry,
  DirectorySort,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
} from "../../api/filePreview";

export type {
  DirectoryEntry,
  DirectorySort,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
};

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
  | "error";

// PreviewFrame is one entry in the navigation history. Walking into a
// directory pushes the frame being left; Back pops it. Frames hold the fully
// loaded state, so stepping back is instant while the preview id is still
// live and falls back to a re-resolve once it has expired.
export interface PreviewFrame {
  requestedPath: string;
  model: PreviewModel;
  text: PreviewTextContent | null;
  listing: PreviewListing | null;
}

export interface PreviewState {
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
  model: PreviewModel;
  text: PreviewTextContent | null;
  listing: PreviewListing | null;
  /** Renderers call this when native media/blob loading fails. */
  onError: (message: string) => void;
  /** Open another path in this viewer, pushing the current one onto history. */
  onNavigate: (path: string) => void;
  /** Append the next page of a directory listing. */
  onLoadMore: () => void;
  /** Re-list the current directory with a new ordering or hidden filter. */
  onListOptionsChange: (options: { sort?: DirectorySort; showHidden?: boolean }) => void;
  loadingMore: boolean;
}

export type PreviewRenderer = (props: PreviewRendererProps) => JSX.Element;
