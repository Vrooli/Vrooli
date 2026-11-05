import type { StyleFilter } from "../../api/studio";
import { strings } from "../../consts/strings";
import type { AxisValues } from "../../hooks/useStudio";
import { useTranslation } from "../../i18n";

/**
 * The five taxonomy axes plus placement, as always-visible facets.
 *
 * They are a row of selects rather than a menu because the taxonomy is this
 * product's actual contribution: the operator's question is "show me the
 * cyanotype ones" or "show me what can go in a split panel", and burying that
 * behind a control makes the catalog a scroll instead of a search.
 *
 * The options come from the styles that exist, not from the enum. An axis value
 * with nothing behind it would be a facet that always returns nothing — the
 * same defect as a subject no generator draws, one layer up.
 */
export function AxisFilters({
  values,
  filter,
  onChange,
  resultCount,
}: {
  values: AxisValues;
  filter: StyleFilter;
  onChange: (next: StyleFilter) => void;
  resultCount: number;
}) {
  const { t } = useTranslation();

  const axes: Array<{ key: keyof StyleFilter; label: string; options: string[] }> = [
    { key: "role", label: t(strings.pages.catalog.axisRole), options: values.role },
    { key: "subject", label: t(strings.pages.catalog.axisSubject), options: values.subject },
    { key: "treatment", label: t(strings.pages.catalog.axisTreatment), options: values.treatment },
    { key: "lineage", label: t(strings.pages.catalog.axisLineage), options: values.lineage },
    { key: "placement", label: t(strings.pages.catalog.axisPlacement), options: values.placement },
  ];

  const active = axes.filter((axis) => filter[axis.key]);

  return (
    <div className="flex flex-col gap-3" data-testid="catalog-axis-filters">
      <div className="flex flex-wrap gap-3">
        {axes.map((axis) => (
          <label key={axis.key} className="flex flex-col gap-1 text-sm font-medium">
            {axis.label}
            <select
              className="min-h-11 min-w-40 rounded-control border border-app-border bg-app-surface px-3"
              value={filter[axis.key] ?? ""}
              data-testid={`catalog-filter-${axis.key}`}
              onChange={(event) =>
                onChange({ ...filter, [axis.key]: event.target.value || undefined })
              }
            >
              <option value="">{t(strings.pages.catalog.axisAny)}</option>
              {axis.options.map((option) => (
                <option key={option} value={option}>
                  {option}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-3 text-sm text-app-muted-foreground">
        <span data-testid="catalog-result-count">
          {t(strings.pages.catalog.resultCount, { count: resultCount })}
        </span>
        {active.length > 0 ? (
          <button
            type="button"
            className="min-h-11 rounded-control border border-app-border px-3 text-sm font-medium"
            onClick={() => onChange({})}
          >
            {t(strings.pages.catalog.clearFilters)}
          </button>
        ) : null}
      </div>
    </div>
  );
}
