import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
// `Cell` and its status enum come from the shared package the coverage
// service embeds, not from the coverage package's own re-export: comparing a
// shared-package status against the coverage-package enum is a type error,
// and silencing it would be silencing a real mismatch.
import { CellStatus } from "@vrooli/proto-types/infrastructure-manager/v1/shared/cell_pb";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { formatNumber } from "../../i18n/format";
import { EmptyState } from "../../components/ui/empty-state";
import { ConfidenceChip, Figure, RatioConfidence } from "../../components/ui/instrument-status";
import { Lamp, LampLegend } from "../../components/instrument/Lamp";
import { LegendPlate } from "../../components/instrument/LegendPlate";
import { StatPlate, StatStrip } from "../../components/instrument/StatPlate";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchCells, fetchCoverage } from "../../api/reliability";
import type { SignalState } from "../../theme/instrument";
import {
  CONFIDENCE_KEYS,
  confidenceName,
  projectionName,
  projectionSignal,
  rationaleFor,
  reasonFor,
  weakestConfidence,
} from "./model";

/**
 * The Coverage page.
 *
 * One page that answers, for every coverage space this instrument grades: what
 * ratio was computed, what denominator it was computed against, how much the
 * owner is prepared to claim about that denominator, and which cells are open
 * loop and for how long.
 *
 * THREE HONESTY RULES ARE STRUCTURAL HERE, not stylistic:
 *
 *  1. A space that could not be read carries `available=false` and zeroed
 *     counters on the wire. Those zeros are NOT rendered. An unread space
 *     shows an em dash and its failure reason, because "we could not look" and
 *     "we looked and found nothing" are different facts with different owners.
 *  2. No ratio is printed without its denominator confidence and the rationale
 *     for that confidence, in the same visual unit.
 *  3. A MISSING cell is dated. `gap_open_days` is `0` both for a gap declared
 *     today and for a gap nobody ever dated, so an undated gap is rendered as
 *     undated and sorted to the top rather than shown as a zero-day gap.
 */
