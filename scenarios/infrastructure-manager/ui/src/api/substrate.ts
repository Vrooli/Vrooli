import {
  ConfidenceLevel,
  Projection,
} from "@vrooli/proto-types/infrastructure-manager/v1/coverage/coverage_pb";
import {
  BandVerdict as WireBand,
  CellStatus,
  Observation,
  Rung as WireRung,
  TrustVerdict as WireTrust,
  type LadderCell,
} from "@vrooli/proto-types/infrastructure-manager/v1/ladder/ladder_pb";
import {
  HostOS,
  Qualification,
  ResolutionStatus,
  type PlatformEntry,
} from "@vrooli/proto-types/infrastructure-manager/v1/portability/portability_pb";

import { coverageClient, fetchLadder, fetchPortabilityGrid } from "./reliability";
import { RUNG_ORDER, type Rung, type SignalState } from "../theme/instrument";
import type {
  CheckPlatformCoverage,
  DeviceClassNode,
  PortabilityRow,
  RungDetail,
  SourceStatus,
  SubstrateBoard,
} from "../features/substrate/model";

/**
 * Assembles the Substrate Board from the instrument's read surfaces.
 *
 * THE CONTRACT THIS FILE ENFORCES: a source that does not answer produces a
 * `SourceStatus` with a non-VALID verdict and a reason. It NEVER produces an
 * empty class list the board would then render as "nothing attached", and it
 * never substitutes a default for a value it did not read. That distinction is
 * the scenario's entire trust model — an owner outage must not read as a
 * coverage collapse — so it is enforced here at the boundary rather than left
 * to each component to remember.
 */

const VERDICT_VALID = "VALID";
const VERDICT_UNAVAILABLE = "UNAVAILABLE";

/** Wire rung -> the view model's rung key. */
const RUNG_NAMES: Record<number, Rung> = {
  [WireRung.IDENTITY]: "IDENTITY",
  [WireRung.TELEMETRY]: "TELEMETRY",
  [WireRung.EVIDENCE]: "EVIDENCE",
  [WireRung.CONTROL]: "CONTROL",
  [WireRung.ANTICIPATION]: "ANTICIPATION",
};

const TRUST_NAMES: Record<number, string> = {
  [WireTrust.VALID]: "VALID",
  [WireTrust.GHOST]: "GHOST",
  [WireTrust.SATURATED]: "SATURATED",
  [WireTrust.SHELVED]: "SHELVED",
  [WireTrust.UNIT_MISMATCH]: "UNIT_MISMATCH",
  [WireTrust.UNAVAILABLE]: "UNAVAILABLE",
  [WireTrust.UNTRUSTED]: "UNTRUSTED",
};

/**
 * Projects one graded ladder cell onto a lamp state.
 *
 * The ORDER of these tests is the honesty rule made executable, and this is the
 * most load-bearing function in the board:
 *
 *  1. An UNREAD observation means the source stayed silent. That is a fact
 *     about the INSTRUMENT and outranks everything else, because reporting the
 *     plant from a source that did not answer is exactly the failure the trust
 *     model exists to stop.
 *  2. NOT_APPLICABLE means the rung is meaningless for this class and is graded
 *     elsewhere. It must not read as a gap, or the board manufactures blindness
 *     that nobody declared.
 *  3. An authored MISSING status is DECLARED BLINDNESS — dated, owned, and the
 *     single thing this whole surface was built to make visible.
 *  4. UNMEASURABLE and UNAVAILABLE stay distinct: "the host refused the read"
 *     is a different fact, with a different fix, from "the mechanism is absent
 *     on this host".
 *  5. BLOCKED means a lower rung is blind, so this rung's claim is unsupported
 *     rather than false. PARTIAL, never COVERED.
 *  6. Only a MEASURED observation may light, and a measured reading that is out
 *     of band is an EXCURSION rather than coverage.
 */
