// Action history component showing recent recovery action executions
// [REQ:HEAL-ACTION-001]
import { useQuery } from "@tanstack/react-query";
import { Clock, CheckCircle2, XCircle, Loader2, Activity } from "lucide-react";
import { fetchActionHistory, type ActionLog } from "../../lib/api";

interface ActionHistoryProps {
  checkId?: string;
  limit?: number;
}

function formatRelativeTime(timestamp: string): string {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}h ago`;
  const diffDays = Math.floor(diffHour / 24);
  return `${diffDays}d ago`;
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function ActionLogRow({ log }: { log: ActionLog }) {
  return (
    <div className="flex min-w-0 items-start gap-3 p-3 transition-colors hover:bg-surface-overlay/40">
      <div className="mt-0.5 shrink-0">
        {log.success ? (
          <CheckCircle2 size={16} className="text-accent-success" />
        ) : (
          <XCircle size={16} className="text-accent-danger" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
          <span className="text-sm font-medium text-text-primary">
            {log.actionId}
          </span>
          <span className="text-xs text-text-muted">on</span>
          <span className="break-all font-mono text-xs text-text-muted">{log.checkId}</span>
        </div>
        <p className="mt-0.5 break-words text-xs text-text-muted">{log.message}</p>
        {log.error && (
          <p className="mt-1 break-words text-xs text-accent-danger">{log.error}</p>
        )}
      </div>
      <div className="shrink-0 text-right">
        <p className="text-xs text-text-muted" title={new Date(log.timestamp).toLocaleString()}>
          {formatRelativeTime(log.timestamp)}
        </p>
        <p className="text-xs text-text-muted/70">{formatDuration(log.durationMs)}</p>
      </div>
    </div>
  );
}

export function ActionHistory({ checkId, limit = 20 }: ActionHistoryProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["action-history", checkId],
    queryFn: () => fetchActionHistory(checkId),
    refetchInterval: 30000,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8 text-text-muted">
        <Loader2 size={20} className="mr-2 animate-spin" />
        Loading action history...
      </div>
    );
  }

  if (error) {
    return (
      <div className="py-8 text-center text-text-muted">
        <XCircle size={32} className="mx-auto mb-2 opacity-50" />
        <p>Failed to load action history</p>
      </div>
    );
  }

  const logs = data?.logs || [];

  if (logs.length === 0) {
    return (
      <div className="py-8 text-center text-text-muted">
        <Activity size={32} className="mx-auto mb-2 opacity-50" />
        <p>No actions have been executed yet</p>
        <p className="mt-1 text-xs">
          Use the "Actions" button on resource checks to start, stop, or restart services
        </p>
      </div>
    );
  }

  const displayLogs = logs.slice(0, limit);

  return (
    <div className="divide-y divide-border-default/70">
      {displayLogs.map((log) => (
        <ActionLogRow key={`${log.id}-${log.timestamp}`} log={log} />
      ))}
      {logs.length > limit && (
        <div className="p-3 text-center text-xs text-text-muted">
          Showing {limit} of {logs.length} actions
        </div>
      )}
    </div>
  );
}

// Compact version for inline display
export function ActionHistoryCompact({ checkId }: { checkId: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ["action-history", checkId],
    queryFn: () => fetchActionHistory(checkId),
    staleTime: 30000,
  });

  if (isLoading || !data?.logs || data.logs.length === 0) {
    return null;
  }

  const lastAction = data.logs[0];
  if (!lastAction) {
    return null;
  }

  return (
    <div className="mt-1 flex min-w-0 items-center gap-2 text-xs text-text-muted">
      <Clock size={10} />
      <span className="min-w-0 break-words">
        Last action: {lastAction.actionId} {formatRelativeTime(lastAction.timestamp)}
        {lastAction.success ? (
          <CheckCircle2 size={10} className="ml-1 inline text-accent-success" />
        ) : (
          <XCircle size={10} className="ml-1 inline text-accent-danger" />
        )}
      </span>
    </div>
  );
}
