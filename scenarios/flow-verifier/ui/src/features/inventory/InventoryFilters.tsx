import { useTranslation } from "../../i18n";

export type StatusKey = "passed" | "failed" | "error" | "none";
export type LanguageKey = "all" | "ts" | "go";
export type SortKey = "flowId" | "language" | "status" | "finishedAt";
export type SortDir = "asc" | "desc";

export interface InventoryFilterState {
  root: string;
  search: string;
  language: LanguageKey;
  status: StatusKey[];
  sort: { key: SortKey; dir: SortDir };
}

interface Props {
  value: InventoryFilterState;
  onChange: (next: InventoryFilterState) => void;
  onReload: () => void;
  onVerifyAll: () => void;
  verifyingAll?: boolean;
  flowsCount: number;
}

const STATUS_OPTIONS: StatusKey[] = ["passed", "failed", "error", "none"];

export function InventoryFilters({
  value,
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
    <div
      data-testid="inventory-filters"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3"
    >
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-1 min-w-[220px] flex-col gap-1 text-xs text-app-muted-foreground">
          <span>{t("inventory.search", { defaultValue: "Search" })}</span>
          <input
            data-testid="inventory-search"
            type="search"
            value={value.search}
            placeholder={t("inventory.searchPlaceholder", { defaultValue: "Flow id…" })}
            onChange={(e) => onChange({ ...value, search: e.target.value })}
            className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          />
        </label>
        <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
          <span>{t("inventory.rootLabel", { defaultValue: "Root path" })}</span>
          <input
            data-testid="inventory-root"
            value={value.root}
            onChange={(e) => onChange({ ...value, root: e.target.value })}
            className="h-9 w-40 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          />
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
        <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
          <span>{t("inventory.sort", { defaultValue: "Sort" })}</span>
          <div className="flex h-9 items-stretch overflow-hidden rounded-control border border-app-border">
            <select
              data-testid="inventory-sort-key"
              value={value.sort.key}
              onChange={(e) =>
                onChange({
                  ...value,
                  sort: { ...value.sort, key: e.target.value as SortKey },
                })
              }
              className="bg-app-surface px-2 text-sm text-app-foreground"
            >
              <option value="flowId">{t("inventory.sortFlow", { defaultValue: "Flow id" })}</option>
              <option value="language">{t("inventory.sortLang", { defaultValue: "Language" })}</option>
              <option value="status">{t("inventory.sortStatus", { defaultValue: "Last status" })}</option>
              <option value="finishedAt">{t("inventory.sortWhen", { defaultValue: "Last verified" })}</option>
            </select>
            <button
              type="button"
              data-testid="inventory-sort-dir"
              aria-label={t("inventory.sortDir", { defaultValue: "Toggle sort direction" })}
              onClick={() =>
                onChange({
                  ...value,
                  sort: { ...value.sort, dir: value.sort.dir === "asc" ? "desc" : "asc" },
                })
              }
              className="border-l border-app-border bg-app-surface-muted px-2 text-sm text-app-foreground hover:bg-app-surface"
            >
              {value.sort.dir === "asc"
                ? t("inventory.sortDirAsc", { defaultValue: "↑" })
                : t("inventory.sortDirDesc", { defaultValue: "↓" })}
            </button>
          </div>
        </label>
        <div className="ms-auto flex items-end gap-2">
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
        </div>
      </div>
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
    </div>
  );
}
