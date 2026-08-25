import { useQuery } from "@tanstack/react-query";
import type { Reading } from "@vrooli/proto-types/infrastructure-manager/v1/condition/condition_pb";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { formatNumber } from "../../i18n/format";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { Figure, StatusToken, TrustTriple } from "../../components/ui/instrument-status";
import { Lamp } from "../../components/instrument/Lamp";
import { LegendPlate } from "../../components/instrument/LegendPlate";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchCondition, fetchTrust } from "../../api/reliability";
import { bandName, trustName } from "./model";

/**
 * The Condition page.
 *
 * One page that answers: of everything this instrument read, what can actually
 * be believed?
 *
 * TRUST IS GRADED BEFORE BAND, and the order is load-bearing. A reading whose
 * trust verdict is not VALID carries NO band verdict here — not "in band", not
 * "out of band", not a quietly-omitted column. Banding a number the instrument
 * could not vouch for is precisely how a dashboard learns to report health it
 * never measured, which is the `untrusted-is-not-banded` claim in
 * `experience/pages/condition.json`.
 *
 * An unreachable SOURCE is rendered as instrument state on the chrome strip,
 * never as a plant reading, so an owner outage cannot read as a condition
 * collapse.
 */
export function ConditionPage() {
  const { t } = useTranslation();
  const condition = useQuery({ queryKey: ["reliability", "condition"], queryFn: fetchCondition });
  const trust = useQuery({ queryKey: ["reliability", "trust"], queryFn: fetchTrust });

  const triple = trust.data?.trust;
  const distribution = Object.fromEntries(
    (triple?.distribution ?? []).map((item) => [trustName(item.verdict), item.count]),
  );
  const readings = condition.data?.readings ?? [];
  const sources = condition.data?.sources ?? [];
  const trustedCount = readings.filter((reading) => trustName(reading.trustVerdict) === "VALID").length;

  /**
   * State is derived PER SURFACE, from the data that surface actually depends
   * on — not once for the page.
   *
   * A single page-level state marks every region degraded the moment any one
   * source is unavailable, which is both inaccurate and self-defeating: this
   * host returns 78 trusted readings while six declared sources have no typed
   * reader, and a shared state reported the readings table as degraded anyway.
   * The region that REPORTS the degradation is `source-availability`; the
   * regions that read fine say so.
   */
  const readingsState: ExperienceSurfaceState = condition.isLoading
    ? "loading"
    : condition.error
      ? "error"
      : readings.length === 0
        ? "empty"
        : "ready";
  const trustState: ExperienceSurfaceState = trust.isLoading
    ? "loading"
    : trust.error
      ? "error"
      : "ready";
  // The chrome is the one region whose job is to carry the bad news, so it is
  // the only one that goes partial when a declared source did not answer.
  const sourcesState: ExperienceSurfaceState = condition.isLoading
    ? "loading"
    : condition.error
      ? "error"
      : sources.length === 0
        ? "empty"
        : sources.some((source) => !source.available)
          ? "partial"
          : "ready";
  // `state` remains the page-level reading for the body's branch logic, which
  // asks "is there data to show?" — that is the readings question.
  const state: ExperienceSurfaceState = readingsState;

  /**
   * The live-region announcement follows the WORST state across the page, so a
   * screen-reader user hears about a degraded source even though the region
   * that degraded is not the one they are reading.
   */
  const statusMessage =
    readingsState === "loading" || trustState === "loading" || sourcesState === "loading"
      ? t(strings.pages.condition.reading)
      : readingsState === "error" || trustState === "error" || sourcesState === "error"
        ? t(strings.pages.condition.unavailableBody)
        : sourcesState === "partial"
          ? t(strings.pages.condition.sourceUnavailable)
          : undefined;

  return (
    <section
      data-testid="page-condition"
      aria-labelledby="condition-heading"
      className="flex flex-col gap-space-xl"
    >
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.condition.eyebrow)}
        </p>
        <h1 id="condition-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.condition.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.condition.description)}</p>
      </header>

      {/* ------------------------------------------------------- instrument -- */}
      <section aria-labelledby="condition-sources-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="condition-sources-heading" legend={t(strings.pages.condition.sourcesHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.condition.sourcesNote)}
        </p>
        <ExperienceSurface
          surfaceId="source-availability"
          state={sourcesState}
          data-testid="condition-availability"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.condition.reading)}</p>
          ) : sources.length > 0 ? (
            <ul className="panel p-space-md flex flex-col gap-space-2xs list-none m-0">
              {sources.map((source) => (
                <li key={source.source} className="flex flex-wrap items-center gap-space-sm">
                  <Lamp
                    state={source.available ? "COVERED" : "SOURCE_DOWN"}
                    subject={source.source}
                    reason={source.available ? undefined : source.reason || undefined}
                  />
                  <span className="font-mono text-body-sm">{source.source}</span>
                  <span className={source.available ? "text-body-sm text-app-muted-foreground" : "blind-note"}>
                    {source.available
                      ? t(strings.pages.condition.sourceReadable)
                      : t(strings.pages.condition.sourceUnreadable, {
                          reason: source.reason || t(strings.pages.condition.noReason),
                        })}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title={t(strings.pages.condition.noSourceReportTitle)}
              description={t(strings.pages.condition.noSourceReportBody)}
            />
          )}
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------------ trust -- */}
      <section aria-labelledby="condition-trust-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="condition-trust-heading" legend={t(strings.pages.condition.trustHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.condition.trustNote)}</p>
        <ExperienceSurface
          surfaceId="trust-distribution"
          state={trustState}
          data-testid="condition-trust"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.condition.reading)}</p>
          ) : (
            <TrustTriple
              value={{
                distribution,
                checked: triple?.checkedDenominator ?? 0,
                total: triple?.total ?? 0,
              }}
            />
          )}
        </ExperienceSurface>
      </section>

      {/* --------------------------------------------------------- readings -- */}
      <section aria-labelledby="condition-readings-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="condition-readings-heading"
          legend={t(strings.pages.condition.readingsHeading)}
          aside={condition.data ? formatNumber(readings.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.condition.readingsNote)}</p>
        {/* The readings region declares no `empty` lifecycle state in
            `experience/pages/condition.json`, and that is deliberate: an
            empty space is not a distinct readiness state here, it is a
            finding. It is rendered as one, in the body. */}
        <ExperienceSurface
          surfaceId="readings"
          state={readingsState}
          data-testid="condition-readings"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.condition.reading)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.condition.unavailableTitle)}
              description={t(strings.pages.condition.unavailableBody)}
            />
          ) : readings.length === 0 ? (
            <EmptyState
              title={t(strings.pages.condition.emptyTitle)}
              description={t(strings.pages.condition.emptyBody)}
            />
          ) : (
            <div className="scroller">
              <table className="annunciator">
                <caption>
                  {t(strings.pages.condition.readingsCaption, {
                    shown: formatNumber(readings.length),
                    trusted: formatNumber(trustedCount),
                  })}
                </caption>
                <thead>
                  <tr>
                    <th scope="col">{t(strings.pages.condition.colCell)}</th>
                    <th scope="col">{t(strings.pages.condition.colValue)}</th>
                    <th scope="col">{t(strings.pages.condition.colSource)}</th>
                    <th scope="col">{t(strings.pages.condition.colTrust)}</th>
                    <th scope="col">{t(strings.pages.condition.colBand)}</th>
                  </tr>
                </thead>
                <tbody>
                  {readings.map((reading) => (
                    <ReadingRow key={reading.id} reading={reading} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ExperienceSurface>
      </section>

      {/* ---------------------------------------------------------- history -- */}
      <section aria-labelledby="condition-history-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="condition-history-heading" legend={t(strings.pages.condition.historyHeading)} />
        <ExperienceSurface surfaceId="history" state="empty" data-testid="condition-history">
          <EmptyState
            title={t(strings.pages.condition.historyEmptyTitle)}
            description={t(strings.pages.condition.historyEmptyBody)}
          />
        </ExperienceSurface>
      </section>
    </section>
  );
}

/**
 * One reading.
 *
 * Two facts are withheld rather than fabricated. A reading whose source could
 * not answer carries `value: 0` on the wire, so no value is printed for it —
 * a zero the instrument never measured is worse than no figure at all. And a
 * reading that is not trusted is not banded: it takes NOT_EVALUATED and states
 * why, in the same row, so nobody has to infer it from an absence.
 */
function ReadingRow({ reading }: { reading: Reading }) {
  const { t } = useTranslation();
  const verdict = trustName(reading.trustVerdict);
  const trusted = verdict === "VALID";
  const measured = verdict !== "UNAVAILABLE";
  return (
    <tr className={trusted ? undefined : "bg-app-surface-muted border-l-2 border-l-app-border-lit"}>
      <th scope="row" className="font-mono whitespace-nowrap">
        {reading.cellRef}
      </th>
      <td className="font-mono">
        <Figure value={measured ? `${formatNumber(reading.value)} ${reading.unit}` : null} />
        {measured ? null : (
          <span className="blind-note block">
            {reading.unavailableReason || t(strings.pages.condition.valueUnavailable)}
          </span>
        )}
      </td>
      <td className="font-mono text-body-sm text-app-muted-foreground">{reading.source}</td>
      <td>
        <StatusToken verdict={verdict} />
      </td>
      <td>
        {trusted ? (
          <StatusToken verdict={bandName(reading.bandVerdict)} />
        ) : (
          <>
            <StatusToken verdict="NOT_EVALUATED" />
            <span className="blind-note block">{t(strings.pages.condition.untrustedNotBanded)}</span>
          </>
        )}
      </td>
    </tr>
  );
}
