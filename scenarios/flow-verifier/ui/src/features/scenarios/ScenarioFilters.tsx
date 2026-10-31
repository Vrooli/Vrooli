// ScenarioFilters wires the scenarios-specific filter dropdowns and
// bulk-action buttons into the shared ListToolbar shell. Search +
// sort + outer panel come from the toolbar; the flows/errors selects
// and the Generate/Clear/Reload buttons remain here.
import { ListToolbar } from "../listing/ListToolbar";
import type { SortDir } from "../listing/types";
import { useTranslation } from "../../i18n";

export type ScenarioFlowsKey = "any" | "has" | "empty";
export type ScenarioErrorsKey = "any" | "with" | "clean";
export type ScenarioSortKey = "name" | "flowCount" | "id";
export type ScenarioSortDir = SortDir;

export interface ScenarioFilterState {
  search: string;
  flows: ScenarioFlowsKey;
  errors: ScenarioErrorsKey;
  sort: { key: ScenarioSortKey; dir: ScenarioSortDir };
}

export interface ScenarioFiltersProps {
  value: ScenarioFilterState;
  onChange: (next: ScenarioFilterState) => void;
  onReload: () => void;
  onGenerateAll: () => void;
  onClearAll: () => void;
  generatingAll?: boolean;
  clearingAll?: boolean;
  scenariosCount: number;
  selectedCount: number;
}

export function ScenarioFilters({
  value,
  onChange,
  onReload,
  onGenerateAll,
  onClearAll,
  generatingAll,
  clearingAll,
  scenariosCount,
  selectedCount,
}: ScenarioFiltersProps) {
  const { t } = useTranslation();
  const bulkActive = selectedCount > 0;

  return (
    <ListToolbar
      testId="scenario-filters"
      searchTestId="scenario-search"
      searchValue={value.search}
      searchPlaceholder={t("scenarios.searchPlaceholder", { defaultValue: "Name, id, description…" })}
      onSearchChange={(next) => onChange({ ...value, search: next })}
      filters={
        <>
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            <span>{t("scenarios.flowsFilter", { defaultValue: "Flows" })}</span>
            <select
              data-testid="scenario-flows"
              value={value.flows}
              onChange={(e) => onChange({ ...value, flows: e.target.value as ScenarioFlowsKey })}
              className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
            >
              <option value="any">{t("scenarios.flowsAny", { defaultValue: "Any" })}</option>
              <option value="has">{t("scenarios.flowsHas", { defaultValue: "Has flows" })}</option>
              <option value="empty">{t("scenarios.flowsEmpty", { defaultValue: "Empty" })}</option>
            </select>
          </label>
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            <span>{t("scenarios.errorsFilter", { defaultValue: "Errors" })}</span>
            <select
              data-testid="scenario-errors"
              value={value.errors}
              onChange={(e) => onChange({ ...value, errors: e.target.value as ScenarioErrorsKey })}
              className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
            >
              <option value="any">{t("scenarios.errorsAny", { defaultValue: "Any" })}</option>
              <option value="with">{t("scenarios.errorsWith", { defaultValue: "Has errors" })}</option>
              <option value="clean">{t("scenarios.errorsClean", { defaultValue: "Clean" })}</option>
            </select>
          </label>
        </>
      }
      sort={{
        options: [
          { value: "name", label: t("scenarios.sortName", { defaultValue: "Name" }) },
          { value: "flowCount", label: t("scenarios.sortFlows", { defaultValue: "Flow count" }) },
          { value: "id", label: t("scenarios.sortId", { defaultValue: "Id" }) },
        ],
        value: value.sort,
        onChange: (sort) => onChange({ ...value, sort }),
        testIdPrefix: "scenario",
      }}
      actions={
        <>
          <button
            type="button"
            data-testid="scenario-reload"
            onClick={onReload}
            className="inline-flex h-9 items-center rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground hover:bg-app-surface-muted"
          >
            {t("scenarios.reload", { defaultValue: "Reload" })}
          </button>
          <button
            type="button"
            data-testid="scenario-generate-all"
            onClick={onGenerateAll}
            disabled={generatingAll || clearingAll || scenariosCount === 0}
            className="inline-flex h-9 items-center rounded-control bg-app-primary px-4 text-sm font-medium text-app-primary-foreground hover:brightness-95 disabled:opacity-60"
          >
            {generatingAll
              ? t("scenarios.generatingSelected", { defaultValue: "Generating…" })
              : bulkActive
                ? t("scenarios.generateSelected", { defaultValue: "Generate selected" })
                : t("scenarios.generateAll", { defaultValue: "Generate all" })}
          </button>
          <button
            type="button"
            data-testid="scenario-clear-all"
            onClick={onClearAll}
            disabled={generatingAll || clearingAll || scenariosCount === 0}
            className="inline-flex h-9 items-center rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground hover:bg-app-surface-muted disabled:opacity-60"
          >
            {clearingAll
              ? t("scenarios.clearingSelected", { defaultValue: "Clearing…" })
              : bulkActive
                ? t("scenarios.clearSelected", { defaultValue: "Clear selected" })
                : t("scenarios.clearAll", { defaultValue: "Clear all" })}
          </button>
        </>
      }
      summary={
        <p data-testid="scenario-filters-summary" className="text-xs text-app-muted-foreground">
          {bulkActive
            ? t("scenarios.summarySelected", {
                defaultValue: "{{count}} selected of {{total}}",
                count: selectedCount,
                total: scenariosCount,
              })
            : t("scenarios.summary", {
                defaultValue: "{{count}} scenario",
                defaultValue_other: "{{count}} scenarios",
                count: scenariosCount,
              })}
        </p>
      }
    />
  );
}
