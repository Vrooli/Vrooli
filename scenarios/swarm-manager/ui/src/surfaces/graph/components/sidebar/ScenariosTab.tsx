import { useQuery } from "@tanstack/react-query";
import { Package } from "lucide-react";
import { scenariosService } from "../../../../services";
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

function remediationState(scenario: Awaited<ReturnType<typeof scenariosService.list>>[number]): string {
  return scenario.health?.remediation?.[0]?.state ?? "none";
}

function sortedScenarios(scenarios: Awaited<ReturnType<typeof scenariosService.list>>, sort: SortConfig) {
  const direction = sort.direction === "asc" ? 1 : -1;
  return [...scenarios].sort((left, right) => {
    if (sort.field === "alphabetical") return left.displayName.localeCompare(right.displayName) * direction;
    if (sort.field === "status") return left.status.localeCompare(right.status) * direction;
    if (sort.field === "recency") return (left.health?.observedAt ?? "").localeCompare(right.health?.observedAt ?? "") * -direction;
    return (left.priority - right.priority) * direction || left.name.localeCompare(right.name);
  });
}

export function ScenariosTab({ searchQuery, filters, sort, onItemClick, onClearSearch }: ScenariosTabProps) {
  const { data = [], isLoading, error } = useQuery({ queryKey: ["scenarios"], queryFn: () => scenariosService.list(), staleTime: 15_000 });
  const filtered = sortedScenarios(data.filter((scenario) => (
    (!searchQuery || matchesSearch(searchQuery, scenario.name, scenario.displayName, scenario.description ?? ""))
    && (filters.lifecycle.length === 0 || filters.lifecycle.includes(scenario.status))
    && (filters.evidenceStates.length === 0 || filters.evidenceStates.includes(scenario.health?.evidenceState ?? "unavailable"))
    && (filters.remediationStates.length === 0 || filters.remediationStates.includes(remediationState(scenario)))
  )), sort);

  if (isLoading) return <div className="p-3 text-sm text-slate-400">Loading scenarios…</div>;
  if (error) return <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">Failed to load scenarios: {error.message}</div>;
  if (filtered.length === 0) return <SidebarEmptyState icon={Package} title="No scenarios match." hint="Adjust search or filters to find a scenario." query={searchQuery} onClearSearch={onClearSearch} />;

  return <div className="space-y-1.5" data-testid="scenarios-tab">{filtered.map((scenario) => {
    const health = scenario.health;
    return <button key={scenario.name} type="button" onClick={() => onItemClick(`scenario/${scenario.name}`)} className="w-full rounded-lg border border-slate-800/80 bg-slate-900/60 p-2.5 text-left transition-colors hover:border-slate-700 hover:bg-slate-900" aria-label={`Open ${scenario.displayName}`} data-testid={`scenario-row-${scenario.name}`}>
      <div className="flex items-center justify-between gap-2"><span className="truncate text-sm font-medium text-slate-100">{scenario.displayName}</span><span className="rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-400">{scenario.status}</span></div>
      <div className="mt-1 flex items-center justify-between text-xs text-slate-500"><span className="truncate">{scenario.name}</span><span className={health?.evidenceState === "fresh" ? "text-emerald-300" : "text-amber-300"}>{health?.evidenceState ?? "unavailable"}</span></div>
    </button>;
  })}</div>;
}
