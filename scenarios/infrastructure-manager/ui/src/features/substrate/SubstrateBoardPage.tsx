import { useMemo, useState } from "react";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { LegendPlate } from "../../components/instrument/LegendPlate";
import { Lamp, LampLegend } from "../../components/instrument/Lamp";
import { StatPlate, StatStrip } from "../../components/instrument/StatPlate";
import { AnnunciatorGrid, type AnnunciatorRow } from "../../components/instrument/AnnunciatorGrid";
import { EmptyState } from "../../components/ui/empty-state";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { RUNG_ORDER, type SignalState } from "../../theme/instrument";
import { DeviceConstellation, describeConstellation } from "./DeviceConstellation";
import { DeviceDrilldown, type DeviceDrilldownLabels } from "./DeviceDrilldown";
import { PortabilityLegend, PortabilityMatrix, QUALIFICATION_ORDER } from "./PortabilityMatrix";
import {
  isUnseen,
  ladderCoverage,
  rungBlindDays,
  rungReasons,
  rungStates,
  ungradedCells,
  unseenClasses,
  type SubstrateBoard,
} from "./model";
import { useSubstrateBoard } from "./useSubstrateBoard";

/**
 * The Substrate Board.
 *
 * One page that answers, for this machine: what is attached to it, which rungs
 * of the observability ladder are covered for each of those things, which are
 * dark and for how long, and — on a second axis — how much of the platform's
 * capability surface is actually proven per operating system.
 *
 * READ-ONLY BY CONTRACT. Operating-model rule 3 gives this scenario no
 * actuation right: it has no controller letter. Nothing on this page offers an
 * edit or an action affordance, and the experience contract asserts that.
 *
 * NO FIGURE IS HARD-CODED. Every number here is computed from the board read,
 * and each traces to a CLI verb returning the same value. A board that cannot
 * be checked against the instrument is decoration.
 */

/**
 * The key rendered beneath the constellation.
 *
 * All eight states are listed rather than only those present in the current
 * read, because the reader needs the vocabulary to interpret what is ABSENT
 * from the board as much as what is on it.
 */
const STATES_RENDERED: readonly SignalState[] = [
  "COVERED",
  "PARTIAL",
  "EXCURSION",
  "UNMEASURABLE",
  "UNAVAILABLE",
  "NOT_APPLICABLE",
  "BLIND",
  "SOURCE_DOWN",
];

