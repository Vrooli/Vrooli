import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { Task, InsightReport } from '@/types/api';
import { Button } from '@/components/ui/button';
import { Loader2, TrendingUp, AlertCircle, Plus } from 'lucide-react';
import { InsightReportViewer } from './InsightReportViewer';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';

interface InsightsTabProps {
  task: Task;
}

export function InsightsTab({ task }: InsightsTabProps) {
  const queryClient = useQueryClient();
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);
  const [generateLimit, setGenerateLimit] = useState('10');
  const [generateStatusFilter, setGenerateStatusFilter] = useState('failed,timeout');

  // Load insight reports for this task
  const {
    data: reports,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: queryKeys.insights.task(task.id),
    queryFn: () => api.getTaskInsights(task.id),
    staleTime: 30000,
  });

  // Generate insight report mutation
  const generateMutation = useMutation({
    mutationFn: () =>
      api.generateInsightReport(task.id, {
        limit: parseInt(generateLimit, 10),
        status_filter: generateStatusFilter,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.insights.task(task.id) });
      refetch();
    },
  });

  // Apply suggestion mutation
  const [applyingId, setApplyingId] = useState<string | null>(null);
  const applySuggestionMutation = useMutation({
    mutationFn: ({
      reportId,
      suggestionId,
    }: {
      reportId: string;
      suggestionId: string;
    }) => api.applySuggestion(task.id, reportId, suggestionId),
    onMutate: ({ suggestionId }) => {
      setApplyingId(suggestionId);
    },
    onSuccess: (_, { reportId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.insights.task(task.id) });
      // Refresh the selected report
      if (selectedReportId === reportId) {
        refetch();
      }
    },
    onError: (error, { suggestionId }) => {
      alert(`Failed to apply suggestion: ${error instanceof Error ? error.message : 'Unknown error'}`);
    },
    onSettled: () => {
      setApplyingId(null);
    },
  });

  const selectedReport = reports?.find((r) => r.id === selectedReportId) || reports?.[0] || null;

  // Auto-select first report when loaded
  if (reports && reports.length > 0 && !selectedReportId) {
    setSelectedReportId(reports[0].id);
  }

  if (isLoading) {
    return (
      <div className="text-center py-12 text-slate-400">
        <Loader2 className="h-8 w-8 mx-auto mb-3 animate-spin" />
        <p className="text-sm">Loading insights...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-center py-12 text-red-400">
        <AlertCircle className="h-12 w-12 mx-auto mb-3 opacity-50" />
        <p className="text-sm">Failed to load insights</p>
        <p className="text-xs text-slate-500 mt-1">
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
        <Button onClick={() => refetch()} variant="outline" size="sm" className="mt-4">
          Retry
        </Button>
      </div>
    );
  }

  if (!reports || reports.length === 0) {
    return (
      <div className="space-y-6">
        <div className="text-center py-12 text-slate-400">
          <TrendingUp className="h-12 w-12 mx-auto mb-3 opacity-50" />
          <p className="text-sm mb-4">No insight reports generated yet</p>
          <p className="text-xs text-slate-500 mb-6">
            Insight reports analyze execution history to identify patterns and suggest improvements
          </p>
        </div>

        <div className="bg-slate-800/50 rounded-lg p-6 border border-slate-700/50">
          <h3 className="text-sm font-semibold text-slate-200 mb-4">Generate Insight Report</h3>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="limit" className="text-xs text-slate-300">
                  Executions to Analyze
                </Label>
                <Select value={generateLimit} onValueChange={setGenerateLimit}>
                  <SelectTrigger id="limit">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="5">Last 5</SelectItem>
                    <SelectItem value="10">Last 10</SelectItem>
                    <SelectItem value="20">Last 20</SelectItem>
                    <SelectItem value="50">Last 50</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="status-filter" className="text-xs text-slate-300">
                  Status Filter
                </Label>
                <Select value={generateStatusFilter} onValueChange={setGenerateStatusFilter}>
                  <SelectTrigger id="status-filter">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="failed,timeout">Failed & Timeout</SelectItem>
                    <SelectItem value="failed">Failed Only</SelectItem>
                    <SelectItem value="timeout">Timeout Only</SelectItem>
                    <SelectItem value="all">All Statuses</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <Button
              onClick={() => generateMutation.mutate()}
              disabled={generateMutation.isPending}
              className="w-full"
            >
              {generateMutation.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Generating...
                </>
              ) : (
                <>
                  <Plus className="mr-2 h-4 w-4" />
                  Generate Insight Report
                </>
              )}
            </Button>

            {generateMutation.isError && (
              <p className="text-xs text-red-400">
                Failed to generate report:{' '}
                {generateMutation.error instanceof Error
                  ? generateMutation.error.message
                  : 'Unknown error'}
              </p>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Report Selector */}
      <div className="flex items-center gap-4">
        <div className="flex-1">
          <Select value={selectedReportId || ''} onValueChange={setSelectedReportId}>
            <SelectTrigger>
              <SelectValue placeholder="Select a report" />
            </SelectTrigger>
            <SelectContent>
              {reports.map((report) => (
                <SelectItem key={report.id} value={report.id}>
                  {new Date(report.generated_at).toLocaleString()} ({report.execution_count}{' '}
                  executions)
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          onClick={() => generateMutation.mutate()}
          disabled={generateMutation.isPending}
          size="sm"
          variant="outline"
        >
          {generateMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plus className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Report Viewer */}
      {selectedReport && (
        <InsightReportViewer
          report={selectedReport}
          onApplySuggestion={(suggestionId) =>
            applySuggestionMutation.mutate({ reportId: selectedReport.id, suggestionId })
          }
          applyingId={applyingId ?? undefined}
        />
      )}
    </div>
  );
}
