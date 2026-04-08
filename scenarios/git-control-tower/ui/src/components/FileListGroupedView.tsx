import {
  File,
  FilePlus,
  FileX,
  AlertTriangle,
  Plus,
  Minus,
  Trash2,
  ChevronDown,
  ChevronRight,
  ClipboardCheck,
} from "lucide-react";
import { Button } from "./ui/button";
import type { RepoFilesStatus, RepoFileStats } from "../lib/api";
import { FileSection } from "./FileSection";
import { type FileCategory, type SelectedFileEntry, type GroupingRule, summarizeFileStats } from "./FileListTypes";

interface GroupEntry {
  id: string;
  label: string;
  displayPrefix: string;
  files: Record<FileCategory, string[]>;
}

export interface FileListGroupedViewProps {
  groupedSections: GroupEntry[];
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
  compactHeader: boolean;
  isMobile: boolean;
  collapsedGroups: Set<string>;
  confirmingGroup: string | null;
  onStagePaths?: (paths: string[]) => void;
  onDiscardPaths?: (paths: string[], untracked: boolean) => void;
  onOpenReview?: (scenarioSlug: string) => void;
  onToggleGroupCollapse: (groupId: string) => void;
  onSetConfirmingGroup: (groupId: string | null) => void;
  handleDiscardUnstaged: (path: string) => void;
  handleDiscardUntracked: (path: string) => void;
  handleIgnoreFile: (path: string, level?: "project" | "group", groupDir?: string) => void;
  handleOpenMobileActions: (file: string) => void;
  handleFileContextMenu?: (file: string, event: React.MouseEvent) => void;
  mobileSelectionMode: boolean;
  handleLongPress: (file: string, staged: boolean) => void;
  handleMobileTap: (file: string, staged: boolean, mode: "toggle" | "range") => void;
  openGroupMetrics: (groupFiles: Record<string, string[]>, groupLabel: string) => void;
  openGroupCategoryMetrics: (paths: string[], category: "staged" | "unstaged" | "untracked", groupLabel: string) => void;
  openFileMetrics: (path: string, category?: FileCategory) => void;
}

