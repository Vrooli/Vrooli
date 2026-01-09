import type { SuiteExecutionResult } from "../../lib/api";
import { Button } from "../ui/button";
import { StatusPill } from "./StatusPill";
import { formatRelative, formatDuration } from "../../lib/formatters";

interface ExecutionCardProps {
  execution: SuiteExecutionResult;
  onPrefill?: () => void;
  onFocus?: () => void;
}

export function ExecutionCard({ execution, onPrefill, onFocus }: ExecutionCardProps) {
  return (
    <div className="mt-6 rounded-2xl border border-white/5 bg-black/20 p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Scenario</p>
          <h3 className="text-xl font-semibold">{execution.scenarioName}</h3>
        </div>
        <StatusPill status={execution.success ? "completed" : "failed"} />
      </div>
      <p className="mt-2 text-sm text-slate-400">
        {execution.preset ? `${execution.preset} preset · ` : ""}
        Completed {formatRelative(execution.completedAt)}
      </p>
      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <div className="rounded-xl border border-white/5 p-3 text-sm">
          <p className="text-xs uppercase tracking-wide text-slate-400">Phase summary</p>
          <p className="mt-2 text-base text-white">
            {execution.phaseSummary.passed}/{execution.phaseSummary.total} passed
          </p>
        </div>
        <div className="rounded-xl border border-white/5 p-3 text-sm">
          <p className="text-xs uppercase tracking-wide text-slate-400">Duration</p>
          <p className="mt-2 text-base text-white">
            {formatDuration(execution.phaseSummary.durationSeconds)}
          </p>
        </div>
      </div>
      {(onPrefill || onFocus) && (
        <div className="mt-4 flex flex-wrap gap-2">
          {onPrefill && (
            <Button variant="outline" size="sm" onClick={onPrefill}>
              Rerun tests
            </Button>
          )}
          {onFocus && (
            <Button
              variant="outline"
              size="sm"
              className="border-transparent text-slate-200 hover:border-white/40"
              onClick={onFocus}
            >
              View scenario
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
