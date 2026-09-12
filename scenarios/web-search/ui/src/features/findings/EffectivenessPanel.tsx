import { useQuery } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { FindingEffectiveness } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

import { findingsClient } from "../../api/clients";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { errorMessage } from "../../lib/errorMessage";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";

/**
 * EffectivenessPanel surfaces the OT-P2-001 usage-telemetry signal: each
 * finding's surfaced/used counts, when it was last surfaced, and the blended
 * effective score (age-decayed confidence × usage factor), highest-first. It is
 * a read-only readout — it never mutates findings; the GC consumes the same
 * signal to pick supersede candidates.
 */
export function EffectivenessPanel() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["findings", "effectiveness"],
    queryFn: async () => findingsClient.listEffectiveness({ limit: 50 }),
  });

  const items = query.data?.items ?? [];

  return (
    <section
      data-testid={selectors.findings.effectivenessPanel}
      aria-labelledby="effectiveness-heading"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-col gap-1">
        <h3 id="effectiveness-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.findings.effectivenessHeading)}
        </h3>
        <p className="text-xs text-app-muted-foreground">{t(strings.findings.effectivenessHint)}</p>
      </div>

      {query.isLoading && (
        <p data-testid={selectors.findings.effectivenessLoading} className="text-sm text-app-muted-foreground">
          {t(strings.findings.effectivenessLoading)}
        </p>
      )}
      {query.error != null && (
        <p data-testid={selectors.findings.effectivenessError} className="text-sm text-app-danger">
          {t(strings.findings.effectivenessError, { message: errorMessage(query.error, t) })}
        </p>
      )}
      {query.data && items.length === 0 && (
        <p data-testid={selectors.findings.effectivenessEmpty} className="text-sm text-app-muted-foreground">
          {t(strings.findings.effectivenessEmpty)}
        </p>
      )}
      {items.length > 0 && (
        <ul data-testid={selectors.findings.effectivenessList} className="flex flex-col gap-2">
          {items.map((item) => (
            <EffectivenessRow key={item.finding?.id ?? Math.random()} item={item} />
          ))}
        </ul>
      )}
    </section>
  );
}

function EffectivenessRow({ item }: { item: FindingEffectiveness }) {
  const { t } = useTranslation();
  const lastSurfaced = item.lastSurfacedAt
    ? formatDate(timestampDate(item.lastSurfacedAt), { dateStyle: "medium" })
    : null;

  return (
    <li
      data-testid={selectors.findings.effectivenessItem}
      data-finding={item.finding?.id}
      className="flex flex-col gap-1 rounded-control bg-app-surface-muted px-3 py-2"
    >
      <span className="truncate text-sm text-app-foreground">{item.finding?.claim}</span>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-app-muted-foreground">
        <span className="rounded-pill bg-app-primary/10 px-2 py-0.5 font-mono text-app-foreground">
          {t(strings.findings.effectivenessScore, { value: item.effectiveScore.toFixed(2) })}
        </span>
        <span>{t(strings.findings.effectivenessSurfaced, { count: item.surfacedCount })}</span>
        {item.usedCount > 0 && <span>{t(strings.findings.effectivenessUsed, { count: item.usedCount })}</span>}
        <span>
          {lastSurfaced
            ? t(strings.findings.effectivenessLastSurfaced, { when: lastSurfaced })
            : t(strings.findings.effectivenessNeverSurfaced)}
        </span>
      </div>
    </li>
  );
}
