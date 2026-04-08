import { DiscardConfirmationModal, type DiscardFile } from "./components/DiscardConfirmationModal";
import { DeleteConfirmationModal } from "./components/DeleteConfirmationModal";
import { UpstreamInfoModal } from "./components/UpstreamInfoModal";
import { SettingsModal } from "./components/SettingsModal";
import { FileSearchModal } from "./components/FileSearchModal";
import { MobileFileSearch } from "./components/MobileFileSearch";
import type { LayoutPreset, LayoutSection } from "./components/LayoutSettingsModal";
import type { GroupingRule } from "./components/FileList";
import type { FileViewMode, SyncStatusResponse } from "./lib/api";
import type { PendingDeletePath } from "./App.types";

export interface AppModalsProps {
  // Settings
  isSettingsOpen: boolean;
  onCloseSettings: () => void;
  repoDir: string | undefined;
  repoId: string | null;
  syncStatus: SyncStatusResponse | undefined;
  layoutPreset: LayoutPreset;
  primaryPanel: LayoutSection;
  onChangePreset: (preset: LayoutPreset) => void;
  onChangePrimary: (panel: LayoutSection) => void;
  onResetLayout: () => void;
  fileViewMode: FileViewMode;
  onToggleGrouping: () => void;
  groupingRules: GroupingRule[];
  onChangeGroupingRules: (rules: GroupingRule[]) => void;

  // Upstream info
  isUpstreamInfoOpen: boolean;
  onCloseUpstreamInfo: () => void;
  localBranch: string | undefined;
  upstreamRef: string | undefined;
  upstreamAhead: number;
  upstreamBehind: number;

  // Discard confirmation
  pendingDiscardFiles: DiscardFile[] | null;
  isDiscardLoading: boolean;
  onConfirmDiscard: () => void;
  onCancelDiscard: () => void;

  // Delete confirmation
  pendingDeletePath: PendingDeletePath | null;
  isDeleteLoading: boolean;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;

  // File search (desktop)
  isFileSearchOpen: boolean;
  onCloseFileSearch: () => void;
  onSelectAnyFile: (path: string, lineNumber?: number) => void;

  /** When true, render MobileFileSearch instead of FileSearchModal */
  isMobile: boolean;
}

export function AppModals(props: AppModalsProps) {
  return (
    <>
      <SettingsModal
        isOpen={props.isSettingsOpen}
        repoDir={props.repoDir}
        repoId={props.repoId}
        syncStatus={props.syncStatus}
        preset={props.layoutPreset}
        primaryPanel={props.primaryPanel}
        onChangePreset={props.onChangePreset}
        onChangePrimary={props.onChangePrimary}
        onResetLayout={props.onResetLayout}
        groupingEnabled={props.fileViewMode === "grouped"}
        onToggleGrouping={props.onToggleGrouping}
        groupingRules={props.groupingRules}
        onChangeGroupingRules={props.onChangeGroupingRules}
        onClose={props.onCloseSettings}
      />
      <UpstreamInfoModal
        isOpen={props.isUpstreamInfoOpen}
        localBranch={props.localBranch}
        upstreamRef={props.upstreamRef}
        ahead={props.upstreamAhead}
        behind={props.upstreamBehind}
        repoId={props.repoId}
        onClose={props.onCloseUpstreamInfo}
      />
      <DiscardConfirmationModal
        isOpen={props.pendingDiscardFiles !== null}
        files={props.pendingDiscardFiles ?? []}
        isLoading={props.isDiscardLoading}
        onConfirm={props.onConfirmDiscard}
        onCancel={props.onCancelDiscard}
      />
      <DeleteConfirmationModal
        isOpen={props.pendingDeletePath !== null}
        path={props.pendingDeletePath?.path ?? ""}
        isDirectory={props.pendingDeletePath?.isDir ?? false}
        isLoading={props.isDeleteLoading}
        onConfirm={props.onConfirmDelete}
        onCancel={props.onCancelDelete}
      />
      {props.isMobile ? (
        <MobileFileSearch
          isOpen={props.isFileSearchOpen}
          onClose={props.onCloseFileSearch}
          onSelectFile={props.onSelectAnyFile}
          repoId={props.repoId}
        />
      ) : (
        <FileSearchModal
          isOpen={props.isFileSearchOpen}
          onClose={props.onCloseFileSearch}
          onSelectFile={props.onSelectAnyFile}
          repoId={props.repoId}
        />
      )}
    </>
  );
}
