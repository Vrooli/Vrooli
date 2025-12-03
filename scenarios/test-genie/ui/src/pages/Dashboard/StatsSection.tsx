import { ArrowRight } from "lucide-react";
import { Button } from "../../components/ui/button";
import { StatusPill } from "../../components/cards/StatusPill";
import { selectors } from "../../consts/selectors";
import { formatRelative } from "../../lib/formatters";
import type { SuiteExecutionResult, SuiteRequest } from "../../lib/api";
import type { CatalogStats } from "../../types";

interface StatsSectionProps {
  queueMetrics: Array<{ label: string; value: number }>;
  catalogStats: CatalogStats;
  heroExecution?: SuiteExecutionResult;
  actionableRequest?: SuiteRequest | null;
  lastFailedExecution?: SuiteExecutionResult;
  onViewQueue: () => void;
  onViewHistory: () => void;
  onViewScenarios: () => void;
  onRerunExecution: () => void;
}

export function StatsSection({
  queueMetrics,
  catalogStats,
  heroExecution,
  actionableRequest,
  lastFailedExecution,
  onViewQueue,
  onViewHistory,
  onViewScenarios,
  onRerunExecution
}: StatsSectionProps) {
  return (
    <section
      className="rounded-2xl border border-white/10 bg-gradient-to-br from-white/[0.05] to-white/[0.01] p-6"
      data-testid={selectors.dashboard.stats}
    >
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-xs uppercase tracking-[0.25em] text-slate-400">Overview</p>
          <h2 className="mt-2 text-2xl font-semibold">Current status</h2>
          <p className="mt-2 text-sm text-slate-300">
            Quick view of your test queue, recent runs, and scenario health.
          </p>
        </div>
      </div>

      <div className="mt-6 grid gap-4 md:grid-cols-3">
        {/* Queue Health */}
        <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Test queue</p>
          <div className="mt-3 grid grid-cols-2 gap-2">
            {queueMetrics.slice(0, 4).map((metric) => (
              <div key={metric.label} className="text-center">
                <p className="text-2xl font-semibold">{metric.value}</p>
                <p className="text-xs text-slate-400">{metric.label}</p>
              </div>
            ))}
          </div>
          <div className="mt-4">
            <Button size="sm" variant="outline" onClick={onViewQueue}>
              View queue
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </article>

        {/* Latest Execution */}
        <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Latest run</p>
          {heroExecution ? (
            <>
              <div className="mt-3 flex items-center justify-between">
                <h3 className="text-lg font-semibold">{heroExecution.scenarioName}</h3>
                <StatusPill status={heroExecution.success ? "passed" : "failed"} />
              </div>
              <p className="mt-2 text-sm text-slate-300">
                {heroExecution.phaseSummary.passed}/{heroExecution.phaseSummary.total} phases ·{" "}
                {formatRelative(heroExecution.completedAt)}
              </p>
              <div className="mt-4 flex gap-2">
                <Button size="sm" variant="outline" onClick={onRerunExecution}>
                  Rerun
                </Button>
                <Button size="sm" variant="outline" onClick={onViewHistory}>
                  History
                </Button>
              </div>
            </>
          ) : (
            <>
              <p className="mt-3 text-sm text-slate-400">No runs recorded yet</p>
              <p className="mt-2 text-sm text-slate-400">
                Run tests to start tracking your coverage
              </p>
            </>
          )}
        </article>

        {/* Scenario Overview */}
        <article className="rounded-2xl border border-white/5 bg-black/30 p-5">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-400">Scenarios</p>
          <div className="mt-3 grid grid-cols-2 gap-2">
            <div className="text-center">
              <p className="text-2xl font-semibold">{catalogStats.tracked}</p>
              <p className="text-xs text-slate-400">Tracked</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-semibold">{catalogStats.failing}</p>
              <p className="text-xs text-slate-400">Failing</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-semibold">{catalogStats.pending}</p>
              <p className="text-xs text-slate-400">Pending</p>
            </div>
            <div className="text-center">
              <p className="text-2xl font-semibold">{catalogStats.idle}</p>
              <p className="text-xs text-slate-400">No runs</p>
            </div>
          </div>
          <div className="mt-4">
            <Button size="sm" variant="outline" onClick={onViewScenarios}>
              View all
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </article>
      </div>
    </section>
  );
}
