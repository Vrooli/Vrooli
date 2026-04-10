import { useState, useEffect, useCallback, useRef } from "react";
import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import {
  type DeploymentProgress,
  type ProgressEvent,
  type StepStatus,
  getInitialSteps,
  updateStepStatus,
} from "../types/progress";

const API_BASE = resolveApiBase({ appendSuffix: true });

export interface UseDeploymentProgressOptions {
  onComplete?: (success: boolean, error?: string) => void;
  onError?: (error: string) => void;
  runId?: string | null;
}

export function useDeploymentProgress(
  deploymentId: string | null,
  options: UseDeploymentProgressOptions = {},
) {
  const [progress, setProgress] = useState<DeploymentProgress | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const { onComplete, onError, runId } = options;

  const disconnect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    setIsConnected(false);
  }, []);

  useEffect(() => {
    if (!deploymentId) {
      disconnect();
      setProgress(null);
      return;
    }

    let url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/progress`, {
      baseUrl: API_BASE,
    });
    if (runId) {
      url += `?run_id=${encodeURIComponent(runId)}`;
    }

    // Initialize progress state
    setProgress({
      currentStep: "",
      currentStepTitle: "Initializing...",
      progress: 0,
      steps: getInitialSteps(),
      preflightResult: undefined,
      isComplete: false,
    });
    setConnectionError(null);

    const eventSource = new EventSource(url);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setIsConnected(true);
      setConnectionError(null);
    };

    eventSource.onerror = () => {
      // EventSource automatically reconnects, but we track the error
      setConnectionError("Connection lost, reconnecting...");
      setIsConnected(false);
    };

    // Handle step_started events
    eventSource.addEventListener("step_started", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => ({
          currentStep: data.step,
          currentStepTitle: data.step_title || data.step,
          progress: data.progress,
          steps: updateStepStatus(prev?.steps, data.step, "running"),
          preflightResult: prev?.preflightResult,
          isComplete: false,
        }));
      } catch {
        // Ignore parse errors
      }
    });

    // Handle step_completed events
    eventSource.addEventListener("step_completed", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => ({
          currentStep: prev?.currentStep ?? data.step,
          currentStepTitle: prev?.currentStepTitle ?? data.step_title ?? "",
          progress: data.progress,
          steps: updateStepStatus(prev?.steps, data.step, "completed"),
          preflightResult: prev?.preflightResult,
          isComplete: false,
        }));
      } catch {
        // Ignore parse errors
      }
    });

    // Handle progress_update events (for reconnection)
    eventSource.addEventListener("progress_update", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => {
          let steps = prev?.steps ?? getInitialSteps();
          const stepIndex = steps.findIndex((s) => s.id === data.step);
          if (stepIndex !== -1) {
            steps = steps.map((s, index) => {
              if (index < stepIndex) {
                return { ...s, status: "completed" as StepStatus };
              }
              if (index === stepIndex) {
                return { ...s, status: "running" as StepStatus };
              }
              return s;
            });
          }

          return {
            currentStep: data.step,
            currentStepTitle: data.step_title || data.step,
            progress: data.progress,
            steps,
            preflightResult: prev?.preflightResult,
            isComplete: false,
          };
        });
      } catch {
        // Ignore parse errors
      }
    });

    // Handle preflight_result events
    eventSource.addEventListener("preflight_result", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => ({
          currentStep: prev?.currentStep ?? "",
          currentStepTitle: prev?.currentStepTitle ?? "",
          progress: prev?.progress ?? 0,
          steps: prev?.steps ?? getInitialSteps(),
          preflightResult: data.preflight_result,
          error: prev?.error,
          isComplete: prev?.isComplete ?? false,
        }));
      } catch {
        // Ignore parse errors
      }
    });

    // Handle completed events
    eventSource.addEventListener("completed", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => ({
          currentStep: "completed",
          currentStepTitle: data.message || "Deployment complete",
          progress: 100,
          steps: prev?.steps.map((s) => ({ ...s, status: "completed" as StepStatus })) ?? [],
          preflightResult: prev?.preflightResult,
          isComplete: true,
        }));
        eventSource.close();
        setIsConnected(false);
        onComplete?.(true);
      } catch {
        // Ignore parse errors
      }
    });

    // Handle deployment_error events (renamed from "error" to avoid browser SSE conflict)
    eventSource.addEventListener("deployment_error", (e) => {
      try {
        const data = JSON.parse(e.data) as ProgressEvent;
        setProgress((prev) => {
          // Reconstruct step states: all steps before failed step = completed, failed step = failed
          let steps = prev?.steps ?? getInitialSteps();
          if (data.step) {
            const stepIndex = steps.findIndex((s) => s.id === data.step);
            if (stepIndex !== -1) {
              steps = steps.map((s, index) => {
                if (index < stepIndex) {
                  return { ...s, status: "completed" as StepStatus };
                }
                if (index === stepIndex) {
                  return { ...s, status: "failed" as StepStatus };
                }
                return s;
              });
            }
          }

          return {
            currentStep: data.step || prev?.currentStep || "",
            currentStepTitle: prev?.currentStepTitle || "",
            progress: data.progress,
            steps,
            error: data.error,
            preflightResult: prev?.preflightResult,
            isComplete: true,
          };
        });
        eventSource.close();
        setIsConnected(false);
        const errorMsg = data.error || "Deployment failed";
        onError?.(errorMsg);
        onComplete?.(false, errorMsg);
      } catch {
        // Ignore parse errors
      }
    });

    return () => {
      eventSource.close();
      eventSourceRef.current = null;
    };
  }, [deploymentId, runId, disconnect, onComplete, onError]);

  const reset = useCallback(() => {
    disconnect();
    setProgress(null);
    setConnectionError(null);
  }, [disconnect]);

  return {
    progress,
    isConnected,
    connectionError,
    disconnect,
    reset,
  };
}
