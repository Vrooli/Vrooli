import {
  ClipboardCheck,
  RefreshCw,
  Search,
  Settings
} from "lucide-react";
import type { RepoStatus, HealthResponse, SyncStatusResponse } from "../lib/api";
import type { ViewingCommit } from "./HistoryModeHeader";
import type { ViewingFileBlame } from "./BlameModeHeader";
import { BranchSelector, type BranchActions, type RepoActions } from "./BranchSelector";
import { HealthIndicator } from "./HealthIndicator";
import { FileStatsBadges } from "./FileStatsBadges";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { HistoryModeHeader } from "./HistoryModeHeader";
import { BlameModeHeader } from "./BlameModeHeader";
import { SyncButton } from "./SyncButton";
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
  /** Actionable repository-health findings; drives the settings badge. */
  healthIssueCount?: number;
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
  healthIssueCount = 0,
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
}: StatusHeaderProps) {
  const { isHealthy, cleanDetails } =
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
      <div className="flex items-center gap-4">
        {/* Branch Info */}
        <div className="flex items-center gap-2" data-testid="branch-info">
          <BranchSelector
            status={status}
            syncStatus={syncStatus}
            actions={branchActions}
            repoActions={repoActions}
            onRepoChange={onRepoChange}
            onOpenUpstreamInfo={onOpenUpstreamInfo}
            commitOid={status?.branch.oid}
          />
        </div>

        {/* Sync Button */}
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
          onClick={onOpenReview}
          aria-label="Scenario review"
          size="xs"
          surface="ghost"
          title="Scenario review"
          data-testid="review-button"
        >
          <ClipboardCheck className="h-4 w-4 text-slate-400" />
        </IconButton>

        <IconButton
          onClick={onOpenFileSearch}
          aria-label="Search files (Ctrl+K)"
          size="xs"
          surface="ghost"
          title="Search files (Ctrl+K)"
          data-testid="file-search-button"
        >
          <Search className="h-4 w-4 text-slate-400" />
        </IconButton>

        <IconButton
          onClick={onOpenSettings}
          aria-label={
            healthIssueCount > 0
              ? `Open settings (${healthIssueCount} health ${healthIssueCount === 1 ? "issue" : "issues"})`
              : "Open settings"
          }
          data-testid="settings-button"
        >
          <span className="relative inline-flex">
            <Settings className="h-4 w-4 text-slate-400" />
            {/* Actionable findings only, so a lit dot always means something to do. */}
            {healthIssueCount > 0 && (
              <span
                className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-amber-400 ring-2 ring-slate-900"
                data-testid="settings-health-badge"
              />
            )}
          </span>
        </IconButton>

        <IconButton
          onClick={onRefresh}
          disabled={isLoading}
          aria-label="Refresh status"
          size="xs"
          surface="ghost"
          data-testid="refresh-button"
        >
          <RefreshCw className={`h-4 w-4 text-slate-400 ${isLoading ? "animate-spin" : ""}`} />
        </IconButton>
      </div>
    </header>
  );
}
