import { useCallback, useEffect, useMemo, useState, type MutableRefObject } from "react";
import type { Run, RunDiff, RunEvent, Task } from "../types";
import { ApprovalState, RunStatus } from "../types";
import type { UseRunEventStoreReturn } from "./useRunEventStore";

interface UseSelectedRunControllerOptions {
  runs: Run[];
  tasks: Task[];
  routeRunId?: string;
  isDeselectingRef: MutableRefObject<boolean>;
  onGetRun: (id: string) => Promise<Run>;
  onGetEvents: (id: string, options?: { afterSequence?: bigint }) => Promise<RunEvent[]>;
  onGetDiff: (id: string) => Promise<RunDiff>;
  onGetTask: (id: string) => Promise<Task>;
  runEventStore: UseRunEventStoreReturn;
  wsSubscribe: (runId: string) => void;
  wsUnsubscribe: (runId: string) => void;
}

function shouldLoadDiff(run: Run): boolean {
  return (
    run.status === RunStatus.NEEDS_REVIEW ||
    run.status === RunStatus.COMPLETE ||
    run.approvalState !== ApprovalState.NONE
  );
}

export function useSelectedRunController({
  runs,
  tasks,
  routeRunId,
  isDeselectingRef,
  onGetRun,
  onGetEvents,
  onGetDiff,
  onGetTask,
  runEventStore,
  wsSubscribe,
  wsUnsubscribe,
}: UseSelectedRunControllerOptions) {
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const [diff, setDiff] = useState<RunDiff | null>(null);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [diffLoading, setDiffLoading] = useState(false);
  const [extraTasks, setExtraTasks] = useState<Record<string, Task>>({});

  const taskById = useMemo(
    () => new Map(tasks.map((task) => [task.id, task])),
    [tasks]
  );

  const getTaskById = useCallback(
    (taskId: string) => extraTasks[taskId] ?? taskById.get(taskId) ?? null,
    [extraTasks, taskById]
  );

  const getTaskTitle = useCallback(
    (taskId: string) => getTaskById(taskId)?.title || "Unknown Task",
    [getTaskById]
  );

  const resolvedRuns = useMemo(() => {
    const snapshots = runEventStore.state.runsById;
    return runs.map((run) => {
      const snapshot = snapshots[run.id];
      return snapshot ? ({ ...run, ...snapshot } as Run) : run;
    });
  }, [runs, runEventStore.state.runsById]);

  const syncRunDetails = useCallback(
    async (runId: string) => {
      try {
        const latest = await onGetRun(runId);
        runEventStore.actions.runSnapshotLoaded(latest);
        setSelectedRun((prev) => (prev && prev.id === runId ? ({ ...prev, ...latest } as Run) : prev));

        if (!getTaskById(latest.taskId)) {
          const task = await onGetTask(latest.taskId);
          setExtraTasks((prev) => ({ ...prev, [task.id]: task }));
        }
      } catch (err) {
        console.error("Failed to sync run details:", err);
      }
    },
    [getTaskById, onGetRun, onGetTask, runEventStore.actions]
  );

  const selectedRunId = selectedRun?.id ?? null;
  const events = selectedRunId ? runEventStore.getRunEvents(selectedRunId) : [];

  useEffect(() => {
    if (!selectedRunId) return;
    runEventStore.actions.subscribeRun(selectedRunId);
    wsSubscribe(selectedRunId);
    return () => {
      runEventStore.actions.unsubscribeRun(selectedRunId);
      wsUnsubscribe(selectedRunId);
    };
  }, [selectedRunId, runEventStore.actions, wsSubscribe, wsUnsubscribe]);

  useEffect(() => {
    if (!selectedRunId) return;
    const snapshot = runEventStore.state.runsById[selectedRunId];
    if (snapshot) {
      setSelectedRun((prev) => (prev && prev.id === selectedRunId ? ({ ...prev, ...snapshot } as Run) : prev));
    }
  }, [selectedRunId, runEventStore.state.runsById]);

  const loadRunDetails = useCallback(
    async (run: Run) => {
      setSelectedRun(run);
      setDiff(null);
      runEventStore.actions.runSnapshotLoaded(run);
      runEventStore.actions.subscribeRun(run.id);
      void syncRunDetails(run.id);

      setEventsLoading(true);
      try {
        const evts = await onGetEvents(run.id);
        runEventStore.actions.eventsGapFilled(run.id, evts || []);
      } catch (err) {
        console.error("Failed to load events:", err);
      } finally {
        setEventsLoading(false);
      }

      if (shouldLoadDiff(run)) {
        setDiffLoading(true);
        try {
          const diffResult = await onGetDiff(run.id);
          setDiff(diffResult);
        } catch (err) {
          console.error("Failed to load diff:", err);
        } finally {
          setDiffLoading(false);
        }
      }
    },
    [onGetEvents, onGetDiff, runEventStore.actions, syncRunDetails]
  );

  useEffect(() => {
    if (!selectedRun) return;
    const updatedRun = resolvedRuns.find((run) => run.id === selectedRun.id);
    if (updatedRun && updatedRun !== selectedRun) {
      setSelectedRun(updatedRun);
    }
  }, [resolvedRuns, selectedRun]);

  useEffect(() => {
    if (isDeselectingRef.current) return;
    if (!routeRunId || resolvedRuns.length === 0) return;
    if (selectedRunId === routeRunId) return;
    const run = resolvedRuns.find((candidate) => candidate.id === routeRunId);
    if (run) {
      loadRunDetails(run);
    }
  }, [isDeselectingRef, routeRunId, resolvedRuns, selectedRunId, loadRunDetails]);

  return {
    selectedRun,
    setSelectedRun,
    selectedRunId,
    diff,
    events,
    eventsLoading,
    diffLoading,
    resolvedRuns,
    getTaskById,
    getTaskTitle,
    loadRunDetails,
  };
}
