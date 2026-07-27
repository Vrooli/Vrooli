/**
 * Hook for managing pipeline investigations and fix tasks.
 */

import { useState, useCallback, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createTask,
  listTasks,
  getTask,
  stopTask,
  getAgentManagerStatus,
} from "../lib/api";
import {
  InvestigationStatus,
  TaskType,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/tasks_pb";
import type { CreateTaskInput } from "../lib/api/tasks";

/**
 * Hook to fetch the agent-manager status.
 */
export function useAgentManagerStatus() {
  return useQuery({
    queryKey: ["agent-manager-status"],
    queryFn: getAgentManagerStatus,
    staleTime: 30000, // Cache for 30s
    retry: false, // Don't retry if agent-manager is down
  });
}

/**
 * Hook to fetch tasks for a pipeline.
 */
export function useTasks(pipelineId: string | null, limit = 50) {
  return useQuery({
    queryKey: ["pipeline-tasks", pipelineId, limit],
    queryFn: async () => {
      if (!pipelineId) return [];
      return listTasks(pipelineId, limit);
    },
    enabled: !!pipelineId,
    refetchInterval: (query) => {
      const data = query.state.data;
      // Poll frequently if there's a running task
      const hasRunning = data?.some(
        (task) =>
          task.status === InvestigationStatus.PENDING ||
          task.status === InvestigationStatus.RUNNING,
      );
      return hasRunning ? 3000 : false;
    },
  });
}

/**
 * Hook to fetch a single task.
 */
export function useTaskDetails(
  pipelineId: string | null,
  taskId: string | null,
) {
  return useQuery({
    queryKey: ["pipeline-task", pipelineId, taskId],
    queryFn: async () => {
      if (!pipelineId || !taskId) return null;
      return getTask(pipelineId, taskId);
    },
    enabled: !!pipelineId && !!taskId,
    refetchInterval: (query) => {
      const data = query.state.data;
      // Poll frequently while running
      if (
        data?.status === InvestigationStatus.PENDING ||
        data?.status === InvestigationStatus.RUNNING
      ) {
        return 2000;
      }
      return false;
    },
  });
}

/**
 * Hook to create a new task (investigate or fix).
 */
export function useCreateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      pipelineId,
      request,
    }: {
      pipelineId: string;
      request: CreateTaskInput;
    }) => {
      return createTask(pipelineId, request);
    },
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["pipeline-tasks", variables.pipelineId],
      });
    },
  });
}

/**
 * Hook to stop a running task.
 */
export function useStopTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      pipelineId,
      taskId,
    }: {
      pipelineId: string;
      taskId: string;
    }) => {
      return stopTask(pipelineId, taskId);
    },
    onSuccess: (_, variables) => {
      void queryClient.invalidateQueries({
        queryKey: ["pipeline-tasks", variables.pipelineId],
      });
      void queryClient.invalidateQueries({
        queryKey: ["pipeline-task", variables.pipelineId, variables.taskId],
      });
    },
  });
}

/**
 * Combined hook for managing tasks on a pipeline.
 * Provides convenience methods for triggering, viewing, and stopping tasks.
 */
export function usePipelineInvestigation(pipelineId: string | null) {
  const queryClient = useQueryClient();
  const [activeTaskId, setActiveTaskId] = useState<string | null>(null);
  const [showReport, setShowReport] = useState(false);

  // Agent manager status
  const agentStatus = useAgentManagerStatus();

  // List of tasks for this pipeline
  const tasksQuery = useTasks(pipelineId);

  // Active task details
  const activeTask = useTaskDetails(pipelineId, activeTaskId);

  // Mutations
  const createMutation = useCreateTask();
  const stopMutation = useStopTask();

  // Reset state when pipeline changes
  useEffect(() => {
    setActiveTaskId(null);
    setShowReport(false);
  }, [pipelineId]);

  // Auto-track latest task (prefer running, fallback to most recent)
  useEffect(() => {
    const tasks = tasksQuery.data ?? [];
    if (tasks.length === 0) {
      if (activeTaskId) {
        setActiveTaskId(null);
      }
      return;
    }

    // Prefer running task
    const running = tasks.find(
      (task) =>
        task.status === InvestigationStatus.PENDING ||
        task.status === InvestigationStatus.RUNNING,
    );
    if (running && running.id !== activeTaskId) {
      setActiveTaskId(running.id);
      return;
    }

    // If no running task and none tracked, track the most recent one
    const hasActive = activeTaskId
      ? tasks.some((task) => task.id === activeTaskId)
      : false;
    if (!activeTaskId || !hasActive) {
      // Tasks are sorted by creation time descending, so first is most recent.
      const firstTask = tasks[0];
      if (firstTask) {
        setActiveTaskId(firstTask.id);
      }
    }
  }, [tasksQuery.data, activeTaskId]);

  // Create new task (investigate or fix)
  const trigger = useCallback(
    async (request: CreateTaskInput) => {
      if (!pipelineId) return;
      const task = await createMutation.mutateAsync({ pipelineId, request });
      setActiveTaskId(task.id);
      return task;
    },
    [pipelineId, createMutation],
  );

  // Stop active task
  const stop = useCallback(async () => {
    if (!pipelineId || !activeTaskId) return;
    await stopMutation.mutateAsync({ pipelineId, taskId: activeTaskId });
  }, [pipelineId, activeTaskId, stopMutation]);

  // Create a fix task from a completed investigation
  const triggerFix = useCallback(
    async (
      sourceInvestigationId: string,
      options: Omit<CreateTaskInput, "taskType" | "sourceInvestigationId">,
    ) => {
      if (!pipelineId) return;
      const task = await createMutation.mutateAsync({
        pipelineId,
        request: {
          ...options,
          taskType: TaskType.FIX,
          sourceInvestigationId,
        },
      });
      // Track the new fix task
      setActiveTaskId(task.id);
      return task;
    },
    [pipelineId, createMutation],
  );

  // View task report
  const viewReport = useCallback((taskId: string) => {
    setActiveTaskId(taskId);
    setShowReport(true);
  }, []);

  // Close report modal
  const closeReport = useCallback(() => {
    setShowReport(false);
  }, []);

  // Computed state
  const isAgentAvailable = agentStatus.data?.available ?? false;
  const isRunning =
    activeTask.data?.status === InvestigationStatus.RUNNING ||
    activeTask.data?.status === InvestigationStatus.PENDING;
  const isTriggering = createMutation.isPending;
  const isStopping = stopMutation.isPending;

  return {
    // Agent manager status
    agentStatus: agentStatus.data,
    isAgentAvailable,
    isAgentLoading: agentStatus.isLoading,

    // Tasks list
    tasks: tasksQuery.data ?? [],
    isLoadingTasks: tasksQuery.isLoading,

    // Active task
    activeTask: activeTask.data,
    activeTaskId,
    isRunning,

    // Report modal
    showReport,
    viewReport,
    closeReport,

    // Actions
    trigger,
    triggerFix,
    stop,
    isTriggering,
    isStopping,
    triggerError: createMutation.error,
    stopError: stopMutation.error,

    // Refresh
    refresh: () => {
      void queryClient.invalidateQueries({
        queryKey: ["pipeline-tasks", pipelineId],
      });
      if (activeTaskId) {
        void queryClient.invalidateQueries({
          queryKey: ["pipeline-task", pipelineId, activeTaskId],
        });
      }
    },
  };
}
