// Check detail modal for drill-down view
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { useState, useEffect, useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X, Download, Clock, AlertCircle, CheckCircle, AlertTriangle, Info, BookOpen, CheckCircle2, XCircle, Zap, Loader2 } from "lucide-react";
import {
  fetchCheckHistory, HealthStatus, type HistoryEntry, type SubCheck, type CheckHistoryResponse,
  fetchConfig, fetchDefaults, setCheckAutoHeal, fetchCheckActions, executeAction,
  type ActionResult
} from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";
import { StatusIcon } from "./StatusIcon";
import { StatusSparkline } from "./StatusSparkline";
import { ActionButtons } from "./ActionButtons";
import { exportCheckHistoryToCSV } from "../lib/export";
import { navigateToCheckDocs } from "../lib/docs";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";

interface CheckDetailModalProps {
  checkId: string;
  onClose: () => void;
}

type TabId = "details" | "history";

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

function formatInterval(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  return `${hours}h`;
}

// Sub-check row component
function SubCheckRow({ subCheck }: { subCheck: SubCheck }) {
  const Icon = subCheck.passed ? CheckCircle2 : XCircle;
  const colorClass = subCheck.passed ? "text-emerald-400" : "text-red-400";

  return (
    <div className="flex items-center gap-2 text-xs">
      <Icon size={12} className={colorClass} />
      <span className={subCheck.passed ? "text-slate-400" : "text-slate-300"}>
        {subCheck.name}
      </span>
      {subCheck.detail && (
        <span className="text-slate-500">- {subCheck.detail}</span>
      )}
    </div>
  );
}

