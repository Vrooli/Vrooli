import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  CheckCircle2,
  XCircle,
  Loader2,
  Eye,
  Activity,
  Image,
  LineChart,
  X,
} from 'lucide-react';
import { useDashboardStore, type RecentExecution } from '@stores/dashboardStore';
import { useExecutionStore } from '../store';
import { TabEmptyState, ExecutionsEmptyPreview } from '@/views/DashboardView/previews';
import { ExecutionCard, type ExecutionCardData } from './ExecutionCard';
import { ExecutionFilters, type StatusFilter } from './ExecutionFilters';
import { InlineExecutionViewer } from '../InlineExecutionViewer';
import { logger } from '@utils/logger';
import toast from 'react-hot-toast';

interface ExecutionsTabProps {
  onViewExecution: (executionId: string, workflowId: string) => void;
  onNavigateToHome?: () => void;
  onCreateWorkflow?: () => void;
  /** Called when re-run is requested - if not provided, navigates to record page */
  onRerunWorkflow?: (workflowId: string) => void;
}

/** Convert RecentExecution to ExecutionCardData */
const toCardData = (execution: RecentExecution): ExecutionCardData => ({
  id: execution.id,
  workflowId: execution.workflowId,
  workflowName: execution.workflowName,
  projectName: execution.projectName,
  status: execution.status,
  startedAt: execution.startedAt,
  completedAt: execution.completedAt,
  error: execution.error,
});