function cellState(cell: LadderCell): SignalState {
  const extended = cell as LadderCell & { reasonCode?: string };
  if (extended.reasonCode === "host_not_sampled") return "HOST_NOT_SAMPLED";
  if (cell.observation === Observation.UNREAD) return "SOURCE_DOWN";
  if (cell.observation === Observation.NOT_APPLICABLE) return "NOT_APPLICABLE";
  if (cell.trust === WireTrust.UNAVAILABLE) return "SOURCE_DOWN";
  if (cell.status === CellStatus.MISSING) return "BLIND";
  if (cell.observation === Observation.UNMEASURABLE) return "UNMEASURABLE";
  if (cell.observation === Observation.UNAVAILABLE) return "UNAVAILABLE";
  if (cell.observation === Observation.BLOCKED) return "PARTIAL";
  if (cell.observation === Observation.MEASURED) {
    // `graded` gates the excursion: a cell with no bar has nothing to be out
    // of, so it cannot be an excursion however its value reads.
    if (cell.graded && cell.band === WireBand.OUT_OF_BAND) return "EXCURSION";
    if (cell.status === CellStatus.IN_REACH) return "PARTIAL";
    return "COVERED";
  }
  // An observation this UI does not recognise renders as an excursion so it is
  // VISIBLE, never as the most flattering state on the list.
  return "EXCURSION";
}

/** An empty proto string means "not supplied"; it must not render as content. */
const text = (value: string): string | null => (value.trim() === "" ? null : value);

function toRungDetail(cell: LadderCell): RungDetail {
	const extended = cell as LadderCell & { reasonCode?: string };
	return {
    state: cellState(cell),
    cellRef: text(cell.cellRef),
    question: text(cell.question),
		reason: text(cell.reason) ?? text(cell.unavailableReason),
		reasonCode: text(extended.reasonCode ?? ""),
    mechanism: text(cell.mechanism),
    remediation: text(cell.remediation),
    blockedBy: RUNG_NAMES[cell.blockedBy] ?? null,
    trust: TRUST_NAMES[cell.trust] ?? null,
    graded: cell.graded,
    ungradedReason: text(cell.ungradedReason),
    provisional: cell.provisional,
    blindDays: null,
  };
}

/**
 * A rung the ladder authored no cell for.
 *
 * This is NOT a source outage, and calling it one would assert an instrument
 * fault that is not happening. The owner's space document simply declares no
 * cell for this rung on this class: the question has not been asked. That is a
 * fact about the DENOMINATOR, which is exactly why the substrate space reports
 * `PARTIAL` confidence — the cell set is known to be under-declared.
 */
function unauthoredRung(): RungDetail {
  return {
    state: "UNAUTHORED",
    cellRef: null,
    question: null,
		reason: "the substrate space authors no cell for this rung on this class",
	reasonCode: "unauthored",
    mechanism: null,
    remediation: null,
    blockedBy: null,
    trust: null,
    graded: false,
    ungradedReason: "no cell is authored for this rung, so there is nothing to grade",
    provisional: false,
    blindDays: null,
  };
}

export async function fetchSubstrateBoard(): Promise<SubstrateBoard> {
  const sources: SourceStatus[] = [];
  const [ladder, portability, coverage] = await Promise.all([
    readLadder(sources),
    readPortability(sources),
    readSubstrateDenominator(),
  ]);

  // Blindness dates come from the coverage cells, keyed by the authored cell
  // reference each ladder rung answers. A dated gap is the differentiator of
  // this whole surface; a gap with no age is the failure it exists to prevent.
  const classes = ladder.classes.map((node) => ({
    ...node,
    rungs: RUNG_ORDER.reduce(
      (acc, rung) => {
        const detail = node.rungs[rung];
        acc[rung] = {
          ...detail,
          blindDays: detail.cellRef === null ? null : (coverage.gapDaysByCell[detail.cellRef] ?? null),
        };
        return acc;
      },
      {} as Record<Rung, RungDetail>,
    ),
  }));

  return {
    host: ladder.host,
    classes,
    portability,
    sources,
    checkPlatforms: ladder.checkPlatforms,
    coverageAvailable: ladder.coverageAvailable,
    coverageReason: ladder.coverageReason,
    denominator: coverage.denominator,
  };
}

interface LadderRead {
  host: SubstrateBoard["host"];
  classes: readonly DeviceClassNode[];
  checkPlatforms: readonly CheckPlatformCoverage[];
  coverageAvailable: boolean;
  coverageReason: string | null;
}

const EMPTY_LADDER: LadderRead = {
  host: { name: "host", os: "" },
  classes: [],
  checkPlatforms: [],
  coverageAvailable: false,
  coverageReason: null,
};

