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
}

export function useDeploymentProgress(
  deploymentId: string | null,
  options: UseDeploymentProgressOptions = {},
) {
  const [progress, setProgress] = useState<DeploymentProgress | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);
  const { onComplete, onError } = options;

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

    const url = buildApiUrl(`/deployments/${encodeURIComponent(deploymentId)}/progress`, {
      baseUrl: API_BASE,
    });

    // Initialize progress state
    setProgress({
      currentStep: "",
      currentStepTitle: "Initializing...",
      progress: 0,
      steps: getInitialSteps(),
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
          // Update steps up to current step as completed
          let steps = prev?.steps ?? getInitialSteps();
          let foundCurrent = false;
          steps = steps.map((s) => {
            if (s.id === data.step) {
              foundCurrent = true;
              return { ...s, status: "running" as StepStatus };
            }
            if (!foundCurrent) {
              return { ...s, status: "completed" as StepStatus };
            }
            return s;
          });

          return {
            currentStep: data.step,
            currentStepTitle: data.step_title || data.step,
            progress: data.progress,
            steps,
            isComplete: false,
          };
        });
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
          isComplete: true,
        }));
        eventSource.close();
        setIsConnected(false);
        onComplete?.(true);
      } catch {
        // Ignore parse errors
      }
    });

    // Handle error events
    eventSource.addEventListener("error", (e) => {
      try {
        // Check if this is an SSE error event or our custom error event
        if (e instanceof MessageEvent && e.data) {
          const data = JSON.parse(e.data) as ProgressEvent;
          setProgress((prev) => {
            // Reconstruct step states: all steps before failed step = completed, failed step = failed
            let steps = prev?.steps ?? getInitialSteps();
            if (data.step) {
              let foundFailed = false;
              steps = steps.map((s) => {
                if (s.id === data.step) {
                  foundFailed = true;
                  return { ...s, status: "failed" as StepStatus };
                }
                if (!foundFailed) {
                  return { ...s, status: "completed" as StepStatus };
                }
                return s;
              });
            }

            return {
              currentStep: data.step || prev?.currentStep || "",
              currentStepTitle: prev?.currentStepTitle || "",
              progress: data.progress,
              steps,
              error: data.error,
              isComplete: true,
            };
          });
          eventSource.close();
          setIsConnected(false);
          const errorMsg = data.error || "Deployment failed";
          onError?.(errorMsg);
          onComplete?.(false, errorMsg);
        }
      } catch {
        // Connection error, not our custom error event
      }
    });

    return () => {
      eventSource.close();
      eventSourceRef.current = null;
    };
  }, [deploymentId, disconnect, onComplete, onError]);

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
