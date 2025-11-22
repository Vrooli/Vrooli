import { useState, useEffect, useMemo, useCallback, useRef } from "react";
import {
  Clock,
  CheckCircle,
  XCircle,
  Loader,
  AlertCircle,
  Eye,
  Filter,
  RefreshCw,
  ChevronDown,
} from "lucide-react";
import { format, formatDistanceToNow } from "date-fns";
import { useExecutionStore } from "../stores/executionStore";
import { logger } from "../utils/logger";
import { usePopoverPosition } from "../hooks/usePopoverPosition";
import { selectors } from "../consts/selectors";

interface ExecutionHistoryProps {
  workflowId?: string;
  onSelectExecution?: (execution: ExecutionSummary) => void;
}

interface ExecutionSummary {
  id: string;
  workflow_id: string;
  workflow_name?: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  started_at: string;
  completed_at?: string;
  progress: number;
  current_step?: string;
  error?: string;
}

type StatusFilter = "all" | "completed" | "failed" | "running" | "cancelled";

const STATUS_FILTERS: StatusFilter[] = [
  "all",
  "completed",
  "failed",
  "running",
  "cancelled",
];

function ExecutionHistory({
  workflowId,
  onSelectExecution,
}: ExecutionHistoryProps) {
  const [executions, setExecutions] = useState<ExecutionSummary[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isMobileFilterOpen, setIsMobileFilterOpen] = useState(false);

  const mobileFilterContainerRef = useRef<HTMLDivElement | null>(null);
  const mobileFilterButtonRef = useRef<HTMLButtonElement | null>(null);
  const mobileFilterDropdownRef = useRef<HTMLDivElement | null>(null);

  const { floatingStyles: mobileFilterStyles } = usePopoverPosition(
    mobileFilterButtonRef,
    mobileFilterDropdownRef,
    {
      isOpen: isMobileFilterOpen,
      placementPriority: ["bottom-start", "bottom-end", "top-start", "top-end"],
      matchReferenceWidth: true,
    },
  );

  const loadExecutions = useExecutionStore((state) => state.loadExecutions);
  const storeExecutions = useExecutionStore((state) => state.executions);

  const fetchExecutions = useCallback(
    async (options?: { skipSpinner?: boolean }) => {
      if (!options?.skipSpinner) {
        setIsLoading(true);
      }
      setError(null);
      try {
        // Use the store's loadExecutions to fetch and populate the executions array
        await loadExecutions(workflowId);

        // Get the executions from the store
        const storeExecutions = useExecutionStore.getState().executions;
        setExecutions(storeExecutions as unknown as ExecutionSummary[]);
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load executions";
        setError(message);
        logger.error(
          "Failed to load execution history",
          {
            component: "ExecutionHistory",
            action: "fetchExecutions",
            workflowId,
          },
          err,
        );
      } finally {
        setIsLoading(false);
      }
    },
    [loadExecutions, workflowId],
  );

  const handleRefresh = async () => {
    setIsRefreshing(true);
    await fetchExecutions();
    setIsRefreshing(false);
  };

  useEffect(() => {
    const existing = useExecutionStore.getState()
      .executions as unknown as ExecutionSummary[];
    if (existing.length > 0) {
      setExecutions(existing);
    }
    fetchExecutions({ skipSpinner: existing.length > 0 });
  }, [fetchExecutions]);

  useEffect(() => {
    setExecutions(storeExecutions as unknown as ExecutionSummary[]);
  }, [storeExecutions]);

  useEffect(() => {
    if (!isMobileFilterOpen) {
      return;
    }

    const handleClickOutside = (event: MouseEvent | TouchEvent) => {
      if (!mobileFilterContainerRef.current) {
        return;
      }
      if (!mobileFilterContainerRef.current.contains(event.target as Node)) {
        setIsMobileFilterOpen(false);
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    document.addEventListener("touchstart", handleClickOutside);

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("touchstart", handleClickOutside);
    };
  }, [isMobileFilterOpen]);

  const filteredExecutions = useMemo(() => {
    if (statusFilter === "all") {
      return executions;
    }
    return executions.filter((exec) => exec.status === statusFilter);
  }, [executions, statusFilter]);

  const getStatusIcon = (status: ExecutionSummary["status"]) => {
    switch (status) {
      case "running":
        return <Loader size={16} className="animate-spin text-blue-400" />;
      case "completed":
        return <CheckCircle size={16} className="text-green-400" />;
      case "failed":
        return <XCircle size={16} className="text-red-400" />;
      case "cancelled":
        return <AlertCircle size={16} className="text-yellow-400" />;
      default:
        return <Clock size={16} className="text-gray-400" />;
    }
  };

  const getStatusBadge = (status: ExecutionSummary["status"]) => {
    const baseClasses = "px-2 py-1 text-xs font-medium rounded-full";
    switch (status) {
      case "running":
        return (
          <span
            className={`${baseClasses} bg-blue-500/20 text-blue-400`}
            data-testid="execution-status-running"
          >
            Running
          </span>
        );
      case "completed":
        return (
          <span
            className={`${baseClasses} bg-green-500/20 text-green-400`}
            data-testid="execution-status-completed"
          >
            Completed
          </span>
        );
      case "failed":
        return (
          <span
            className={`${baseClasses} bg-red-500/20 text-red-400`}
            data-testid="execution-status-failed"
          >
            Failed
          </span>
        );
      case "cancelled":
        return (
          <span
            className={`${baseClasses} bg-yellow-500/20 text-yellow-400`}
            data-testid="execution-status-cancelled"
          >
            Cancelled
          </span>
        );
      default:
        return (
          <span
            className={`${baseClasses} bg-gray-500/20 text-gray-400`}
            data-testid="execution-status-pending"
          >
            Pending
          </span>
        );
    }
  };

  const calculateDuration = (execution: ExecutionSummary): string => {
    if (!execution.completed_at) {
      return "In progress...";
    }

    const start = new Date(execution.started_at);
    const end = new Date(execution.completed_at);
    const durationMs = end.getTime() - start.getTime();

    if (durationMs < 1000) {
      return `${durationMs}ms`;
    } else if (durationMs < 60000) {
      return `${(durationMs / 1000).toFixed(1)}s`;
    } else {
      const minutes = Math.floor(durationMs / 60000);
      const seconds = Math.floor((durationMs % 60000) / 1000);
      return `${minutes}m ${seconds}s`;
    }
  };

  const formatTimestamp = (timestamp: string) => {
    try {
      const date = new Date(timestamp);
      const isRecent = Date.now() - date.getTime() < 24 * 60 * 60 * 1000; // Within last 24 hours

      if (isRecent) {
        return formatDistanceToNow(date, { addSuffix: true });
      }
      return format(date, "MMM d, yyyy HH:mm");
    } catch {
      return "Unknown";
    }
  };

  const statusCounts = useMemo(() => {
    return {
      all: executions.length,
      completed: executions.filter((e) => e.status === "completed").length,
      failed: executions.filter((e) => e.status === "failed").length,
      running: executions.filter((e) => e.status === "running").length,
      cancelled: executions.filter((e) => e.status === "cancelled").length,
    };
  }, [executions]);

  const formatFilterLabel = (filter: StatusFilter) =>
    filter.charAt(0).toUpperCase() + filter.slice(1);

  if (isLoading && executions.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader
            size={32}
            className="animate-spin text-flow-accent mx-auto mb-4"
          />
          <div className="text-gray-400">Loading execution history...</div>
        </div>
      </div>
    );
  }

  if (error && executions.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <AlertCircle size={48} className="text-red-400 mx-auto mb-4" />
          <h3 className="text-lg font-semibold text-white mb-2">
            Failed to Load Executions
          </h3>
          <p className="text-gray-400 mb-4">{error}</p>
          <button
            onClick={() => fetchExecutions()}
            className="px-4 py-2 bg-flow-accent text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex flex-col h-full"
      data-testid={selectors.executions.list.root}
    >
      {/* Header with filters */}
      <div className="p-4 border-b border-gray-800 space-y-2">
        {/* Desktop filters */}
        <div className="hidden md:flex items-center gap-2 flex-wrap">
          <Filter size={14} className="text-gray-500" />
          {STATUS_FILTERS.map((filter) => (
            <button
              key={filter}
              data-testid={selectors.executions.filters.filter({ filter })}
              onClick={() => setStatusFilter(filter)}
              className={`px-3 py-1.5 text-sm rounded-lg transition-colors ${
                statusFilter === filter
                  ? "bg-flow-accent text-white"
                  : "bg-flow-node text-gray-400 hover:text-white hover:bg-gray-700"
              }`}
            >
              {formatFilterLabel(filter)} ({statusCounts[filter]})
            </button>
          ))}
          <button
            data-testid={selectors.executions.list.refreshButton}
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="ml-auto p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Refresh"
          >
            <RefreshCw
              size={16}
              className={isRefreshing ? "animate-spin" : ""}
            />
          </button>
        </div>

        {/* Mobile filter dropdown */}
        <div className="flex items-center gap-2 md:hidden">
          <div className="relative flex-1" ref={mobileFilterContainerRef}>
            <button
              ref={mobileFilterButtonRef}
              type="button"
              onClick={() => setIsMobileFilterOpen((prev) => !prev)}
              className="w-full flex items-center justify-between gap-2 rounded-lg border border-gray-700 bg-flow-node px-3 py-2 text-sm text-gray-200 hover:border-flow-accent hover:text-white transition-colors"
              aria-haspopup="listbox"
              aria-expanded={isMobileFilterOpen}
            >
              <span className="flex items-center gap-2">
                <Filter size={14} />
                <span>
                  {formatFilterLabel(statusFilter)} (
                  {statusCounts[statusFilter]})
                </span>
              </span>
              <ChevronDown
                size={16}
                className={`transition-transform ${isMobileFilterOpen ? "rotate-180" : ""}`}
              />
            </button>
            {isMobileFilterOpen && (
              <div
                ref={mobileFilterDropdownRef}
                style={mobileFilterStyles}
                className="z-20 rounded-lg border border-gray-700 bg-flow-node shadow-lg overflow-hidden"
              >
                {STATUS_FILTERS.map((filter) => (
                  <button
                    key={filter}
                    type="button"
                    onClick={() => {
                      setStatusFilter(filter);
                      setIsMobileFilterOpen(false);
                    }}
                    className={`w-full flex items-center justify-between px-4 py-2 text-sm transition-colors ${
                      statusFilter === filter
                        ? "bg-flow-accent/20 text-white"
                        : "text-gray-300 hover:bg-gray-700/50 hover:text-white"
                    }`}
                  >
                    <span>{formatFilterLabel(filter)}</span>
                    <span className="text-xs text-gray-400">
                      {statusCounts[filter]}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>
          <button
            type="button"
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="ml-auto p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title="Refresh"
          >
            <RefreshCw
              size={16}
              className={isRefreshing ? "animate-spin" : ""}
            />
          </button>
        </div>
      </div>

      {/* Execution list */}
      <div
        className="flex-1 overflow-auto"
        data-testid={selectors.executions.list.list}
      >
        {filteredExecutions.length === 0 ? (
          <div className="flex items-center justify-center h-full p-8">
            <div className="text-center">
              <Clock size={48} className="text-gray-600 mx-auto mb-4" />
              <h3 className="text-lg font-semibold text-white mb-2">
                {statusFilter === "all"
                  ? "No Executions Yet"
                  : `No ${statusFilter} executions`}
              </h3>
              <p className="text-gray-400">
                {statusFilter === "all"
                  ? "Execute workflows to see their history here"
                  : `Change the filter to see other executions`}
              </p>
            </div>
          </div>
        ) : (
          <div className="divide-y divide-gray-800">
            {filteredExecutions.map((execution) => (
              <div
                key={execution.id}
                data-testid={selectors.executions.list.item}
                data-execution-id={execution.id}
                data-execution-status={execution.status}
                onClick={() => onSelectExecution?.(execution)}
                className={`p-4 cursor-pointer transition-all hover:bg-flow-node ${
                  onSelectExecution
                    ? "hover:border-l-4 hover:border-flow-accent"
                    : ""
                }`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3 flex-1 min-w-0">
                    <div className="mt-0.5">
                      {getStatusIcon(execution.status)}
                    </div>
                    <div className="flex-1 min-w-0">
                      {/* Execution ID */}
                      <div className="flex items-center gap-2 mb-1">
                        <span className="text-sm font-medium text-white">
                          #{execution.id.slice(0, 8)}
                        </span>
                        {getStatusBadge(execution.status)}
                      </div>

                      {/* Workflow name (if available) */}
                      {execution.workflow_name && (
                        <div className="text-sm text-gray-400 mb-1">
                          {execution.workflow_name}
                        </div>
                      )}

                      {/* Current step or error */}
                      {execution.status === "running" &&
                        execution.current_step && (
                          <div className="text-xs text-blue-400 mb-1">
                            Current: {execution.current_step}
                          </div>
                        )}
                      {execution.status === "failed" && execution.error && (
                        <div
                          className="text-xs text-red-400 mb-1 truncate"
                          title={execution.error}
                        >
                          Error: {execution.error}
                        </div>
                      )}

                      {/* Timestamp and duration */}
                      <div className="flex items-center gap-4 text-xs text-gray-500">
                        <div className="flex items-center gap-1">
                          <Clock size={12} />
                          <span>{formatTimestamp(execution.started_at)}</span>
                        </div>
                        <span>Duration: {calculateDuration(execution)}</span>
                      </div>

                      {/* Progress bar for running executions */}
                      {execution.status === "running" && (
                        <div className="mt-2 w-full bg-gray-700 rounded-full h-1.5">
                          <div
                            className="bg-flow-accent h-1.5 rounded-full transition-all duration-300"
                            style={{ width: `${execution.progress || 0}%` }}
                          />
                        </div>
                      )}
                    </div>
                  </div>

                  {/* View button */}
                  {onSelectExecution && (
                    <button
                      data-testid={selectors.executions.actions.viewButton}
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectExecution(execution);
                      }}
                      className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                      title="View execution details"
                    >
                      <Eye size={14} />
                      <span>View</span>
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export default ExecutionHistory;
