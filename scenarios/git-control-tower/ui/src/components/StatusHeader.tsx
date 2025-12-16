import {
  GitBranch,
  GitCommit,
  ArrowUp,
  ArrowDown,
  CheckCircle,
  AlertCircle,
  Circle,
  RefreshCw
} from "lucide-react";
import { Badge } from "./ui/badge";
import type { RepoStatus, HealthResponse } from "../lib/api";

interface StatusHeaderProps {
  status?: RepoStatus;
  health?: HealthResponse;
  isLoading: boolean;
  onRefresh: () => void;
}

export function StatusHeader({ status, health, isLoading, onRefresh }: StatusHeaderProps) {
  const isHealthy = health?.readiness ?? false;

  return (
    <header
      className="flex items-center justify-between px-4 py-3 border-b border-slate-800 bg-slate-900/80 backdrop-blur-sm"
      data-testid="status-header"
    >
      <div className="flex items-center gap-6">
        {/* Branch Info */}
        <div className="flex items-center gap-2" data-testid="branch-info">
          <GitBranch className="h-4 w-4 text-slate-400" />
          <span className="font-mono text-sm text-slate-200">
            {status?.branch.head || "—"}
          </span>
          {status?.branch.upstream && (
            <span className="text-xs text-slate-500">
              → {status.branch.upstream}
            </span>
          )}
          {(status?.branch.ahead ?? 0) > 0 && (
            <Badge variant="info" className="gap-1">
              <ArrowUp className="h-3 w-3" />
              {status?.branch.ahead}
            </Badge>
          )}
          {(status?.branch.behind ?? 0) > 0 && (
            <Badge variant="warning" className="gap-1">
              <ArrowDown className="h-3 w-3" />
              {status?.branch.behind}
            </Badge>
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
            <span className="text-xs text-slate-500">Working tree clean</span>
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
