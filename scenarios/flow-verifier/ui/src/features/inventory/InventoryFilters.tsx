// InventoryFilters wires the inventory-specific scenario/language
// selects and status chip-bank into the shared ListToolbar shell.
// Search + sort + outer panel come from the toolbar.
import type { ScenarioSummary } from "../../api/scenarios";
import { useTranslation } from "../../i18n";
import { ListToolbar } from "../listing/ListToolbar";
import type { SortDir as ListingSortDir } from "../listing/types";

export type StatusKey = "passed" | "failed" | "error" | "none";
export type LanguageKey = "all" | "ts" | "go";
export type SortKey = "flowId" | "language" | "status" | "finishedAt";
export type SortDir = ListingSortDir;

export interface InventoryFilterState {
  scenarioId: string;
  search: string;
  language: LanguageKey;
  status: StatusKey[];
  sort: { key: SortKey; dir: SortDir };
}

interface Props {
  value: InventoryFilterState;
  scenarios: ScenarioSummary[];
  onChange: (next: InventoryFilterState) => void;
  onReload: () => void;
  onVerifyAll: () => void;
  verifyingAll?: boolean;
  flowsCount: number;
}

const STATUS_OPTIONS: StatusKey[] = ["passed", "failed", "error", "none"];

export function InventoryFilters({
  value,
  scenarios,
  onChange,
  onReload,
  onVerifyAll,
  verifyingAll,
  flowsCount,
}: Props) {
  const { t } = useTranslation();

  const toggleStatus = (s: StatusKey) => {
    const next = value.status.includes(s)
      ? value.status.filter((x) => x !== s)
      : [...value.status, s];
    onChange({ ...value, status: next });
  };

  return (
    <ListToolbar
      testId="inventory-filters"
      searchTestId="inventory-search"
      searchValue={value.search}
      searchPlaceholder={t("inventory.searchPlaceholder", { defaultValue: "Flow id…" })}
      onSearchChange={(next) => onChange({ ...value, search: next })}
      filters={
        <>
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            <span>{t("inventory.scenarioLabel", { defaultValue: "Scenario" })}</span>
            <select
              data-testid="inventory-scenario"
              value={value.scenarioId}
              onChange={(e) => onChange({ ...value, scenarioId: e.target.value })}
              className="h-9 min-w-[12rem] rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
            >
              <option value="">
                {t("inventory.scenarioAll", { defaultValue: "All scenarios" })}
              </option>
              {scenarios.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.displayName} ({s.flowCount})
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            <span>{t("inventory.language", { defaultValue: "Language" })}</span>
            <select
              data-testid="inventory-language"
              value={value.language}
              onChange={(e) => onChange({ ...value, language: e.target.value as LanguageKey })}
              className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
            >
              <option value="all">{t("inventory.langAll", { defaultValue: "All" })}</option>
              <option value="ts">{t("inventory.langTs", { defaultValue: "TypeScript" })}</option>
              <option value="go">{t("inventory.langGo", { defaultValue: "Go" })}</option>
            </select>
          </label>
        </>
      }
      sort={{
        options: [
          { value: "flowId", label: t("inventory.sortFlow", { defaultValue: "Flow id" }) },
          { value: "language", label: t("inventory.sortLang", { defaultValue: "Language" }) },
          { value: "status", label: t("inventory.sortStatus", { defaultValue: "Last status" }) },
          { value: "finishedAt", label: t("inventory.sortWhen", { defaultValue: "Last verified" }) },
        ],
        value: value.sort,
        onChange: (sort) => onChange({ ...value, sort }),
        testIdPrefix: "inventory",
      }}
      actions={
        <>
          <button
            type="button"
            data-testid="inventory-reload"
            onClick={onReload}
            className="inline-flex h-9 items-center rounded-control border border-app-border bg-app-surface px-3 text-sm text-app-foreground hover:bg-app-surface-muted"
          >
            {t("inventory.reload", { defaultValue: "Reload" })}
          </button>
          <button
            type="button"
            data-testid="inventory-verify-all"
            onClick={onVerifyAll}
            disabled={verifyingAll || flowsCount === 0}
            className="inline-flex h-9 items-center rounded-control bg-app-primary px-4 text-sm font-medium text-app-primary-foreground hover:brightness-95 disabled:opacity-60"
          >
            {verifyingAll
              ? t("inventory.verifyingAll", { defaultValue: "Verifying…" })
              : t("inventory.verifyAll", { defaultValue: "Verify all" })}
          </button>
        </>
      }
      summary={
        <div
          data-testid="inventory-status-filters"
          role="group"
          aria-label={t("inventory.statusFilter", { defaultValue: "Filter by status" })}
          className="flex flex-wrap gap-1"
        >
          {STATUS_OPTIONS.map((s) => {
            const active = value.status.includes(s);
            return (
              <button
                key={s}
                type="button"
                data-testid={`inventory-status-${s}`}
                aria-pressed={active}
                onClick={() => toggleStatus(s)}
                className={[
                  "h-7 rounded-pill border px-3 text-xs",
                  active
                    ? "border-app-primary bg-app-primary/15 text-app-foreground"
                    : "border-app-border bg-app-surface text-app-muted-foreground hover:text-app-foreground",
                ].join(" ")}
              >
                {t(`inventory.status.${s}`, { defaultValue: s })}
              </button>
            );
          })}
        </div>
      }
    />
  );
}
