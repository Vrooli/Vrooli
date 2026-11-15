import { useRef, useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import type { AskFacets } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { ListFilter, X } from "lucide-react";

import { chipClass } from "../../components/chipClass";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";
import { filterCount, SORTS, SORT_HOW, SORT_LABEL, URGENCIES, type ListView, type Urgency } from "./askListView";
import { URGENCY_LABEL } from "./askView";

const isUrgency = (value: string): value is Urgency => URGENCIES.some((urgency) => urgency === value);

/** `values` with `value` added or taken away. */
function toggled<T>(values: readonly T[], value: T): T[] {
  return values.includes(value) ? values.filter((existing) => existing !== value) : [...values, value];
}

/**
 * The sort and filter controls of a decision list: a sort menu always in view,
 * a Filters button that opens the choices (with how many decisions each would
 * show), and the filters in force as chips that can each be removed in one tap.
 */
export function ListControls({ view, facets, matching, closingHours, onChange, onClear }: {
  view: ListView;
  /** What the server says each filter would show; absent until the first page is read. */
  facets: AskFacets | undefined;
  /** How many decisions the current filters leave, across every page. */
  matching: number;
  /** How many hours before its deadline an ask counts as closing soon, as the Hub reports it. */
  closingHours: number;
  onChange: (view: ListView) => void;
  onClear: () => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const opener = useRef<HTMLButtonElement>(null);
  const active = filterCount(view);
  const urgencyName = (urgency: Urgency) => t(URGENCY_LABEL[urgency]);
  const urgencyFacets = (facets?.urgencies ?? []).flatMap(({ value, count }) => (isUrgency(value) ? [{ value, count }] : []));

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <span aria-hidden="true" className="shrink-0 text-sm text-app-muted-foreground">{t(strings.asks.sort.label)}</span>
        <div className="min-w-0 flex-1" data-testid={selectors.asks.sort}>
          <Select
            aria-label={`${t(strings.asks.sort.label)}: ${t(SORT_LABEL[view.sort])}`}
            value={view.sort}
            options={SORTS.map((value) => ({ value, label: t(SORT_LABEL[value]) }))}
            onValueChange={(value) => { const next = SORTS.find((sort) => sort === value); if (next) onChange({ ...view, sort: next }); }}
          />
        </div>
        <Button
          ref={opener}
          type="button"
          variant="outline"
          className="gap-2"
          disabled={!facets}
          onClick={() => setOpen(true)}
          data-testid={selectors.asks.filtersOpen}
        >
          <ListFilter size={16} aria-hidden="true" />
          {active > 0 ? t(strings.asks.filters.openActive, { count: active }) : t(strings.asks.filters.open)}
        </Button>
      </div>
      <p data-testid={selectors.asks.sortHow} className="text-sm text-app-muted-foreground">{t(SORT_HOW[view.sort], { count: closingHours })}</p>
      {active > 0 && (
        <ul aria-label={t(strings.asks.filters.active)} data-testid={selectors.asks.activeFilters} className="flex flex-wrap gap-2">
          {view.urgencies.map((urgency) => (
            <li key={`u-${urgency}`}>
              <RemovableChip name={urgencyName(urgency)} onRemove={() => onChange({ ...view, urgencies: view.urgencies.filter((value) => value !== urgency) })} />
            </li>
          ))}
          {view.senders.map((sender) => (
            <li key={`s-${sender}`}>
              <RemovableChip name={sender} onRemove={() => onChange({ ...view, senders: view.senders.filter((value) => value !== sender) })} />
            </li>
          ))}
        </ul>
      )}
      {open && facets && (
        <ResponsiveDialog
          open
          onClose={() => setOpen(false)}
          title={t(strings.asks.filters.title)}
          closeLabel={t(strings.common.close)}
          size="md"
          contentPadding="comfortable"
          returnFocusRef={opener}
          testId={selectors.asks.filtersSheet}
          footer={(
            <div className="flex w-full flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              {active > 0 && (
                <Button type="button" variant="ghost" onClick={onClear} data-testid={selectors.asks.filtersClear}>
                  {t(strings.pages.history.clearFilters)}
                </Button>
              )}
              <Button type="button" variant="primary" onClick={() => setOpen(false)} data-testid={selectors.asks.filtersShow}>
                {t(strings.asks.filters.show, { count: matching })}
              </Button>
            </div>
          )}
        >
          <div className="flex flex-col gap-6">
            <FacetGroup legend={t(strings.asks.filters.urgency)}>
              {urgencyFacets.map(({ value, count }) => (
                <FacetChip
                  key={value}
                  label={urgencyName(value)}
                  count={count}
                  selected={view.urgencies.includes(value)}
                  testId={selectors.asks.urgencyChip({ urgency: value })}
                  onToggle={() => onChange({ ...view, urgencies: URGENCIES.filter((urgency) => toggled(view.urgencies, value).includes(urgency)) })}
                />
              ))}
            </FacetGroup>
            {facets.senders.length > 0 && (
              <FacetGroup legend={t(strings.asks.filters.sender)}>
                {facets.senders.map((facet) => (
                  <FacetChip
                    key={facet.value}
                    label={facet.value}
                    count={facet.count}
                    selected={view.senders.includes(facet.value)}
                    testId={selectors.asks.senderChip}
                    onToggle={() => onChange({ ...view, senders: toggled(view.senders, facet.value) })}
                  />
                ))}
              </FacetGroup>
            )}
          </div>
        </ResponsiveDialog>
      )}
    </div>
  );
}

function FacetGroup({ legend, children }: { legend: string; children: React.ReactNode }) {
  return (
    <fieldset className="flex min-w-0 flex-col gap-2">
      <legend className="mb-2 text-sm font-semibold">{legend}</legend>
      <div className="flex flex-wrap gap-2">{children}</div>
    </fieldset>
  );
}

function FacetChip({ label, count, selected, testId, onToggle }: { label: string; count: number; selected: boolean; testId: string; onToggle: () => void }) {
  return (
    <button type="button" aria-pressed={selected} data-testid={testId} onClick={onToggle} className={cn(chipClass(selected), "max-w-full gap-2")}>
      <span className="min-w-0 [overflow-wrap:anywhere]">{label}</span>
      <span className={cn("tabular-nums", selected ? "opacity-80" : "text-app-muted-foreground")}>{count}</span>
    </button>
  );
}

function RemovableChip({ name, onRemove }: { name: string; onRemove: () => void }) {
  const { t } = useTranslation();
  return (
    <button
      type="button"
      aria-label={t(strings.asks.filters.remove, { name })}
      onClick={onRemove}
      className={cn(chipClass(true), "max-w-full gap-1.5")}
    >
      <span className="min-w-0 [overflow-wrap:anywhere]">{name}</span>
      <X size={14} aria-hidden="true" className="shrink-0" />
    </button>
  );
}
