// Check detail modal for drill-down view
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useQuery } from "@tanstack/react-query";
import { useEffect, useCallback } from "react";
import { X, Download, Clock, AlertCircle, CheckCircle, AlertTriangle, Info } from "lucide-react";
import { fetchCheckHistory, HealthStatus } from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";
import { StatusIcon } from "./StatusIcon";
import { exportCheckHistoryToCSV } from "../lib/export";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";

interface CheckDetailModalProps {
  checkId: string;
  onClose: () => void;
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
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

export function CheckDetailModal({ checkId, onClose }: CheckDetailModalProps) {
  const { getTitle, getMetadata } = useCheckMetadata();
  const metadata = getMetadata(checkId);
  const title = getTitle(checkId);
  const showCheckId = title !== checkId;

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["check-history", checkId],
    queryFn: () => fetchCheckHistory(checkId),
    refetchInterval: 30000,
  });

  // Handle escape key
  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleEsc);
    return () => window.removeEventListener("keydown", handleEsc);
  }, [onClose]);

  // Handle export
  const handleExport = useCallback(() => {
    if (!data?.history) return;
    exportCheckHistoryToCSV(checkId, data.history);
  }, [checkId, data?.history]);

  // Calculate summary stats
  const stats = data?.history
    ? {
        total: data.history.length,
        ok: data.history.filter((h) => h.status === "ok").length,
        warning: data.history.filter((h) => h.status === "warning").length,
        critical: data.history.filter((h) => h.status === "critical").length,
      }
    : null;

  const uptimePercent = stats && stats.total > 0 ? ((stats.ok / stats.total) * 100).toFixed(1) : "100.0";

  return (
    <div
      className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      data-testid="check-detail-modal"
    >
      <div className="bg-slate-900 border border-white/10 rounded-xl max-w-2xl w-full max-h-[80vh] flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10">
          <div>
            <h2 id="modal-title" className="text-lg font-semibold text-slate-200">
              {title}
            </h2>
            {showCheckId && (
              <p className="text-xs text-slate-600 font-mono">{checkId}</p>
            )}
            <p className="text-xs text-slate-500">
              {metadata?.description || "Check History & Details"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleExport}
              disabled={!data?.history?.length}
              className="flex items-center gap-1.5 px-2 py-1 text-xs rounded border border-white/10 bg-white/5 text-slate-400 hover:bg-white/10 hover:text-slate-200 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Export history to CSV"
            >
              <Download size={14} />
              Export
            </button>
            <button
              onClick={onClose}
              className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-slate-200 transition-colors"
              aria-label="Close modal"
            >
              <X size={20} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12 text-slate-500">
              <Clock className="animate-spin mr-2" size={20} />
              Loading history...
            </div>
          ) : error ? (
            <ErrorDisplay error={error} onRetry={() => refetch()} compact />
          ) : (
            <>
              {/* Importance Notice */}
              {metadata?.importance && (
                <div className="flex items-start gap-2 p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
                  <Info size={16} className="text-blue-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm text-blue-300 font-medium">Why This Matters</p>
                    <p className="text-xs text-blue-200/80 mt-0.5">{metadata.importance}</p>
                  </div>
                </div>
              )}

              {/* Stats Summary */}
              {stats && (
                <div className="grid grid-cols-4 gap-3">
                  <div className="p-3 rounded-lg bg-white/5 text-center">
                    <div className="text-xl font-bold text-slate-200">{stats.total}</div>
                    <div className="text-xs text-slate-500">Total</div>
                  </div>
                  <div className="p-3 rounded-lg bg-emerald-500/10 text-center">
                    <div className="text-xl font-bold text-emerald-400">{stats.ok}</div>
                    <div className="text-xs text-emerald-500/80 flex items-center justify-center gap-1">
                      <CheckCircle size={12} /> OK
                    </div>
                  </div>
                  <div className="p-3 rounded-lg bg-amber-500/10 text-center">
                    <div className="text-xl font-bold text-amber-400">{stats.warning}</div>
                    <div className="text-xs text-amber-500/80 flex items-center justify-center gap-1">
                      <AlertTriangle size={12} /> Warn
                    </div>
                  </div>
                  <div className="p-3 rounded-lg bg-red-500/10 text-center">
                    <div className="text-xl font-bold text-red-400">{stats.critical}</div>
                    <div className="text-xs text-red-500/80 flex items-center justify-center gap-1">
                      <AlertCircle size={12} /> Crit
                    </div>
                  </div>
                </div>
              )}

              {/* Uptime Bar */}
              {stats && stats.total > 0 && (
                <div className="p-3 rounded-lg bg-white/5">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm text-slate-400">Uptime</span>
                    <span
                      className={`text-sm font-medium ${
                        parseFloat(uptimePercent) >= 99
                          ? "text-emerald-400"
                          : parseFloat(uptimePercent) >= 90
                          ? "text-amber-400"
                          : "text-red-400"
                      }`}
                    >
                      {uptimePercent}%
                    </span>
                  </div>
                  <div className="h-2 bg-slate-800 rounded-full overflow-hidden">
                    <div className="h-full flex">
                      <div
                        className="h-full bg-emerald-500"
                        style={{ width: `${(stats.ok / stats.total) * 100}%` }}
                      />
                      <div
                        className="h-full bg-amber-500"
                        style={{ width: `${(stats.warning / stats.total) * 100}%` }}
                      />
                      <div
                        className="h-full bg-red-500"
                        style={{ width: `${(stats.critical / stats.total) * 100}%` }}
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* History List */}
              <div className="space-y-2">
                <h3 className="text-sm font-medium text-slate-400">Recent History</h3>
                {data?.history && data.history.length > 0 ? (
                  <div className="space-y-1 max-h-64 overflow-y-auto">
                    {data.history.map((entry, idx) => (
                      <div
                        key={`${entry.timestamp}-${idx}`}
                        className="flex items-start gap-3 p-2 rounded-lg bg-white/[0.02] hover:bg-white/[0.04] transition-colors"
                      >
                        <div className="flex-shrink-0 mt-0.5">
                          <StatusIcon status={entry.status as HealthStatus} size={14} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-slate-300 truncate">{entry.message}</p>
                          <div className="flex items-center gap-2 text-xs text-slate-500">
                            <span>{formatTimestamp(entry.timestamp)}</span>
                            <span className="text-slate-600">({formatRelativeTime(entry.timestamp)})</span>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-8 text-slate-500">
                    <Clock size={32} className="mx-auto mb-2 opacity-50" />
                    <p>No history available</p>
                    <p className="text-xs mt-1">Run a health check tick to see history</p>
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-white/10 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium rounded-lg bg-white/10 text-slate-300 hover:bg-white/15 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
