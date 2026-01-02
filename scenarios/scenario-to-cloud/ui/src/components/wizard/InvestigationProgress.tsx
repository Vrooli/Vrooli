import { Loader2, CheckCircle2, XCircle, StopCircle, Bot } from "lucide-react";
import { Card, CardContent } from "../ui/card";
import { Button } from "../ui/button";
import { useDeploymentInvestigation } from "../../hooks/useInvestigation";
import type { InvestigationStatus } from "../../types/investigation";

interface InvestigationProgressProps {
  deploymentId: string;
  investigationId?: string;
  onViewReport?: () => void;
}

function StatusIcon({ status }: { status: InvestigationStatus }) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="h-5 w-5 text-emerald-400" />;
    case "failed":
      return <XCircle className="h-5 w-5 text-red-400" />;
    case "cancelled":
      return <StopCircle className="h-5 w-5 text-slate-400" />;
    case "running":
    case "pending":
    default:
      return <Loader2 className="h-5 w-5 text-blue-400 animate-spin" />;
  }
}

function statusLabel(status: InvestigationStatus, isFixApplication: boolean): string {
  if (isFixApplication) {
    switch (status) {
      case "pending":
        return "Preparing fix application...";
      case "running":
        return "Agent applying fixes...";
      case "completed":
        return "Fix application complete";
      case "failed":
        return "Fix application failed";
      case "cancelled":
        return "Fix application cancelled";
      default:
        return "Unknown status";
    }
  }
  switch (status) {
    case "pending":
      return "Preparing investigation...";
    case "running":
      return "Agent investigating VPS...";
    case "completed":
      return "Investigation complete";
    case "failed":
      return "Investigation failed";
    case "cancelled":
      return "Investigation cancelled";
    default:
      return "Unknown status";
  }
}

export function InvestigationProgress({
  deploymentId,
  investigationId,
  onViewReport,
}: InvestigationProgressProps) {
  const {
    activeInvestigation,
    isRunning,
    stop,
    isStopping,
    viewReport,
  } = useDeploymentInvestigation(deploymentId);

  // Use provided investigationId or the active one
  const inv = activeInvestigation;
  if (!inv) {
    return null;
  }

  const status = inv.status;
  const isComplete = status === "completed" || status === "failed" || status === "cancelled";
  const isFailed = status === "failed";
  const isSuccess = status === "completed" && !inv.error_message;

  // Check if this is a fix application (operation_mode starts with "fix-application:")
  const isFixApplication = inv.details?.operation_mode?.startsWith("fix-application") ?? false;

  // For completed status, always show 100%. For failed/cancelled, show actual progress (where it stopped)
  const displayProgress = status === "completed" ? 100 : (inv.progress ?? 0);

  // Determine card styling based on status
  const cardClassName = isFailed
    ? "border-red-500/30 bg-red-500/5"
    : isSuccess
      ? "border-emerald-500/30 bg-emerald-500/5"
      : "border-blue-500/30 bg-blue-500/5";

  return (
    <Card className={cardClassName}>
      <CardContent className="py-4">
        <div className="flex items-center gap-3 mb-4">
          <Bot className={`h-5 w-5 ${isFailed ? "text-red-400" : isSuccess ? "text-emerald-400" : "text-blue-400"}`} />
          <span className="font-medium text-slate-200">
            {isFixApplication ? "Fix Application" : "Deployment Investigation"}
          </span>
          <span className="text-xs text-slate-500 font-mono ml-auto">
            {inv.id.slice(0, 8)}
          </span>
        </div>

        {/* Progress bar */}
        <div className="mb-3">
          <div className="w-full bg-slate-700 rounded-full h-2 overflow-hidden">
            <div
              className={`h-2 rounded-full transition-all duration-500 ease-out ${
                status === "failed"
                  ? "bg-gradient-to-r from-red-600 to-red-500"
                  : status === "cancelled"
                    ? "bg-slate-500"
                    : status === "completed"
                      ? "bg-gradient-to-r from-blue-500 to-emerald-500"
                      : "bg-gradient-to-r from-blue-500 to-blue-400"
              }`}
              style={{ width: `${Math.min(displayProgress, 100)}%` }}
            />
          </div>
        </div>

        {/* Status */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <StatusIcon status={status} />
            <span className={`text-sm ${isFailed ? "text-red-300" : "text-slate-300"}`}>
              {statusLabel(status, isFixApplication)}
            </span>
          </div>
          <span className="text-sm text-slate-400 font-mono">{displayProgress}%</span>
        </div>

        {/* Error message */}
        {inv.error_message && (
          <div className="mt-3 text-sm text-red-400 bg-red-500/10 rounded-md p-2 border border-red-500/20">
            {inv.error_message}
          </div>
        )}

        {/* No findings message for failed investigations/fixes */}
        {isFailed && !inv.error_message && !inv.findings && (
          <div className="mt-3 text-sm text-amber-400 bg-amber-500/10 rounded-md p-2 border border-amber-500/20">
            {isFixApplication
              ? "Fix application failed without details. The agent may have encountered an error."
              : "Investigation failed without details. The agent may have encountered an error."}
          </div>
        )}

        {/* Actions */}
        <div className="flex items-center gap-2 mt-4">
          {/* Show View Report for any completed investigation with findings */}
          {isComplete && inv.findings && (
            <Button
              size="sm"
              onClick={() => {
                viewReport(inv.id);
                onViewReport?.();
              }}
            >
              View Report
            </Button>
          )}

          {/* Show View Details for failed investigations even without findings */}
          {isComplete && !inv.findings && (inv.error_message || inv.details) && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                viewReport(inv.id);
                onViewReport?.();
              }}
            >
              View Details
            </Button>
          )}

          {isRunning && (
            <Button
              size="sm"
              variant="ghost"
              onClick={stop}
              disabled={isStopping}
            >
              {isStopping ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <>
                  <StopCircle className="h-4 w-4 mr-1" />
                  Stop
                </>
              )}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
