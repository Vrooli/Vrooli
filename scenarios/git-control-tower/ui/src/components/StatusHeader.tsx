import {
  GitBranch,
  GitCommit,
  ArrowUp,
  ArrowDown,
  CheckCircle,
  AlertCircle,
  Circle,
  RefreshCw,
  LayoutGrid,
  History,
  X
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "../App";
import { BranchSelector, type BranchActions } from "./BranchSelector";

interface StatusHeaderProps {
  status?: RepoStatus;
  health?: HealthResponse;
  syncStatus?: SyncStatusResponse;
  branchActions?: BranchActions;
  isLoading: boolean;
  onRefresh: () => void;
  onOpenLayoutSettings: () => void;
  onOpenUpstreamInfo?: () => void;
  // History mode props
  viewingCommit?: ViewingCommit | null;
  onExitHistoryMode?: () => void;
}

export function StatusHeader({
  status,
  health,
  syncStatus,
  branchActions,
  isLoading,
  onRefresh,
  onOpenLayoutSettings,
  onOpenUpstreamInfo,
  viewingCommit,
  onExitHistoryMode
}: StatusHeaderProps) {
  const isHealthy = health?.readiness ?? false;
  const ahead = syncStatus?.ahead ?? status?.branch.ahead ?? 0;
  const behind = syncStatus?.behind ?? status?.branch.behind ?? 0;
  const branchName = syncStatus?.branch ?? status?.branch.head ?? "";
  const upstreamRef = syncStatus?.upstream ?? status?.branch.upstream ?? "";
  const upstreamBranch = upstreamRef ? upstreamRef.split("/").slice(1).join("/") : "";
  const trackingMismatch = Boolean(
    branchName && upstreamBranch && branchName !== upstreamBranch
  );
  const isHistoryMode = Boolean(viewingCommit);
  const cleanDetails = [
    ahead > 0 ? `${ahead} ahead` : "",
    behind > 0 ? `${behind} behind` : ""
  ]
    .filter(Boolean)
    .join(", ");

  // History mode header - different layout showing commit info
  if (isHistoryMode && viewingCommit) {
    return (
      <header
        className="relative z-30 flex items-center justify-between px-4 py-3 border-b border-amber-800/50 bg-amber-950/30 backdrop-blur-sm"
        data-testid="status-header"
      >
        <div className="flex items-center gap-4 min-w-0 flex-1">
          {/* History mode indicator */}
          <div className="flex items-center gap-2 flex-shrink-0">
            <History className="h-4 w-4 text-amber-400" />
            <Badge variant="warning" className="text-xs">
              Viewing History
            </Badge>
          </div>

          {/* Commit info */}
          <div className="flex items-center gap-3 min-w-0" data-testid="history-commit-info">
            <div className="flex items-center gap-2 flex-shrink-0">
              <GitCommit className="h-4 w-4 text-amber-400" />
              <span className="font-mono text-sm text-amber-200">
                {viewingCommit.hash.substring(0, 7)}
              </span>
            </div>
            <span className="text-sm text-slate-300 truncate" title={viewingCommit.subject}>
              {viewingCommit.subject}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3 flex-shrink-0">
          {/* Commit metadata */}
          {viewingCommit.author && (
            <span className="text-xs text-slate-500 hidden sm:block">
              by {viewingCommit.author}
            </span>
          )}

          {/* Exit history mode button */}
          <Button
            variant="outline"
            size="sm"
            onClick={onExitHistoryMode}
            className="gap-1.5 border-amber-600/50 text-amber-200 hover:bg-amber-900/30"
            data-testid="exit-history-mode"
          >
            <X className="h-3.5 w-3.5" />
            Back to Working Directory
          </Button>
        </div>
      </header>
    );
  }

  return (
    <header
      className="relative z-30 flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-slate-900/80 backdrop-blur-sm"
      data-testid="status-header"
    >
      <div className="flex items-center gap-6">
        {/* Branch Info */}
        <div className="flex items-center gap-2" data-testid="branch-info">
          {branchActions ? (
            <BranchSelector status={status} syncStatus={syncStatus} actions={branchActions} />
          ) : (
            <>
              <GitBranch className="h-4 w-4 text-slate-400" />
              <span className="font-mono text-sm text-slate-200">
                {status?.branch.head || "—"}
              </span>
              {status?.branch.upstream && (
                <span className="text-xs text-slate-500">
                  → {status.branch.upstream}
                </span>
              )}
              {ahead > 0 && (
                <Badge variant="info" className="gap-1">
                  <ArrowUp className="h-3 w-3" />
                  {ahead}
                </Badge>
              )}
              {behind > 0 && (
                <Badge variant="warning" className="gap-1">
                  <ArrowDown className="h-3 w-3" />
                  {behind}
                </Badge>
              )}
            </>
          )}
          {upstreamRef && (
            <button
              type="button"
              onClick={onOpenUpstreamInfo}
              className="rounded-md focus:outline-none focus-visible:ring-1 focus-visible:ring-slate-400/60 disabled:opacity-50"
              aria-label={`Open upstream details for ${upstreamRef}`}
              disabled={!onOpenUpstreamInfo}
            >
              <Badge variant={trackingMismatch ? "warning" : "default"} className="gap-1">
                {trackingMismatch ? "Tracks" : "Upstream"} {upstreamRef}
              </Badge>
            </button>
          )}
        </div>

        {/* Commit OID */}
        {status?.branch.oid && (
          <div className="flex items-center gap-2 text-slate-500" data-testid="commit-oid">
            <GitCommit className="h-4 w-4" />
            <span className="font-mono text-xs">
              {status.branch.oid.substring(0, 7)}
            </span>
          </div>
        )}
      </div>

      <div className="flex items-center gap-4">
        {/* File Stats */}
        <div className="flex items-center gap-3" data-testid="file-stats">
          {(status?.summary.staged ?? 0) > 0 && (
            <Badge variant="staged">
              {status?.summary.staged} staged
            </Badge>
          )}
          {(status?.summary.unstaged ?? 0) > 0 && (
            <Badge variant="unstaged">
              {status?.summary.unstaged} modified
            </Badge>
          )}
          {(status?.summary.untracked ?? 0) > 0 && (
            <Badge variant="untracked">
              {status?.summary.untracked} untracked
            </Badge>
          )}
          {(status?.summary.conflicts ?? 0) > 0 && (
            <Badge variant="conflict">
              {status?.summary.conflicts} conflicts
            </Badge>
          )}
          {status &&
           status.summary.staged === 0 &&
           status.summary.unstaged === 0 &&
           status.summary.untracked === 0 && (
            <span className="text-xs text-slate-500">
              {cleanDetails ? `Working tree clean (${cleanDetails})` : "Working tree clean"}
            </span>
          )}
        </div>

        {/* Health Status */}
        <div
          className="flex items-center gap-2"
          data-testid="health-status"
          title={isHealthy ? "All systems healthy" : "System issues detected"}
        >
          {isHealthy ? (
            <CheckCircle className="h-4 w-4 text-emerald-500" />
          ) : health ? (
            <AlertCircle className="h-4 w-4 text-amber-500" />
          ) : (
            <Circle className="h-4 w-4 text-slate-600" />
          )}
        </div>

        <button
          onClick={onOpenLayoutSettings}
          className="p-2 rounded-md hover:bg-slate-800 transition-colors"
          data-testid="layout-settings-button"
          aria-label="Open layout settings"
        >
          <LayoutGrid className="h-4 w-4 text-slate-400" />
        </button>

        {/* Refresh Button */}
        <button
          onClick={onRefresh}
          disabled={isLoading}
          className="p-2 rounded-md hover:bg-slate-800 transition-colors disabled:opacity-50"
          data-testid="refresh-button"
          aria-label="Refresh status"
        >
          <RefreshCw className={`h-4 w-4 text-slate-400 ${isLoading ? "animate-spin" : ""}`} />
        </button>
      </div>
    </header>
  );
}
