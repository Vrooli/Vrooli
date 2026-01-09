import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { Task, InsightReport } from '@/types/api';
import { Button } from '@/components/ui/button';
import { Loader2, TrendingUp, AlertCircle, Plus, Eye, Info, XCircle, RefreshCw } from 'lucide-react';
import { InsightReportViewer } from './InsightReportViewer';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Card } from '@/components/ui/card';

interface InsightsTabProps {
  task: Task;
}

// Helper to extract user-friendly error message from API errors
function getErrorMessage(error: unknown): string {
  if (!error) return 'Unknown error';

  // Check if it's an Error object with a message
  if (error instanceof Error) {
    try {
      // Try to parse JSON from the error message (e.g., "API Error (400): {...}")
      const match = error.message.match(/API Error \(\d+\): (.+)/);
      if (match) {
        const jsonStr = match[1];
        const parsed = JSON.parse(jsonStr);
        return parsed.error || parsed.message || error.message;
      }
    } catch {
      // If parsing fails, return the original message
    }
    return error.message;
  }

  // If it's an object with an error property
  if (typeof error === 'object' && error !== null && 'error' in error) {
    return String((error as any).error);
  }

  return String(error);
}

export function InsightsTab({ task }: InsightsTabProps) {
  const queryClient = useQueryClient();
  const [selectedReportId, setSelectedReportId] = useState<string | null>(null);
  const [generateLimit, setGenerateLimit] = useState('10');
  const [generateStatusFilter, setGenerateStatusFilter] = useState('failed,timeout');
  const [previewDialogOpen, setPreviewDialogOpen] = useState(false);
  const [previewPrompt, setPreviewPrompt] = useState('');
  const [editedPrompt, setEditedPrompt] = useState('');

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

  // Preview prompt mutation
  const previewMutation = useMutation({
    mutationFn: () =>
      api.previewInsightPrompt(task.id, {
        limit: parseInt(generateLimit, 10),
        status_filter: generateStatusFilter,
      }),
    onSuccess: (data) => {
      setPreviewPrompt(data.prompt);
      setEditedPrompt(data.prompt);
      setPreviewDialogOpen(true);
    },
  });

  // Generate with custom prompt mutation
  const generateWithPromptMutation = useMutation({
    mutationFn: (customPrompt: string) =>
      api.generateInsightReportWithPrompt(task.id, {
        limit: parseInt(generateLimit, 10),
        status_filter: generateStatusFilter,
        custom_prompt: customPrompt,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.insights.task(task.id) });
      setPreviewDialogOpen(false);
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

  // Render dialog at the top level (always visible)
  const previewDialog = (
    <Dialog open={previewDialogOpen} onOpenChange={setPreviewDialogOpen}>
      <DialogContent className="max-w-4xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Preview & Edit Analysis Prompt</DialogTitle>
          <DialogDescription>
            Review and customize the prompt before generating the insight report
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Info Tip */}
          <div className="rounded-md border border-blue-500/30 bg-blue-500/10 p-4 text-sm">
            <div className="flex items-start gap-3">
              <Info className="h-5 w-5 text-blue-400 mt-0.5 flex-shrink-0" />
              <div className="flex-1">
                <h4 className="font-semibold text-blue-100 mb-1">What happens when you submit?</h4>
                <ul className="text-slate-300 space-y-1 text-xs">
                  <li>• An AI agent analyzes the last <strong>{generateLimit}</strong> executions with status: <strong>{generateStatusFilter}</strong></li>
                  <li>• It identifies common failure patterns, blockers, and inefficiencies</li>
                  <li>• The agent generates actionable suggestions to improve prompts and code</li>
                  <li>• Results are saved as an insight report you can review and apply</li>
                </ul>
                <p className="text-slate-400 mt-2 text-xs italic">
                  Tip: You can edit the prompt below to focus the analysis on specific areas or add custom instructions
                </p>
              </div>
            </div>
          </div>

          {/* Prompt Editor */}
          <div className="space-y-2">
            <Label htmlFor="prompt-editor" className="text-sm font-semibold">
              Analysis Prompt
            </Label>
            <Textarea
              id="prompt-editor"
              value={editedPrompt}
              onChange={(e) => setEditedPrompt(e.target.value)}
              className="min-h-[400px] font-mono text-xs"
              placeholder="Loading prompt..."
            />
            <p className="text-xs text-slate-500">
              This prompt will be sent to the AI agent. Feel free to customize it to focus on specific issues or add context.
            </p>
          </div>

          {/* Comparison */}
          {editedPrompt !== previewPrompt && (
            <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-200">
              <Info className="inline h-4 w-4 mr-1.5" />
              You've modified the prompt. Click "Generate Report" to use your custom version.
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={() => {
              setEditedPrompt(previewPrompt);
            }}
            disabled={editedPrompt === previewPrompt}
          >
            Reset to Default
          </Button>
          <Button
            variant="outline"
            onClick={() => setPreviewDialogOpen(false)}
          >
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (editedPrompt.trim()) {
                generateWithPromptMutation.mutate(editedPrompt);
              }
            }}
            disabled={!editedPrompt.trim() || generateWithPromptMutation.isPending}
          >
            {generateWithPromptMutation.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Generating...
              </>
            ) : (
              <>
                <Plus className="mr-2 h-4 w-4" />
                Generate Report
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );

  if (isLoading) {
    return (
      <>
        {previewDialog}
        <div className="text-center py-12 text-slate-400">
          <Loader2 className="h-8 w-8 mx-auto mb-3 animate-spin" />
          <p className="text-sm">Loading insights...</p>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        {previewDialog}
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
      </>
    );
  }

  if (!reports || reports.length === 0) {
    return (
      <>
        {previewDialog}
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

            <div className="grid grid-cols-2 gap-3">
              <Button
                onClick={() => previewMutation.mutate()}
                disabled={previewMutation.isPending || generateMutation.isPending}
                variant="outline"
              >
                {previewMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Loading...
                  </>
                ) : (
                  <>
                    <Eye className="mr-2 h-4 w-4" />
                    Preview & Edit
                  </>
                )}
              </Button>
              <Button
                onClick={() => generateMutation.mutate()}
                disabled={generateMutation.isPending || previewMutation.isPending}
              >
                {generateMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Generating...
                  </>
                ) : (
                  <>
                    <Plus className="mr-2 h-4 w-4" />
                    Generate Now
                  </>
                )}
              </Button>
            </div>

            {(generateMutation.isError || previewMutation.isError) && (() => {
              const error = generateMutation.error || previewMutation.error;
              const errorMsg = getErrorMessage(error);
              const isFilterError = errorMsg.includes('No executions found matching filter') ||
                                   errorMsg.includes('none match your selected status filter');
              const canRetryWithAll = isFilterError && generateStatusFilter !== 'all';

              return (
                <Card className="bg-red-500/10 border-red-500/30 p-4">
                  <div className="flex items-start gap-3">
                    <XCircle className="h-5 w-5 text-red-400 mt-0.5 flex-shrink-0" />
                    <div className="flex-1">
                      <h4 className="font-semibold text-red-200 mb-1">Unable to Generate Insight Report</h4>
                      <p className="text-sm text-red-100 mb-3">
                        {errorMsg}
                      </p>
                      {canRetryWithAll && (
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setGenerateStatusFilter('all');
                            // Reset errors
                            generateMutation.reset();
                            previewMutation.reset();
                            // Retry immediately
                            setTimeout(() => {
                              if (previewMutation.isError) {
                                previewMutation.mutate();
                              } else if (generateMutation.isError) {
                                generateMutation.mutate();
                              }
                            }, 100);
                          }}
                          className="bg-red-900/50 border-red-500/50 hover:bg-red-900/70 hover:border-red-500/70 text-red-100"
                        >
                          <RefreshCw className="h-4 w-4 mr-2" />
                          Try with All Statuses
                        </Button>
                      )}
                    </div>
                  </div>
                </Card>
              );
            })()}
          </div>
        </div>
      </div>
      </>
    );
  }

  return (
    <>
      {previewDialog}
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
    </>
  );
}
