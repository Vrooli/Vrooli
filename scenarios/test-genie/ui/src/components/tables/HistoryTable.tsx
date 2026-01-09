import { useState, useMemo } from "react";
import { Search } from "lucide-react";
import { Button } from "../ui/button";
import { StatusPill } from "../cards/StatusPill";
import { PhaseResultCard } from "../cards/PhaseResultCard";
import { selectors } from "../../consts/selectors";
import { formatRelative, formatDuration } from "../../lib/formatters";
import type { SuiteExecutionResult } from "../../lib/api";

interface HistoryTableProps {
  executions: SuiteExecutionResult[];
  onRerunClick: (execution: SuiteExecutionResult) => void;
  onScenarioClick: (scenarioName: string) => void;
  isLoading?: boolean;
}

export function HistoryTable({
  executions,
  onRerunClick,
  onScenarioClick,
  isLoading
}: HistoryTableProps) {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<"all" | "passed" | "failed">("all");

  const filtered = useMemo(() => {
    let result = executions;

    // Filter by search
    const trimmed = search.trim().toLowerCase();
    if (trimmed) {
      result = result.filter((e) => e.scenarioName.toLowerCase().includes(trimmed));
    }

    // Filter by status
    if (statusFilter === "passed") {
      result = result.filter((e) => e.success);
    } else if (statusFilter === "failed") {
      result = result.filter((e) => !e.success);
    }

    return result;
  }, [executions, search, statusFilter]);

  return (
    <div data-testid={selectors.runs.historyTable}>
      <div className="mb-4 flex flex-wrap items-center gap-4">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            className="w-full rounded-full border border-white/10 bg-black/40 pl-9 pr-4 py-2 text-sm text-white focus:outline-none focus:ring-2 focus:ring-cyan-400"
            placeholder="Search by scenario..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="flex gap-1 rounded-full border border-white/10 bg-black/30 p-1">
          {(["all", "passed", "failed"] as const).map((status) => (
            <button
              key={status}
              type="button"
              onClick={() => setStatusFilter(status)}
              className={`rounded-full px-4 py-1.5 text-sm capitalize transition ${
                statusFilter === status
                  ? "bg-white/10 text-white"
                  : "text-slate-400 hover:text-white"
              }`}
            >
              {status}
            </button>
          ))}
        </div>
        {search && (
          <Button variant="outline" size="sm" onClick={() => setSearch("")}>
            Clear
          </Button>
        )}
      </div>

      <div className="space-y-4">
        {isLoading && filtered.length === 0 && (
          <p className="text-sm text-slate-400">Loading test history...</p>
        )}
        {!isLoading && filtered.length === 0 && (
          <p className="text-sm text-slate-400">
            {search
              ? `No runs match "${search}"`
              : statusFilter !== "all"
                ? `No ${statusFilter} runs found`
                : "No test runs recorded yet. Run tests to populate this history."}
          </p>
        )}
        {filtered.map((execution) => (
          <article
            key={execution.executionId}
            className="rounded-2xl border border-white/5 bg-black/30 p-5"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs uppercase tracking-wide text-slate-400">
                  {execution.preset ? `${execution.preset} preset` : "Custom phases"}
                </p>
                <h3
                  className="text-xl font-semibold cursor-pointer hover:text-cyan-400"
                  onClick={() => onScenarioClick(execution.scenarioName)}
                >
                  {execution.scenarioName}
                </h3>
              </div>
              <StatusPill status={execution.success ? "passed" : "failed"} />
            </div>

            <div className="mt-3 flex flex-wrap gap-6 text-sm text-slate-300">
              <span>
                {execution.phaseSummary.passed}/{execution.phaseSummary.total} phases passed
              </span>
              <span>Duration: {formatDuration(execution.phaseSummary.durationSeconds)}</span>
              <span>Completed {formatRelative(execution.completedAt)}</span>
              {execution.phaseSummary.observationCount > 0 && (
                <span>{execution.phaseSummary.observationCount} observation(s)</span>
              )}
            </div>

            {execution.phases.length > 0 && (
              <div className="mt-4 grid gap-2 md:grid-cols-2 lg:grid-cols-3">
                {execution.phases.slice(0, 6).map((phase) => (
                  <PhaseResultCard
                    key={`${execution.executionId}-${phase.name}`}
                    phase={phase}
                  />
                ))}
              </div>
            )}

            <div className="mt-4 flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() => onRerunClick(execution)}
              >
                Rerun tests
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="border-transparent text-slate-200 hover:border-white/40"
                onClick={() => onScenarioClick(execution.scenarioName)}
              >
                View scenario
              </Button>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}
