import { useState } from "react";
import {
  ClipboardCheck,
  Menu,
  RefreshCw,
  Settings,
  Search
} from "lucide-react";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "./HistoryModeHeader";
import type { ViewingFileBlame } from "./BlameModeHeader";
import { BranchSelector, type BranchActions, type RepoActions } from "./BranchSelector";
import { HealthIndicator } from "./HealthIndicator";
import { FileStatsBadges } from "./FileStatsBadges";
import { SyncButton } from "./SyncButton";
import { HistoryModeHeader } from "./HistoryModeHeader";
import { BlameModeHeader } from "./BlameModeHeader";
import { useHeaderState } from "../hooks/useHeaderState";

interface MobileHeaderProps {
  status?: RepoStatus;
  health?: HealthResponse;
  syncStatus?: SyncStatusResponse;
  branchActions: BranchActions;
  repoActions?: RepoActions;
  onRepoChange?: (repoId: string | null) => void;
  isLoading: boolean;
  onRefresh: () => void;
  onOpenSettings: () => void;
  onOpenUpstreamInfo?: () => void;
  onOpenFileSearch?: () => void;
  onOpenReview?: () => void;
  viewingCommit?: ViewingCommit | null;
  onExitHistoryMode?: () => void;
  viewingFileBlame?: ViewingFileBlame | null;
  onExitBlameMode?: () => void;
  onPush?: () => void;
  onPull?: () => void;
  isPushing?: boolean;
  isPulling?: boolean;
}

export function MobileHeader({
  status,
  health,
  syncStatus,
  branchActions,
  repoActions,
  onRepoChange,
  isLoading,
  onRefresh,
  onOpenSettings,
  onOpenUpstreamInfo,
  onOpenFileSearch,
  onOpenReview,
  viewingCommit,
  onExitHistoryMode,
  viewingFileBlame,
  onExitBlameMode,
  onPush,
  onPull,
  isPushing,
  isPulling
}: MobileHeaderProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const { isHealthy } = useHeaderState(status, health, syncStatus);

  if (viewingFileBlame && onExitBlameMode) {
    return <BlameModeHeader file={viewingFileBlame} onExit={onExitBlameMode} compact />;
  }

  if (viewingCommit && onExitHistoryMode) {
    return <HistoryModeHeader commit={viewingCommit} onExit={onExitHistoryMode} compact />;
  }

  const stagedCount = status?.summary.staged ?? 0;
  const unstagedCount = status?.summary.unstaged ?? 0;
  const untrackedCount = status?.summary.untracked ?? 0;
  const conflictCount = status?.summary.conflicts ?? 0;
  const isClean =
    stagedCount === 0 && unstagedCount === 0 && untrackedCount === 0 && conflictCount === 0;

  return (
    <>
      <header
        className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/95 backdrop-blur-sm pt-safe"
        data-testid="mobile-header"
      >
        {/* Left: Branch info */}
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <BranchSelector
            status={status}
            syncStatus={syncStatus}
            actions={branchActions}
            repoActions={repoActions}
            onRepoChange={onRepoChange}
            onOpenUpstreamInfo={onOpenUpstreamInfo}
            variant="mobile"
            commitOid={status?.branch.oid}
          />
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-1">
          {onPush && onPull && (
            <SyncButton
              ahead={syncStatus?.ahead ?? 0}
              behind={syncStatus?.behind ?? 0}
              canPush={syncStatus?.can_push ?? false}
              canPull={syncStatus?.can_pull ?? false}
              onPush={onPush}
              onPull={onPull}
              isPushing={isPushing ?? false}
              isPulling={isPulling ?? false}
              warning={syncStatus?.safety_warnings?.join("; ")}
            />
          )}

          <div className="p-2">
            <HealthIndicator health={health} isHealthy={isHealthy} />
          </div>

          {onOpenReview && (
            <button
              onClick={onOpenReview}
              className="p-3 rounded-lg hover:bg-slate-800 active:bg-slate-700 transition-colors touch-target"
              aria-label="Scenario review"
              data-testid="mobile-review-button"
            >
              <ClipboardCheck className="h-5 w-5 text-slate-400" />
            </button>
          )}

          {onOpenFileSearch && (
            <button
              onClick={onOpenFileSearch}
              className="p-3 rounded-lg hover:bg-slate-800 active:bg-slate-700 transition-colors touch-target"
              aria-label="Search files"
              data-testid="mobile-search-button"
            >
              <Search className="h-5 w-5 text-slate-400" />
            </button>
          )}

          <button
            onClick={onRefresh}
            disabled={isLoading}
            className="p-3 rounded-lg hover:bg-slate-800 active:bg-slate-700 transition-colors disabled:opacity-50 touch-target"
            aria-label="Refresh"
            data-testid="mobile-refresh-button"
          >
            <RefreshCw
              className={`h-5 w-5 text-slate-400 ${isLoading ? "animate-spin" : ""}`}
            />
          </button>

          <button
            onClick={() => setMenuOpen(true)}
            className="p-3 rounded-lg hover:bg-slate-800 active:bg-slate-700 transition-colors touch-target"
            aria-label="Open menu"
            data-testid="mobile-menu-button"
          >
            <Menu className="h-5 w-5 text-slate-400" />
          </button>
        </div>
      </header>

      {/* Menu Bottom Sheet */}
      <BottomSheet
        isOpen={menuOpen}
        onClose={() => setMenuOpen(false)}
        title="Settings & Info"
      >
        <div className="space-y-2">
          {/* Repository info section */}
          <div className="rounded-xl border border-slate-800/60 bg-slate-900/40 p-4 mb-4">
            <div className="text-xs text-slate-500 uppercase tracking-wide mb-2">
              Repository
            </div>
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm text-slate-300">Branch</span>
                <span className="font-mono text-sm text-slate-100">
                  {status?.branch.head || "—"}
                </span>
              </div>
              {status?.branch.oid && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-300">Commit</span>
                  <span className="font-mono text-xs text-slate-400">
                    {status.branch.oid.substring(0, 7)}
                  </span>
                </div>
              )}
              {status?.branch.upstream && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-slate-300">Upstream</span>
                  <span className="font-mono text-xs text-slate-400 truncate max-w-[150px]">
                    {status.branch.upstream}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* File stats */}
          {!isClean && (
            <div className="rounded-xl border border-slate-800/60 bg-slate-900/40 p-4 mb-4">
              <div className="text-xs text-slate-500 uppercase tracking-wide mb-2">
                Changes
              </div>
              <div className="flex flex-wrap gap-2">
                <FileStatsBadges
                  staged={stagedCount}
                  unstaged={unstagedCount}
                  untracked={untrackedCount}
                  conflicts={conflictCount}
                />
              </div>
            </div>
          )}

          {/* Action buttons */}
          <BottomSheetAction
            icon={<Settings className="h-5 w-5 text-slate-300" />}
            label="Settings"
            description="Layout, grouping, and credentials"
            onClick={() => {
              setMenuOpen(false);
              onOpenSettings();
            }}
          />

        </div>
      </BottomSheet>
    </>
  );
}
