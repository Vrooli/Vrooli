import { useQuery } from "@tanstack/react-query";
import { useLayoutEffect, useMemo, useState } from "react";
import { Loader2, LockKeyhole } from "lucide-react";
import { fetchClosure, fetchScenarios } from "../../api/selection";
import { CardShell } from "@vrooli/react-component-library/CardShell/1";
import { FilterBar } from "@vrooli/react-component-library/FilterBar/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { i18n } from "../../i18n";

interface Props { selected: Set<string>; onToggle: (name: string) => void; target?: string; }

// ScenarioCatalogStep is the V2 entry point. Resources are visible only as
// manifest-derived consequences of a scenario choice, never as the primary
// configuration authority.
export function ScenarioCatalogStep({ selected, onToggle, target = "local" }: Props) {
  const [search, setSearch] = useState("");
  const [activeFilterIds, setActiveFilterIds] = useState<string[]>([]);
  const { data, isLoading, error } = useQuery({ queryKey: ["selection-scenarios", target], queryFn: () => fetchScenarios(target) });
  const closure = useQuery({ queryKey: ["selection-closure", target], queryFn: () => fetchClosure(target) });
  const scenarios = data?.scenarios ?? [];
  const normalizedSearch = search.trim().toLowerCase();
  const visibleScenarios = useMemo(() => scenarios.filter((scenario) => {
    const isSelected = scenario.systemRequired || selected.has(scenario.name);
    const matchesSelection = activeFilterIds.length === 0 || (activeFilterIds.includes("selected") ? isSelected : !isSelected);
    if (!matchesSelection) return false;
    if (!normalizedSearch) return true;
    return scenario.name.toLowerCase().includes(normalizedSearch) || (scenario.description ?? "").toLowerCase().includes(normalizedSearch);
  }), [activeFilterIds, normalizedSearch, scenarios, selected]);
  const impliedResources = closure.data?.resources ?? [];
  const pulledIn = (closure.data?.scenarios ?? []).filter((scenario) => !scenario.direct);
  useLayoutEffect(() => {
    const searchInput = document.querySelector<HTMLInputElement>(`input[aria-label="${i18n.t("onboarding.catalog.search")}"]`);
    searchInput?.setAttribute("data-testid", "scenario-search");
  }, [isLoading, error]);
  return <div data-testid="step-select-scenarios">
    <h1 className="text-xl font-semibold sm:text-2xl">{i18n.t("onboarding.catalog.heading")}</h1>
    <p className="mt-2 text-sm text-muted sm:text-base">{i18n.t("onboarding.catalog.intro")}</p>
    {isLoading && <div className="flex items-center gap-2 py-16 text-muted" role="status"><Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" /> {i18n.t("onboarding.catalog.loading")}</div>}
    {error && <p data-testid="catalog-error" className="py-10 text-danger" role="alert">{i18n.t("onboarding.catalog.error")}</p>}
    {!isLoading && !error && <>
      <div className="mt-6">
        <div className="mb-3 block text-sm font-medium"><span className="block">{i18n.t("onboarding.catalog.show")}</span>
          <Select
            id="scenario-filter"
            data-testid="scenario-filter"
            aria-label={i18n.t("onboarding.catalog.filter")}
            value={activeFilterIds[0] ?? "all"}
            onValueChange={(value) => setActiveFilterIds(value === "all" ? [] : [value])}
            options={[
              { value: "all", label: i18n.t("onboarding.catalog.all") },
              { value: "selected", label: i18n.t("onboarding.catalog.selected") },
              { value: "available", label: i18n.t("onboarding.catalog.available") },
            ]}
            className="mt-1"
          />
        </div>
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
      const checked = scenario.systemRequired || selected.has(scenario.name);
      return <CardShell
        key={scenario.name}
        testId={`scenario-card-${scenario.name}`}
        className="scenario-choice-card"
        interactive={!scenario.systemRequired}
        selectLabel={i18n.t("onboarding.catalog.selectScenario", { name: scenario.name })}
        selection={{
          selectionMode: true,
          selected: checked,
          disabled: scenario.systemRequired,
          disabledReason: scenario.systemRequired ? i18n.t("onboarding.catalog.requiredScenario") : undefined,
          onToggleSelect: () => onToggle(scenario.name),
        }}
      >
        <div className="scenario-choice-card__body">
          <div className="scenario-choice-card__heading">
            <span className="scenario-choice-card__name">{scenario.name}</span>
            {scenario.systemRequired && <span className="scenario-choice-card__badge" data-testid="locked-badge" role="note"><LockKeyhole aria-hidden="true" />{i18n.t("onboarding.catalog.requiredScenario")}</span>}
          </div>
          {scenario.description && <p className="scenario-choice-card__description">{scenario.description}</p>}
          <div className="scenario-choice-card__meta"><span>{i18n.t("onboarding.catalog.uses")}</span><span className="scenario-choice-card__resources">{scenario.resources.length ? scenario.resources.map((resource) => <span key={resource}>{resource}</span>) : i18n.t("onboarding.catalog.noResources")}</span></div>
        </div>
      </CardShell>;
    })}</div>
      {visibleScenarios.length === 0 && <p className="mt-6 rounded-lg border border-muted p-4 text-sm text-muted" role="status">{i18n.t("onboarding.catalog.noMatch")}</p>}
      <p data-testid="cascade-note" role="note" className="mt-5 text-xs text-muted">{pulledIn.length > 0 ? i18n.t("onboarding.catalog.pulledIn", { names: pulledIn.map((scenario) => scenario.name).join(", ") }) : i18n.t("onboarding.catalog.noAdditional")}</p>
      <p data-testid="resource-rollup" className="mt-2 text-xs text-muted">{impliedResources.length > 0 ? i18n.t("onboarding.catalog.implies", { names: impliedResources.map((resource) => resource.name).join(", ") }) : i18n.t("onboarding.catalog.impliesNone")}</p>
    </>}
  </div>;
}
