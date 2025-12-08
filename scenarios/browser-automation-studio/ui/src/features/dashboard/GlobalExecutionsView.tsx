import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { Search, X, ChevronLeft, Clock, Filter, RefreshCw, CheckCircle, XCircle, Loader, Ban, AlertCircle } from 'lucide-react';
import { getConfig } from '../../config';
import { logger } from '../../utils/logger';
import toast from 'react-hot-toast';
import { parseProjectList } from '../../utils/projectProto';

type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

interface ExecutionItem {
  id: string;
  workflowId: string;
  workflowName: string;
  projectId?: string;
  projectName?: string;
  status: ExecutionStatus;
  startedAt: Date;
  completedAt?: Date;
  duration?: number;
  error?: string;
}

interface GlobalExecutionsViewProps {
  onBack: () => void;
  onViewExecution: (executionId: string, workflowId: string) => void;
}

const formatRelativeTime = (date: Date): string => {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
};

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
};

type StatusFilter = 'all' | ExecutionStatus;

const StatusIcon: React.FC<{ status: ExecutionStatus; size?: number }> = ({ status, size = 16 }) => {
  switch (status) {
    case 'completed':
      return <CheckCircle size={size} className="text-green-400" />;
    case 'failed':
      return <XCircle size={size} className="text-red-400" />;
    case 'running':
      return <Loader size={size} className="text-blue-400 animate-spin" />;
    case 'pending':
      return <Clock size={size} className="text-yellow-400" />;
    case 'cancelled':
      return <Ban size={size} className="text-gray-400" />;
    default:
      return <AlertCircle size={size} className="text-gray-400" />;
  }
};

