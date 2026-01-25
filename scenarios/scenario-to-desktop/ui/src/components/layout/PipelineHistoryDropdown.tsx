/**
 * Pipeline history modal showing past pipelines for the current scenario.
 * Read-only view of historical pipelines with status badges and timestamps.
 */

import { useState, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";
import { X, RefreshCw, Clock, CheckCircle, XCircle, AlertCircle, Loader2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";
import { usePipelineStore } from "../../store";
import type { VerbosePipelineStatus } from "../../lib/api";

interface PipelineHistoryDropdownProps {
  open: boolean;
  onClose: () => void;
}

/** Format Unix timestamp to locale string */
function formatTimestamp(unix: number | undefined): string {
  if (!unix) return "-";
  return new Date(unix * 1000).toLocaleString();
}

/** Get status icon and color */
function getStatusDisplay(status: string | undefined): { icon: typeof CheckCircle; color: string } {
  switch (status) {
    case "completed":
      return { icon: CheckCircle, color: "text-green-400" };
    case "failed":
      return { icon: XCircle, color: "text-red-400" };
    case "cancelled":
      return { icon: AlertCircle, color: "text-yellow-400" };
    case "running":
    case "pending":
      return { icon: Loader2, color: "text-blue-400" };
    default:
      return { icon: AlertCircle, color: "text-slate-400" };
  }
}

export function PipelineHistoryDropdown({ open, onClose }: PipelineHistoryDropdownProps) {
  if (!open) return null;

  return <PipelineHistoryContent onClose={onClose} />;
}

function PipelineHistoryContent({ onClose }: { onClose: () => void }) {
  const [pipelines, setPipelines] = useState<VerbosePipelineStatus[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const scenarioName = usePipelineStore((s) => s.scenarioName);
  const currentPipelineId = usePipelineStore((s) => s.pipelineId);
  const loadPipelineHistory = usePipelineStore((s) => s.loadPipelineHistory);

  const fetchHistory = useCallback(async () => {
    if (!scenarioName) {
      setPipelines([]);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const history = await loadPipelineHistory(10);
      setPipelines(history);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load history");
    } finally {
      setIsLoading(false);
    }
  }, [scenarioName, loadPipelineHistory]);

  useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  return createPortal(
    <div
      className="fixed inset-0 z-[99999] flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose();
        }
      }}
    >
      <Card className="w-full max-w-2xl max-h-[80vh] flex flex-col border-slate-800 bg-slate-950/90 shadow-xl">
        <CardHeader className="flex flex-row items-start justify-between gap-4 shrink-0">
          <div className="space-y-1">
            <CardTitle className="text-lg text-slate-100">Pipeline History</CardTitle>
            <p className="text-sm text-slate-400">
              {scenarioName ? `Historical pipelines for ${scenarioName}` : "Select a scenario to view history"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={fetchHistory}
              disabled={isLoading}
              className="h-8 gap-2"
              title="Refresh history"
            >
              <RefreshCw className={`h-4 w-4 ${isLoading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onClose}
              className="h-8 w-8 p-0"
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="flex-1 overflow-auto">
          {error && (
            <div className="rounded-lg bg-red-950/50 border border-red-800 p-3 text-sm text-red-300">
              {error}
            </div>
          )}

          {!error && isLoading && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-slate-400" />
            </div>
          )}

          {!error && !isLoading && pipelines.length === 0 && (
            <div className="text-center py-8 text-slate-400">
              No pipeline history found.
            </div>
          )}

          {!error && !isLoading && pipelines.length > 0 && (
            <div className="space-y-3">
              {pipelines.map((pipeline) => {
                const { icon: StatusIcon, color } = getStatusDisplay(pipeline.status);
                const isCurrent = pipeline.pipeline_id === currentPipelineId;

                return (
                  <div
                    key={pipeline.pipeline_id}
                    className={`rounded-lg border p-3 ${
                      isCurrent
                        ? "border-blue-500/50 bg-blue-950/20"
                        : "border-slate-800 bg-slate-900/50"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <code className="text-xs font-mono text-slate-300 truncate">
                            {pipeline.pipeline_id}
                          </code>
                          {isCurrent && (
                            <Badge variant="outline" className="text-blue-400 border-blue-500/50">
                              Current
                            </Badge>
                          )}
                        </div>
                        <div className="mt-2 flex items-center gap-4 text-xs text-slate-400">
                          <div className="flex items-center gap-1">
                            <Clock className="h-3 w-3" />
                            <span>{formatTimestamp(pipeline.started_at)}</span>
                          </div>
                          {pipeline.completed_at && (
                            <div className="flex items-center gap-1">
                              <CheckCircle className="h-3 w-3" />
                              <span>{formatTimestamp(pipeline.completed_at)}</span>
                            </div>
                          )}
                        </div>
                      </div>
                      <Badge variant="outline" className={`flex items-center gap-1.5 ${color}`}>
                        <StatusIcon className={`h-3 w-3 ${pipeline.status === "running" ? "animate-spin" : ""}`} />
                        {pipeline.status}
                      </Badge>
                    </div>

                    {/* Stage progress */}
                    {pipeline.stage_order && pipeline.stage_order.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-1">
                        {pipeline.stage_order.map((stage) => {
                          const stageResult = pipeline.stages?.[stage];
                          const stageStatus = stageResult?.status ?? "pending";
                          const statusColors: Record<string, string> = {
                            completed: "bg-green-500",
                            failed: "bg-red-500",
                            running: "bg-blue-500",
                            skipped: "bg-slate-600",
                            pending: "bg-slate-700",
                          };
                          return (
                            <div
                              key={stage}
                              className={`px-2 py-0.5 rounded text-xs ${
                                statusColors[stageStatus] ?? "bg-slate-700"
                              } text-white`}
                              title={`${stage}: ${stageStatus}`}
                            >
                              {stage}
                            </div>
                          );
                        })}
                      </div>
                    )}

                    {/* Error message if failed */}
                    {pipeline.error && (
                      <div className="mt-2 text-xs text-red-400 truncate" title={pipeline.error}>
                        {pipeline.error}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>,
    document.body
  );
}
