/**
 * WorkflowInfoCard Component
 *
 * Displays workflow information in a "ready to run" state when a workflow
 * is selected but not yet executing. Shows workflow details, execution
 * history stats, and provides actions to run or change the workflow.
 */

import { useEffect, useState } from 'react';
import { Play, RefreshCw, CheckCircle2, XCircle, Clock, BarChart3 } from 'lucide-react';
import { getConfig } from '@/config';

interface WorkflowStats {
  execution_count: number;
  last_execution?: string;
  success_rate?: number;
  avg_duration_ms?: number;
}

interface WorkflowDetails {
  id: string;
  name: string;
  description?: string;
  node_count?: number;
  stats?: WorkflowStats;
}

interface WorkflowInfoCardProps {
  workflowId: string;
  workflowName: string;
  onRun: () => void;
  onChangeWorkflow: () => void;
}

export function WorkflowInfoCard({
  workflowId,
  workflowName,
  onRun,
  onChangeWorkflow,
}: WorkflowInfoCardProps) {
  const [workflow, setWorkflow] = useState<WorkflowDetails | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchWorkflowDetails = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const config = await getConfig();
        const response = await fetch(`${config.API_URL}/workflows/${workflowId}`);
        if (!response.ok) {
          throw new Error(`Failed to fetch workflow: ${response.status}`);
        }
        const data = await response.json();
        setWorkflow(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load workflow');
      } finally {
        setIsLoading(false);
      }
    };

    fetchWorkflowDetails();
  }, [workflowId]);

  const formatDuration = (ms?: number) => {
    if (!ms) return null;
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const formatLastExecution = (dateStr?: string) => {
    if (!dateStr) return 'Never';
    const date = new Date(dateStr);
    if (Number.isNaN(date.getTime())) return 'Never';
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    return date.toLocaleDateString();
  };

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="w-8 h-8 border-2 border-gray-300 border-t-flow-accent rounded-full animate-spin" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 p-8">
        <div className="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center mb-4">
          <XCircle size={32} className="text-red-500" />
        </div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">
          Failed to Load Workflow
        </h3>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">{error}</p>
        <button
          onClick={onChangeWorkflow}
          className="px-4 py-2 text-sm font-medium text-flow-accent hover:underline"
        >
          Select a different workflow
        </button>
      </div>
    );
  }

  const stats = workflow?.stats;
  const successRate = stats?.success_rate ?? 0;

  return (
    <div className="flex-1 flex flex-col items-center justify-center bg-gray-50 dark:bg-gray-900 p-8">
      {/* Workflow icon and name */}
      <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-flow-accent/20 to-blue-500/20 flex items-center justify-center mb-6 shadow-lg">
        <Play size={40} className="text-flow-accent" />
      </div>

      <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-2 text-center">
        {workflow?.name ?? workflowName}
      </h2>

      {workflow?.description && (
        <p className="text-gray-500 dark:text-gray-400 text-center max-w-md mb-6">
          {workflow.description}
        </p>
      )}

      {/* Stats grid */}
      {stats && stats.execution_count > 0 && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8 w-full max-w-lg">
          {/* Execution count */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 text-center border border-gray-200 dark:border-gray-700">
            <div className="text-2xl font-bold text-gray-900 dark:text-white">
              {stats.execution_count}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Total Runs
            </div>
          </div>

          {/* Success rate */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 text-center border border-gray-200 dark:border-gray-700">
            <div className={`text-2xl font-bold ${
              successRate >= 80 ? 'text-green-500' :
              successRate >= 50 ? 'text-yellow-500' :
              'text-red-500'
            }`}>
              {Math.round(successRate)}%
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 flex items-center justify-center gap-1">
              {successRate >= 80 ? <CheckCircle2 size={12} /> : <BarChart3 size={12} />}
              Success
            </div>
          </div>

          {/* Last run */}
          <div className="bg-white dark:bg-gray-800 rounded-xl p-4 text-center border border-gray-200 dark:border-gray-700">
            <div className="text-lg font-bold text-gray-900 dark:text-white">
              {formatLastExecution(stats.last_execution)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 flex items-center justify-center gap-1">
              <Clock size={12} />
              Last Run
            </div>
          </div>

          {/* Avg duration */}
          {stats.avg_duration_ms && (
            <div className="bg-white dark:bg-gray-800 rounded-xl p-4 text-center border border-gray-200 dark:border-gray-700">
              <div className="text-lg font-bold text-gray-900 dark:text-white">
                {formatDuration(stats.avg_duration_ms)}
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 flex items-center justify-center gap-1">
                <RefreshCw size={12} />
                Avg Time
              </div>
            </div>
          )}
        </div>
      )}

      {/* No executions yet message */}
      {(!stats || stats.execution_count === 0) && (
        <div className="bg-blue-50 dark:bg-blue-900/20 rounded-xl p-4 mb-8 max-w-md text-center">
          <p className="text-sm text-blue-600 dark:text-blue-400">
            This workflow hasn't been executed yet. Click Run to start your first execution.
          </p>
        </div>
      )}

      {/* Action buttons */}
      <div className="flex items-center gap-4">
        <button
          onClick={onRun}
          className="flex items-center gap-2 px-6 py-3 bg-flow-accent text-white font-medium rounded-xl hover:bg-blue-600 transition-colors shadow-lg shadow-flow-accent/25"
        >
          <Play size={20} />
          Run Workflow
        </button>

        <button
          onClick={onChangeWorkflow}
          className="px-4 py-3 text-gray-600 dark:text-gray-400 font-medium rounded-xl hover:bg-gray-200 dark:hover:bg-gray-800 transition-colors"
        >
          Change Workflow
        </button>
      </div>
    </div>
  );
}
