import { useEffect, useMemo, useState } from "react";
import { Card } from "../components/ui/card";
import { executionService } from "../services";
import { useExecutionStore } from "../stores";
import type { ExecutionRecord } from "../types";

const STATUS_GROUPS = [
  { id: "pending", title: "Pending / Scheduled", statuses: ["pending", "scheduled"] as const },
  { id: "running", title: "Running", statuses: ["running"] as const },
  { id: "completed", title: "Completed", statuses: ["completed"] as const },
  { id: "failed", title: "Failed", statuses: ["failed", "canceled"] as const },
];

export function ExecutionPage() {
  const { items, status, error, isRefreshing, fetchExecutions, upsertExecution } = useExecutionStore();
  const [busyId, setBusyId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [modeFilter, setModeFilter] = useState<string>("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [backlogFilter, setBacklogFilter] = useState("");
  const [fromFilter, setFromFilter] = useState("");
  const [toFilter, setToFilter] = useState("");

  useEffect(() => {
    void fetchExecutions();
    const interval = window.setInterval(() => {
      void fetchExecutions({ force: true });
    }, 6000);
    return () => window.clearInterval(interval);
  }, [fetchExecutions]);

  const filteredItems = useMemo(() => {
    return items.filter((item) => {
      if (statusFilter && item.status !== statusFilter) {
        return false;
      }
      if (modeFilter && item.mode !== modeFilter) {
        return false;
      }
      if (sourceFilter && !(item.startedBy ?? "").toLowerCase().includes(sourceFilter.toLowerCase())) {
        return false;
      }
      if (backlogFilter) {
        const backlogValue = `${item.backlogKind}/${item.backlogName}`.toLowerCase();
        if (!backlogValue.includes(backlogFilter.toLowerCase())) {
          return false;
        }
      }
      if (fromFilter) {
        const fromDate = new Date(fromFilter).getTime();
        const createdDate = new Date(item.createdAt).getTime();
        if (!Number.isNaN(fromDate) && !Number.isNaN(createdDate) && createdDate < fromDate) {
          return false;
        }
      }
      if (toFilter) {
        const toDate = new Date(toFilter).getTime();
        const createdDate = new Date(item.createdAt).getTime();
        if (!Number.isNaN(toDate) && !Number.isNaN(createdDate) && createdDate > toDate) {
          return false;
        }
      }
      return true;
    });
  }, [items, statusFilter, modeFilter, sourceFilter, backlogFilter, fromFilter, toFilter]);

  const groups = useMemo(() => {
    return STATUS_GROUPS.map((group) => ({
      ...group,
      items: filteredItems.filter((item) => (group.statuses as readonly string[]).includes(item.status)),
    }));
  }, [filteredItems]);

  const runAction = async (executionId: string, action: "start" | "cancel" | "retry") => {
    setBusyId(executionId);
    try {
      let updated: ExecutionRecord;
      if (action === "start") {
        updated = await executionService.start(executionId);
      } else if (action === "cancel") {
        updated = await executionService.cancel(executionId);
      } else {
        updated = await executionService.retry(executionId);
      }
      upsertExecution(updated);
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="space-y-6" data-testid="execution-page">
      <Card padding="default" centered={false}>
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="text-xl font-semibold text-slate-100">Execution Control</h2>
            <p className="mt-1 text-sm text-slate-300">Track and control backlog execution runs.</p>
          </div>
          <button
            type="button"
            className="rounded border border-slate-500 px-3 py-1 text-sm text-slate-100 hover:border-slate-300"
            onClick={() => void fetchExecutions({ force: true })}
            disabled={status === "loading" || isRefreshing}
          >
            Refresh
          </button>
        </div>
        <div className="mt-4 grid grid-cols-1 gap-2 md:grid-cols-6">
          <input
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            placeholder="Status"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value)}
          />
          <input
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            placeholder="Mode"
            value={modeFilter}
            onChange={(event) => setModeFilter(event.target.value)}
          />
          <input
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            placeholder="Team source"
            value={sourceFilter}
            onChange={(event) => setSourceFilter(event.target.value)}
          />
          <input
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            placeholder="Scenario/Backlog"
            value={backlogFilter}
            onChange={(event) => setBacklogFilter(event.target.value)}
          />
          <input
            type="datetime-local"
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            value={fromFilter}
            onChange={(event) => setFromFilter(event.target.value)}
          />
          <input
            type="datetime-local"
            className="rounded border border-slate-600 bg-slate-900 px-2 py-1 text-xs text-slate-100"
            value={toFilter}
            onChange={(event) => setToFilter(event.target.value)}
          />
        </div>
      </Card>

      {status === "loading" && items.length === 0 ? (
        <Card padding="lg" centered>
          <p className="text-sm text-slate-300">Loading execution runs...</p>
        </Card>
      ) : null}

      {error ? (
        <Card padding="lg" centered>
          <p className="text-sm text-red-300">{error.message}</p>
        </Card>
      ) : null}

      {groups.map((group) => (
        <Card key={group.id} padding="default" centered={false}>
          <h3 className="text-base font-semibold text-slate-100">{group.title}</h3>
          {group.items.length === 0 ? (
            <p className="mt-2 text-sm text-slate-400">No runs.</p>
          ) : (
            <div className="mt-3 space-y-3">
              {group.items.map((item) => (
                <div key={item.executionId} className="rounded border border-slate-700 p-3">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <p className="font-mono text-xs text-slate-300">{item.executionId}</p>
                      <p className="text-sm text-slate-100">
                        {item.backlogKind}/{item.backlogName} ({item.mode})
                      </p>
                      <p className="text-xs text-slate-400">
                        status={item.status}
                        {item.runId ? ` | run=${item.runId}` : ""}
                        {item.taskId ? ` | task=${item.taskId}` : ""}
                      </p>
                      {item.failureReason ? <p className="mt-1 text-xs text-red-300">{item.failureReason}</p> : null}
                    </div>
                    <div className="flex items-center gap-2">
                      {(item.status === "pending" || item.status === "scheduled") && (
                        <button
                          type="button"
                          className="rounded border border-emerald-500 px-2 py-1 text-xs text-emerald-300"
                          disabled={busyId === item.executionId}
                          onClick={() => void runAction(item.executionId, "start")}
                        >
                          Start
                        </button>
                      )}
                      {(item.status === "pending" || item.status === "scheduled" || item.status === "running") && (
                        <button
                          type="button"
                          className="rounded border border-amber-500 px-2 py-1 text-xs text-amber-300"
                          disabled={busyId === item.executionId}
                          onClick={() => void runAction(item.executionId, "cancel")}
                        >
                          Cancel
                        </button>
                      )}
                      {item.status === "failed" && (
                        <button
                          type="button"
                          className="rounded border border-cyan-500 px-2 py-1 text-xs text-cyan-300"
                          disabled={busyId === item.executionId}
                          onClick={() => void runAction(item.executionId, "retry")}
                        >
                          Retry
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>
      ))}
    </div>
  );
}
