import { useQuery } from "@tanstack/react-query";
import { useLayoutEffect, useMemo, useState } from "react";
import { Check, Loader2, LockKeyhole } from "lucide-react";
import { fetchV2Closure, fetchV2Scenarios } from "../../lib/api";
import { cn } from "../../lib/utils";
import { Button } from "@vrooli/react-component-library/Button/2";
import { FilterBar } from "@vrooli/react-component-library/FilterBar/1";
import { i18n } from "../../i18n";

interface Props { selected: Set<string>; onToggle: (name: string) => void; }

// ScenarioCatalogStep is the V2 entry point. Resources are visible only as
// manifest-derived consequences of a scenario choice, never as the primary
// configuration authority.
export function ScenarioCatalogStep({ selected, onToggle }: Props) {
  const [search, setSearch] = useState("");
  const [activeFilterIds, setActiveFilterIds] = useState<string[]>([]);
  const { data, isLoading, error } = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const closure = useQuery({ queryKey: ["v2-closure"], queryFn: fetchV2Closure });
  const scenarios = data?.scenarios ?? [];
  const normalizedSearch = search.trim().toLowerCase();
  const visibleScenarios = useMemo(() => scenarios.filter((scenario) => {
    const isSelected = scenario.system_required || selected.has(scenario.name);
    const matchesSelection = activeFilterIds.length === 0 || (activeFilterIds.includes("selected") ? isSelected : !isSelected);
    if (!matchesSelection) return false;
    if (!normalizedSearch) return true;
    return scenario.name.toLowerCase().includes(normalizedSearch) || (scenario.description ?? "").toLowerCase().includes(normalizedSearch);
  }), [activeFilterIds, normalizedSearch, scenarios, selected]);
  const impliedResources = closure.data?.resources ?? [];
  const pulledIn = (closure.data?.scenarios ?? []).filter((scenario) => !scenario.direct);
  useLayoutEffect(() => {
    const searchInput = document.querySelector<HTMLInputElement>('input[aria-label="Search scenarios"]');
    searchInput?.setAttribute("data-testid", "scenario-search");
  }, [isLoading, error]);
  return <div data-testid="step-select-scenarios">
    <h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.catalog.heading")}</h1>
    <p className="mt-2 text-sm text-muted sm:text-base">{i18n.t("onboarding.catalog.intro")}</p>
    {isLoading && <div className="flex items-center gap-2 py-16 text-muted" role="status"><Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" /> {i18n.t("onboarding.catalog.loading")}</div>}
    {error && <p data-testid="catalog-error" className="py-10 text-danger" role="alert">{i18n.t("onboarding.catalog.error")}</p>}
    {!isLoading && !error && <>
      <div className="mt-6">
        <label className="mb-3 block text-sm font-medium" htmlFor="scenario-filter">{i18n.t("onboarding.catalog.show")}
          <select id="scenario-filter" data-testid="scenario-filter" aria-label={i18n.t("onboarding.catalog.filter")} value={activeFilterIds[0] ?? "all"} onChange={(event) => setActiveFilterIds(event.target.value === "all" ? [] : [event.target.value])} className="mt-1 min-h-11 w-full rounded-md border border-muted bg-surface px-3 py-2">
            <option value="all">{i18n.t("onboarding.catalog.all")}</option>
            <option value="selected">{i18n.t("onboarding.catalog.selected")}</option>
            <option value="available">{i18n.t("onboarding.catalog.available")}</option>
          </select>
        </label>
        <FilterBar
          query={search}
          onQueryChange={setSearch}
          options={[{ id: "selected", label: i18n.t("onboarding.catalog.selected") }, { id: "available", label: i18n.t("onboarding.catalog.available") }]}
          activeFilterIds={activeFilterIds}
          onActiveFilterIdsChange={setActiveFilterIds}
          queryLabel={i18n.t("onboarding.catalog.search")}
          applyLabel={i18n.t("onboarding.catalog.applySearch")}
          resetLabel={i18n.t("onboarding.catalog.clear")}
        />
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-2" role="list" data-testid="scenario-list">{visibleScenarios.map((scenario) => {
      const checked = scenario.system_required || selected.has(scenario.name);
      return <Button key={scenario.name} variant="ghost" type="button" disabled={scenario.system_required} onClick={() => onToggle(scenario.name)} data-testid={`scenario-card-${scenario.name}`} aria-pressed={checked} style={{ minBlockSize: "var(--tap-target-min)", whiteSpace: "normal" }} className={cn("min-h-11 w-full rounded-xl border p-4 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus/50", checked ? "border-primary/50 bg-primary/10" : "border-muted bg-surface-muted hover:bg-surface-subtle", scenario.system_required && "cursor-not-allowed opacity-90")}>
        <div className="flex items-start gap-3"><span className={cn("mt-0.5 flex h-5 w-5 items-center justify-center rounded border", checked ? "border-primary bg-primary" : "border-muted")}>{checked && <Check className="h-3 w-3 text-foreground" aria-hidden="true" />}</span><span className="min-w-0 flex-1"><span className="flex items-center gap-2 font-medium">{scenario.name}{scenario.system_required && <span data-testid="locked-badge" role="note"><LockKeyhole className="h-3.5 w-3.5 text-warning" aria-label={i18n.t("onboarding.catalog.requiredScenario")} /></span>}</span>{scenario.description && <span className="mt-1 block text-xs text-muted">{scenario.description}</span>}<span className="mt-2 block text-xs text-muted">{i18n.t("onboarding.catalog.uses")} {scenario.resources.length ? scenario.resources.join(", ") : i18n.t("onboarding.catalog.noResources")}</span></span></div>
      </Button>;
    })}</div>
      {visibleScenarios.length === 0 && <p className="mt-6 rounded-lg border border-muted p-4 text-sm text-muted" role="status">{i18n.t("onboarding.catalog.noMatch")}</p>}
      <p data-testid="cascade-note" role="note" className="mt-5 text-xs text-muted">{pulledIn.length > 0 ? i18n.t("onboarding.catalog.pulledIn", { names: pulledIn.map((scenario) => scenario.name).join(", ") }) : i18n.t("onboarding.catalog.noAdditional")}</p>
      <p data-testid="resource-rollup" className="mt-2 text-xs text-muted">{impliedResources.length > 0 ? i18n.t("onboarding.catalog.implies", { names: impliedResources.map((resource) => resource.name).join(", ") }) : i18n.t("onboarding.catalog.impliesNone")}</p>
    </>}
  </div>;
}
