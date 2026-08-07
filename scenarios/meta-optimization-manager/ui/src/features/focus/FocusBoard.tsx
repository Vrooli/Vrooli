import { useQuery } from "@tanstack/react-query";

import { focusClient } from "../../api/focus";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cellStatusLabel, gapAxisLabel, projectionLabel } from "../labels";

/**
 * FocusBoard renders the ranked next-best gaps (OT-P0-002) and the gaps registry
 * (OT-P0-003). The focus list is impact × importance ordered; the registry lists
 * every known gap with its scope + status.
 */
export function FocusBoard() {
  const { t } = useTranslation();
  const focus = useQuery({ queryKey: ["focus", "next"], queryFn: () => focusClient.getFocus({}) });
  const gaps = useQuery({ queryKey: ["focus", "gaps"], queryFn: () => focusClient.listGaps({}) });

  if (focus.isLoading || gaps.isLoading) {
    return (
      <p data-testid={selectors.focus.loading} className="text-app-muted-foreground">
        {t(strings.common.loading)}
      </p>
    );
  }
  if (focus.error || gaps.error) {
    return (
      <p data-testid={selectors.focus.error} className="text-red-400">
        {t(strings.common.error)}
      </p>
    );
  }

  const items = focus.data?.items ?? [];
  const registry = gaps.data?.gaps ?? [];
  if (items.length === 0 && registry.length === 0) {
    return (
      <p data-testid={selectors.focus.empty} className="text-app-muted-foreground">
        {t(strings.pages.focus.empty)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <section aria-labelledby="focus-ranked-heading" className="flex flex-col gap-3">
        <h3 id="focus-ranked-heading" className="text-lg font-semibold">
          {t(strings.pages.focus.focusHeading)}
        </h3>
        <ol className="flex flex-col gap-2">
          {items.map((it) => {
            const gap = it.gap;
            if (!gap) {
              return null;
            }
            return (
              <li
                key={gap.id}
                data-testid={selectors.focus.item}
                className="rounded-panel border border-app-border bg-app-surface p-3"
              >
                <div className="flex items-center gap-2">
                  <span className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                    {it.priorityScore.toFixed(2)} {t(strings.pages.focus.priorityLabel)}
                  </span>
                  <span className="text-sm font-medium">{gap.title}</span>
                  <span data-testid="focus-axis" className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                    {gapAxisLabel(gap.axis)}
                  </span>
                </div>
                <p className="mt-1 text-xs text-app-muted-foreground">{it.rationale}</p>
                {gap.recurrence ? (
                  <p data-testid="focus-evidence" className="text-xs text-app-muted-foreground">
                    {evidenceLabel(gap.recurrence, gap.evidenceSource, gap.evidenceLocator)}
                  </p>
                ) : null}
              </li>
            );
          })}
        </ol>
      </section>

      <section aria-labelledby="focus-registry-heading" className="flex flex-col gap-3">
        <h3 id="focus-registry-heading" className="text-lg font-semibold">
          {t(strings.pages.focus.gapsHeading)}
        </h3>
        <ul className="flex flex-col gap-2">
          {registry.map((gap) => (
            <li
              key={gap.id}
              data-testid={selectors.focus.gap}
              className="rounded-panel border border-app-border bg-app-surface p-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs uppercase text-app-muted-foreground">
                  {gap.global ? t(strings.pages.focus.globalBadge) : projectionLabel(gap.projection)}
                </span>
                <span data-testid="focus-axis" className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                  {gapAxisLabel(gap.axis)}
                </span>
                <span className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                  {cellStatusLabel(gap.status)}
                </span>
                <span className="text-sm font-medium">{gap.title}</span>
              </div>
              {gap.approaches.length > 0 && (
                <ul className="mt-1 list-disc ps-5 text-xs text-app-muted-foreground">
                  {gap.approaches.map((a, i) => (
                    <li key={i}>{a}</li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function evidenceLabel(recurrence: number, source: string, locator: string): string {
  return [`recurrence=${recurrence}`, source, locator].filter(Boolean).join(" · ");
}
