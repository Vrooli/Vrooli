import { memo, useEffect, useMemo } from "react";
import { Package } from "lucide-react";
import { useScenariosStore } from "../../../../stores";
import type { Scenario } from "../../../../types";
import { scenariosService } from "../../../../services";
import { CollectionList as BaseCollectionList, type CollectionListProps } from "@vrooli/react-component-library/CollectionList/1.0.0";

function CollectionList<T>(props: CollectionListProps<T>) { return <BaseCollectionList {...props} virtualize />; }
import { matchesSearch } from "./useSidebarSearch";
import { SidebarEmptyState } from "./SidebarEmptyState";
import type { ScenarioFilters, SortConfig } from "./types";
import { ScenarioSummaryCard } from "../../../../components/scenario/scenario-summary-card";

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
  return <CollectionList
    items={filtered}
    getKey={(scenario) => scenario.name}
    label="Scenarios"
    onOpen={(scenario) => onItemClick(`scenario/${scenario.name}`)}
    selection={{ mode: "none", enterOn: ["shortcut"] }}
    actions={[
      { id: "open", label: "Open", onSelect: (selected) => { const scenario = selected[0]; if (scenario) onItemClick(`scenario/${scenario.name}`); } },
      { id: "restart", label: "Restart", bulk: true, disabled: (scenario) => scenario.status === "running" ? false : "Only running scenarios can be restarted", onSelect: async (selected) => { for (const scenario of selected) await scenariosService.restart(scenario.name); } },
    ]}
    renderItem={(scenario) => <ScenarioSummaryCard scenario={scenario} />}
    className="space-y-1.5"
  />;
}

export const ScenariosTab = memo(ScenariosTabImpl);
