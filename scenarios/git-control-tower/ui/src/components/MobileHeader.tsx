import { useState } from "react";
import {
  GitBranch,
  GitCommit,
  Menu,
  RefreshCw,
  CheckCircle,
  AlertCircle,
  Circle,
  ArrowUp,
  ArrowDown,
  Settings,
  LayoutGrid,
  History,
  X
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { BottomSheet, BottomSheetAction } from "./ui/bottom-sheet";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "../App";
import { BranchSelector, type BranchActions } from "./BranchSelector";

interface MobileHeaderProps {
  status?: RepoStatus;
  health?: HealthResponse;
  syncStatus?: SyncStatusResponse;
  branchActions?: BranchActions;
  isLoading: boolean;
  onRefresh: () => void;
  onOpenLayoutSettings: () => void;
  onOpenGroupingSettings?: () => void;
  // History mode props
  viewingCommit?: ViewingCommit | null;
  onExitHistoryMode?: () => void;
}

export function MobileHeader({
  status,
  health,
  syncStatus,
  branchActions,
  isLoading,
  onRefresh,
  onOpenLayoutSettings,
  onOpenGroupingSettings,
  viewingCommit,
  onExitHistoryMode
}: MobileHeaderProps) {
  const [menuOpen, setMenuOpen] = useState(false);

  const isHealthy = health?.readiness ?? false;
  const ahead = status?.branch.ahead ?? syncStatus?.ahead ?? 0;
  const behind = status?.branch.behind ?? syncStatus?.behind ?? 0;
  const stagedCount = status?.summary.staged ?? 0;
  const unstagedCount = status?.summary.unstaged ?? 0;
  const untrackedCount = status?.summary.untracked ?? 0;
  const conflictCount = status?.summary.conflicts ?? 0;
  const isHistoryMode = Boolean(viewingCommit);

  const isClean =
    stagedCount === 0 &&
    unstagedCount === 0 &&
    untrackedCount === 0 &&
    conflictCount === 0;

  // History mode header - different layout
  if (isHistoryMode && viewingCommit) {
    return (
      <header
        className="flex items-center justify-between px-3 py-2 border-b border-amber-800/50 bg-amber-950/30 backdrop-blur-sm pt-safe"
        data-testid="mobile-header"
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <History className="h-4 w-4 text-amber-400 flex-shrink-0" />
          <Badge variant="warning" className="flex-shrink-0 text-xs">
            History
          </Badge>
          <GitCommit className="h-3.5 w-3.5 text-amber-400 flex-shrink-0" />
          <span className="font-mono text-xs text-amber-200">
            {viewingCommit.hash.substring(0, 7)}
          </span>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={onExitHistoryMode}
          className="gap-1 border-amber-600/50 text-amber-200 hover:bg-amber-900/30 text-xs px-2"
        >
          <X className="h-3.5 w-3.5" />
          Exit
        </Button>
      </header>
    );
  }

  return (
    <>
      <header
        className="flex items-center justify-between px-3 py-2 border-b border-slate-800 bg-slate-900/95 backdrop-blur-sm pt-safe"
        data-testid="mobile-header"
      >
        {/* Left: Branch info */}
        <div className="flex items-center gap-2 min-w-0 flex-1">
          {branchActions ? (
            <BranchSelector
              status={status}
              syncStatus={syncStatus}
              actions={branchActions}
              variant="mobile"
            />
          ) : (
            <>
              <GitBranch className="h-4 w-4 text-slate-400 flex-shrink-0" />
              <span className="font-mono text-sm text-slate-200 truncate">
                {status?.branch.head || "—"}
              </span>
              {ahead > 0 && (
                <Badge variant="info" className="gap-0.5 flex-shrink-0">
                  <ArrowUp className="h-3 w-3" />
                  {ahead}
                </Badge>
              )}
              {behind > 0 && (
                <Badge variant="warning" className="gap-0.5 flex-shrink-0">
                  <ArrowDown className="h-3 w-3" />
                  {behind}
                </Badge>
              )}
            </>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-1">
          {/* Health indicator */}
          <div className="p-2">
            {isHealthy ? (
              <CheckCircle className="h-4 w-4 text-emerald-500" />
            ) : health ? (
              <AlertCircle className="h-4 w-4 text-amber-500" />
            ) : (
              <Circle className="h-4 w-4 text-slate-600" />
            )}
          </div>

          {/* Refresh button */}
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

          {/* Menu button */}
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
                {stagedCount > 0 && (
                  <Badge variant="staged">{stagedCount} staged</Badge>
                )}
                {unstagedCount > 0 && (
                  <Badge variant="unstaged">{unstagedCount} modified</Badge>
                )}
                {untrackedCount > 0 && (
                  <Badge variant="untracked">{untrackedCount} untracked</Badge>
                )}
                {conflictCount > 0 && (
                  <Badge variant="conflict">{conflictCount} conflicts</Badge>
                )}
              </div>
            </div>
          )}

          {/* Action buttons */}
          <BottomSheetAction
            icon={<LayoutGrid className="h-5 w-5 text-slate-300" />}
            label="Layout Settings"
            description="Change panel arrangement"
            onClick={() => {
              setMenuOpen(false);
              onOpenLayoutSettings();
            }}
          />

          {onOpenGroupingSettings && (
            <BottomSheetAction
              icon={<Settings className="h-5 w-5 text-slate-300" />}
              label="Grouping Settings"
              description="Configure file grouping rules"
              onClick={() => {
                setMenuOpen(false);
                onOpenGroupingSettings();
              }}
            />
          )}
        </div>
      </BottomSheet>
    </>
  );
}
