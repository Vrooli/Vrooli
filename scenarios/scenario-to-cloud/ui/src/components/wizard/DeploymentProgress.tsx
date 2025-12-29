import { CheckCircle2, Circle, Loader2, XCircle, WifiOff } from "lucide-react";
import { Alert } from "../ui/alert";
import { Card, CardContent } from "../ui/card";
import { useDeploymentProgress } from "../../hooks/useDeploymentProgress";
import { type StepStatus, DEPLOYMENT_STEPS } from "../../types/progress";

interface DeploymentProgressProps {
  deploymentId: string;
  onComplete: (success: boolean, error?: string) => void;
}

function StepIcon({ status }: { status: StepStatus }) {
  switch (status) {
    case "completed":
      return <CheckCircle2 className="h-4 w-4 text-emerald-400" />;
    case "running":
      return <Loader2 className="h-4 w-4 text-blue-400 animate-spin" />;
    case "failed":
      return <XCircle className="h-4 w-4 text-red-400" />;
    default:
      return <Circle className="h-4 w-4 text-slate-600" />;
  }
}

function getStepStatus(
  progress: { steps: Array<{ id: string; status: StepStatus }> } | null,
  stepId: string,
): StepStatus {
  if (!progress?.steps) return "pending";
  const step = progress.steps.find((s) => s.id === stepId);
  return step?.status ?? "pending";
}

export function DeploymentProgress({ deploymentId, onComplete }: DeploymentProgressProps) {
  const { progress, isConnected, connectionError } = useDeploymentProgress(deploymentId, {
    onComplete,
  });

  const progressPercent = progress?.progress ?? 0;
  const currentTitle = progress?.currentStepTitle ?? "Initializing...";
  const isComplete = progress?.isComplete ?? false;

  return (
    <div className="space-y-6">
      {/* Error display - shown prominently above progress */}
      {progress?.error && (
        <Alert variant="error" title="Deployment Failed">
          {progress.error}
        </Alert>
      )}

      <Card>
        <CardContent className="py-6">
          {/* Connection status - only show when not complete */}
          {connectionError && !isComplete && (
            <div className="flex items-center gap-2 text-amber-400 text-sm mb-4">
              <WifiOff className="h-4 w-4" />
              {connectionError}
            </div>
          )}

          {/* Progress bar */}
          <div className="mb-4">
            <div className="w-full bg-slate-700 rounded-full h-3 overflow-hidden">
              <div
                className={`h-3 rounded-full transition-all duration-500 ease-out ${
                  progress?.error
                    ? "bg-gradient-to-r from-red-600 to-red-500"
                    : "bg-gradient-to-r from-blue-500 to-emerald-500"
                }`}
                style={{ width: `${Math.min(progressPercent, 100)}%` }}
              />
            </div>
          </div>

          {/* Current step info */}
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-2">
              {!isComplete && <Loader2 className="h-4 w-4 text-blue-400 animate-spin" />}
              {isComplete && !progress?.error && (
                <CheckCircle2 className="h-4 w-4 text-emerald-400" />
              )}
              {progress?.error && <XCircle className="h-4 w-4 text-red-400" />}
              <span className="text-sm font-medium text-slate-200">{currentTitle}</span>
            </div>
            <span className="text-sm text-slate-400 font-mono">
              {Math.round(progressPercent)}%
            </span>
          </div>

          {/* Step list */}
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {DEPLOYMENT_STEPS.map((step) => {
              const status = getStepStatus(progress, step.id);
              return (
                <div
                  key={step.id}
                  className={`flex items-center gap-3 text-sm py-1 ${
                    status === "completed"
                      ? "text-slate-500"
                      : status === "running"
                        ? "text-slate-200 font-medium"
                        : status === "failed"
                          ? "text-red-400"
                          : "text-slate-600"
                  }`}
                >
                  <StepIcon status={status} />
                  <span>{step.title}</span>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>

      {/* Connection indicator - only show when actively streaming */}
      {!isComplete && (
        <div className="flex items-center justify-center gap-2 text-xs text-slate-500">
          <div
            className={`w-2 h-2 rounded-full ${
              isConnected ? "bg-emerald-400" : "bg-amber-400"
            }`}
          />
          {isConnected ? "Connected" : "Reconnecting..."}
        </div>
      )}
    </div>
  );
}
