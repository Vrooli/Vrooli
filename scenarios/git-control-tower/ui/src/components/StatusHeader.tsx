import {
  GitCommit,
  RefreshCw,
  Search,
  Settings
} from "lucide-react";
import { Badge } from "./ui/badge";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "./HistoryModeHeader";
import type { ViewingFileBlame } from "./BlameModeHeader";
import { BranchSelector, type BranchActions, type RepoActions } from "./BranchSelector";
import { HealthIndicator } from "./HealthIndicator";
import { FileStatsBadges } from "./FileStatsBadges";
import { IconButton } from "./IconButton";
import { HistoryModeHeader } from "./HistoryModeHeader";
import { BlameModeHeader } from "./BlameModeHeader";
import { useHeaderState } from "../hooks/useHeaderState";

interface StatusHeaderProps {
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
  viewingCommit?: ViewingCommit | null;
  onExitHistoryMode?: () => void;
  viewingFileBlame?: ViewingFileBlame | null;
  onExitBlameMode?: () => void;
}

export function StatusHeader({
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
  viewingCommit,
  onExitHistoryMode,
  viewingFileBlame,
  onExitBlameMode
}: StatusHeaderProps) {
  const { isHealthy, upstreamRef, trackingMismatch, cleanDetails } =
    useHeaderState(status, health, syncStatus);

  if (viewingFileBlame && onExitBlameMode) {
    return <BlameModeHeader file={viewingFileBlame} onExit={onExitBlameMode} />;
  }

  if (viewingCommit && onExitHistoryMode) {
    return <HistoryModeHeader commit={viewingCommit} onExit={onExitHistoryMode} />;
  }

  return (
    <header
      className="relative z-30 flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-slate-900/80 backdrop-blur-sm"
      data-testid="status-header"
    >
      <div className="flex items-center gap-6">
        {/* Branch Info */}
        <div className="flex items-center gap-2" data-testid="branch-info">
          <BranchSelector
            status={status}
            syncStatus={syncStatus}
            actions={branchActions}
            repoActions={repoActions}
            onRepoChange={onRepoChange}
            onOpenUpstreamInfo={onOpenUpstreamInfo}
          />
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
        <FileStatsBadges
          staged={status?.summary.staged ?? 0}
          unstaged={status?.summary.unstaged ?? 0}
          untracked={status?.summary.untracked ?? 0}
          conflicts={status?.summary.conflicts ?? 0}
          cleanDetails={cleanDetails}
          variant="compact"
        />

        <HealthIndicator health={health} isHealthy={isHealthy} />

        <IconButton
          onClick={onOpenFileSearch}
          label="Search files (Ctrl+K)"
          title="Search files (Ctrl+K)"
          data-testid="file-search-button"
        >
          <Search className="h-4 w-4 text-slate-400" />
        </IconButton>

        <IconButton
          onClick={onOpenSettings}
          label="Open settings"
          data-testid="settings-button"
        >
          <Settings className="h-4 w-4 text-slate-400" />
        </IconButton>

        <IconButton
          onClick={onRefresh}
          disabled={isLoading}
          label="Refresh status"
          data-testid="refresh-button"
        >
          <RefreshCw className={`h-4 w-4 text-slate-400 ${isLoading ? "animate-spin" : ""}`} />
        </IconButton>
      </div>
    </header>
  );
}
