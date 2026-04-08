import {
  File,
  FilePlus,
  FileX,
  AlertTriangle,
  Plus,
  Minus,
} from "lucide-react";
import type { RepoFilesStatus, RepoFileStats } from "../lib/api";
import { FileSection } from "./FileSection";
import { type FileCategory, type SelectedFileEntry, type GroupingRule, summarizeFileStats } from "./FileListTypes";

export interface FileListFlatViewProps {
  files?: RepoFilesStatus;
  fileStats?: RepoFileStats;
  binarySet: Set<string>;
  approvedPaths?: Set<string>;
  maxPathChars: number;
  selectedFiles?: SelectedFileEntry[];
  selectedKeySet?: Set<string>;
  selectionKey: (entry: SelectedFileEntry) => string;
  onSelectFile: (path: string, staged: boolean, event: React.MouseEvent<HTMLLIElement>) => void;
  onStageFile: (path: string) => void;
  onUnstageFile: (path: string) => void;
  isStaging: boolean;
  isDiscarding: boolean;
  isIgnoring: boolean;
  confirmingDiscard: string | null;
  onConfirmDiscard: (path: string | null) => void;
  confirmingIgnore: string | null;
  onConfirmIgnore: (path: string | null) => void;
  groupingRules: GroupingRule[];
  handleDiscardUnstaged: (path: string) => void;
  handleDiscardUntracked: (path: string) => void;
  handleIgnoreFile: (path: string, level?: "project" | "group", groupDir?: string) => void;
  handleOpenMobileActions: (file: string) => void;
  handleFileContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode: boolean;
  handleLongPress: (file: string, staged: boolean) => void;
  handleMobileTap: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  openAggregateMetrics: () => void;
  openFileMetrics: (path: string, category?: FileCategory) => void;
}

