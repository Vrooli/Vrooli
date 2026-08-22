import { useQuery } from "@tanstack/react-query";
import type { Finding } from "@vrooli/proto-types/infrastructure-manager/v1/focus/focus_pb";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { formatNumber } from "../../i18n/format";
import { EmptyState } from "../../components/ui/empty-state";
import { Figure } from "../../components/ui/instrument-status";
import { Lamp } from "../../components/instrument/Lamp";
import { LegendPlate } from "../../components/instrument/LegendPlate";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { fetchFocus } from "../../api/reliability";

/**
 * The Focus page.
 *
 * One page that answers: what should be read next, and what rule put it there?
 *
 * READ-ONLY BY CONTRACT. The scenario has no actuation right, so nothing here
 * offers a control that changes the plant — no restart, no shelve, no dismiss.
 *
 * THE EMPTY STATE IS NOT THE UNREAD STATE. "Every source answered and none had
 * a finding" and "no source answered" are different facts with different next
 * actions, so they render as different surfaces and the empty one states how
 * much of the error surface was actually read.
 */
export function FocusPage() {
  const { t } = useTranslation();
  const focus = useQuery({ queryKey: ["reliability", "focus"], queryFn: fetchFocus });

  const data = focus.data;
  const sources = data?.sources ?? [];
  const readSources = sources.filter((source) => source.available);

  /**
   * State is derived PER SURFACE, from what that surface actually reports.
   *
   * A single page-level state marked the SOURCE region degraded because the
   * ranked list was empty, and the ranked list degraded because a source was
   * unavailable — each region answering for the other's problem. The sources
   * region is the one whose job is to report an unread source; the ranked
   * region reports whether anything is ranked.
   */
  const rankedState: ExperienceSurfaceState = focus.isLoading
    ? "loading"
    : focus.error
      ? "error"
      : data?.allSourcesUnavailable
        ? "partial"
        : data?.noFindings || (data?.findings.length ?? 0) === 0
          ? "empty"
          : "ready";
  const sourcesState: ExperienceSurfaceState = focus.isLoading
    ? "loading"
    : focus.error
      ? "error"
      : sources.length === 0
        ? "empty"
        : sources.some((source) => !source.available)
          ? "partial"
          : "ready";

  // Retained for the body's branch logic, which asks "is anything ranked?".
  const state: ExperienceSurfaceState = rankedState;
  const statusMessage =
    state === "loading"
      ? t(strings.pages.focus.reading)
      : state === "error"
        ? t(strings.pages.focus.unavailableBody)
        : state === "partial"
          ? t(strings.pages.focus.sourceUnavailable)
          : undefined;

  return (
    <section data-testid="page-focus" aria-labelledby="focus-heading" className="flex flex-col gap-space-xl">
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.focus.eyebrow)}
        </p>
        <h1 id="focus-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.focus.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.focus.description)}</p>
      </header>

      {/* -------------------------------------------------------- rationale -- */}
      <section aria-labelledby="focus-rationale-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="focus-rationale-heading" legend={t(strings.pages.focus.rationaleHeading)} />
        <ExperienceSurface surfaceId="ranking-rationale" state="static" data-testid="focus-rationale">
          <p className="panel p-space-md m-0 max-w-[66ch] text-body-sm text-app-muted-foreground">
            {t(strings.pages.focus.rationaleBody)}
          </p>
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------- instrument -- */}
      <section aria-labelledby="focus-sources-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="focus-sources-heading"
          legend={t(strings.pages.focus.sourcesHeading)}
          aside={data ? `${formatNumber(readSources.length)} / ${formatNumber(sources.length)}` : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.focus.sourcesNote)}</p>
        <ExperienceSurface
          surfaceId="source-health"
          state={sourcesState}
          data-testid="focus-sources"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.focus.reading)}</p>
          ) : sources.length > 0 ? (
            <ul className="panel p-space-md flex flex-col gap-space-2xs list-none m-0">
              {sources.map((source) => (
                <li key={source.id} className="flex flex-wrap items-center gap-space-sm">
                  <Lamp
                    state={source.available ? "COVERED" : "SOURCE_DOWN"}
                    subject={source.label}
                    reason={source.available ? undefined : source.reason || undefined}
                  />
                  <span className="font-mono text-body-sm">{source.label}</span>
                  {source.available ? (
                    <span className="text-body-sm text-app-muted-foreground">
                      {t(strings.pages.focus.sourceFindings, { value: formatNumber(source.findingCount) })}
                    </span>
                  ) : (
                    <span className="blind-note">
                      {/* An unread source reports 0 findings on the wire. That
                          zero is a wire default, not a measurement. */}
                      {t(strings.pages.focus.sourceUnreadable, {
                        reason: source.reason || t(strings.pages.focus.noReason),
                      })}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title={t(strings.pages.focus.sourcesEmptyTitle)}
              description={t(strings.pages.focus.sourcesEmptyBody)}
            />
          )}
        </ExperienceSurface>
      </section>

      {/* ----------------------------------------------------------- ranked -- */}
      <section aria-labelledby="focus-ranked-heading" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="focus-ranked-heading"
          legend={t(strings.pages.focus.rankedHeading)}
          aside={data && !data.allSourcesUnavailable ? formatNumber(data.findings.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.focus.rankedNote)}</p>
        <ExperienceSurface
          surfaceId="ranked-surface"
          state={rankedState}
          data-testid="focus-surface"
          statusMessage={statusMessage}
        >
          {state === "loading" ? (
            <p className="blind-note">{t(strings.pages.focus.reading)}</p>
          ) : state === "error" ? (
            <EmptyState
              title={t(strings.pages.focus.unavailableTitle)}
              description={t(strings.pages.focus.unavailableBody)}
            />
          ) : !data || data.allSourcesUnavailable ? (
            <EmptyState
              title={t(strings.pages.focus.nothingReadTitle)}
              description={t(strings.pages.focus.nothingReadBody)}
            />
          ) : data.noFindings || data.findings.length === 0 ? (
            <EmptyState
              title={t(strings.pages.focus.emptyTitle)}
              description={t(strings.pages.focus.emptyBody, {
                read: formatNumber(readSources.length),
                total: formatNumber(sources.length),
              })}
            />
          ) : (
            <ol className="grid gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0">
              {data.findings.map((finding) => (
                <FindingRow key={finding.id} finding={finding} />
              ))}
            </ol>
          )}
        </ExperienceSurface>
      </section>

      {/* --------------------------------------------------------- efficacy -- */}
      <section aria-labelledby="focus-efficacy-heading" className="flex flex-col gap-space-sm">
        <LegendPlate id="focus-efficacy-heading" legend={t(strings.pages.focus.efficacyHeading)} />
        <ExperienceSurface surfaceId="efficacy" state="empty" data-testid="focus-efficacy">
          <EmptyState
            title={t(strings.pages.focus.efficacyEmptyTitle)}
            description={t(strings.pages.focus.efficacyEmptyBody)}
          />
        </ExperienceSurface>
      </section>
    </section>
  );
}

/**
 * One ranked finding.
 *
 * The rank is the cascade's output, so it is printed as a real reference and
 * an unranked finding shows an em dash rather than borrowing the position it
 * happens to hold in the list.
 */
function FindingRow({ finding }: { finding: Finding }) {
  const { t } = useTranslation();
  const rank = finding.rationale?.rank;
  return (
    <li className="bg-app-surface p-space-md flex flex-col gap-space-2xs">
      <div className="flex flex-wrap items-baseline gap-space-sm">
        <span className="font-mono text-body-sm uppercase tracking-[0.1em] text-signal-covered">
          {t(strings.pages.focus.rankLabel)}
        </span>
        <span className="font-mono tabular-nums text-app-foreground">
          <Figure value={typeof rank === "number" ? formatNumber(rank) : null} />
        </span>
        <h3 className="font-display text-heading uppercase tracking-[0.06em] m-0">{finding.title}</h3>
        <span className="font-mono text-body-sm text-app-subtle-foreground">{finding.source}</span>
      </div>
      <p className="m-0 text-body-sm text-app-muted-foreground">{finding.message}</p>
      {finding.rationale ? (
        <p className="m-0 font-mono text-body-sm text-app-muted-foreground">
          {finding.rationale.cascadeStage}
          {" · "}
          {finding.rationale.explanation}
        </p>
      ) : (
        <p className="m-0 blind-note">{t(strings.pages.focus.unranked)}</p>
      )}
      {finding.sensorRef ? (
        <p className="m-0 font-mono text-body-sm text-app-subtle-foreground">{finding.sensorRef}</p>
      ) : null}
    </li>
  );
}
