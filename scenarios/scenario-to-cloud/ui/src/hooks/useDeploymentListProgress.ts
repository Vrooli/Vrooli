import { useState, useEffect, useRef, useCallback } from "react";
import { getDeployment, type DeploymentStatus } from "../lib/api";

interface DeploymentProgress {
  progress_step?: string;
  progress_percent: number;
  status: DeploymentStatus;
}

/**
 * Hook to track progress for multiple in-progress deployments in a list view.
 * Polls the API every 2 seconds for each in-progress deployment to get updated progress.
 */
export function useDeploymentListProgress(deploymentIds: string[]) {
  const [progressMap, setProgressMap] = useState<Record<string, DeploymentProgress>>({});
  const [isPolling, setIsPolling] = useState(false);

  // Track which deployments we're polling and previous IDs to detect changes
  const previousIdsRef = useRef<string[]>([]);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchProgress = useCallback(async (ids: string[]) => {
    if (ids.length === 0) return;

    const updates: Record<string, DeploymentProgress> = {};

    await Promise.all(
      ids.map(async (id) => {
        try {
          const res = await getDeployment(id);
          const deployment = res.deployment;
          updates[id] = {
            progress_step: deployment.progress_step,
            progress_percent: deployment.progress_percent ?? 0,
            status: deployment.status,
          };
        } catch {
          // Ignore errors for individual deployments - they might have been deleted
        }
      })
    );

    setProgressMap((prev) => ({ ...prev, ...updates }));
  }, []);

  useEffect(() => {
    // Clear interval when no deployments to track
    if (deploymentIds.length === 0) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      setIsPolling(false);
      return;
    }

    // Check if deploymentIds changed
    const idsChanged =
      deploymentIds.length !== previousIdsRef.current.length ||
      deploymentIds.some((id, i) => id !== previousIdsRef.current[i]);

    if (idsChanged) {
      previousIdsRef.current = deploymentIds;

      // Fetch immediately on change
      fetchProgress(deploymentIds);
    }

    // Set up polling if not already running
    if (!intervalRef.current) {
      setIsPolling(true);
      intervalRef.current = setInterval(() => {
        fetchProgress(deploymentIds);
      }, 2000); // Poll every 2 seconds
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [deploymentIds, fetchProgress]);

  // Pause polling when tab is hidden to save resources
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        // Tab is hidden - pause polling
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
        setIsPolling(false);
      } else {
        // Tab is visible - resume polling if we have deployments to track
        if (deploymentIds.length > 0 && !intervalRef.current) {
          fetchProgress(deploymentIds);
          intervalRef.current = setInterval(() => {
            fetchProgress(deploymentIds);
          }, 2000);
          setIsPolling(true);
        }
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [deploymentIds, fetchProgress]);

  return { progressMap, isPolling };
}
