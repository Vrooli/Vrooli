import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Check, Loader2, LockKeyhole } from "lucide-react";
import { fetchV2Closure, fetchV2Scenarios } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "../ui/button";

interface Props { selected: Set<string>; onToggle: (name: string) => void; }

// StepSelectScenarios is the V2 entry point. Resources are visible only as
// manifest-derived consequences of a scenario choice, never as the primary
// configuration authority.
export function StepSelectScenarios({ selected, onToggle }: Props) {
  const [search, setSearch] = useState("");
  const [selectionFilter, setSelectionFilter] = useState("all");
  const { data, isLoading, error } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const closure = useQuery({ queryKey: ["v2-closure"], queryFn: fetchV2Closure });
  const scenarios = data?.scenarios ?? [];
  const normalizedSearch = search.trim().toLowerCase();
  const visibleScenarios = useMemo(() => scenarios.filter((scenario) => {
    const isSelected = scenario.system_required || selected.has(scenario.name);
    const matchesSelection = selectionFilter === "all" || (selectionFilter === "selected" ? isSelected : !isSelected);
    if (!matchesSelection) return false;
    if (!normalizedSearch) return true;
    return scenario.name.toLowerCase().includes(normalizedSearch) || (scenario.description ?? "").toLowerCase().includes(normalizedSearch);
  }), [normalizedSearch, scenarios, selected, selectionFilter]);
  const impliedResources = closure.data?.resources ?? [];
  const pulledIn = (closure.data?.scenarios ?? []).filter((scenario) => !scenario.direct);
  return <div data-testid="step-select-scenarios">
    <h1 className="text-xl font-semibold sm:text-2xl">Choose capabilities</h1>
    <p className="mt-2 text-sm text-muted sm:text-base">Choose the scenarios this installation should run. Required system capabilities stay enabled.</p>
    {isLoading && <div className="flex items-center gap-2 py-16 text-muted" role="status"><Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" /> Loading scenarios…</div>}
    {error && <p data-testid="catalog-error" className="py-10 text-danger" role="alert">Unable to load scenario manifests. Check the control plane and try again.</p>}
    {!isLoading && !error && <>
      <div className="mt-6 grid gap-3 sm:grid-cols-[minmax(0,1fr)_12rem]">
        <div>
          <label className="block text-sm text-muted" htmlFor="scenario-search">Search scenarios</label>
          <input id="scenario-search" role="searchbox" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Name or description" className="mt-1 w-full rounded-lg border border-muted bg-surface-muted px-3 py-2 text-sm text-foreground placeholder:text-muted focus:border-primary focus:outline-none" data-testid="scenario-search" />
        </div>
        <div>
          <label className="block text-sm text-muted" htmlFor="scenario-filter">Selection</label>
          <select id="scenario-filter" value={selectionFilter} onChange={(event) => setSelectionFilter(event.target.value)} aria-label="Filter by selection state" className="mt-1 min-h-11 w-full rounded-lg border border-muted bg-surface-elevated px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none" data-testid="scenario-filter">
            <option value="all">All scenarios</option>
            <option value="selected">Selected</option>
            <option value="available">Available</option>
          </select>
        </div>
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-2" role="list" data-testid="scenario-list">{visibleScenarios.map((scenario) => {
      const checked = scenario.system_required || selected.has(scenario.name);
      return <Button key={scenario.name} variant="ghost" type="button" disabled={scenario.system_required} onClick={() => onToggle(scenario.name)} data-testid={`scenario-card-${scenario.name}`} aria-pressed={checked} className={cn("w-full rounded-xl border p-4 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50", checked ? "border-primary/50 bg-primary/10" : "border-muted bg-surface-muted hover:bg-surface-subtle", scenario.system_required && "cursor-not-allowed opacity-90")}>
        <div className="flex items-start gap-3"><span className={cn("mt-0.5 flex h-5 w-5 items-center justify-center rounded border", checked ? "border-primary bg-primary" : "border-muted")}>{checked && <Check className="h-3 w-3 text-foreground" aria-hidden="true" />}</span><span className="min-w-0 flex-1"><span className="flex items-center gap-2 font-medium">{scenario.name}{scenario.system_required && <span data-testid="locked-badge" role="note"><LockKeyhole className="h-3.5 w-3.5 text-warning" aria-label="Required system scenario" /></span>}</span>{scenario.description && <span className="mt-1 block text-xs text-muted">{scenario.description}</span>}<span className="mt-2 block text-xs text-muted">Uses: {scenario.resources.length ? scenario.resources.join(", ") : "no local resources"}</span></span></div>
      </Button>;
    })}</div>
      {visibleScenarios.length === 0 && <p className="mt-6 rounded-lg border border-muted p-4 text-sm text-muted" role="status">No scenarios match this filter. Clear it to see the full catalog.</p>}
      <p data-testid="cascade-note" role="note" className="mt-5 text-xs text-muted">{pulledIn.length > 0 ? `Pulled in by the current selection: ${pulledIn.map((scenario) => scenario.name).join(", ")}` : "No additional scenarios are pulled in by the current selection."}</p>
      <p data-testid="resource-rollup" className="mt-2 text-xs text-muted">{impliedResources.length > 0 ? `This selection implies: ${impliedResources.map((resource) => resource.name).join(", ")}` : "This selection implies no local resources."}</p>
    </>}
  </div>;
}
