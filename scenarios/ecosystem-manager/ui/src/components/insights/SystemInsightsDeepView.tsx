/**
 * SystemInsightsDeepView
 * Displays detailed system-wide analysis with cross-task patterns and task-type breakdown
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Sparkles, TrendingUp, Users } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { CrossTaskPatternCard } from './CrossTaskPatternCard';
import { TaskTypeStatsCard } from './TaskTypeStatsCard';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { SystemInsightReport } from '@/types/api';

interface SystemInsightsDeepViewProps {
  sinceDays: number;
}

export function SystemInsightsDeepView({ sinceDays }: SystemInsightsDeepViewProps) {
  const queryClient = useQueryClient();

  const generateMutation = useMutation({
    mutationFn: () => api.generateSystemInsights(sinceDays),
    onSuccess: (report) => {
      // Cache the generated report
      queryClient.setQueryData(['system-insight-report', sinceDays], report);
      queryClient.invalidateQueries({ queryKey: queryKeys.insights.system(sinceDays) });
    },
  });

  const generatedReport = queryClient.getQueryData<SystemInsightReport>([
    'system-insight-report',
    sinceDays,
  ]);

  if (!generatedReport && !generateMutation.isPending) {
    return (
      <Card className="bg-slate-900/70 border-white/5 p-12">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="w-16 h-16 rounded-full bg-purple-500/10 flex items-center justify-center">
            <Sparkles className="h-8 w-8 text-purple-400" />
          </div>
          <div>
            <h3 className="text-lg font-semibold text-white mb-2">
              Generate Deep System Analysis
            </h3>
            <p className="text-sm text-slate-400 mb-4 max-w-md">
              Analyze cross-task patterns, identify systemic issues, and get comprehensive
              statistics broken down by task type and operation.
            </p>
            <Button
              onClick={() => generateMutation.mutate()}
              className="bg-purple-600 hover:bg-purple-700"
            >
              <Sparkles className="h-4 w-4 mr-2" />
              Generate Analysis
            </Button>
          </div>
        </div>
      </Card>
    );
  }

  if (generateMutation.isPending) {
    return (
      <Card className="bg-slate-900/70 border-white/5 p-12">
        <div className="flex flex-col items-center gap-4">
          <Sparkles className="h-12 w-12 text-purple-400 animate-pulse" />
          <div className="text-center">
            <p className="text-white font-medium">Generating deep analysis...</p>
            <p className="text-sm text-slate-400 mt-1">
              Analyzing execution patterns across all tasks
            </p>
          </div>
        </div>
      </Card>
    );
  }

  if (generateMutation.isError) {
    return (
      <Card className="bg-slate-900/70 border-white/5 p-12">
        <div className="flex flex-col items-center gap-4 text-center">
          <p className="text-red-400">Failed to generate analysis</p>
          <p className="text-sm text-slate-500">
            {generateMutation.error instanceof Error
              ? generateMutation.error.message
              : 'Unknown error'}
          </p>
          <Button variant="outline" onClick={() => generateMutation.mutate()}>
            Try Again
          </Button>
        </div>
      </Card>
    );
  }

  if (!generatedReport) {
    return null;
  }

  const hasTaskTypeStats = generatedReport.by_task_type && Object.keys(generatedReport.by_task_type).length > 0;
  const hasOperationStats = generatedReport.by_operation && Object.keys(generatedReport.by_operation).length > 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-white flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-purple-400" />
            Deep System Analysis
          </h3>
          <p className="text-xs text-slate-400 mt-1">
            Generated {new Date(generatedReport.generated_at).toLocaleString()} •{' '}
            {generatedReport.task_count} tasks • {generatedReport.total_executions} executions
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => generateMutation.mutate()}
          disabled={generateMutation.isPending}
        >
          Regenerate
        </Button>
      </div>

      {/* Cross-Task Patterns */}
      {generatedReport.cross_task_patterns && generatedReport.cross_task_patterns.length > 0 && (
        <div>
          <h4 className="text-sm font-semibold text-white mb-3 flex items-center gap-2">
            <Users className="h-4 w-4 text-amber-400" />
            Cross-Task Patterns ({generatedReport.cross_task_patterns.length})
          </h4>
          <div className="space-y-3">
            {generatedReport.cross_task_patterns.map((pattern, idx) => (
              <CrossTaskPatternCard key={`${pattern.id || idx}`} pattern={pattern} />
            ))}
          </div>
        </div>
      )}

      {/* Task Type Breakdown */}
      {hasTaskTypeStats && (
        <div>
          <h4 className="text-sm font-semibold text-white mb-3 flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-blue-400" />
            Performance by Task Type
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {Object.entries(generatedReport.by_task_type).map(([type, stats]) => (
              <TaskTypeStatsCard key={type} label={type} stats={stats} />
            ))}
          </div>
        </div>
      )}

      {/* Operation Breakdown */}
      {hasOperationStats && (
        <div>
          <h4 className="text-sm font-semibold text-white mb-3 flex items-center gap-2">
            <TrendingUp className="h-4 w-4 text-emerald-400" />
            Performance by Operation
          </h4>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {Object.entries(generatedReport.by_operation).map(([operation, stats]) => (
              <TaskTypeStatsCard key={operation} label={operation} stats={stats} />
            ))}
          </div>
        </div>
      )}

      {/* System-Level Suggestions */}
      {generatedReport.system_suggestions && generatedReport.system_suggestions.length > 0 && (
        <Card className="bg-slate-900/70 border-white/5 p-4">
          <h4 className="text-sm font-semibold text-white mb-3">
            System-Wide Recommendations ({generatedReport.system_suggestions.length})
          </h4>
          <div className="space-y-2">
            {generatedReport.system_suggestions.slice(0, 5).map((suggestion, idx) => (
              <div
                key={suggestion.id || idx}
                className="p-3 bg-slate-800/50 rounded border border-white/5"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1">
                    <p className="text-sm font-medium text-white">{suggestion.title}</p>
                    <p className="text-xs text-slate-300 mt-1">{suggestion.description}</p>
                  </div>
                  <span className="text-xs px-2 py-1 bg-purple-500/20 text-purple-200 rounded">
                    {suggestion.priority}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
