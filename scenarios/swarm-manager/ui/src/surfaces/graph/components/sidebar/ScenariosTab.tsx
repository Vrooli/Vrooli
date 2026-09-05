import { memo, useEffect, useMemo } from "react";
import { Activity, AlertTriangle, CircleCheck, Package } from "lucide-react";
import { useScenariosStore } from "../../../../stores";
import type { Scenario } from "../../../../types";
import { matchesSearch } from "./useSidebarSearch";
import { SidebarEmptyState } from "./SidebarEmptyState";
import type { ScenarioFilters, SortConfig } from "./types";

interface ScenariosTabProps {
  searchQuery: string;
  filters: ScenarioFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

function remediationState(scenario: Scenario): string {
  return scenario.health?.remediation?.[0]?.state ?? "none";
}

function sortedScenarios(scenarios: Scenario[], sort: SortConfig) {
  const direction = sort.direction === "asc" ? 1 : -1;
  return [...scenarios].sort((left, right) => {
    if (sort.field === "alphabetical") return left.displayName.localeCompare(right.displayName) * direction;
    if (sort.field === "status") return left.status.localeCompare(right.status) * direction;
    if (sort.field === "recency") return (left.health?.observedAt ?? "").localeCompare(right.health?.observedAt ?? "") * -direction;
    return (left.priority - right.priority) * direction || left.name.localeCompare(right.name);
  });
}

function healthLabel(scenario: Scenario) {
  const state = scenario.health?.evidenceState ?? "unavailable";
  if (state === "fresh") return { label: "Health current", className: "text-emerald-300", Icon: CircleCheck };
  if (state === "stale") return { label: "Health stale", className: "text-amber-300", Icon: AlertTriangle };
  if (state === "no_evidence") return { label: "No test evidence", className: "text-slate-400", Icon: Activity };
  return { label: "Health unavailable", className: "text-amber-300", Icon: AlertTriangle };
}

function ScenarioRow({ scenario, onItemClick }: { scenario: Scenario; onItemClick: (nodeId: string) => void }) {
  const health = healthLabel(scenario);
  return (
    <button type="button" onClick={() => onItemClick(`scenario/${scenario.name}`)} className="w-full rounded-lg border border-slate-800/80 bg-slate-900/60 p-2.5 text-left transition-colors hover:border-slate-700 hover:bg-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400/70" aria-label={`Open ${scenario.displayName}`} data-testid={`scenario-row-${scenario.name}`}>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0"><div className="truncate text-sm font-medium text-slate-100">{scenario.displayName}</div><div className="mt-0.5 line-clamp-1 text-xs text-slate-500">{scenario.description || scenario.name}</div></div>
        <span className="shrink-0 rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-300">{scenario.status}</span>
      </div>
      <div className="mt-2 flex items-center justify-between gap-2 text-[11px]">
        <span className="text-slate-500">P{scenario.priority}{scenario.isGreenfield ? " · greenfield" : ""}</span>
        <span className={`inline-flex shrink-0 items-center gap-1 ${health.className}`}><health.Icon className="h-3 w-3" aria-hidden />{health.label}</span>
      </div>
      {scenario.health?.remediation?.length ? <div className="mt-1 text-[11px] text-amber-300">Needs remediation</div> : null}
    </button>
  );
}

function LoadingSkeleton() { return <div className="space-y-1.5">{[1, 2, 3].map((i) => <div key={i} className="h-20 animate-pulse rounded-lg border border-slate-800/80 bg-slate-900/50" />)}</div>; }

function ScenariosTabImpl({ searchQuery, filters, sort, onItemClick, onClearSearch }: ScenariosTabProps) {
  const scenarios = useScenariosStore((state) => state.scenarios);
  const status = useScenariosStore((state) => state.status);
  const error = useScenariosStore((state) => state.error);
  const fetchScenarios = useScenariosStore((state) => state.fetchScenarios);
  useEffect(() => { void fetchScenarios(); }, [fetchScenarios]);
  const filtered = useMemo(() => sortedScenarios(scenarios.filter((scenario) => (
    (!searchQuery || matchesSearch(searchQuery, scenario.name, scenario.displayName, scenario.description ?? ""))
    && (filters.lifecycle.length === 0 || filters.lifecycle.includes(scenario.status))
    && (filters.evidenceStates.length === 0 || filters.evidenceStates.includes(scenario.health?.evidenceState ?? "unavailable"))
    && (filters.remediationStates.length === 0 || filters.remediationStates.includes(remediationState(scenario)))
  )), sort), [filters, scenarios, searchQuery, sort]);
  if (status === "loading" && scenarios.length === 0) return <LoadingSkeleton />;
  if (error && scenarios.length === 0) return <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">Failed to load scenarios: {error.message}</div>;
  if (filtered.length === 0) return <SidebarEmptyState icon={Package} title="No scenarios match." hint="Adjust search or filters to find a scenario." query={searchQuery} onClearSearch={onClearSearch} />;
  return <div className="space-y-1.5" data-testid="scenarios-tab">{filtered.map((scenario) => <ScenarioRow key={scenario.name} scenario={scenario} onItemClick={onItemClick} />)}</div>;
}

export const ScenariosTab = memo(ScenariosTabImpl);
