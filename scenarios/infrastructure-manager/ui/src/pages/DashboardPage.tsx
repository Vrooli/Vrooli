import { useQuery } from "@tanstack/react-query";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1.1.0";
import { Figure } from "../components/ui/instrument-status";
import { Lamp } from "../components/instrument/Lamp";
import { LegendPlate } from "../components/instrument/LegendPlate";
import { StatPlate, StatStrip } from "../components/instrument/StatPlate";
import { HealthCard } from "../features/health/HealthCard";
import { trustName } from "../features/condition/model";
import { CONFIDENCE_KEYS, weakestConfidence } from "../features/coverage/model";
import { useTranslation } from "../i18n";
import { formatNumber } from "../i18n/format";
import { fetchCoverage, fetchCondition, fetchFocus } from "../api/reliability";
import { ExperienceSurface, type ExperienceSurfaceState } from "../components/experience/ExperienceSurface";

/**
 * The board.
 *
 * The three reliability domains reduced to what an operator needs before
 * choosing a detail route: how much of the cell space could be read, how much
 * of what was read can be believed, and what the cascade would look at first.
 *
 * EVERY FIGURE HERE IS COMPUTED OVER THE SOURCES THAT ANSWERED, and the board
 * says so. A source that did not answer contributes to neither numerator nor
 * denominator, so the board understates rather than inventing a reading — and
 * the unread sources are named, with their reasons, in their own region.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-space-xl"
    >
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.dashboard.eyebrow)}
        </p>
        <h1 id="dashboard-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.dashboard.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      </header>

      <BoardSummary />

      {/* ------------------------------------------------------- instrument -- */}
      <section aria-labelledby="dashboard-instrument-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="dashboard-instrument-heading" legend={t(strings.pages.dashboard.instrumentHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.instrumentNote)}
        </p>
        <HealthCard />
      </section>
    </section>
  );
}