async function readLadder(sources: SourceStatus[]): Promise<LadderRead> {
  try {
    const response = await fetchLadder();
    const ladder = response.ladder;
    if (!ladder) {
      sources.push({
        name: "ladder",
        verdict: VERDICT_UNAVAILABLE,
        reason: "the ladder domain answered without a ladder",
      });
      return EMPTY_LADDER;
    }

    // Every source the ladder itself read is surfaced verbatim, so an outage in
    // any one of them stays attributable instead of being folded into a single
    // undifferentiated failure.
    for (const source of ladder.sources) {
      sources.push({
        name: source.id,
        verdict: source.available ? VERDICT_VALID : VERDICT_UNAVAILABLE,
        reason: text(source.reason),
      });
    }

    // Only cells for the host this instrument runs on can be refined by a live
    // device read; other platforms are reasoned about from declarations alone,
    // so mixing them into one constellation would blend two kinds of evidence.
    const local = ladder.cells.filter(
      (cell) => cell.hostOs === "" || cell.hostOs === ladder.hostOs,
    );
    const byClass = new Map<string, LadderCell[]>();
    for (const cell of local) {
      const bucket = byClass.get(cell.deviceClass);
      if (bucket) bucket.push(cell);
      else byClass.set(cell.deviceClass, [cell]);
    }

    const classes: DeviceClassNode[] = [...byClass.entries()]
      .map(([deviceClass, cells]) => {
        const byRung = new Map<Rung, LadderCell>();
        for (const cell of cells) {
          const rung = RUNG_NAMES[cell.rung];
          if (rung) byRung.set(rung, cell);
        }
        const rungs = RUNG_ORDER.reduce(
          (acc, rung) => {
            const cell = byRung.get(rung);
            acc[rung] = cell ? toRungDetail(cell) : unauthoredRung();
            return acc;
          },
          {} as Record<Rung, RungDetail>,
        );
        // Counts are taken only from cells that actually read. A `0` from an
        // unread cell is a wire default, not a measurement, so a class whose
        // every cell is unread reports `null` rather than claiming it
        // enumerated nothing.
        const read = cells.filter((cell) => cell.observation !== Observation.UNREAD);
        return {
          deviceClass,
          rungs,
          deviceCount: read.length === 0 ? null : Math.max(...read.map((cell) => cell.deviceCount)),
          blindDevices: read.length === 0 ? null : Math.max(...read.map((cell) => cell.blindDevices)),
        };
      })
      .sort((left, right) => left.deviceClass.localeCompare(right.deviceClass));

    return {
      host: { name: ladder.hostOs || "host", os: ladder.hostOs },
      classes,
      checkPlatforms: ladder.checkPlatforms.map((entry) => ({
        hostOs: entry.hostOs,
        applicable: entry.applicable,
        total: entry.total,
        universal: entry.universal,
        available: entry.available,
        reason: text(entry.reason),
      })),
      coverageAvailable: ladder.coverageAvailable,
      coverageReason: text(ladder.coverageReason),
    };
  } catch (error) {
    sources.push({ name: "ladder", verdict: VERDICT_UNAVAILABLE, reason: describeError(error) });
    return EMPTY_LADDER;
  }
}

interface DenominatorRead {
  denominator: SubstrateBoard["denominator"];
  gapDaysByCell: Readonly<Record<string, number>>;
}

/**
 * Reads the substrate projection's authored denominator and the dated ages of
 * its open-loop cells.
 *
 * On failure the confidence reports SKETCH — the honest floor. An unread
 * denominator is not a good one; it is an unknown one, and the board prints
 * this qualifier beside every ratio so the reader can see which they have.
 */
async function readSubstrateDenominator(): Promise<DenominatorRead> {
  try {
    const [coverage, cells] = await Promise.all([
      coverageClient.getCoverage({}),
      coverageClient.listCells({}),
    ]);
    const substrate = coverage.projections.find(
      (projection) => projection.projection === Projection.SUBSTRATE,
    );
    // Keyed both bare and projection-qualified, because a ladder cell may cite
    // either form and a missed key would silently drop a gap's age — which is
    // the one field on this board that must never go quiet.
    const gapDaysByCell: Record<string, number> = {};
    for (const cell of cells.cells) {
      if (cell.gapOpenDays > 0) {
        gapDaysByCell[cell.id] = cell.gapOpenDays;
        gapDaysByCell[`substrate/${cell.id}`] = cell.gapOpenDays;
      }
    }
    return {
      denominator: {
        confidence: confidenceName(substrate?.confidence?.level),
        rationale:
          substrate?.confidence?.rationale ||
          substrate?.unavailableReason ||
          "No denominator rationale was supplied for the substrate projection.",
      },
      gapDaysByCell,
    };
  } catch {
    return {
      denominator: {
        confidence: "SKETCH",
        rationale:
          "The coverage source did not answer, so the substrate denominator could not be read. No ratio on this board is graded against an authored denominator.",
      },
      gapDaysByCell: {},
    };
  }
}