export const ExecutionsTab: React.FC<ExecutionsTabProps> = ({
  onViewExecution,
  onNavigateToHome,
  onCreateWorkflow,
  onRerunWorkflow,
}) => {
  const {
    recentExecutions,
    runningExecutions,
    recentWorkflows,
    isLoadingExecutions,
    fetchRecentExecutions,
    fetchRunningExecutions,
  } = useDashboardStore();
  const { stopExecution, loadExecution, currentExecution, closeViewer } = useExecutionStore();
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [selectedExecutionId, setSelectedExecutionId] = useState<string | null>(null);

  const isViewerOpen = Boolean(currentExecution) && Boolean(selectedExecutionId);

  // Auto-refresh running executions every 5 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      if (runningExecutions.length > 0) {
        void fetchRunningExecutions();
      }
    }, 5000);
    return () => clearInterval(interval);
  }, [runningExecutions.length, fetchRunningExecutions]);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await Promise.all([fetchRecentExecutions(), fetchRunningExecutions()]);
    setIsRefreshing(false);
  }, [fetchRecentExecutions, fetchRunningExecutions]);

  const handleStopExecution = useCallback(async (executionId: string) => {
    try {
      await stopExecution(executionId);
      await fetchRunningExecutions();
    } catch (error) {
      console.error('Failed to stop execution:', error);
    }
  }, [stopExecution, fetchRunningExecutions]);

  const handleSelectExecution = useCallback(async (executionId: string, _workflowId: string) => {
    try {
      setSelectedExecutionId(executionId);
      await loadExecution(executionId);
    } catch (error) {
      logger.error('Failed to load execution', { component: 'ExecutionsTab', executionId }, error);
      toast.error('Failed to load execution details');
      setSelectedExecutionId(null);
    }
  }, [loadExecution]);

  const handleCloseViewer = useCallback(() => {
    setSelectedExecutionId(null);
    closeViewer();
  }, [closeViewer]);

  const handleRerun = useCallback(() => {
    if (!currentExecution) return;
    if (onRerunWorkflow) {
      onRerunWorkflow(currentExecution.workflowId);
    } else {
      // Fallback: navigate to record page (this will be handled by parent)
      onViewExecution(currentExecution.id, currentExecution.workflowId);
    }
  }, [currentExecution, onRerunWorkflow, onViewExecution]);

  const handleRerunFromCard = useCallback((_executionId: string, workflowId: string) => {
    if (onRerunWorkflow) {
      onRerunWorkflow(workflowId);
    }
  }, [onRerunWorkflow]);

  // Combine and filter executions
  const allExecutions = [...runningExecutions, ...recentExecutions];
  const filteredExecutions = allExecutions.filter((execution) => {
    if (statusFilter === 'all') return true;
    if (statusFilter === 'running') return execution.status === 'running' || execution.status === 'pending';
    if (statusFilter === 'completed') return execution.status === 'completed';
    if (statusFilter === 'failed') return execution.status === 'failed' || execution.status === 'cancelled';
    return true;
  });

  const completedExecutions = filteredExecutions.filter(
    (e) => e.status !== 'running' && e.status !== 'pending'
  );

  // Get counts for each status (for notification badges)
  const statusCounts = useMemo(() => {
    const running = allExecutions.filter(e => e.status === 'running' || e.status === 'pending').length;
    const completed = allExecutions.filter(e => e.status === 'completed').length;
    const failed = allExecutions.filter(e => e.status === 'failed' || e.status === 'cancelled').length;
    return {
      all: allExecutions.length,
      running,
      completed,
      failed,
    };
  }, [allExecutions]);

  // For the pulse strip summary
  const failedCount = statusCounts.failed;

  return (
    <div className="flex h-full min-h-[500px]">
      {/* Left side: Execution list */}
      <div className={`flex flex-col ${isViewerOpen ? 'w-1/2 border-r border-gray-800' : 'w-full'} transition-all duration-200`}>
        <div className="flex-1 overflow-auto p-4 space-y-6">
          {/* Pulse strip */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between bg-gradient-to-r from-gray-900/80 via-gray-900 to-blue-900/10 border border-gray-800/80 rounded-2xl p-4">
            <div className="flex flex-wrap items-center gap-2">
              <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-green-500/10 border border-green-500/30 text-green-100">
                <div className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
                {runningExecutions.length} running
              </div>
              <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/30 text-red-100">
                <XCircle size={14} />
                {failedCount} failed
              </div>
              <div className="inline-flex items-center gap-2 px-3 py-2 rounded-lg bg-gray-700/50 text-gray-100 border border-gray-700">
                <LineChart size={14} />
                {allExecutions.length} total
              </div>
            </div>
            {allExecutions.length > 0 && !isViewerOpen && (
              <button
                onClick={() => {
                  const latest = allExecutions[0];
                  handleSelectExecution(latest.id, latest.workflowId);
                }}
                className="hero-button-secondary w-full sm:w-auto justify-center"
              >
                Open latest run
              </button>
            )}
          </div>

          {/* Filter controls */}
          <ExecutionFilters
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            statusCounts={statusCounts}
            onRefresh={handleRefresh}
            isRefreshing={isRefreshing || isLoadingExecutions}
            filters={['all', 'running', 'completed', 'failed']}
            showRefreshLabel
          />

          {/* Running Executions Section */}
          {runningExecutions.length > 0 && (statusFilter === 'all' || statusFilter === 'running') && (
            <div className="space-y-3">
              <h3 className="text-sm font-medium text-gray-300 flex items-center gap-2">
                <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                Running Now ({runningExecutions.length})
              </h3>
              <div className="space-y-2">
                {runningExecutions.map((execution) => (
                  <ExecutionCard
                    key={execution.id}
                    execution={toCardData(execution)}
                    isRunning
                    isSelected={selectedExecutionId === execution.id}
                    onClick={handleSelectExecution}
                    onStop={handleStopExecution}
                  />
                ))}
              </div>
            </div>
          )}

          {/* Recent Executions Section */}
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-gray-300">
              {statusFilter === 'all' ? 'Recent Executions' : `${statusFilter.charAt(0).toUpperCase() + statusFilter.slice(1)} Executions`}
            </h3>

            {isLoadingExecutions ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 size={24} className="animate-spin text-gray-500" />
              </div>
            ) : filteredExecutions.length === 0 ? (
              <TabEmptyState
                icon={<Activity size={22} />}
                title="Watch your automations come alive"
                subtitle={
                  recentWorkflows.length > 0
                    ? `You have ${recentWorkflows.length} workflow${recentWorkflows.length !== 1 ? 's' : ''} ready to run.`
                    : 'Create a workflow, run it, and monitor every step with logs, screenshots, and metrics.'
                }
                preview={<ExecutionsEmptyPreview />}
                variant="polished"
                primaryCta={{
                  label: recentWorkflows.length > 0 ? 'Run a workflow' : 'Create your first workflow',
                  onClick:
                    (recentWorkflows.length > 0 ? onNavigateToHome : onCreateWorkflow) ??
                    onCreateWorkflow ??
                    (() => {}),
                }}
                secondaryCta={
                  recentWorkflows.length > 0 && onCreateWorkflow
                    ? { label: 'Create another workflow', onClick: onCreateWorkflow }
                    : undefined
                }
                progressPath={[
                  { label: 'Create workflow', completed: recentWorkflows.length > 0 },
                  { label: 'Run & monitor', active: true },
                  { label: 'Export results' },
                ]}
                features={[
                  {
                    title: 'Live monitoring',
                    description: 'Follow each browser action in real time with streaming logs.',
                    icon: <Eye size={16} />,
                  },
                  {
                    title: 'Step-by-step checks',
                    description: 'Progress, retries, and errors are captured automatically.',
                    icon: <CheckCircle2 size={16} />,
                  },
                  {
                    title: 'Visual evidence',
                    description: 'Screenshots attach to every critical step for quick validation.',
                    icon: <Image size={16} />,
                  },
                ]}
              />
            ) : (
              <div className="space-y-2">
                {completedExecutions.map((execution) => (
                  <ExecutionCard
                    key={execution.id}
                    execution={toCardData(execution)}
                    isSelected={selectedExecutionId === execution.id}
                    onClick={handleSelectExecution}
                    onRerun={onRerunWorkflow ? handleRerunFromCard : undefined}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Right side: Execution viewer */}
      {isViewerOpen && currentExecution && (
        <div className="w-1/2 flex flex-col bg-gray-900 relative">
          {/* Close button for mobile */}
          <button
            onClick={handleCloseViewer}
            className="md:hidden absolute top-2 right-2 z-10 p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg"
          >
            <X size={20} />
          </button>
          <InlineExecutionViewer
            executionId={currentExecution.id}
            workflowId={currentExecution.workflowId}
            projectId=""
            onClose={handleCloseViewer}
            onRerun={onRerunWorkflow ? handleRerun : undefined}
          />
        </div>
      )}
    </div>
  );
};

export default ExecutionsTab;