export function CoveragePage() {
  const { t } = useTranslation();
  const coverage = useQuery({ queryKey: ["reliability", "coverage"], queryFn: fetchCoverage });
  const cells = useQuery({ queryKey: ["reliability", "cells"], queryFn: fetchCells });

  const projections = useMemo(() => coverage.data?.projections ?? [], [coverage.data]);
  const readProjections = projections.filter((projection) => projection.available);
  const unreadProjections = projections.filter((projection) => !projection.available);

  /**
   * Open-loop cells, undated first and then oldest gap first.
   *
   * The undated ones lead because a gap with no age is the failure this
   * instrument exists to prevent; burying it under the dated ones would hide
   * the only cell on the page nobody can put a clock on.
   */
  const missingCells = useMemo(() => {
    const missing = (cells.data?.cells ?? []).filter((cell) => cell.status === CellStatus.MISSING);
    return [...missing].sort((left, right) => {
      const leftDated = left.gapOpenedOn !== "";
      const rightDated = right.gapOpenedOn !== "";
      if (leftDated !== rightDated) return leftDated ? 1 : -1;
      return right.gapOpenDays - left.gapOpenDays;
    });
  }, [cells.data]);

  const state: ExperienceSurfaceState =
    coverage.isLoading || cells.isLoading ? "loading" : coverage.error || cells.error ? "error" : "ready";
  const statusMessage =
    state === "loading"
      ? t(strings.pages.coverage.reading)
      : state === "error"
        ? t(strings.pages.coverage.sourceUnavailable)
        : undefined;

  const coverageRead = !coverage.isLoading && !coverage.error;
  const cellsRead = !cells.isLoading && !cells.error;

  /**
   * Missing cells summed over the spaces that ANSWERED. An unread space
   * contributes neither to this figure nor to its denominator, so the total
   * understates rather than inventing coverage it could not see.
   */
  const missingInReadSpaces = readProjections.reduce((sum, projection) => sum + projection.missingCount, 0);
  const weakest = weakestConfidence(readProjections);

  const renderedStates: readonly SignalState[] = useMemo(
    () => [...new Set(projections.map(projectionSignal))],
    [projections],
  );

  return (
    <section
      data-testid="page-coverage"
      aria-labelledby="coverage-heading"
      className="flex flex-col gap-space-xl"
    >
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.coverage.eyebrow)}
        </p>
        <h1 id="coverage-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.coverage.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.coverage.description)}</p>
      </header>

      {/* Instrument chrome. Which spaces answered is a fact about the
          instrument, so it is stated here and kept out of every plant figure
          below — an owner outage must never read as a coverage collapse. */}
      <section aria-labelledby="coverage-chrome" className="flex flex-col gap-space-sm">
        <LegendPlate id="coverage-chrome" legend={t(strings.pages.coverage.chromeHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.coverage.chromeNote)}
        </p>
        <div className="panel p-space-md">
          {!coverageRead ? (
            <p className="blind-note">{t(strings.pages.coverage.sourceUnavailable)}</p>
          ) : unreadProjections.length === 0 ? (
            <p className="blind-note">{t(strings.pages.coverage.chromeAllRead)}</p>
          ) : (
            <ul className="flex flex-wrap gap-x-space-md gap-y-space-2xs list-none p-0 m-0">
              {unreadProjections.map((projection) => (
                <li key={projection.projection} className="legend-key">
                  <Lamp
                    state="SOURCE_DOWN"
                    subject={projectionName(projection.projection)}
                    reason={projection.unavailableReason}
                  />
                  <span>{projectionName(projection.projection)}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      <StatStrip label={t(strings.pages.coverage.headlineLabel)}>
        <StatPlate
          value={coverageRead ? `${formatNumber(readProjections.length)} / ${formatNumber(projections.length)}` : null}
          label={t(strings.pages.coverage.statSpacesRead)}
          tone="covered"
        />
        <StatPlate
          value={coverageRead ? formatNumber(missingInReadSpaces) : null}
          label={t(strings.pages.coverage.statCellsMissing)}
          tone={coverageRead && missingInReadSpaces > 0 ? "excursion" : "neutral"}
        />
        <StatPlate
          value={cellsRead ? formatNumber(missingCells.length) : null}
          label={t(strings.pages.coverage.statDatedGaps)}
        />
        <StatPlate
          value={weakest === null ? null : t(CONFIDENCE_KEYS[weakest])}
          label={t(strings.pages.coverage.statWeakestConfidence)}
          tone={weakest === "SKETCH" ? "excursion" : "neutral"}
        />
      </StatStrip>

      {/* ------------------------------------------------------------ grid -- */}
      <section aria-labelledby="coverage-grid-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="coverage-grid-heading"
          legend={t(strings.pages.coverage.gridHeading)}
          aside={coverageRead ? formatNumber(projections.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.coverage.gridNote)}</p>
        <ExperienceSurface surfaceId="cell-grid" state={state} data-testid="coverage-grid" statusMessage={statusMessage}>
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.coverage.reading)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.coverage.unavailableTitle)}
              description={t(strings.pages.coverage.unavailableBody)}
            />
          ) : projections.length === 0 ? (
            <EmptyState
              title={t(strings.pages.coverage.noSpacesTitle)}
              description={t(strings.pages.coverage.noSpacesBody)}
            />
          ) : (
            <>
              <ul className="grid gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0 [grid-template-columns:repeat(auto-fit,minmax(300px,1fr))]">
                {projections.map((projection) => (
                  <li key={projection.projection} className="bg-app-surface p-space-md flex flex-col gap-space-sm">
                    <div className="flex items-center gap-space-sm">
                      <Lamp
                        state={projectionSignal(projection)}
                        subject={projectionName(projection.projection)}
                        reason={reasonFor(projection)}
                      />
                      <h3 className="font-display text-heading uppercase tracking-[0.08em]">
                        {projectionName(projection.projection)}
                      </h3>
                    </div>
                    <RatioConfidence
                      value={{
                        ratio: projection.available ? (projection.ratio?.value ?? null) : null,
                        confidence: confidenceName(projection.confidence?.level),
                        rationale: rationaleFor(projection, t(strings.pages.coverage.noRationale)),
                      }}
                    />
                    <dl className="grid grid-cols-3 gap-space-2xs font-mono text-body-sm">
                      {(
                        [
                          [strings.pages.coverage.countNow, projection.nowCount],
                          [strings.pages.coverage.countInReach, projection.inReachCount],
                          [strings.pages.coverage.countMissing, projection.missingCount],
                        ] as const
                      ).map(([labelKey, count]) => (
                        <div key={labelKey} className="flex flex-col gap-space-3xs">
                          <dt className="uppercase tracking-[0.08em] text-app-subtle-foreground">{t(labelKey)}</dt>
                          <dd className="m-0 text-app-foreground">
                            {/* An unread space reports 0 on the wire. That zero
                                is a wire default, not a measurement. */}
                            <Figure value={projection.available ? formatNumber(count) : null} />
                          </dd>
                        </div>
                      ))}
                    </dl>
                    {projection.available ? null : (
                      <p className="blind-note">{t(strings.pages.coverage.projectionUnavailable)}</p>
                    )}
                  </li>
                ))}
              </ul>
              {renderedStates.length > 0 ? (
                <div className="mt-space-sm">
                  <LampLegend states={renderedStates} />
                </div>
              ) : null}
            </>
          )}
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------ confidence -- */}
      <section aria-labelledby="coverage-confidence-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="coverage-confidence-heading" legend={t(strings.pages.coverage.confidenceHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.coverage.confidenceNote)}
        </p>
        <ExperienceSurface
          surfaceId="confidence"
          state={state}
          data-testid="coverage-confidence"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.coverage.reading)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.coverage.unavailableTitle)}
              description={t(strings.pages.coverage.unavailableBody)}
            />
          ) : (
            <div className="scroller">
              <table className="annunciator">
                <caption>
                  {t(strings.pages.coverage.confidenceCaption, {
                    read: formatNumber(readProjections.length),
                    total: formatNumber(projections.length),
                  })}
                </caption>
                <thead>
                  <tr>
                    <th scope="col">{t(strings.pages.coverage.colSpace)}</th>
                    <th scope="col">{t(strings.pages.coverage.colRatio)}</th>
                    <th scope="col">{t(strings.pages.coverage.colConfidence)}</th>
                    <th scope="col">{t(strings.pages.coverage.colRationale)}</th>
                  </tr>
                </thead>
                <tbody>
                  {projections.map((projection) => (
                    <tr key={projection.projection}>
                      <th scope="row" className="font-mono">
                        {projectionName(projection.projection)}
                      </th>
                      <td className="font-mono">
                        <Figure
                          value={
                            projection.available && projection.ratio
                              ? formatNumber(projection.ratio.value, {
                                  style: "percent",
                                  maximumFractionDigits: 1,
                                })
                              : null
                          }
                        />
                      </td>
                      <td>
                        <ConfidenceChip level={confidenceName(projection.confidence?.level)} />
                      </td>
                      <td className="text-body-sm text-app-muted-foreground">
                        {rationaleFor(projection, t(strings.pages.coverage.noRationale))}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------- open loop -- */}
      <section aria-labelledby="coverage-open-loop-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="coverage-open-loop-heading"
          legend={t(strings.pages.coverage.openLoopHeading)}
          aside={cellsRead ? formatNumber(missingCells.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.coverage.openLoopNote)}</p>
        <ExperienceSurface
          surfaceId="open-loop"
          state={state === "ready" && missingCells.length === 0 ? "empty" : state}
          data-testid="coverage-open-loop"
          statusMessage={state === "loading" ? t(strings.pages.coverage.readingGaps) : statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.coverage.readingGaps)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.coverage.unavailableTitle)}
              description={t(strings.pages.coverage.openLoopUnavailable)}
            />
          ) : missingCells.length === 0 ? (
            <EmptyState
              title={t(strings.pages.coverage.openLoopEmptyTitle)}
              description={t(strings.pages.coverage.openLoopEmptyBody)}
            />
          ) : (
            <div className="scroller">
              <table className="annunciator">
                <caption>
                  {t(strings.pages.coverage.openLoopCaption, {
                    missing: formatNumber(missingCells.length),
                    total: formatNumber(cells.data?.cells.length ?? 0),
                  })}
                </caption>
                <thead>
                  <tr>
                    <th scope="col" className="annunciator__lamp-cell">
                      {t(strings.pages.coverage.colState)}
                    </th>
                    <th scope="col">{t(strings.pages.coverage.colCell)}</th>
                    <th scope="col">{t(strings.pages.coverage.colQuestion)}</th>
                    <th scope="col">{t(strings.pages.coverage.colOpened)}</th>
                    <th scope="col">{t(strings.pages.coverage.colAge)}</th>
                  </tr>
                </thead>
                <tbody>
                  {missingCells.map((cell) => {
                    const dated = cell.gapOpenedOn !== "";
                    return (
                      <tr key={`${projectionName(cell.projection)}-${cell.id}`}>
                        <td className="annunciator__lamp-cell">
                          <Lamp
                            state="BLIND"
                            subject={cell.id}
                            blindDays={dated ? cell.gapOpenDays : undefined}
                          />
                        </td>
                        <th scope="row" className="font-mono whitespace-nowrap">
                          {projectionName(cell.projection)}/{cell.id}
                        </th>
                        <td className="text-body-sm">{cell.question}</td>
                        <td className="font-mono">
                          <Figure value={dated ? cell.gapOpenedOn : null} />
                        </td>
                        <td className="font-mono">
                          {/* `gap_open_days` is 0 for an undated gap and for a
                              gap declared today. Only a dated gap has an age. */}
                          {dated ? (
                            <span className="blind-note__age">
                              {t(strings.pages.coverage.ageDays, { days: formatNumber(cell.gapOpenDays) })}
                            </span>
                          ) : (
                            <span className="blind-note">{t(strings.pages.coverage.undated)}</span>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------- integrity -- */}
      <section aria-labelledby="coverage-integrity-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="coverage-integrity-heading" legend={t(strings.pages.coverage.integrityHeading)} />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.coverage.integrityNote)}</p>
        <ExperienceSurface
          surfaceId="integrity"
          state={state === "ready" && !coverage.data?.integrityFindings.length ? "empty" : state}
          data-testid="coverage-integrity"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.coverage.reading)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.coverage.unavailableTitle)}
              description={t(strings.pages.coverage.integrityUnavailable)}
            />
          ) : coverage.data?.integrityFindings.length ? (
            <ul className="panel p-space-md flex flex-col gap-space-sm list-none m-0">
              {coverage.data.integrityFindings.map((finding) => (
                <li
                  key={`${finding.code}-${finding.location}`}
                  className="flex flex-col gap-space-3xs border-b border-app-border pb-space-sm last:border-b-0 last:pb-0"
                >
                  <div className="flex flex-wrap items-baseline gap-space-sm">
                    <span className="font-mono text-body-sm uppercase tracking-[0.1em] text-signal-excursion">
                      {finding.code}
                    </span>
                    <span className="font-mono text-body-sm text-app-subtle-foreground">{finding.location}</span>
                  </div>
                  <p className="m-0 text-body-sm text-app-muted-foreground">{finding.message}</p>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title={t(strings.pages.coverage.integrityEmptyTitle)}
              description={t(strings.pages.coverage.integrityEmptyBody)}
            />
          )}
        </ExperienceSurface>
      </section>
    </section>
  );
}
