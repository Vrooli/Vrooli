export type SelectionEntry = { path: string; staged: boolean };

export interface PushNotice {
  tone: "success" | "info" | "warning";
  message: string;
}

export interface WarningNotice {
  message: string;
  details?: string;
}

export interface PendingDeletePath {
  path: string;
  isDir: boolean;
}

export interface ViewingFileBlame {
  path: string;
  filename: string;
}

/** Re-export types used by sub-modules for convenience */
export type { LayoutPreset, LayoutSection } from "./components/LayoutSettingsModal";
export type { DiscardFile } from "./components/DiscardConfirmationModal";
export type { GroupingRule } from "./components/FileList";
export type { ViewingCommit } from "./components/HistoryModeHeader";
export type { ReviewTab } from "./hooks";
export type { RepoHistoryEntry, ViewMode, FileViewMode, GroupingRulesConfig } from "./lib/api";