export function FileListGroupedView({
  groupedSections,
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
  compactHeader,
  isMobile,
  collapsedGroups,
  confirmingGroup,
  onStagePaths,
  onDiscardPaths,
  onOpenReview,
  onToggleGroupCollapse,
  onSetConfirmingGroup,
  handleDiscardUnstaged,
  handleDiscardUntracked,
  handleIgnoreFile,
  handleOpenMobileActions,
  handleFileContextMenu,
  mobileSelectionMode,
  handleLongPress,
  handleMobileTap,
  openGroupMetrics,
  openGroupCategoryMetrics,
  openFileMetrics,
}: FileListGroupedViewProps) {
  return (
    <>
      {groupedSections.map((group) => {
        const stageable = [
          ...group.files.unstaged,
          ...group.files.untracked,
          ...group.files.conflicts,
        ];
        const discardTracked = group.files.unstaged;
        const discardUntracked = group.files.untracked;
        const discardCount = discardTracked.length + discardUntracked.length;
        const groupCount =
          group.files.conflicts.length +
          group.files.staged.length +
          group.files.unstaged.length +
          group.files.untracked.length;
        const isGroupCollapsed = collapsedGroups.has(group.id);

        return (
          <div
            key={group.id}
            className="mb-4 rounded-lg border border-slate-800/80 bg-slate-950/40"
            data-testid={`file-group-${group.id}`}
          >
            <div
              className={`flex flex-wrap items-center justify-between gap-2 px-3 py-2 ${isGroupCollapsed ? "" : "border-b border-slate-800/70"}`}
            >
              <button
                type="button"
                className="flex items-center gap-2 min-w-0 hover:bg-slate-800/30 rounded px-1 -ml-1 transition-colors"
                onClick={() => onToggleGroupCollapse(group.id)}
                data-testid={`file-group-toggle-${group.id}`}
              >
                {isGroupCollapsed ? (
                  <ChevronRight className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
                ) : (
                  <ChevronDown className={`text-slate-500 flex-shrink-0 ${isMobile ? "h-4.5 w-4.5" : "h-3.5 w-3.5"}`} />
                )}
                <div className="min-w-0 text-left">
                  <div className={`font-semibold uppercase tracking-wider text-slate-300 ${isMobile ? "text-sm" : "text-xs"}`}>
                    {group.label}
                  </div>
                  {group.displayPrefix && (
                    <div className={`text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`}>
                      {group.displayPrefix}
                    </div>
                  )}
                </div>
              </button>
              <div className={`flex items-center gap-2 text-slate-500 ${isMobile ? "text-sm" : "text-xs"}`}>
                {onOpenReview && group.displayPrefix && /^(scenarios|apps|services)\//.test(group.displayPrefix) ? (
                  <button
                    type="button"
                    className="h-7 px-2 inline-flex items-center gap-1 rounded text-xs text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 transition-colors"
                    onClick={(e) => {
                      e.stopPropagation();
                      onOpenReview(group.displayPrefix?.split("/")[1] ?? "");
                    }}
                    title="Open scenario review"
                  >
                    {groupCount !== undefined && <span>{groupCount} files</span>}
                    <ClipboardCheck className="h-3.5 w-3.5" />
                  </button>
                ) : (
                  <button
                    type="button"
                    className="hover:underline decoration-slate-600 cursor-pointer"
                    onClick={(e) => {
                      e.stopPropagation();
                      openGroupMetrics(group.files, group.label);
                    }}
                  >
                    {groupCount} files
                  </button>
                )}
                {!isGroupCollapsed &&
                  stageable.length > 0 &&
                  onStagePaths && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onStagePaths(stageable)}
                      disabled={isStaging}
                      className={compactHeader ? "h-7 w-7 p-0" : "h-7 px-2"}
                      title="Stage All"
                    >
                      {compactHeader ? <Plus className="h-3 w-3" /> : "Stage All"}
                    </Button>
                  )}
                {!isGroupCollapsed &&
                  discardCount > 0 &&
                  onDiscardPaths && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => onSetConfirmingGroup(group.id)}
                      disabled={isDiscarding}
                      className={`border-red-400/40 text-red-200 hover:bg-red-900/20 ${compactHeader ? "h-7 w-7 p-0" : "h-7 px-2"}`}
                      title="Discard All"
                    >
                      {compactHeader ? <Trash2 className="h-3 w-3" /> : "Discard All"}
                    </Button>
                  )}
              </div>
            </div>
            {!isGroupCollapsed && (
              <>
                {confirmingGroup === group.id &&
                  discardCount > 0 && (
                    <div className="flex items-center justify-between gap-2 px-3 py-2 text-xs text-red-200 bg-red-950/30 border-b border-red-900/40">
                      <span>
                        Discard {discardCount} changes in this group?
                      </span>
                      <div className="flex items-center gap-2">
                        <button
                          type="button"
                          className="px-2 py-1 rounded border border-red-400/40 text-red-100 hover:bg-red-900/30"
                          onClick={() => {
                            if (discardTracked.length > 0) {
                              onDiscardPaths?.(discardTracked, false);
                            }
                            if (discardUntracked.length > 0) {
                              onDiscardPaths?.(discardUntracked, true);
                            }
                            onSetConfirmingGroup(null);
                          }}
                        >
                          Discard
                        </button>
                        <button
                          type="button"
                          className="px-2 py-1 rounded border border-slate-600 text-slate-200 hover:bg-slate-800/50"
                          onClick={() => onSetConfirmingGroup(null)}
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  )}
                <div className="px-2 py-2">
                  <FileSection
                    key={`${group.id}-conflicts`}
                    title="Conflicts"
                    category="conflicts"
                    files={group.files.conflicts}
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
                    changeStats={summarizeFileStats(group.files.conflicts, fileStats?.unstaged)}
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
                    onStatsClick={() => openGroupCategoryMetrics(group.files.conflicts, "unstaged", group.label)}
                    onViewMetrics={openFileMetrics}
                  />
                  <FileSection
                    key={`${group.id}-staged`}
                    title="Staged"
                    category="staged"
                    files={group.files.staged}
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
                    changeStats={summarizeFileStats(group.files.staged, fileStats?.staged)}
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
                    onStatsClick={() => openGroupCategoryMetrics(group.files.staged, "staged", group.label)}
                    onViewMetrics={openFileMetrics}
                  />
                  <FileSection
                    key={`${group.id}-unstaged`}
                    title="Modified"
                    category="unstaged"
                    files={group.files.unstaged}
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
                    changeStats={summarizeFileStats(group.files.unstaged, fileStats?.unstaged)}
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
                    onStatsClick={() => openGroupCategoryMetrics(group.files.unstaged, "unstaged", group.label)}
                    onViewMetrics={openFileMetrics}
                  />
                  <FileSection
                    key={`${group.id}-untracked`}
                    title="Untracked"
                    category="untracked"
                    files={group.files.untracked}
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
                    changeStats={summarizeFileStats(group.files.untracked, fileStats?.untracked)}
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
                    onStatsClick={() => openGroupCategoryMetrics(group.files.untracked, "untracked", group.label)}
                    onViewMetrics={openFileMetrics}
                  />
                </div>
              </>
            )}
          </div>
        );
      })}
    </>
  );
}
