import { useMemo, useState } from "react";

import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { LegendPlate } from "../../components/instrument/LegendPlate";
import { LampLegend } from "../../components/instrument/Lamp";
import { StatPlate, StatStrip } from "../../components/instrument/StatPlate";
import { AnnunciatorGrid, type AnnunciatorRow } from "../../components/instrument/AnnunciatorGrid";
import { EmptyState } from "../../components/ui/empty-state";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { RUNG_ORDER, type SignalState } from "../../theme/instrument";
import { DeviceConstellation, describeConstellation } from "./DeviceConstellation";
import { DeviceDrilldown, type DeviceDrilldownLabels } from "./DeviceDrilldown";
import { PortabilityLegend, PortabilityMatrix, QUALIFICATION_ORDER } from "./PortabilityMatrix";
import { groupByClass, ladderCoverage, unseenDevices, type SubstrateBoard } from "./model";
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

  /**
   * Whether the device-graph source actually answered.
   *
   * This gates every device-derived FIGURE. `board.devices.length` is `0` both
   * when the machine has no devices and when the source did not answer, and
   * printing "0 devices enumerated" for the second case is precisely the
   * fabricated zero this instrument exists to remove. When the source is not
   * VALID the plates render an em dash instead.
   */
  const deviceSourceAnswered = useMemo(
    () => board?.sources.some((source) => source.name === "device-graph" && source.verdict === "VALID") ?? false,
    [board],
  );

  const groups = useMemo(() => (board ? groupByClass(board.devices) : []), [board]);
  const coverage = useMemo(() => (board ? ladderCoverage(board.devices) : null), [board]);
  const unseen = useMemo(() => (board ? unseenDevices(board.devices) : []), [board]);

  const visibleDevices = useMemo(() => {
    if (!board) return [];
    if (!selectedClass) return board.devices;
    return board.devices.filter((device) => device.deviceClass === selectedClass);
  }, [board, selectedClass]);

  const selectedDevice = useMemo(
    () => visibleDevices.find((device) => device.id === selectedDeviceId) ?? null,
    [visibleDevices, selectedDeviceId],
  );

  const rows: readonly AnnunciatorRow[] = useMemo(
    () =>
      visibleDevices.map((device) => ({
        id: device.id,
        name: device.name,
        tag: device.id,
        states: device.rungs,
        reasons: device.reasons,
        blindDays: device.blindDays,
        onSelect: () => setSelectedDeviceId(device.id),
      })),
    [visibleDevices],
  );

  const drilldownLabels: DeviceDrilldownLabels = {
    heading: t(strings.pages.substrate.drilldownHeading),
    identity: t(strings.pages.substrate.fieldIdentity),
    ladder: t(strings.pages.substrate.ladderHeading),
    provenance: t(strings.pages.substrate.fieldProvenance),
    vendor: t(strings.pages.substrate.fieldVendor),
    driver: t(strings.pages.substrate.fieldDriver),
    parent: t(strings.pages.substrate.fieldParent),
    nodes: t(strings.pages.substrate.fieldNodes),
    discoveredBy: t(strings.pages.substrate.fieldDiscoveredBy),
    enrichedBy: t(strings.pages.substrate.fieldEnrichedBy),
    reasonHeading: t(strings.pages.substrate.reasonHeading),
    remediationHeading: t(strings.pages.substrate.remediationHeading),
    noRemediation: t(strings.pages.substrate.noRemediation),
    blindFor: (days: number) => t(strings.pages.substrate.blindForDays, { days }),
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
      <ExperienceSurface surfaceId="source-trust" state={state} statusMessage={statusMessage}>
        <SourceTrustStrip board={board} unavailableLabel={t(strings.pages.substrate.sourceUnavailable)} />
      </ExperienceSurface>

      <ExperienceSurface surfaceId="headline" state={state} statusMessage={statusMessage}>
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
              value={deviceSourceAnswered ? String(unseen.length) : null}
              label={t(strings.pages.substrate.statUnseenDevices)}
              tone={deviceSourceAnswered && unseen.length > 0 ? "excursion" : "neutral"}
            />
            <StatPlate
              value={deviceSourceAnswered ? String(board.devices.length) : null}
              label={t(strings.pages.substrate.statDevices)}
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
          aside={board ? `${groups.length} ${t(strings.pages.substrate.classesSuffix)}` : undefined}
        />
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.substrate.constellationNote)}
        </p>
        <ExperienceSurface surfaceId="constellation" state={state} statusMessage={statusMessage}>
          {board ? (
            <>
              <DeviceConstellation
                hostName={board.host.name}
                groups={groups}
                selectedClass={selectedClass}
                onSelectClass={(deviceClass) => {
                  setSelectedClass((current) => (current === deviceClass ? null : deviceClass));
                  setSelectedDeviceId(null);
                }}
                summary={describeConstellation(board.host.name, groups)}
              />
              <p className="mt-space-sm text-body-sm text-app-muted-foreground">
                {describeConstellation(board.host.name, groups)}
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
        <ExperienceSurface surfaceId="rung-matrix" state={state} statusMessage={statusMessage}>
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
      {selectedDevice ? (
        <section aria-labelledby="substrate-drilldown" className="flex flex-col gap-space-sm">
          <LegendPlate id="substrate-drilldown" legend={t(strings.pages.substrate.drilldownSection)} />
          <div className="panel p-space-md">
            <DeviceDrilldown device={selectedDevice} labels={drilldownLabels} />
          </div>
        </section>
      ) : null}

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
        <ExperienceSurface surfaceId="portability" state={state} statusMessage={statusMessage}>
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
