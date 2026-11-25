import { useMemo, useState } from "react";
import {
  AlertCircle,
  CheckCircle,
  Circle,
  Menu,
  RefreshCw,
  Settings,
  Search,
  Share2
} from "lucide-react";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "./HistoryModeHeader";
import type { ViewingFileBlame } from "./BlameModeHeader";
import { BranchSelector, type BranchActions, type RepoActions } from "./BranchSelector";
import { FileStatsBadges } from "./FileStatsBadges";
import { SyncButton } from "./SyncButton";
import { HistoryModeHeader } from "./HistoryModeHeader";
import { BlameModeHeader } from "./BlameModeHeader";
import { useHeaderState } from "../hooks/useHeaderState";
import { useBranches, useHealth } from "../lib/hooks";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";

interface MobileHeaderProps {
  status?: RepoStatus;
  health?: HealthResponse;
  syncStatus?: SyncStatusResponse;
  branchActions: BranchActions;
  repoId?: string | null;
  repoActions?: RepoActions;
  onRepoChange?: (repoId: string | null) => void;
  isLoading: boolean;
  onRefresh: () => void;
  onOpenSettings: () => void;
  onOpenUpstreamInfo?: () => void;
  onOpenFileSearch?: () => void;
  onOpenSourceDistributions?: () => void;
  viewingCommit?: ViewingCommit | null;
  onExitHistoryMode?: () => void;
  viewingFileBlame?: ViewingFileBlame | null;
  onExitBlameMode?: () => void;
  onPush?: () => void;
  onPull?: () => void;
  isPushing?: boolean;
  isPulling?: boolean;
  /** Phase and elapsed time while a remote operation is running. */
  syncProgressLabel?: string;
}

export function MobileHeader({
  status,
  health,
  syncStatus,
  branchActions,
  repoId,
  repoActions,
  onRepoChange,
  isLoading,
  onRefresh,
  onOpenSettings,
  onOpenUpstreamInfo,
  onOpenFileSearch,
  onOpenSourceDistributions,
  viewingCommit,
  onExitHistoryMode,
  viewingFileBlame,
  onExitBlameMode,
  onPush,
  onPull,
  isPushing,
  isPulling,
  syncProgressLabel
}: MobileHeaderProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const healthQuery = useHealth();
  const currentHealth = health ?? healthQuery.data;
  const branchesQuery = useBranches(repoId, Boolean(repoId));
  const currentBranchActions = useMemo(() => ({
    ...branchActions,
    branches: branchesQuery.data,
    isLoading: branchesQuery.isLoading,
  }), [branchActions, branchesQuery.data, branchesQuery.isLoading]);
  const headerLoading = isLoading || healthQuery.isLoading;
  const { isHealthy } = useHeaderState(status, currentHealth, syncStatus);

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
        className="gct-mobile-header relative z-40 flex items-center justify-between px-2 py-0 border-b border-slate-800"
        data-testid="mobile-header"
      >
        {/* Left: Branch info */}
        <div className="flex items-center gap-1 min-w-0 flex-1">
          <BranchSelector
            status={status}
            syncStatus={syncStatus}
            actions={currentBranchActions}
            repoActions={repoActions}
            onRepoChange={onRepoChange}
            onOpenUpstreamInfo={onOpenUpstreamInfo}
            variant="mobile"
            commitOid={status?.branch.oid}
          />
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-0">
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
              progressLabel={syncProgressLabel}
              warning={syncStatus?.safety_warnings?.join("; ")}
            />
          )}

          <IconButton
            onClick={() => setMenuOpen(true)}
            aria-label="More repository actions"
            size="xs"
            surface="ghost"
            data-testid="mobile-menu-button"
          >
            <Menu className="h-5 w-5 text-slate-400" />
          </IconButton>
        </div>
      </header>

      {/* Menu Bottom Sheet */}
      <BottomSheet
        isOpen={menuOpen}
        onClose={() => setMenuOpen(false)}
        title="Repository tools"
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

          {/* Health status */}
          <div className="rounded-xl border border-slate-800/60 bg-slate-900/40 p-4 mb-4">
            <div className="text-xs text-slate-500 uppercase tracking-wide mb-2">
              System Health
            </div>
            <div className="flex items-center gap-2">
              {isHealthy ? (
                <CheckCircle className="h-4 w-4 text-emerald-500" />
              ) : currentHealth ? (
                <AlertCircle className="h-4 w-4 text-amber-500" />
              ) : (
                <Circle className="h-4 w-4 text-slate-600" />
              )}
              <span className="text-sm text-slate-300">
                {isHealthy ? "All systems healthy" : currentHealth ? "Issues detected" : "Unknown"}
              </span>
            </div>
          </div>

          {/* Less frequent actions live here to keep the mobile chrome compact. */}
          {onOpenFileSearch && (
            <BottomSheetAction
              icon={<Search className="h-5 w-5 text-slate-300" />}
              label="Search files"
              description="Find a file in this repository"
              onClick={() => {
                setMenuOpen(false);
                onOpenFileSearch();
              }}
              testId="mobile-menu-search"
            />
          )}

          {onOpenSourceDistributions && (
            <BottomSheetAction
              icon={<Share2 className="h-5 w-5 text-cyan-400" />}
              label="Source distributions"
              description="Manage source package distributions"
              onClick={() => {
                setMenuOpen(false);
                onOpenSourceDistributions();
              }}
              testId="mobile-menu-source-distributions"
            />
          )}

          {/* Action buttons */}
          <BottomSheetAction
            icon={<RefreshCw className={`h-5 w-5 text-slate-300 ${headerLoading ? "animate-spin" : ""}`} />}
            label="Refresh status"
            description="Re-scan git status"
            disabled={headerLoading}
            onClick={() => {
              setMenuOpen(false);
              onRefresh();
            }}
          />

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