const CONFIDENCE_NAMES: Record<number, SubstrateBoard["denominator"]["confidence"]> = {
  [ConfidenceLevel.AUTHORITATIVE]: "AUTHORITATIVE",
  [ConfidenceLevel.PARTIAL]: "PARTIAL",
};

function confidenceName(level: number | undefined): SubstrateBoard["denominator"]["confidence"] {
  // Anything the enum does not name resolves to SKETCH, never to the most
  // flattering value. An unrecognised confidence is not evidence of a good
  // denominator; it is evidence that nobody stated one.
  return (level !== undefined && CONFIDENCE_NAMES[level]) || "SKETCH";
}

const QUALIFICATION_NAMES: Record<number, string> = {
  [Qualification.QUALIFIED]: "qualified",
  [Qualification.BUILD_VERIFIED]: "build-verified",
  [Qualification.UNQUALIFIED]: "unqualified",
  [Qualification.DEGRADED]: "degraded",
  [Qualification.INELIGIBLE]: "ineligible",
  [Qualification.UNDECLARED]: "undeclared",
};

const STATUS_NAMES: Record<number, string> = {
  [ResolutionStatus.IMPLEMENTED]: "implemented",
  [ResolutionStatus.DEGRADED]: "degraded",
  [ResolutionStatus.INELIGIBLE]: "ineligible",
  [ResolutionStatus.UNWIRED]: "unwired",
  [ResolutionStatus.PEERLESS]: "peerless",
  [ResolutionStatus.STATUS_INVALID]: "status_invalid",
};

const HOST_OS_NAMES: Record<number, string> = {
  [HostOS.HOST_OS_LINUX]: "linux",
  [HostOS.HOST_OS_MACOS]: "macos",
  [HostOS.HOST_OS_WINDOWS]: "windows",
};

/**
 * Reads the capability grid from this scenario's `portability` domain.
 *
 * An unrecognised qualification maps to the literal "unspecified" rather than
 * to any valid rung. The matrix renders an unrecognised rung with the excursion
 * treatment, so a wire value nobody handled becomes visible instead of quietly
 * rendering as the most flattering thing on the list.
 */
async function readPortability(sources: SourceStatus[]): Promise<readonly PortabilityRow[]> {
  try {
    const response = await fetchPortabilityGrid();
    const grid = response.grid;
    if (!grid) {
      sources.push({
        name: "portability",
        verdict: VERDICT_UNAVAILABLE,
        reason: "the portability domain answered without a grid",
      });
      return [];
    }
    sources.push({
      name: "portability",
      verdict: VERDICT_VALID,
      reason: `${grid.manifestsRead} manifests read from ${grid.manifestRoot}`,
    });
    return grid.capabilities.map((entry) => ({
      capability: entry.capability,
      platforms: Object.fromEntries(
        entry.platforms
          .map(
            (platform: PlatformEntry) =>
              [HOST_OS_NAMES[platform.hostOs], toCell(platform)] as const,
          )
          .filter(([os]) => Boolean(os)),
      ),
    }));
  } catch (error) {
    sources.push({
      name: "portability",
      verdict: VERDICT_UNAVAILABLE,
      reason: describeError(error),
    });
    return [];
  }
}

function toCell(platform: PlatformEntry) {
	const extended = platform as PlatformEntry & {
		controls?: string[];
		absent?: string[];
		declarers?: Array<{ name: string; role: string; resolved: boolean; reason: string }>;
	};
	return {
    status: STATUS_NAMES[platform.status] ?? "unspecified",
    qualification: QUALIFICATION_NAMES[platform.qualification] ?? "unspecified",
    implementer: text(platform.implementer),
    mechanism: text(platform.mechanism),
    reason: [platform.reason, platform.qualificationReason].filter(Boolean).join(" — "),
		controls: extended.controls ?? [],
		absent: extended.absent ?? [],
		declarers: (extended.declarers ?? []).map((declarer) => ({
      name: declarer.name,
      role: declarer.role,
      resolved: declarer.resolved,
      reason: declarer.reason,
    })),
  };
}

function describeError(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