export function SubstrateBoardPage() {
  const { t } = useTranslation();
  const query = useSubstrateBoard();
  const [selectedClass, setSelectedClass] = useState<string | null>(null);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string | null>(null);

  const board = query.data ?? null;
  const state: ExperienceSurfaceState = query.isLoading
    ? "loading"
    : query.error
      ? "error"
      : "ready";
  const statusMessage =
    state === "loading"
      ? t(strings.pages.substrate.reading)
      : state === "error"
        ? t(strings.pages.substrate.sourceUnavailable)
        : undefined;

  // Memoised so the derived reads below do not recompute on every render: a
  // bare `?? []` allocates a new array each time and invalidates every
  // dependent memo.
  const classes = useMemo(() => board?.classes ?? [], [board]);

  /**
   * Whether the ladder actually returned any graded class.
   *
   * This gates every class-derived FIGURE. An empty list means the same thing
   * as `0` on the wire — and "the machine has nothing" versus "the source did
   * not answer" are opposite facts. Printing "0 classes" for the second case is
   * precisely the fabricated zero this instrument exists to remove, so when
   * nothing was read the plates render an em dash instead.
   */
  const ladderRead = classes.length > 0;
  const coverage = useMemo(() => ladderCoverage(classes), [classes]);
  const unseen = useMemo(() => unseenClasses(classes), [classes]);
  const ungraded = useMemo(() => ungradedCells(classes), [classes]);

  const visible = useMemo(
    () => (selectedClass ? classes.filter((node) => node.deviceClass === selectedClass) : classes),
    [classes, selectedClass],
  );

  const selectedNode = useMemo(
    () => visible.find((node) => node.deviceClass === selectedDeviceId) ?? null,
    [visible, selectedDeviceId],
  );

  const rows: readonly AnnunciatorRow[] = useMemo(
    () =>
      visible.map((node) => ({
        id: node.deviceClass,
        name: node.deviceClass,
        tag:
          node.deviceCount === null
            ? t(strings.pages.substrate.notRead)
            : `${node.deviceCount} · ${isUnseen(node) ? "unseen" : "seen"}`,
        states: rungStates(node),
        reasons: rungReasons(node),
        blindDays: rungBlindDays(node),
        onSelect: () => setSelectedDeviceId(node.deviceClass),
      })),
    [visible, t],
  );

  const drilldownLabels: DeviceDrilldownLabels = {
    heading: t(strings.pages.substrate.drilldownHeading),
    ladder: t(strings.pages.substrate.ladderHeading),
    devices: t(strings.pages.substrate.drilldownDevices),
    blindDevices: t(strings.pages.substrate.drilldownBlindDevices),
    reasonHeading: t(strings.pages.substrate.reasonHeading),
    remediationHeading: t(strings.pages.substrate.remediationHeading),
    mechanismHeading: t(strings.pages.substrate.mechanismHeading),
    noRemediation: t(strings.pages.substrate.noRemediation),
    ungraded: t(strings.pages.substrate.ungraded),
    provisional: t(strings.pages.substrate.provisional),
    blockedBy: (rung: string) => t(strings.pages.substrate.blockedByRung, { rung }),
    blindFor: (days: number) => t(strings.pages.substrate.blindForDays, { days }),
    notRead: t(strings.pages.substrate.notRead),
  };

  return (
    <section
      data-testid="page-substrate"
      aria-labelledby="substrate-heading"
      className="flex flex-col gap-space-xl"
    >
      <header className="flex flex-col gap-space-2xs">
        <p className="font-mono text-body-sm uppercase tracking-[0.22em] text-app-subtle-foreground">
          {t(strings.pages.substrate.eyebrow)}
        </p>
        <h1 id="substrate-heading" className="font-display text-display uppercase tracking-[0.06em]">
          {t(strings.pages.substrate.title)}
        </h1>
        <p className="max-w-[66ch] text-app-muted-foreground">{t(strings.pages.substrate.description)}</p>
      </header>

      {/* Instrument chrome: source trust, rendered SEPARATELY from plant data so
          an owner outage can never read as a coverage collapse. */}
      <ExperienceSurface surfaceId="source-trust" data-testid="substrate-source-trust" state={state} statusMessage={statusMessage}>
        <SourceTrustStrip board={board} unavailableLabel={t(strings.pages.substrate.sourceUnavailable)} />
        {/* When the space document itself could not be read, NO cell on this
            board carries an authored status — so the board says that outright
            rather than letting the reader assume the statuses are authored. */}
        {board && !board.coverageAvailable ? (
          <p className="blind-note mt-space-2xs">
            {board.coverageReason ?? t(strings.pages.substrate.denominatorUnread)}
          </p>
        ) : null}
      </ExperienceSurface>

      <ExperienceSurface surfaceId="headline" data-testid="substrate-headline" state={state} statusMessage={statusMessage}>
        {state === "error" || !board ? (
          <EmptyState
            title={t(strings.pages.substrate.unavailableTitle)}
            description={t(strings.pages.substrate.unavailableBody)}
          />
        ) : (
          <StatStrip label={t(strings.pages.substrate.headlineLabel)}>
            <StatPlate
              value={coverage ? `${Math.round(coverage.ratio * 1000) / 10}%` : null}
              label={t(strings.pages.substrate.statLadderCoverage)}
              tone="covered"
            />
            <StatPlate
              value={coverage ? `${coverage.covered} / ${coverage.total}` : null}
              label={t(strings.pages.substrate.statRungsCovered)}
            />
            <StatPlate
              value={ladderRead ? String(unseen.length) : null}
              label={t(strings.pages.substrate.statUnseenDevices)}
              tone={ladderRead && unseen.length > 0 ? "excursion" : "neutral"}
            />
            <StatPlate
              value={ladderRead ? String(classes.length) : null}
              label={t(strings.pages.substrate.statDevices)}
            />
            <StatPlate
              value={ladderRead ? String(ungraded) : null}
              label={t(strings.pages.substrate.statUngraded)}
              tone={ungraded > 0 ? "excursion" : "neutral"}
            />
            <StatPlate
              value={board.denominator.confidence}
              label={t(strings.pages.substrate.statDenominator)}
              tone={board.denominator.confidence === "SKETCH" ? "excursion" : "neutral"}
            />
          </StatStrip>
        )}
      </ExperienceSurface>

      {/* -------------------------------------------------------- constellation -- */}
      <section aria-labelledby="substrate-constellation" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="substrate-constellation"
          tag="SB9"
          legend={t(strings.pages.substrate.constellationHeading)}
          aside={board ? `${classes.length} ${t(strings.pages.substrate.classesSuffix)}` : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.substrate.constellationNote)}
        </p>
        <ExperienceSurface surfaceId="constellation" data-testid="substrate-constellation" state={state} statusMessage={statusMessage}>
          {board ? (
            <>
              <DeviceConstellation
                hostName={board.host.name}
                classes={classes}
                selectedClass={selectedClass}
                onSelectClass={(deviceClass) => {
                  setSelectedClass((current) => (current === deviceClass ? null : deviceClass));
                  setSelectedDeviceId(null);
                }}
                summary={describeConstellation(board.host.name, classes)}
              />
              <p className="mt-space-sm text-body-sm text-app-muted-foreground">
                {describeConstellation(board.host.name, classes)}
              </p>
              <div className="mt-space-sm">
                <LampLegend states={STATES_RENDERED} />
              </div>
            </>
          ) : (
            <EmptyState
              title={t(strings.pages.substrate.unavailableTitle)}
              description={t(strings.pages.substrate.unavailableBody)}
            />
          )}
        </ExperienceSurface>
      </section>

      {/* --------------------------------------------------------------- matrix -- */}
      <section aria-labelledby="substrate-matrix" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="substrate-matrix"
          tag="R1–R5"
          legend={t(strings.pages.substrate.matrixHeading)}
          aside={selectedClass ?? undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">{t(strings.pages.substrate.matrixNote)}</p>
        <ExperienceSurface surfaceId="rung-matrix" data-testid="substrate-rung-matrix" state={state} statusMessage={statusMessage}>
          {rows.length > 0 ? (
            <AnnunciatorGrid
              rows={rows}
              rowHeader={t(strings.pages.substrate.deviceColumn)}
              caption={
                board
                  ? t(strings.pages.substrate.matrixCaption, {
                      count: rows.length,
                      confidence: board.denominator.confidence,
                      rationale: board.denominator.rationale,
                    })
                  : ""
              }
            />
          ) : (
            <EmptyState title={t(strings.pages.substrate.noDevicesTitle)} description={t(strings.pages.substrate.noDevicesBody)} />
          )}
        </ExperienceSurface>
      </section>

      {/* ------------------------------------------------------------ drilldown -- */}
      {selectedNode ? (
        <section aria-labelledby="substrate-drilldown" className="flex flex-col gap-space-sm">
          <LegendPlate id="substrate-drilldown" legend={t(strings.pages.substrate.drilldownSection)} />
          <div className="panel p-space-md">
            <DeviceDrilldown node={selectedNode} labels={drilldownLabels} />
          </div>
        </section>
      ) : null}

      {/* ------------------------------------------------- substrate sensing -- */}
      <section aria-labelledby="substrate-checks" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="substrate-checks"
          legend={t(strings.pages.substrate.checkPlatformsHeading)}
          aside={board ? String(board.checkPlatforms.length) : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.substrate.checkPlatformsNote)}
        </p>
        <ExperienceSurface surfaceId="check-platforms" data-testid="substrate-check-platforms" state={state} statusMessage={statusMessage}>
          {board && board.checkPlatforms.length > 0 ? (
            <ul className="flex flex-col gap-px bg-app-border border border-app-border rounded-panel overflow-hidden list-none p-0 m-0">
              {board.checkPlatforms.map((entry) => (
                <li
                  key={entry.hostOs}
                  className="bg-app-surface p-space-sm flex flex-wrap items-center gap-space-sm"
                >
                  <Lamp
                    state={entry.available ? "COVERED" : "SOURCE_DOWN"}
                    subject={entry.hostOs}
                    reason={entry.available ? undefined : (entry.reason ?? undefined)}
                  />
                  <span className="font-mono text-body-sm uppercase tracking-[0.08em]">
                    {entry.hostOs}
                  </span>
                  {/* Always with its denominator: "4 checks apply on windows"
                      is unreadable without the population it came from. */}
                  <span className="text-body-sm text-app-muted-foreground tabular-nums">
                    {t(strings.pages.substrate.checkPlatformRow, {
                      applicable: entry.applicable,
                      total: entry.total,
                      universal: entry.universal,
                    })}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <EmptyState
              title={t(strings.pages.substrate.unavailableTitle)}
              description={t(strings.pages.substrate.unavailableBody)}
            />
          )}
        </ExperienceSurface>
      </section>

      {/* ---------------------------------------------------------- portability -- */}
      <section aria-labelledby="substrate-portability" className="flex flex-col gap-space-sm">
        <LegendPlate
          id="substrate-portability"
          legend={t(strings.pages.substrate.portabilityHeading)}
          aside={board ? `${board.portability.length}` : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.substrate.portabilityNote)}
        </p>
        <ExperienceSurface surfaceId="portability" data-testid="substrate-portability" state={state} statusMessage={statusMessage}>
          {board && board.portability.length > 0 ? (
            <>
              <PortabilityMatrix
                rows={board.portability}
                operatingSystems={OPERATING_SYSTEMS}
                rowHeader={t(strings.pages.substrate.capabilityColumn)}
                caption={t(strings.pages.substrate.portabilityCaption, { count: board.portability.length })}
              />
              <div className="mt-space-sm">
                <PortabilityLegend rungs={QUALIFICATION_ORDER} />
              </div>
            </>
          ) : (
            <EmptyState
              title={t(strings.pages.substrate.unavailableTitle)}
              description={t(strings.pages.substrate.unavailableBody)}
            />
          )}
        </ExperienceSurface>
      </section>
    </section>
  );
}

/**
 * The three operating systems the resolver's vocabulary covers. This is fixed
 * because `deployability.HostOS` is a closed vocabulary; adding one means adding
 * a constant there first, which is the correct order.
 */
const OPERATING_SYSTEMS: readonly string[] = ["linux", "macos", "windows"];

function SourceTrustStrip({
  board,
  unavailableLabel,
}: {
  board: SubstrateBoard | null;
  unavailableLabel: string;
}) {
  if (!board) {
    return (
      <p className="blind-note" role="status">
        {unavailableLabel}
      </p>
    );
  }
  return (
    <ul className="flex flex-wrap gap-x-space-md gap-y-space-2xs list-none p-0 m-0">
      {board.sources.map((source) => (
        <li key={source.name} className="legend-key" title={source.reason ?? undefined}>
          <span
            className={
              source.verdict === "VALID" ? "lamp lamp--covered" : "lamp lamp--unavailable"
            }
            aria-hidden="true"
          >
            <span>{source.verdict === "VALID" ? "●" : "—"}</span>
          </span>
          <span>
            {source.name} · {source.verdict}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** Exported so tests can assert the rung column count without re-deriving it. */
export const RUNG_COLUMN_COUNT = RUNG_ORDER.length;
