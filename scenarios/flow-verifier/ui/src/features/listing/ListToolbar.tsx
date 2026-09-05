// ListToolbar is the visual shell both inventory + scenarios filter
// strips render inside. It owns the surface (border, panel padding,
// stack direction) and the search input + sort dropdown; everything
// else slots through `filters` and `actions` so the two call sites
// stay independent — duplicate before extract, then extract only
// what genuinely overlaps.
import type { ReactNode } from "react";

import { useTranslation } from "../../i18n";
import { flipDir, type SortDir } from "./types";

interface SortOption<TKey extends string> {
  value: TKey;
  label: string;
}

interface Props<TKey extends string> {
  testId: string;
  searchTestId?: string;
  searchValue: string;
  searchPlaceholder?: string;
  onSearchChange: (next: string) => void;
  filters?: ReactNode;
  actions?: ReactNode;
  summary?: ReactNode;
  sort?: {
    options: ReadonlyArray<SortOption<TKey>>;
    value: { key: TKey; dir: SortDir };
    onChange: (next: { key: TKey; dir: SortDir }) => void;
    testIdPrefix: string;
  };
}

export function ListToolbar<TKey extends string>({
  testId,
  searchTestId,
  searchValue,
  searchPlaceholder,
  onSearchChange,
  filters,
  actions,
  summary,
  sort,
}: Props<TKey>) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={testId}
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-3"
    >
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-1 min-w-[220px] flex-col gap-1 text-xs text-app-muted-foreground">
          <span>{t("list.search", { defaultValue: "Search" })}</span>
          <input
            data-testid={searchTestId ?? `${testId}-search`}
            type="search"
            value={searchValue}
            placeholder={searchPlaceholder}
            onChange={(e) => onSearchChange(e.target.value)}
            className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          />
        </label>
        {filters}
        {sort && (
          <label className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            <span>{t("list.sort", { defaultValue: "Sort" })}</span>
            <div className="flex h-9 items-stretch overflow-hidden rounded-control border border-app-border">
              <select
                data-testid={`${sort.testIdPrefix}-sort-key`}
                value={sort.value.key}
                onChange={(e) =>
                  sort.onChange({ ...sort.value, key: e.target.value as TKey })
                }
                className="bg-app-surface px-2 text-sm text-app-foreground"
              >
                {sort.options.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.label}
                  </option>
                ))}
              </select>
              <button
                type="button"
                data-testid={`${sort.testIdPrefix}-sort-dir`}
                aria-label={t("list.sortDir", { defaultValue: "Toggle sort direction" })}
                onClick={() => sort.onChange({ ...sort.value, dir: flipDir(sort.value.dir) })}
                className="border-l border-app-border bg-app-surface-muted px-2 text-sm text-app-foreground hover:bg-app-surface"
              >
                {sort.value.dir === "asc" ? "↑" : "↓"}
              </button>
            </div>
          </label>
        )}
        {actions && <div className="ms-auto flex items-end gap-2">{actions}</div>}
      </div>
      {summary}
    </div>
  );
}