export function FileListFlatView({
  files,
  fileStats,
  binarySet,
  approvedPaths,
  maxPathChars,
  selectedFiles,
  selectedKeySet,
  selectionKey,
  onSelectFile,
  onStageFile,
  onUnstageFile,
  isStaging,
  isDiscarding,
  isIgnoring,
  confirmingDiscard,
  onConfirmDiscard,
  confirmingIgnore,
  onConfirmIgnore,
  groupingRules,
  handleDiscardUnstaged,
  handleDiscardUntracked,
  handleIgnoreFile,
  handleOpenMobileActions,
  handleFileContextMenu,
  mobileSelectionMode,
  handleLongPress,
  handleMobileTap,
  openAggregateMetrics,
  openFileMetrics,
}: FileListFlatViewProps) {
  return (
    <>
      {/* Conflicts - Always show first if any */}
      <FileSection
        title="Conflicts"
        category="conflicts"
        files={files?.conflicts ?? []}
        fileStatuses={files?.statuses}
        binaryFiles={binarySet}
        approvedFiles={approvedPaths}
        maxPathChars={maxPathChars}
        icon={<AlertTriangle className="h-3.5 w-3.5 text-red-500" />}
        selectedFiles={selectedFiles}
        selectedKeySet={selectedKeySet}
        selectionKey={selectionKey}
        onSelectFile={onSelectFile}
        onAction={onStageFile}
        actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
        actionLabel="Stage file"
        isLoading={isStaging}
        changeStats={summarizeFileStats(files?.conflicts ?? [], fileStats?.unstaged)}
        onIgnore={handleIgnoreFile}
        isIgnoring={isIgnoring}
        confirmingIgnore={confirmingIgnore}
        onConfirmIgnore={onConfirmIgnore}
        groupingRules={groupingRules}
        onOpenMobileActions={handleOpenMobileActions}
        onContextMenu={handleFileContextMenu}
        mobileSelectionMode={mobileSelectionMode}
        onLongPress={handleLongPress}
        onMobileTap={handleMobileTap}
        onStatsClick={openAggregateMetrics}
        onViewMetrics={openFileMetrics}
      />

      {/* Staged Changes */}
      <FileSection
        title="Staged"
        category="staged"
        files={files?.staged ?? []}
        fileStatuses={files?.statuses}
        binaryFiles={binarySet}
        approvedFiles={approvedPaths}
        maxPathChars={maxPathChars}
        icon={<FilePlus className="h-3.5 w-3.5 text-emerald-500" />}
        selectedFiles={selectedFiles}
        selectedKeySet={selectedKeySet}
        selectionKey={selectionKey}
        onSelectFile={onSelectFile}
        onAction={onUnstageFile}
        actionIcon={<Minus className="h-3 w-3 text-slate-400" />}
        actionLabel="Unstage file"
        isLoading={isStaging}
        changeStats={summarizeFileStats(files?.staged ?? [], fileStats?.staged)}
        onIgnore={handleIgnoreFile}
        isIgnoring={isIgnoring}
        confirmingIgnore={confirmingIgnore}
        onConfirmIgnore={onConfirmIgnore}
        groupingRules={groupingRules}
        onOpenMobileActions={handleOpenMobileActions}
        onContextMenu={handleFileContextMenu}
        mobileSelectionMode={mobileSelectionMode}
        onLongPress={handleLongPress}
        onMobileTap={handleMobileTap}
        onStatsClick={openAggregateMetrics}
        onViewMetrics={openFileMetrics}
      />

      {/* Unstaged Changes */}
      <FileSection
        title="Modified"
        category="unstaged"
        files={files?.unstaged ?? []}
        fileStatuses={files?.statuses}
        binaryFiles={binarySet}
        approvedFiles={approvedPaths}
        maxPathChars={maxPathChars}
        icon={<FileX className="h-3.5 w-3.5 text-amber-500" />}
        selectedFiles={selectedFiles}
        selectedKeySet={selectedKeySet}
        selectionKey={selectionKey}
        onSelectFile={onSelectFile}
        onAction={onStageFile}
        actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
        actionLabel="Stage file"
        isLoading={isStaging}
        changeStats={summarizeFileStats(files?.unstaged ?? [], fileStats?.unstaged)}
        onDiscard={handleDiscardUnstaged}
        isDiscarding={isDiscarding}
        confirmingDiscard={confirmingDiscard}
        onConfirmDiscard={onConfirmDiscard}
        onIgnore={handleIgnoreFile}
        isIgnoring={isIgnoring}
        confirmingIgnore={confirmingIgnore}
        onConfirmIgnore={onConfirmIgnore}
        groupingRules={groupingRules}
        onOpenMobileActions={handleOpenMobileActions}
        onContextMenu={handleFileContextMenu}
        mobileSelectionMode={mobileSelectionMode}
        onLongPress={handleLongPress}
        onMobileTap={handleMobileTap}
        onStatsClick={openAggregateMetrics}
        onViewMetrics={openFileMetrics}
      />

      {/* Untracked Files */}
      <FileSection
        title="Untracked"
        category="untracked"
        files={files?.untracked ?? []}
        fileStatuses={files?.statuses}
        binaryFiles={binarySet}
        approvedFiles={approvedPaths}
        maxPathChars={maxPathChars}
        icon={<File className="h-3.5 w-3.5 text-slate-500" />}
        selectedFiles={selectedFiles}
        selectedKeySet={selectedKeySet}
        selectionKey={selectionKey}
        onSelectFile={onSelectFile}
        onAction={onStageFile}
        actionIcon={<Plus className="h-3 w-3 text-slate-400" />}
        actionLabel="Stage file"
        isLoading={isStaging}
        changeStats={summarizeFileStats(files?.untracked ?? [], fileStats?.untracked)}
        defaultExpanded={false}
        onDiscard={handleDiscardUntracked}
        isDiscarding={isDiscarding}
        confirmingDiscard={confirmingDiscard}
        onConfirmDiscard={onConfirmDiscard}
        onIgnore={handleIgnoreFile}
        isIgnoring={isIgnoring}
        confirmingIgnore={confirmingIgnore}
        onConfirmIgnore={onConfirmIgnore}
        groupingRules={groupingRules}
        onOpenMobileActions={handleOpenMobileActions}
        onContextMenu={handleFileContextMenu}
        mobileSelectionMode={mobileSelectionMode}
        onLongPress={handleLongPress}
        onMobileTap={handleMobileTap}
        onStatsClick={openAggregateMetrics}
        onViewMetrics={openFileMetrics}
      />
    </>
  );
}