export function CheckDetailModal({ checkId, onClose }: CheckDetailModalProps) {
  const { getTitle, getMetadata } = useCheckMetadata();
  const queryClient = useQueryClient();
  const metadata = getMetadata(checkId);
  const title = getTitle(checkId);
  const showCheckId = title !== checkId;
  const [activeTab, setActiveTab] = useState<TabId>("details");
  const [autoHealResult, setAutoHealResult] = useState<ActionResult | null>(null);

  const { data, isLoading, error, refetch } = useQuery<CheckHistoryResponse>({
    queryKey: ["check-history", checkId],
    queryFn: () => fetchCheckHistory(checkId),
    refetchInterval: 30000,
  });

  // Fetch config to get current auto-heal state
  const { data: config } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
    staleTime: 30000,
  });

  // Fetch defaults to know the default auto-heal state
  const { data: defaults } = useQuery({
    queryKey: ["config-defaults"],
    queryFn: fetchDefaults,
    staleTime: 60000,
  });

  // Fetch available recovery actions for the "Heal Now" button
  const { data: actionsData } = useQuery({
    queryKey: ["check-actions", checkId],
    queryFn: () => fetchCheckActions(checkId),
    staleTime: 30000,
  });

  // Determine current auto-heal state
  const autoHealEnabled = useMemo(() => {
    const configCheck = config?.checks?.[checkId];
    const defaultCheck = defaults?.checks?.[checkId];
    return configCheck?.autoHeal ?? defaultCheck?.autoHeal ?? false;
  }, [config, defaults, checkId]);

  // Determine if check is enabled
  const checkEnabled = useMemo(() => {
    const configCheck = config?.checks?.[checkId];
    const defaultCheck = defaults?.checks?.[checkId];
    return configCheck?.enabled ?? defaultCheck?.enabled ?? true;
  }, [config, defaults, checkId]);

  // Mutation for toggling auto-heal
  const toggleAutoHealMutation = useMutation({
    mutationFn: (autoHeal: boolean) => setCheckAutoHeal(checkId, autoHeal),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config"] });
    },
  });

  // Find the primary healing action (restart for resources, first available otherwise)
  const primaryHealAction = useMemo(() => {
    if (!actionsData?.actions) return null;
    const available = actionsData.actions.filter((a) => a.available);
    const restart = available.find((a) => a.id === "restart");
    if (restart) return restart;
    return available.find((a) => a.id !== "logs") || null;
  }, [actionsData]);

  // Mutation for executing auto-heal action
  const executeHealMutation = useMutation({
    mutationFn: (actionId: string) => executeAction(checkId, actionId),
    onSuccess: (result) => {
      setAutoHealResult(result);
      queryClient.invalidateQueries({ queryKey: ["status"] });
      queryClient.invalidateQueries({ queryKey: ["check-actions", checkId] });
      queryClient.invalidateQueries({ queryKey: ["check-history", checkId] });
      queryClient.invalidateQueries({ queryKey: ["action-history"] });
    },
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
  const stats = useMemo(() => {
    if (!data?.history) return null;
    return {
      total: data.history.length,
      ok: data.history.filter((h) => h.status === "ok").length,
      warning: data.history.filter((h) => h.status === "warning").length,
      critical: data.history.filter((h) => h.status === "critical").length,
    };
  }, [data?.history]);

  // Get recent statuses for sparkline
  const recentStatuses = useMemo(() => {
    if (!data?.history) return [];
    return data.history.slice(0, 24).map((h) => h.status as HealthStatus);
  }, [data?.history]);

  // Get latest entry for details - explicitly typed to satisfy TS
  const latestEntry = (data?.history?.[0] ?? undefined) as HistoryEntry | undefined;

  const uptimePercent = stats && stats.total > 0 ? ((stats.ok / stats.total) * 100).toFixed(1) : "100.0";

  // Check if this check has details or sub-checks
  const hasDetails = latestEntry?.details && Object.keys(latestEntry.details).length > 0;
  const hasSubChecks = latestEntry?.details?.subChecks && Array.isArray(latestEntry.details.subChecks);

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
      <div className="bg-slate-900 border border-white/10 rounded-xl max-w-2xl w-full max-h-[85vh] flex flex-col shadow-2xl">
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
              onClick={() => {
                onClose();
                navigateToCheckDocs(checkId);
              }}
              className="flex items-center gap-1.5 px-2 py-1 text-xs rounded border border-white/10 bg-white/5 text-slate-400 hover:bg-blue-500/20 hover:text-blue-400 hover:border-blue-500/30 transition-colors"
              title="View documentation for this check"
              data-testid="modal-learn-more"
            >
              <BookOpen size={14} />
              Docs
            </button>
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

              {/* Auto-Heal Controls */}
              <div className="flex items-center justify-between p-3 rounded-lg bg-white/5 border border-white/10">
                <div className="flex items-center gap-3">
                  <Zap size={18} className={autoHealEnabled ? "text-blue-400" : "text-slate-500"} />
                  <div>
                    <p className="text-sm font-medium text-slate-200">Auto-Heal</p>
                    <p className="text-xs text-slate-500">
                      {autoHealEnabled
                        ? "Automatically recover when unhealthy"
                        : "Manual intervention required when unhealthy"}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {/* Auto-Heal Toggle */}
                  <button
                    onClick={() => toggleAutoHealMutation.mutate(!autoHealEnabled)}
                    disabled={toggleAutoHealMutation.isPending || !checkEnabled}
                    title={!checkEnabled ? "Enable check first to use auto-heal" : (autoHealEnabled ? "Disable auto-heal" : "Enable auto-heal")}
                    className={`relative w-12 h-6 rounded-full transition-colors ${
                      autoHealEnabled && checkEnabled ? "bg-blue-500" : "bg-slate-600"
                    } ${!checkEnabled ? "opacity-50 cursor-not-allowed" : ""}`}
                  >
                    {toggleAutoHealMutation.isPending ? (
                      <div className="absolute inset-0 flex items-center justify-center">
                        <Loader2 size={14} className="animate-spin text-white" />
                      </div>
                    ) : (
                      <span
                        className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
                          autoHealEnabled && checkEnabled ? "translate-x-6" : ""
                        }`}
                      />
                    )}
                  </button>

                  {/* Heal Now Button */}
                  {primaryHealAction && (
                    <button
                      onClick={() => executeHealMutation.mutate(primaryHealAction.id)}
                      disabled={executeHealMutation.isPending}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg bg-blue-500/20 text-blue-400 border border-blue-500/30 hover:bg-blue-500/30 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      title={`Run ${primaryHealAction.name} now`}
                    >
                      {executeHealMutation.isPending ? (
                        <Loader2 size={12} className="animate-spin" />
                      ) : (
                        <Zap size={12} />
                      )}
                      Heal Now
                    </button>
                  )}
                </div>
              </div>

              {/* Auto-Heal Result */}
              {autoHealResult && (
                <div className={`p-3 rounded-lg border ${
                  autoHealResult.success
                    ? "bg-emerald-500/10 border-emerald-500/20"
                    : "bg-red-500/10 border-red-500/20"
                }`}>
                  <div className="flex items-start gap-2">
                    {autoHealResult.success ? (
                      <CheckCircle2 size={16} className="text-emerald-400 mt-0.5 flex-shrink-0" />
                    ) : (
                      <XCircle size={16} className="text-red-400 mt-0.5 flex-shrink-0" />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className={`text-sm font-medium ${autoHealResult.success ? "text-emerald-300" : "text-red-300"}`}>
                        {autoHealResult.message}
                      </p>
                      {autoHealResult.output && (
                        <pre className="mt-2 p-2 text-xs font-mono bg-black/30 rounded overflow-x-auto whitespace-pre-wrap max-h-24 overflow-y-auto text-slate-400">
                          {autoHealResult.output}
                        </pre>
                      )}
                      {autoHealResult.error && (
                        <p className="mt-1 text-xs text-red-400">{autoHealResult.error}</p>
                      )}
                      <button
                        onClick={() => setAutoHealResult(null)}
                        className="mt-2 text-xs text-slate-500 hover:text-slate-300"
                      >
                        Dismiss
                      </button>
                    </div>
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

              {/* Uptime with Sparkline */}
              {stats && stats.total > 0 && (
                <div className="p-3 rounded-lg bg-white/5">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-sm text-slate-400">Uptime</span>
                      <span
                        className={`text-lg font-semibold ${
                          parseFloat(uptimePercent) >= 99
                            ? "text-emerald-400"
                            : parseFloat(uptimePercent) >= 90
                            ? "text-amber-400"
                            : "text-red-400"
                        }`}
                      >
                        {uptimePercent}%
                      </span>
                      {metadata?.intervalSeconds && (
                        <span className="text-xs text-slate-500 flex items-center gap-1">
                          <Clock size={10} />
                          every {formatInterval(metadata.intervalSeconds)}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-slate-500">Recent:</span>
                      <StatusSparkline statuses={recentStatuses} maxBars={24} barHeight={20} />
                    </div>
                  </div>
                </div>
              )}

              {/* Actions (for resource checks) */}
              {metadata?.category && (
                <ActionButtons checkId={checkId} category={metadata.category} />
              )}

              {/* Tab Navigation */}
              <div className="flex items-center gap-1 border-b border-white/10">
                <button
                  onClick={() => setActiveTab("details")}
                  className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                    activeTab === "details"
                      ? "border-blue-400 text-blue-400"
                      : "border-transparent text-slate-500 hover:text-slate-300"
                  }`}
                >
                  Details
                </button>
                <button
                  onClick={() => setActiveTab("history")}
                  className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                    activeTab === "history"
                      ? "border-blue-400 text-blue-400"
                      : "border-transparent text-slate-500 hover:text-slate-300"
                  }`}
                >
                  History ({data?.count || 0})
                </button>
              </div>

              {/* Tab Content */}
              {activeTab === "details" && (
                <div className="space-y-4">
                  {/* Latest status message */}
                  {latestEntry ? (
                    <div className="p-3 rounded-lg bg-white/[0.03]">
                      <div className="flex items-start gap-2">
                        <StatusIcon status={latestEntry.status as HealthStatus} size={16} />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-slate-300">{latestEntry.message}</p>
                          <p className="text-xs text-slate-500 mt-1">
                            {formatTimestamp(latestEntry.timestamp)} ({formatRelativeTime(latestEntry.timestamp)})
                          </p>
                        </div>
                      </div>
                    </div>
                  ) : null}

                  {/* Sub-checks */}
                  {hasSubChecks && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-slate-400">Sub-checks</h4>
                      <div className="p-3 rounded-lg bg-white/[0.03] space-y-1.5">
                        {(latestEntry?.details?.subChecks as SubCheck[]).map((sc, idx) => (
                          <SubCheckRow key={idx} subCheck={sc} />
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Raw details */}
                  {hasDetails && !hasSubChecks && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-slate-400">Raw Details</h4>
                      <div className="p-3 rounded-lg bg-black/30 text-xs font-mono">
                        <pre className="overflow-x-auto whitespace-pre-wrap text-slate-400">
                          {JSON.stringify(latestEntry?.details, null, 2)}
                        </pre>
                      </div>
                    </div>
                  )}

                  {/* No details available */}
                  {!hasDetails && !hasSubChecks && !latestEntry && (
                    <div className="text-center py-8 text-slate-500">
                      <Info size={32} className="mx-auto mb-2 opacity-50" />
                      <p>No details available</p>
                      <p className="text-xs mt-1">Run a health check tick to see details</p>
                    </div>
                  )}
                </div>
              )}

              {activeTab === "history" && (
                <div className="space-y-2">
                  {data?.history && data.history.length > 0 ? (
                    <div className="space-y-1 max-h-80 overflow-y-auto">
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
              )}
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