const StatusBadge: React.FC<{ status: ExecutionStatus }> = ({ status }) => {
  const config = {
    completed: { bg: 'bg-green-500/20', text: 'text-green-400', label: 'Completed' },
    failed: { bg: 'bg-red-500/20', text: 'text-red-400', label: 'Failed' },
    running: { bg: 'bg-blue-500/20', text: 'text-blue-400', label: 'Running' },
    pending: { bg: 'bg-yellow-500/20', text: 'text-yellow-400', label: 'Pending' },
    cancelled: { bg: 'bg-gray-500/20', text: 'text-gray-400', label: 'Cancelled' },
  };
  const { bg, text, label } = config[status] ?? config.cancelled;

  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${bg} ${text}`}>
      <StatusIcon status={status} size={12} />
      {label}
    </span>
  );
};

export const GlobalExecutionsView: React.FC<GlobalExecutionsViewProps> = ({
  onBack,
  onViewExecution,
}) => {
  const [executions, setExecutions] = useState<ExecutionItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [showFilters, setShowFilters] = useState(false);

  const fetchAllExecutions = useCallback(async (refresh = false) => {
    if (refresh) {
      setIsRefreshing(true);
    } else {
      setIsLoading(true);
    }

    try {
      const config = await getConfig();

      // Fetch projects for names
      const projectsResponse = await fetch(`${config.API_URL}/projects`);
      const projectsData = await projectsResponse.json();
      const projects = parseProjectList(projectsData);
      const projectsMap = new Map<string, string>();
      projects.forEach((p) => projectsMap.set(p.id, p.name));

      // Fetch workflows for names
      const workflowsResponse = await fetch(`${config.API_URL}/workflows?limit=500`);
      const workflowsData = await workflowsResponse.json();
      const workflowsMap = new Map<string, { name: string; projectId?: string; projectName?: string }>();
      if (Array.isArray(workflowsData.workflows)) {
        workflowsData.workflows.forEach((w: Record<string, unknown>) => {
          const projectId = String(w.project_id ?? w.projectId ?? '');
          workflowsMap.set(String(w.id), {
            name: String(w.name ?? 'Untitled'),
            projectId,
            projectName: projectsMap.get(projectId),
          });
        });
      }

      // Fetch all executions
      const response = await fetch(`${config.API_URL}/executions?limit=200`);
      if (!response.ok) {
        throw new Error(`Failed to fetch executions: ${response.status}`);
      }
      const data = await response.json();

      const executionItems: ExecutionItem[] = Array.isArray(data.executions)
        ? data.executions.map((e: Record<string, unknown>) => {
            const workflowId = String(e.workflow_id ?? e.workflowId ?? '');
            const workflowInfo = workflowsMap.get(workflowId);
            const statusValue = String(e.status ?? 'pending');
            const validStatuses = ['pending', 'running', 'completed', 'failed', 'cancelled'];
            const status = validStatuses.includes(statusValue) ? statusValue as ExecutionStatus : 'pending';

            const startedAt = new Date(String(e.started_at ?? e.startedAt ?? new Date().toISOString()));
            const completedAt = e.completed_at || e.completedAt
              ? new Date(String(e.completed_at ?? e.completedAt))
              : undefined;
            const duration = completedAt
              ? completedAt.getTime() - startedAt.getTime()
              : undefined;

            return {
              id: String(e.id ?? e.execution_id ?? ''),
              workflowId,
              workflowName: workflowInfo?.name ?? 'Unknown Workflow',
              projectId: workflowInfo?.projectId,
              projectName: workflowInfo?.projectName,
              status,
              startedAt,
              completedAt,
              duration,
              error: e.error ? String(e.error) : undefined,
            };
          })
        : [];

      // Sort by startedAt descending
      executionItems.sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime());

      setExecutions(executionItems);
    } catch (error) {
      logger.error('Failed to fetch all executions', { component: 'GlobalExecutionsView', action: 'fetchAllExecutions' }, error);
      toast.error('Failed to load executions');
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, []);

  useEffect(() => {
    fetchAllExecutions();

    // Poll for updates every 15 seconds
    const interval = setInterval(() => {
      fetchAllExecutions(true);
    }, 15000);

    return () => clearInterval(interval);
  }, [fetchAllExecutions]);

  // Filter executions
  const filteredExecutions = useMemo(() => {
    let result = executions;

    // Filter by status
    if (statusFilter !== 'all') {
      result = result.filter((e) => e.status === statusFilter);
    }

    // Filter by search term
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      result = result.filter(
        (e) =>
          e.workflowName.toLowerCase().includes(term) ||
          e.projectName?.toLowerCase().includes(term) ||
          e.id.toLowerCase().includes(term)
      );
    }

    return result;
  }, [executions, statusFilter, searchTerm]);

  // Stats
  const stats = useMemo(() => {
    const running = executions.filter((e) => e.status === 'running' || e.status === 'pending').length;
    const completed = executions.filter((e) => e.status === 'completed').length;
    const failed = executions.filter((e) => e.status === 'failed').length;
    return { running, completed, failed, total: executions.length };
  }, [executions]);

  const renderExecutionItem = (execution: ExecutionItem) => (
    <div
      key={execution.id}
      onClick={() => onViewExecution(execution.id, execution.workflowId)}
      className="group flex items-center gap-3 p-3 rounded-lg hover:bg-gray-700/50 cursor-pointer transition-colors border border-transparent hover:border-gray-600"
    >
      <div className="flex items-center justify-center w-10 h-10 bg-gray-700/50 rounded-lg transition-colors">
        <StatusIcon status={execution.status} size={20} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-white truncate">{execution.workflowName}</span>
          <StatusBadge status={execution.status} />
        </div>
        <div className="text-xs text-gray-500 truncate">
          {execution.projectName && <span>{execution.projectName} &middot; </span>}
          <Clock className="inline w-3 h-3 mr-1" />
          {formatRelativeTime(execution.startedAt)}
          {execution.duration && (
            <span> &middot; {formatDuration(execution.duration)}</span>
          )}
        </div>
        {execution.error && (
          <div className="text-xs text-red-400 truncate mt-1">{execution.error}</div>
        )}
      </div>
      <svg className="w-4 h-4 text-gray-500 group-hover:text-gray-300 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
      </svg>
    </div>
  );

  return (
    <div className="flex-1 flex flex-col min-h-[100svh] overflow-hidden bg-flow-bg">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur supports-[backdrop-filter]:bg-flow-bg/90">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={onBack}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                title="Back to Dashboard"
              >
                <ChevronLeft size={20} />
              </button>
              <div>
                <h1 className="text-xl font-bold text-white">All Executions</h1>
                <p className="text-sm text-gray-400">
                  {isLoading ? 'Loading...' : `${filteredExecutions.length} of ${stats.total} execution${stats.total !== 1 ? 's' : ''}`}
                </p>
              </div>
            </div>
            <button
              onClick={() => fetchAllExecutions(true)}
              disabled={isRefreshing}
              className="flex items-center gap-2 px-3 py-1.5 text-gray-400 hover:text-white bg-gray-800/50 hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors disabled:opacity-50"
              title="Refresh"
            >
              <RefreshCw size={14} className={isRefreshing ? 'animate-spin' : ''} />
              <span className="text-sm hidden sm:inline">Refresh</span>
            </button>
          </div>

          {/* Stats Bar */}
          {!isLoading && stats.total > 0 && (
            <div className="mt-3 flex items-center gap-4">
              {stats.running > 0 && (
                <div className="flex items-center gap-1.5 text-sm">
                  <span className="w-2 h-2 bg-blue-400 rounded-full animate-pulse" />
                  <span className="text-blue-400 font-medium">{stats.running} running</span>
                </div>
              )}
              <div className="flex items-center gap-1.5 text-sm text-gray-500">
                <CheckCircle size={14} className="text-green-400" />
                <span>{stats.completed} completed</span>
              </div>
              {stats.failed > 0 && (
                <div className="flex items-center gap-1.5 text-sm text-gray-500">
                  <XCircle size={14} className="text-red-400" />
                  <span>{stats.failed} failed</span>
                </div>
              )}
            </div>
          )}

          {/* Search and Filters */}
          <div className="mt-4 flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500" />
              <input
                type="text"
                placeholder="Search executions..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-10 pr-10 py-2 bg-flow-node border border-gray-700 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-flow-accent focus:ring-2 focus:ring-flow-accent/50"
              />
              {searchTerm && (
                <button
                  onClick={() => setSearchTerm('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
                >
                  <X size={16} />
                </button>
              )}
            </div>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`flex items-center gap-2 px-3 py-2 rounded-lg border transition-colors ${
                showFilters || statusFilter !== 'all'
                  ? 'bg-flow-accent/20 border-flow-accent text-flow-accent'
                  : 'bg-gray-800/50 border-gray-700 text-gray-400 hover:text-white'
              }`}
            >
              <Filter size={16} />
              <span className="text-sm">Filters</span>
              {statusFilter !== 'all' && (
                <span className="px-1.5 py-0.5 bg-flow-accent/30 rounded text-xs">1</span>
              )}
            </button>
          </div>

          {/* Filter Options */}
          {showFilters && (
            <div className="mt-3 p-3 bg-gray-800/50 rounded-lg border border-gray-700">
              <label className="block text-xs text-gray-400 mb-2">Status</label>
              <div className="flex flex-wrap gap-2">
                {[
                  { value: 'all', label: 'All' },
                  { value: 'running', label: 'Running' },
                  { value: 'pending', label: 'Pending' },
                  { value: 'completed', label: 'Completed' },
                  { value: 'failed', label: 'Failed' },
                  { value: 'cancelled', label: 'Cancelled' },
                ].map((option) => (
                  <button
                    key={option.value}
                    onClick={() => setStatusFilter(option.value as StatusFilter)}
                    className={`px-3 py-1.5 rounded-md text-sm transition-colors ${
                      statusFilter === option.value
                        ? 'bg-flow-accent text-white'
                        : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Content */}
      <div className="flex-1 min-h-0 overflow-auto px-4 py-6 sm:px-6">
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="animate-pulse flex items-center gap-3 p-3 bg-gray-800/30 rounded-lg">
                <div className="w-10 h-10 bg-gray-700 rounded-lg" />
                <div className="flex-1">
                  <div className="h-4 bg-gray-700 rounded w-1/3 mb-2" />
                  <div className="h-3 bg-gray-700/50 rounded w-1/2" />
                </div>
              </div>
            ))}
          </div>
        ) : filteredExecutions.length === 0 ? (
          <div className="text-center py-12">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800 flex items-center justify-center">
              {statusFilter !== 'all' ? (
                <Filter size={28} className="text-gray-500" />
              ) : searchTerm ? (
                <Search size={28} className="text-gray-500" />
              ) : (
                <Clock size={28} className="text-gray-500" />
              )}
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              {statusFilter !== 'all'
                ? `No ${statusFilter} executions`
                : searchTerm
                ? 'No executions found'
                : 'No executions yet'}
            </h3>
            <p className="text-gray-400">
              {statusFilter !== 'all'
                ? `There are no executions with status "${statusFilter}"`
                : searchTerm
                ? `No executions match "${searchTerm}"`
                : 'Run a workflow to see execution history'}
            </p>
            {(searchTerm || statusFilter !== 'all') && (
              <button
                onClick={() => {
                  setSearchTerm('');
                  setStatusFilter('all');
                }}
                className="mt-4 text-flow-accent hover:text-blue-400 text-sm"
              >
                Clear filters
              </button>
            )}
          </div>
        ) : (
          <div className="space-y-1">
            {filteredExecutions.map((execution) => renderExecutionItem(execution))}
          </div>
        )}
      </div>
    </div>
  );
};
