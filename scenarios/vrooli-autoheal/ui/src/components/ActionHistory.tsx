// Action history component showing recent recovery action executions
// [REQ:HEAL-ACTION-001]
import { useQuery } from "@tanstack/react-query";
import { Clock, CheckCircle2, XCircle, Loader2, Activity } from "lucide-react";
import { fetchActionHistory, type ActionLog } from "../lib/api";

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
    <div className="flex items-start gap-3 p-3 hover:bg-white/[0.02] transition-colors">
      <div className="flex-shrink-0 mt-0.5">
        {log.success ? (
          <CheckCircle2 size={16} className="text-emerald-400" />
        ) : (
          <XCircle size={16} className="text-red-400" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-slate-300">
            {log.actionId}
          </span>
          <span className="text-xs text-slate-500">on</span>
          <span className="text-xs font-mono text-slate-400">{log.checkId}</span>
        </div>
        <p className="text-xs text-slate-400 mt-0.5">{log.message}</p>
        {log.error && (
          <p className="text-xs text-red-400 mt-1">{log.error}</p>
        )}
      </div>
      <div className="flex-shrink-0 text-right">
        <p className="text-xs text-slate-500" title={new Date(log.timestamp).toLocaleString()}>
          {formatRelativeTime(log.timestamp)}
        </p>
        <p className="text-xs text-slate-600">{formatDuration(log.durationMs)}</p>
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
      <div className="flex items-center justify-center py-8 text-slate-500">
        <Loader2 size={20} className="animate-spin mr-2" />
        Loading action history...
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-8 text-slate-500">
        <XCircle size={32} className="mx-auto mb-2 opacity-50" />
        <p>Failed to load action history</p>
      </div>
    );
  }

  const logs = data?.logs || [];

  if (logs.length === 0) {
    return (
      <div className="text-center py-8 text-slate-500">
        <Activity size={32} className="mx-auto mb-2 opacity-50" />
        <p>No actions have been executed yet</p>
        <p className="text-xs mt-1">
          Use the "Actions" button on resource checks to start, stop, or restart services
        </p>
      </div>
    );
  }

  const displayLogs = logs.slice(0, limit);

  return (
    <div className="divide-y divide-white/5">
      {displayLogs.map((log) => (
        <ActionLogRow key={`${log.id}-${log.timestamp}`} log={log} />
      ))}
      {logs.length > limit && (
        <div className="p-3 text-center text-xs text-slate-500">
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

  return (
    <div className="flex items-center gap-2 text-xs text-slate-500 mt-1">
      <Clock size={10} />
      <span>
        Last action: {lastAction.actionId} {formatRelativeTime(lastAction.timestamp)}
        {lastAction.success ? (
          <CheckCircle2 size={10} className="inline ml-1 text-emerald-400" />
        ) : (
          <XCircle size={10} className="inline ml-1 text-red-400" />
        )}
      </span>
    </div>
  );
}
