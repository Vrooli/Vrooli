import { useQuery } from "@tanstack/react-query";
import { CheckCircle2, XCircle } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { fleetClient, type DomainCoverage, type MeasureSummary } from "../../api/fleet";
import { compareDomainStatus, domainStatusMeta, tierMeta } from "./coverage";

/**
 * ScenarioCoverageCard is the per-scenario drill-down: it runs
 * `ValidationService.ValidateScenario` (static, no behavioral probe) and
 * renders one row per derived domain — status chip, measure count, worst tier,
 * and (for waived/not-expected rows) the reason — with each covered domain's
 * measures nested beneath it. Domains are ordered by urgency (UNCOVERED first)
 * so a gap is always at the top.
 */
export function ScenarioCoverageCard({ scenario }: { scenario?: string }) {
  const { t } = useTranslation();

  const query = useQuery({
    queryKey: ["validate-scenario", scenario],
    queryFn: () => fleetClient.validateScenario({ scenario: scenario ?? "", probe: false }),
    enabled: Boolean(scenario),
  });

  if (!scenario) {
    return (
      <section
        data-testid={selectors.fleet.detail.hint}
        className="rounded-xl border border-dashed border-app-border bg-app-surface/40 p-4 text-app-muted-foreground"
      >
        {t(strings.fleet.detail.hint)}
      </section>
    );
  }

  const data = query.data;
  const domains = data ? [...data.domains].sort((a, b) => compareDomainStatus(a.status, b.status) || a.domain.localeCompare(b.domain)) : [];

  return (
    <section
      data-testid={selectors.fleet.detail.card}
      aria-label={t(strings.fleet.detail.title)}
      className="rounded-xl border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.fleet.detail.title)}</h2>
        <span className="font-mono text-xs text-app-muted-foreground">{scenario}</span>
        {data && (
          <span
            data-testid={selectors.fleet.detail.status}
            data-passed={data.passed}
            className={[
              "ms-auto inline-flex items-center gap-1 rounded-control px-2 py-0.5 text-xs font-medium",
              data.passed
                ? "border border-emerald-500/40 bg-emerald-500/10 text-emerald-300"
                : "border border-red-500/40 bg-red-500/10 text-red-300",
            ].join(" ")}
          >
            {data.passed ? (
              <CheckCircle2 aria-hidden="true" className="h-3.5 w-3.5" />
            ) : (
              <XCircle aria-hidden="true" className="h-3.5 w-3.5" />
            )}
            {data.passed ? t(strings.fleet.passed) : t(strings.fleet.failed)}
          </span>
        )}
      </div>

      {query.isLoading && (
        <p data-testid={selectors.fleet.detail.loading} className="mt-4 text-app-muted-foreground">
          {t(strings.fleet.detail.loading)}
        </p>
      )}

      {query.error && (
        <p data-testid={selectors.fleet.detail.error} className="mt-4 text-red-400">
          {errorMessage(query.error, t)}
        </p>
      )}

      {data && domains.length === 0 && (
        <p data-testid={selectors.fleet.detail.empty} className="mt-4 text-app-muted-foreground">
          {t(strings.fleet.detail.empty)}
        </p>
      )}

      {domains.length > 0 && (
        <ul data-testid={selectors.fleet.detail.domains} className="mt-4 flex flex-col gap-2">
          {domains.map((domain) => (
            <DomainRow key={domain.domain} domain={domain} />
          ))}
        </ul>
      )}
    </section>
  );
}

function DomainRow({ domain }: { domain: DomainCoverage }) {
  const { t } = useTranslation();
  const status = domainStatusMeta(domain.status);
  const tier = tierMeta(domain.tier);
  const reason = domain.waiverReason || domain.note;

  return (
    <li
      data-testid={selectors.fleet.domainRow({ domain: domain.domain })}
      className="rounded-lg border border-app-border bg-app-background/40 p-3"
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className={["rounded px-1.5 py-0.5 text-xs font-semibold uppercase", status.chipClass].join(" ")}>
          {t(status.labelKey)}
        </span>
        <span className="font-medium">{domain.domain}</span>
        {domain.measureCount > 0 && (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.fleet.detail.measureCount, { count: domain.measureCount })}
          </span>
        )}
        {domain.measureCount > 0 && (
          <span className={["ms-auto rounded px-1.5 py-0.5 text-xs font-semibold uppercase", tier.chipClass].join(" ")}>
            {t(tier.labelKey)}
          </span>
        )}
      </div>

      {reason && (
        <p className="mt-1.5 text-sm text-app-foreground/80">
          <span className="font-semibold">
            {domain.waiverReason ? t(strings.fleet.detail.waiverLabel) : t(strings.fleet.detail.noteLabel)}:{" "}
          </span>
          {reason}
        </p>
      )}

      {domain.measures.length > 0 && (
        <ul className="mt-2 flex flex-col gap-1.5">
          {domain.measures.map((measure) => (
            <MeasureRow key={measure.name} measure={measure} />
          ))}
        </ul>
      )}
    </li>
  );
}

function MeasureRow({ measure }: { measure: MeasureSummary }) {
  const { t } = useTranslation();
  const tier = tierMeta(measure.tier);

  return (
    <li className="rounded border border-app-border/60 bg-app-surface/60 px-2.5 py-1.5">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-mono text-xs font-medium">{measure.name}</span>
        <span className="rounded bg-app-surface-muted px-1 py-0.5 text-[0.65rem] uppercase text-app-muted-foreground">
          {measure.effect}
        </span>
        <span className={["rounded px-1 py-0.5 text-[0.65rem] font-semibold uppercase", tier.chipClass].join(" ")}>
          {t(tier.labelKey)}
        </span>
        {measure.probeDetail && (
          <span
            data-passed={measure.probePassed}
            className={[
              "ms-auto text-[0.65rem]",
              measure.probePassed ? "text-emerald-300" : "text-red-300",
            ].join(" ")}
          >
            {measure.probePassed ? t(strings.fleet.detail.probePassed) : t(strings.fleet.detail.probeFailed)}
          </span>
        )}
      </div>
      {measure.intent && <p className="mt-0.5 text-xs text-app-muted-foreground">{measure.intent}</p>}
      {measure.tierNote && <p className="mt-0.5 text-xs text-amber-300/90">{measure.tierNote}</p>}
    </li>
  );
}
