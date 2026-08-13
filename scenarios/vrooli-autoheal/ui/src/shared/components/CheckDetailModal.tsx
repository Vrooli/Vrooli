// Check detail modal for drill-down view
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { memo, Profiler, useState, useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { X, Download, Clock, AlertCircle, CheckCircle, AlertTriangle, Info, BookOpen, CheckCircle2, XCircle, Zap, Loader2 } from "lucide-react";
import {
  fetchCheckHistory, type HealthStatus, type HistoryEntry, type SubCheck, type CheckHistoryResponse,
  fetchConfig, fetchDefaults, setCheckAutoHeal, fetchCheckActions, executeAction,
  type ActionResult, type RecoveryAction, normalizeHealthStatus
} from "../../lib/api";
import { formatRelativeTime } from "../../lib/utils";
import { CodePreview } from "./CodePreview";
import { ErrorDisplay } from "./ErrorDisplay";
import { StatusIcon } from "./StatusIcon";
import { StatusSparkline } from "./StatusSparkline";
import { ActionButtons } from "./ActionButtons";
import { exportCheckHistoryToCSV } from "../../lib/export";
import { navigateToCheckDocs } from "../../lib/docs";
import { onProfilerRender } from "../../lib/profiler";
import { useCheckMetadata } from "../contexts/CheckMetadataContext";
import { useEscapeKey } from "../../hooks/useEscapeKey";
import { Notice, NoticeTitle, TabTrigger } from "../ui/composites";
import { Button, ModalContent, ModalOverlay, Switch } from "../ui/primitives";

interface CheckDetailModalProps {
  checkId: string;
  onClose: () => void;
}

type TabId = "details" | "history";
const HISTORY_VISIBLE_STEP = 80;

function isSubCheck(value: unknown): value is SubCheck {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return typeof record.name === "string" && typeof record.passed === "boolean";
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
  const colorClass = subCheck.passed ? "text-accent-success" : "text-accent-danger";

  return (
    <div className="flex items-center gap-2 text-xs">
      <Icon size={12} className={colorClass} />
      <span className={subCheck.passed ? "text-text-muted" : "text-text-primary"}>
        {subCheck.name}
      </span>
      {subCheck.detail && (
        <span className="text-text-muted/80">- {subCheck.detail}</span>
      )}
    </div>
  );
}

interface HistoryRowView {
  key: string;
  message: string;
  status: HealthStatus;
  timestampLabel: string;
  relativeLabel: string;
}

const HistoryRow = memo(function HistoryRow({ entry }: { entry: HistoryRowView }) {
  return (
    <div className="flex items-start gap-3 rounded-lg bg-surface-overlay/30 p-2 transition-colors hover:bg-surface-overlay/50">
      <div className="flex-shrink-0 mt-0.5">
        <StatusIcon status={entry.status} size={14} />
      </div>
      <div className="flex-1 min-w-0">
        <p className="truncate text-sm text-text-primary">{entry.message}</p>
        <div className="flex items-center gap-2 text-xs text-text-muted">
          <span>{entry.timestampLabel}</span>
          <span className="text-text-muted/80">({entry.relativeLabel})</span>
        </div>
      </div>
    </div>
  );
});

function CheckDetailModalImpl({ checkId, onClose }: CheckDetailModalProps) {
  const { getTitle, getMetadata } = useCheckMetadata();
  const queryClient = useQueryClient();
  const metadata = getMetadata(checkId);
  const title = getTitle(checkId);
  const showCheckId = title !== checkId;
  const [activeTab, setActiveTab] = useState<TabId>("details");
  const [historyVisibleCount, setHistoryVisibleCount] = useState(HISTORY_VISIBLE_STEP);
  const [autoHealResult, setAutoHealResult] = useState<ActionResult | null>(null);
  const [confirmHealAction, setConfirmHealAction] = useState<RecoveryAction | null>(null);

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

  // Find the primary healing action - prefer actual healing actions over diagnostic ones
  const primaryHealAction = useMemo(() => {
    if (!actionsData?.actions) return null;
    const available = actionsData.actions.filter((a) => a.available);

    // Prefer restart action (most common healing action for resources)
    const restart = available.find((a) => a.id === "restart");
    if (restart) return restart;

    // Then prefer actual healing actions over diagnostic/informational ones
    // Diagnostic actions typically just show information, not fix problems
    const diagnosticIds = ["list", "logs", "status", "diagnose", "info"];
    const healingActions = available.filter((a) => !diagnosticIds.includes(a.id));
    if (healingActions.length > 0) return healingActions[0];

    // Fallback to first available (excluding logs which is purely informational)
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

  useEscapeKey(onClose);

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
    return data.history.slice(0, 24).map((h) => normalizeHealthStatus(h.status, "ok"));
  }, [data?.history]);

  const visibleHistory = useMemo<HistoryRowView[]>(() => {
    if (!data?.history) return [];
    return data.history.slice(0, historyVisibleCount).map((entry, idx) => ({
      key: `${entry.timestamp}-${idx}`,
      message: entry.message,
      status: normalizeHealthStatus(entry.status, "ok"),
      timestampLabel: formatTimestamp(entry.timestamp),
      relativeLabel: formatRelativeTime(entry.timestamp),
    }));
  }, [data?.history, historyVisibleCount]);

  const remainingHistoryCount = Math.max((data?.history?.length ?? 0) - visibleHistory.length, 0);
  const showMoreHistory = useCallback(() => {
    setHistoryVisibleCount((count) => count + HISTORY_VISIBLE_STEP);
  }, []);

  // Get latest entry for details
  const latestEntry: HistoryEntry | undefined = data?.history?.[0];
  const parsedSubChecks = useMemo<SubCheck[]>(() => {
    const subChecksValue = latestEntry?.details?.subChecks;
    if (!Array.isArray(subChecksValue)) return [];
    return subChecksValue.filter(isSubCheck);
  }, [latestEntry?.details?.subChecks]);

  const uptimePercent = stats && stats.total > 0 ? ((stats.ok / stats.total) * 100).toFixed(1) : "100.0";

  // Check if this check has details or sub-checks
  const hasDetails = latestEntry?.details && Object.keys(latestEntry.details).length > 0;
  const hasSubChecks = parsedSubChecks.length > 0;

  return (
    <ModalOverlay
      onDismiss={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
      data-testid="check-detail-modal"
    >
      <ModalContent size="md" className="max-h-[92vh] sm:max-h-[85vh]">
        {/* Header */}
        <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border-default/70 p-3 sm:p-4">
          <div className="min-w-0">
            <h2 id="modal-title" className="text-lg font-semibold text-text-primary">
              {title}
            </h2>
            {showCheckId && (
              <p className="font-mono text-xs text-text-muted/70">{checkId}</p>
            )}
            <p className="text-xs text-text-muted">
              {metadata?.description || "Check History & Details"}
            </p>
          </div>
          <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:justify-end">
            <Button
              onClick={() => {
                onClose();
                navigateToCheckDocs(checkId);
              }}
              variant="outline"
              size="sm"
              className="gap-1.5 text-xs text-text-muted hover:border-accent-primary/30 hover:bg-accent-primary/20 hover:text-accent-primary"
              title="View documentation for this check"
              data-testid="modal-learn-more"
            >
              <BookOpen size={14} />
              <span className="hidden sm:inline">Docs</span>
            </Button>
            <Button
              onClick={handleExport}
              disabled={!data?.history?.length}
              variant="outline"
              size="sm"
              className="gap-1.5 text-xs text-text-muted hover:bg-surface-overlay/70 hover:text-text-primary"
              title="Export history to CSV"
            >
              <Download size={14} />
              <span className="hidden sm:inline">Export</span>
            </Button>
            <Button
              onClick={onClose}
              variant="outline"
              size="icon"
              className="text-text-muted hover:bg-surface-overlay/70 hover:text-text-primary"
              aria-label="Close modal"
            >
              <X size={20} />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 space-y-4 overflow-y-auto p-3 sm:p-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-12 text-text-muted">
              <Clock className="mr-2 animate-spin" size={20} />
              Loading history...
            </div>
          ) : error ? (
            <ErrorDisplay error={error} onRetry={() => refetch()} compact />
          ) : (
            <>
              {/* Importance Notice */}
              {metadata?.importance && (
                <Notice tone="info" className="flex items-start gap-2">
                  <Info size={16} className="mt-0.5 shrink-0 text-accent-primary" />
                  <div>
                    <NoticeTitle tone="info">Why This Matters</NoticeTitle>
                    <p className="mt-0.5 text-xs text-text-muted">{metadata.importance}</p>
                  </div>
                </Notice>
              )}

              {/* Auto-Heal Controls */}
              <div className="rounded-lg border border-border-default/70 bg-surface-overlay/40 p-3">
                <div className="flex items-center gap-3">
                  <Zap size={18} className={autoHealEnabled ? "text-accent-primary" : "text-text-muted"} />
                  <div>
                    <p className="text-sm font-medium text-text-primary">Auto-Heal</p>
                    <p className="text-xs text-text-muted">
                      {autoHealEnabled
                        ? "Automatically recover when unhealthy"
                        : "Manual intervention required when unhealthy"}
                    </p>
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap items-center gap-3 sm:mt-2 sm:justify-end">
                  {/* Auto-Heal Toggle */}
                  <Switch
                    checked={autoHealEnabled && checkEnabled}
                    onCheckedChange={(checked) => toggleAutoHealMutation.mutate(checked)}
                    disabled={toggleAutoHealMutation.isPending || !checkEnabled}
                    title={!checkEnabled ? "Enable check first to use auto-heal" : (autoHealEnabled ? "Disable auto-heal" : "Enable auto-heal")}
                    tone="primary"
                  />

                  {/* Heal Now Button */}
                  {primaryHealAction && (
                    <button
                      onClick={() => {
                        if (primaryHealAction.dangerous) {
                          setConfirmHealAction(primaryHealAction);
                        } else {
                          executeHealMutation.mutate(primaryHealAction.id);
                        }
                      }}
                      disabled={executeHealMutation.isPending}
                      className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
                        primaryHealAction.dangerous
                          ? "border-accent-warning/30 bg-accent-warning/20 text-accent-warning hover:bg-accent-warning/30"
                          : "border-accent-primary/30 bg-accent-primary/20 text-accent-primary hover:bg-accent-primary/30"
                      }`}
                      title={`Run ${primaryHealAction.name} now${primaryHealAction.dangerous ? " (requires confirmation)" : ""}`}
                    >
                      {executeHealMutation.isPending ? (
                        <Loader2 size={12} className="animate-spin" />
                      ) : primaryHealAction.dangerous ? (
                        <AlertTriangle size={12} />
                      ) : (
                        <Zap size={12} />
                      )}
                      Heal Now
                    </button>
                  )}
                </div>
              </div>

              {/* Confirmation Dialog for Dangerous Actions */}
              {confirmHealAction && (
                <Notice tone="warning">
                  <div className="flex items-start gap-2">
                    <AlertTriangle size={16} className="mt-0.5 shrink-0 text-accent-warning" />
                    <div className="flex-1">
                      <NoticeTitle tone="warning">Confirm Action</NoticeTitle>
                      <p className="mt-1 text-xs text-text-muted">
                        Are you sure you want to run <strong>{confirmHealAction.name}</strong>?
                        {confirmHealAction.description && (
                          <span className="mt-1 block text-text-muted/80">{confirmHealAction.description}</span>
                        )}
                      </p>
                      <div className="mt-3 flex gap-2">
                        <Button
                          onClick={() => {
                            executeHealMutation.mutate(confirmHealAction.id);
                            setConfirmHealAction(null);
                          }}
                          disabled={executeHealMutation.isPending}
                          size="sm"
                          className="border-accent-warning bg-accent-warning text-text-inverse hover:bg-accent-warning/80"
                        >
                          {executeHealMutation.isPending ? "Executing..." : "Confirm"}
                        </Button>
                        <Button
                          onClick={() => setConfirmHealAction(null)}
                          disabled={executeHealMutation.isPending}
                          size="sm"
                          variant="outline"
                        >
                          Cancel
                        </Button>
                      </div>
                    </div>
                  </div>
                </Notice>
              )}

              {/* Auto-Heal Result */}
              {autoHealResult && (
                <Notice tone={autoHealResult.success ? "success" : "danger"}>
                  <div className="flex items-start gap-2">
                    {autoHealResult.success ? (
                      <CheckCircle2 size={16} className="mt-0.5 shrink-0 text-accent-success" />
                    ) : (
                      <XCircle size={16} className="mt-0.5 shrink-0 text-accent-danger" />
                    )}
                    <div className="flex-1 min-w-0">
                      <p className={`text-sm font-medium ${autoHealResult.success ? "text-accent-success" : "text-accent-danger"}`}>
                        {autoHealResult.message}
                      </p>
                      {autoHealResult.output && (
                        <CodePreview code={autoHealResult.output} language="text" maxHeight="6rem" className="mt-2" />
                      )}
                      {autoHealResult.error && (
                        <p className="mt-1 text-xs text-accent-danger">{autoHealResult.error}</p>
                      )}
                      <button
                        onClick={() => setAutoHealResult(null)}
                        className="mt-2 text-xs text-text-muted transition-colors hover:text-text-primary"
                      >
                        Dismiss
                      </button>
                    </div>
                  </div>
                </Notice>
              )}

              {/* Stats Summary */}
              {stats && (
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div className="rounded-lg bg-surface-overlay/40 p-3 text-center">
                    <div className="text-xl font-bold text-text-primary">{stats.total}</div>
                    <div className="text-xs text-text-muted">Total</div>
                  </div>
                  <div className="rounded-lg bg-accent-success/10 p-3 text-center">
                    <div className="text-xl font-bold text-accent-success">{stats.ok}</div>
                    <div className="flex items-center justify-center gap-1 text-xs text-accent-success">
                      <CheckCircle size={12} /> OK
                    </div>
                  </div>
                  <div className="rounded-lg bg-accent-warning/10 p-3 text-center">
                    <div className="text-xl font-bold text-accent-warning">{stats.warning}</div>
                    <div className="flex items-center justify-center gap-1 text-xs text-accent-warning">
                      <AlertTriangle size={12} /> Warn
                    </div>
                  </div>
                  <div className="rounded-lg bg-accent-danger/10 p-3 text-center">
                    <div className="text-xl font-bold text-accent-danger">{stats.critical}</div>
                    <div className="flex items-center justify-center gap-1 text-xs text-accent-danger">
                      <AlertCircle size={12} /> Crit
                    </div>
                  </div>
                </div>
              )}

              {/* Uptime with Sparkline */}
              {stats && stats.total > 0 && (
                <div className="rounded-lg bg-surface-overlay/40 p-3">
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex flex-wrap items-center gap-2 sm:gap-3">
                      <span className="text-sm text-text-muted">Uptime</span>
                      <span
                        className={`text-lg font-semibold ${
                          parseFloat(uptimePercent) >= 99
                            ? "text-accent-success"
                            : parseFloat(uptimePercent) >= 90
                            ? "text-accent-warning"
                            : "text-accent-danger"
                        }`}
                      >
                        {uptimePercent}%
                      </span>
                      {metadata?.intervalSeconds && (
                        <span className="flex items-center gap-1 text-xs text-text-muted">
                          <Clock size={10} />
                          every {formatInterval(metadata.intervalSeconds)}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2 overflow-x-auto">
                      <span className="text-xs text-text-muted">Recent:</span>
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
              <div className="flex items-center gap-1 overflow-x-auto border-b border-border-default/70">
                <TabTrigger
                  onClick={() => setActiveTab("details")}
                  active={activeTab === "details"}
                  className="shrink-0"
                >
                  Details
                </TabTrigger>
                <TabTrigger
                  onClick={() => {
                    setActiveTab("history");
                    setHistoryVisibleCount(HISTORY_VISIBLE_STEP);
                  }}
                  active={activeTab === "history"}
                  className="shrink-0"
                >
                  History ({data?.count || 0})
                </TabTrigger>
              </div>

              {/* Tab Content */}
              {activeTab === "details" && (
                <div className="space-y-4">
                  {/* Latest status message */}
                  {latestEntry ? (
                    <div className="rounded-lg bg-surface-overlay/30 p-3">
                      <div className="flex items-start gap-2">
                        <StatusIcon status={normalizeHealthStatus(latestEntry.status, "ok")} size={16} />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-text-primary">{latestEntry.message}</p>
                          <p className="mt-1 text-xs text-text-muted">
                            {formatTimestamp(latestEntry.timestamp)} ({formatRelativeTime(latestEntry.timestamp)})
                          </p>
                        </div>
                      </div>
                    </div>
                  ) : null}

                  {/* Sub-checks */}
                  {hasSubChecks && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-text-muted">Sub-checks</h4>
                      <div className="space-y-1.5 rounded-lg bg-surface-overlay/30 p-3">
                        {parsedSubChecks.map((sc, idx) => (
                          <SubCheckRow key={idx} subCheck={sc} />
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Raw details */}
                  {hasDetails && !hasSubChecks && (
                    <div className="space-y-2">
                      <h4 className="text-sm font-medium text-text-muted">Raw Details</h4>
                      <CodePreview code={latestEntry?.details} language="json" />
                    </div>
                  )}

                  {/* No details available */}
                  {!hasDetails && !hasSubChecks && !latestEntry && (
                    <div className="py-8 text-center text-text-muted">
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
                    <div className="max-h-80 space-y-1 overflow-y-auto">
                      {visibleHistory.map((entry) => (
                        <HistoryRow key={entry.key} entry={entry} />
                      ))}
                      {remainingHistoryCount > 0 && (
                        <Button
                          onClick={showMoreHistory}
                          size="sm"
                          variant="outline"
                          className="mt-2 w-full justify-center text-xs"
                        >
                          Show more ({remainingHistoryCount} remaining)
                        </Button>
                      )}
                    </div>
                  ) : (
                    <div className="py-8 text-center text-text-muted">
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
        <div className="flex justify-end border-t border-border-default/70 p-4">
          <Button
            onClick={onClose}
            variant="outline"
          >
            Close
          </Button>
        </div>
      </ModalContent>
    </ModalOverlay>
  );
}

export function CheckDetailModal(props: CheckDetailModalProps) {
  return (
    <Profiler id="CheckDetailModal" onRender={onProfilerRender}>
      <CheckDetailModalImpl {...props} />
    </Profiler>
  );
}
