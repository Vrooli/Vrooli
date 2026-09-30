export type SelectionEntry = { path: string; staged: boolean };

export interface PushNotice {
  tone: "success" | "info" | "warning" | "error";
  message: string;
  /** Verbatim git output, shown under the message. */
  detail?: string;
  /** Notices the operator must acknowledge (failures) do not auto-dismiss. */
  sticky?: boolean;
}

/** Which remote operation is running, and how far it has got. */
export interface SyncActivity {
  kind: "push" | "pull";
  /** "preflight" refreshes remote state; "transfer" is the git process itself. */
  phase: "preflight" | "transfer";
  startedAt: number;
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