function BoardSummary() {
  const { t } = useTranslation();
  const coverage = useQuery({ queryKey: ["reliability", "coverage"], queryFn: fetchCoverage });
  // Keyed by what it actually fetches. Sharing a key with a different query
  // function hands one response to both callers, which is how two surfaces
  // start reporting the same number for two different questions.
  const condition = useQuery({ queryKey: ["reliability", "condition"], queryFn: fetchCondition });
  const focus = useQuery({ queryKey: ["reliability", "focus"], queryFn: fetchFocus });

  const projections = coverage.data?.projections ?? [];
  const readProjections = projections.filter((projection) => projection.available);
  const missingInReadSpaces = readProjections.reduce((sum, projection) => sum + projection.missingCount, 0);
  const weakest = weakestConfidence(readProjections);

  const readings = condition.data?.readings ?? [];
  const trusted = readings.filter((reading) => trustName(reading.trustVerdict) === "VALID").length;

  const findings = focus.data?.findings ?? [];
  const focusSources = focus.data?.sources ?? [];
  const conditionSources = condition.data?.sources ?? [];
  const allSourcesUnavailable = focus.data?.allSourcesUnavailable ?? false;

  const coverageRead = !coverage.isLoading && !coverage.error;
  const conditionRead = !condition.isLoading && !condition.error;
  const focusRead = !focus.isLoading && !focus.error;

  /**
   * State is derived PER SURFACE, from the queries that surface actually reads.
   *
   * A single page-level state marked every region degraded the moment any one
   * declared source was unavailable — so the headline figures reported as
   * degraded because a focus source had not answered, and the ranked-findings
   * region reported as degraded because a condition source had not. Each region
   * now answers only for its own reads, and `source-availability` is the one
   * region whose job is to carry the bad news.
   */
  const headlineState: ExperienceSurfaceState =
    coverage.isLoading || condition.isLoading || focus.isLoading
      ? "loading"
      : coverage.error && condition.error && focus.error
        ? "error"
        : coverage.error || condition.error || focus.error
          ? "partial"
          : "ready";
  const findingsState: ExperienceSurfaceState = focus.isLoading
    ? "loading"
    : focus.error
      ? "error"
      : allSourcesUnavailable
        ? "partial"
        : findings.length === 0
          ? "empty"
          : "ready";
  const sourcesState: ExperienceSurfaceState =
    condition.isLoading || focus.isLoading
      ? "loading"
      : condition.error || focus.error
        ? "error"
        : conditionSources.length === 0 && focusSources.length === 0
          ? "empty"
          : conditionSources.some((source) => !source.available) ||
              focusSources.some((source) => !source.available)
            ? "partial"
            : "ready";

  // Retained for the body's branch logic, which asks "are the figures ready?".
  const state: ExperienceSurfaceState = headlineState;

  /** The announcement follows the worst state anywhere on the page. */
  const statusMessage =
    headlineState === "loading" || findingsState === "loading" || sourcesState === "loading"
      ? t(strings.pages.dashboard.reading)
      : sourcesState === "partial" || headlineState !== "ready" || findingsState === "partial"
        ? t(strings.pages.dashboard.sourceUnavailable)
        : undefined;

  return (
    <>
      {/* ------------------------------------------------------- confidence -- */}
      <section aria-labelledby="dashboard-confidence-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="dashboard-confidence-heading" legend={t(strings.pages.dashboard.confidenceHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.confidenceNote)}
        </p>
        <ExperienceSurface
          surfaceId="confidence-header"
          state={headlineState}
          data-testid="board-confidence-header"
          statusMessage={statusMessage}
        >
          <StatStrip label={t(strings.pages.dashboard.headlineLabel)}>
            <StatPlate
              value={coverageRead ? formatNumber(missingInReadSpaces) : null}
              label={t(strings.pages.dashboard.statMissingCells)}
              tone={coverageRead && missingInReadSpaces > 0 ? "excursion" : "neutral"}
            />
            <StatPlate
              // Only VALID readings are counted, and the denominator states how
              // many were returned, so an untrusted reading can never be quoted
              // as a trusted one.
              value={conditionRead ? `${formatNumber(trusted)} / ${formatNumber(readings.length)}` : null}
              label={t(strings.pages.dashboard.statTrustedReadings)}
              tone="covered"
            />
            <StatPlate
              value={focusRead && !allSourcesUnavailable ? formatNumber(findings.length) : null}
              label={t(strings.pages.dashboard.statRankedFindings)}
            />
            <StatPlate
              value={weakest === null ? null : t(CONFIDENCE_KEYS[weakest])}
              label={t(strings.pages.dashboard.statWeakestConfidence)}
              tone={weakest === "SKETCH" ? "excursion" : "neutral"}
            />
          </StatStrip>
        </ExperienceSurface>
      </section>

      {/* ---------------------------------------------------------- ranked -- */}
      <section aria-labelledby="dashboard-findings-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="dashboard-findings-heading"
          legend={t(strings.pages.dashboard.findingsHeading)}
          aside={focusRead && !allSourcesUnavailable ? formatNumber(findings.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.findingsNote)}
        </p>
        <ExperienceSurface
          surfaceId="ranked-findings"
          state={findingsState}
          data-testid="board-ranked-findings"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.dashboard.reading)}</p>
          ) : !focusRead || allSourcesUnavailable ? (
            <EmptyState
              title={t(strings.pages.dashboard.findingsUnavailableTitle)}
              description={t(strings.pages.dashboard.findingsUnavailableBody)}
            />
          ) : findings.length === 0 ? (
            <EmptyState
              title={t(strings.pages.dashboard.findingsEmptyTitle)}
              description={t(strings.pages.dashboard.findingsEmptyBody)}
            />
          ) : (
            <ol className="grid gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0">
              {findings.slice(0, 3).map((finding) => (
                <li key={finding.id} className="bg-app-surface p-space-md flex flex-wrap items-baseline gap-space-sm">
                  <span className="font-mono tabular-nums text-signal-covered">
                    <Figure
                      value={
                        typeof finding.rationale?.rank === "number"
                          ? formatNumber(finding.rationale.rank)
                          : null
                      }
                    />
                  </span>
                  <span className="font-display text-heading uppercase tracking-[0.06em]">{finding.title}</span>
                  <span className="font-mono text-body-sm text-app-subtle-foreground">{finding.source}</span>
                </li>
              ))}
            </ol>
          )}
        </ExperienceSurface>
      </section>

      {/* ---------------------------------------------------------- sources -- */}
      <section aria-labelledby="dashboard-sources-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="dashboard-sources-heading" legend={t(strings.pages.dashboard.sourcesHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.dashboard.sourcesNote)}
        </p>
        <ExperienceSurface
          surfaceId="source-availability"
          state={sourcesState}
          data-testid="board-source-availability"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.dashboard.reading)}</p>
          ) : conditionSources.length > 0 || focusSources.length > 0 ? (
            <ul className="panel p-space-md flex flex-col gap-space-2xs list-none m-0">
              {[
                ...conditionSources.map((source) => ({
                  key: `condition:${source.source}`,
                  name: source.source,
                  available: source.available,
                  reason: source.reason,
                })),
                ...focusSources.map((source) => ({
                  key: `focus:${source.id}`,
                  name: source.label,
                  available: source.available,
                  reason: source.reason,
                })),
              ].map((source) => (
                <li key={source.key} className="flex flex-wrap items-center gap-space-sm">
                  <Lamp
                    state={source.available ? "COVERED" : "SOURCE_DOWN"}
                    subject={source.name}
                    reason={source.available ? undefined : source.reason || undefined}
                  />
                  <span className="font-mono text-body-sm">{source.name}</span>
                  {source.available ? null : <span className="blind-note">{source.reason}</span>}
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title={t(strings.pages.dashboard.sourcesEmptyTitle)}
              description={t(strings.pages.dashboard.sourcesEmptyBody)}
            />
          )}
        </ExperienceSurface>
      </section>
    </>
  );
}
