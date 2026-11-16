import type { ComponentType, LazyExoticComponent } from "react";

import type {
  DirectoryEntry,
  DirectorySort,
  PreviewArchive,
  PreviewSource,
  PreviewCandidate,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
} from "../source/types.js";

export type {
  DirectoryEntry,
  DirectorySort,
  PreviewCandidate,
  PreviewKind,
  PreviewListing,
  PreviewModel,
  PreviewTextContent,
};

/** How an open archive is being browsed; part of the viewer's navigation history. */
export interface ArchiveView { folder: string; query: string; sort: "name" | "size"; showHidden: boolean; page: number }
export const DEFAULT_ARCHIVE_VIEW: ArchiveView = { folder: "", query: "", sort: "name", showHidden: false, page: 1 };

/**
 * The uniform contract every renderer receives. `text` is non-null only for
 * text-kind previews, `listing` only for directories; blob kinds read
 * `model.blobHref`.
 */
export interface PreviewRendererProps {
  archive?: PreviewArchive | null;
  archiveView?: ArchiveView;
  onArchiveFolder?: (folder: string) => void;
  onArchiveEntry?: (entry: { id: string; name: string }) => void;
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
  onListOptionsChange: (options: { sort?: DirectorySort; showHidden?: boolean }) => void;
  loadingMore: boolean;
  /** Retain the folder and its options while refreshing the current listing. */
  listingRefreshing?: boolean;
  /**
   * The source the viewer is showing. Lets a renderer ask it for thumbnails or
   * resolve relative image paths (the Markdown reader); absent when a renderer
   * is mounted outside a viewer.
   */
  source?: PreviewSource;
  /** A heading to scroll to (GitHub-style slug); honoured by the Markdown and Word readers. */
  anchor?: string;
  /** Pick mode: rows carry a checkbox and file rows toggle instead of opening. */
  selection?: DirectorySelection;
}

/**
 * Turns the folder browser into a picker (attach from the host). Folders still
 * open on tap, their checkbox selects them, while a file row's only action is
 * to toggle, since opening it would leave the picker.
 */
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

/** A renderer component; registry entries are lazy, so render inside Suspense. */
export type PreviewRenderer = ComponentType<PreviewRendererProps> | LazyExoticComponent<ComponentType<PreviewRendererProps>>;
