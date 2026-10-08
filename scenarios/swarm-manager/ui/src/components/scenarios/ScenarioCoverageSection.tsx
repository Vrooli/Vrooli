/**
 * ScenarioCoverageSection
 *
 * Fetches GET /scenarios/{name}/context and surfaces the scenario's full
 * coverage picture to the operator:
 *   - Goals whose derived scope targets this scenario (with rollup)
 *   - Backlog items targeting this scenario but not in a goal's derived scope
 *   - Combined completion rollup across everything
 *
 * This is the surface that answers "what's being done about scenario X?"
 * without requiring the operator to dig through the graph topology lens.
 */

import { Target, FileText } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { DetailSection } from "../detail/DetailSection";
import { EntityLink } from "../ui/entity-link";
import { RollupProgressBar } from "../ui/rollup-progress-bar";
import { GoalProgressCard } from "../goals/GoalProgressCard";
import { ScenarioFixHistorySection } from "./ScenarioFixHistorySection";
import { scenariosService } from "../../services";
import { defaultQueryOptions } from "../../lib";

export interface ScenarioCoverageSectionProps {
  scenarioName: string;
}

/** Full coverage view for a scenario: goals + unassigned items + rollup. */
export function ScenarioCoverageSection({ scenarioName }: ScenarioCoverageSectionProps) {
  const navigate = useNavigate();
  const { data, error, isLoading } = useQuery({
    queryKey: ["scenario-context", scenarioName],
    queryFn: () => scenariosService.getContext(scenarioName),
    enabled: !!scenarioName,
    ...defaultQueryOptions,
  });

  if (isLoading) {
    return (
      <DetailSection title="Associated Goals & Backlog" icon={Target}>
        <p className="text-sm text-slate-500">Loading coverage…</p>
      </DetailSection>
    );
  }

  if (error) {
    return (
      <DetailSection title="Associated Goals & Backlog" icon={Target}>
        <p className="text-sm text-red-300">Failed to load scenario coverage.</p>
      </DetailSection>
    );
  }

  if (!data) return null;

  const empty = data.goals.length === 0 && data.orphanItems.length === 0;

  return (
    <DetailSection
      title="Associated Goals & Backlog"
      icon={Target}
      data-testid="scenario-coverage-section"
    >
      <div className="space-y-5">
        <div className="rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3">
          <div className="text-xs uppercase tracking-[0.18em] text-slate-500">Rollup</div>
          <div className="mt-2 flex flex-wrap gap-3 text-[12px] text-slate-300">
            <span>{data.rollup.total} total</span>
            <span className="text-emerald-400">{data.rollup.completed} done</span>
            <span className="text-purple-400">{data.rollup.inProgress} active</span>
            {data.rollup.failed > 0 && <span className="text-red-400">{data.rollup.failed} failed</span>}
            <span className="text-slate-500">{data.rollup.pending} pending</span>
          </div>
          {data.rollup.total > 0 && (
            <RollupProgressBar rollup={data.rollup} barHeight="h-1.5" className="mt-3" />
          )}
        </div>

        {empty && (
          <p className="text-sm text-slate-400" data-testid="scenario-coverage-empty">
            No goals or backlog items target this scenario yet. Consider
            creating a goal or backlog item once there is concrete work to
            pursue.
          </p>
        )}

        {data.goals.length > 0 && (
          <div className="space-y-2" data-testid="scenario-coverage-goals">
            <div className="flex items-baseline justify-between">
              <h3 className="text-sm font-semibold text-slate-100">Goals</h3>
              <span className="rounded-full border border-slate-700/80 bg-slate-900/70 px-2 py-0.5 text-[11px] text-slate-400">
                {data.goals.length}
              </span>
            </div>
            <div className="grid min-w-0 gap-2">
              {data.goals.map((goal) => (
                <GoalProgressCard
                  key={goal.name}
                  title={goal.title || goal.name}
                  subtitle={`${goal.status} · ${goal.name}`}
                  priority={goal.priority}
                  completed={goal.rollup.completed}
                  total={goal.rollup.total}
                  inProgress={goal.rollup.inProgress}
                  failed={goal.rollup.failed}
                  pending={goal.rollup.pending}
                  onOpen={() => navigate(`/goals/${encodeURIComponent(goal.name)}`)}
                  data-testid={`scenario-coverage-goal-${goal.name}`}
                />
              ))}
            </div>
          </div>
        )}

        <ScenarioFixHistorySection fixes={data.fixes} />

        {data.orphanItems.length > 0 && (
          <div className="space-y-2" data-testid="scenario-coverage-orphans">
            <div className="flex items-baseline justify-between">
              <h3 className="text-sm font-semibold text-slate-100">
                Orphan items
                <span className="ml-2 text-xs font-normal text-slate-500">
                  (targeting but not in any goal scope)
                </span>
              </h3>
              <span className="rounded-full border border-slate-700/80 bg-slate-900/70 px-2 py-0.5 text-[11px] text-slate-400">
                {data.orphanItems.length}
              </span>
            </div>
            <div className="grid gap-2">
              {data.orphanItems.map((orphan) => (
                <div
                  key={`${orphan.kind}/${orphan.name}`}
                  className="flex items-start justify-between gap-3 rounded-2xl border border-slate-800/80 bg-slate-900/55 p-3"
                >
                  <div className="flex min-w-0 items-start gap-2">
                    <FileText className="mt-0.5 h-4 w-4 shrink-0 text-slate-500" />
                    <div className="min-w-0">
                      <EntityLink
                        entityType="backlog"
                        kind={orphan.kind}
                        name={orphan.name}
                        label={orphan.title || `${orphan.kind}/${orphan.name}`}
                      />
                      <p className="mt-1 truncate text-[11px] text-slate-500">
                        {orphan.kind}/{orphan.name}
                      </p>
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-col items-end gap-1 text-[11px] text-slate-500">
                    <span>{orphan.status}</span>
                    {orphan.priority > 0 && <span>P{orphan.priority}</span>}
                    {orphan.archivedAt && (
                      <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-amber-300">
                        Archived
                      </span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </DetailSection>
  );
}
