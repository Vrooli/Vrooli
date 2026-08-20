import { RUNG_ORDER, type Rung, type SignalState } from "../../theme/instrument";

/**
 * The Substrate Board's view model.
 *
 * This is deliberately a separate layer from the wire types. The board renders
 * a JOIN across three owners — the device graph collected by `system-monitor`,
 * the substrate cells authored by `vrooli-autoheal`, and the capability grid
 * served by this scenario's `portability` domain — and a presentational
 * component that had to understand three proto packages would be impossible to
 * reason about. `api/substrate.ts` owns the translation; everything under
 * `features/substrate/` speaks only the types below.
 *
 * The unit is the DEVICE CLASS, not the individual device, because that is the
 * unit the instrument actually grades: a ladder cell is
 * `(device class, rung, host OS)`. It is also the only unit that reads — this
 * host alone reports 58 PCI functions, and 58 nodes around a hub is decoration
 * rather than a finding. Per-class device counts carry the population so a
 * class is never quoted without it.
 *
 * ONE RULE GOVERNS EVERY TYPE HERE: an absent value is never `0` and never
 * `false`. It is `null`, or an explicit state with a reason. The board exists to
 * make blindness visible, so its view model must be incapable of hiding it.
 */

/** Per-rung detail for one class. Every field may be absent; none defaults. */
export interface RungDetail {
  state: SignalState;
  /** The authored substrate cell this rung answers, e.g. `substrate/SB10`. */
  cellRef: string | null;
  /** The question the cell asks, from the owner's space document. */
  question: string | null;
  /** Why the state is what it is. Required for every non-covered state. */
  reason: string | null;
  /** The interface the reading was, or would be, taken from. */
  mechanism: string | null;
  /** The declared host change that would close the gap. */
  remediation: string | null;
  /** The lower rung whose blindness suppressed this grade, if any. */
  blockedBy: Rung | null;
  /** Trust verdict on the reading behind this rung. */
  trust: string | null;
  /** Whether a setpoint bar actually graded this cell. */
  graded: boolean;
  /** When `graded` is false, why. Never empty when `graded` is false. */
  ungradedReason: string | null;
  /** Whether the bar behind this cell is still provisional. */
  provisional: boolean;
  /** How long this has been blind, from the coverage model's `gap_open_days`. */
  blindDays: number | null;
}

export interface DeviceClassNode {
  /** Stable key: the device class as the collector names it. */
  deviceClass: string;
  /** One graded detail per ladder rung. */
  rungs: Readonly<Record<Rung, RungDetail>>;
  /** How many devices of this class the graph enumerated; `null` if unread. */
  deviceCount: number | null;
  /** How many of those have no covered rung; `null` if unread. */
  blindDevices: number | null;
}

/** One row of the portability matrix: a capability across operating systems. */
export interface PortabilityRow {
  capability: string;
  /** Keyed by host OS: linux, macos, windows. */
  platforms: Readonly<Record<string, PortabilityCell>>;
}

export interface PortabilityCell {
  status: string;
  qualification: string;
  implementer: string | null;
  mechanism: string | null;
  reason: string;
}

/**
 * A source the board read from, and whether it answered.
 *
 * This is INSTRUMENT state, not plant state, and the board renders it on the
 * chrome rather than in the data. An owner outage must never read as a coverage
 * collapse — that distinction is the scenario's whole trust model.
 */
export interface SourceStatus {
  name: string;
  /** VALID | UNAVAILABLE | UNTRUSTED | ... from the trust vocabulary. */
  verdict: string;
  reason: string | null;
}

/** Substrate sensing declared for one host OS, always with its denominator. */
export interface CheckPlatformCoverage {
  hostOs: string;
  applicable: number;
  total: number;
  /**
   * Checks declaring no platform at all. An empty platform declaration means
   * the check applies EVERYWHERE, not that its platform is unknown, so these
   * are the reason a host OS can have applicable checks while no check names it.
   */
  universal: number;
  available: boolean;
  reason: string | null;
}

export interface SubstrateBoard {
  host: { name: string; os: string };
  classes: readonly DeviceClassNode[];
  portability: readonly PortabilityRow[];
  sources: readonly SourceStatus[];
  checkPlatforms: readonly CheckPlatformCoverage[];
  /**
   * Whether the substrate space document itself could be read. When false, no
   * cell carries an authored status and no ratio on the board is meaningful.
   */
  coverageAvailable: boolean;
  coverageReason: string | null;
  denominator: {
    confidence: "AUTHORITATIVE" | "PARTIAL" | "SKETCH";
    rationale: string;
  };
}

/**
 * Counts covered rungs over gradeable rungs across every class.
 *
 * Returns `null` — never `0` — when there is nothing to count, because a board
 * that prints "0%" when it read nothing is indistinguishable from a board that
 * read everything and found nothing, and those are opposite facts.
 *
 * Only `COVERED` counts toward the numerator. `PARTIAL` deliberately does not:
 * counting partial coverage as coverage is the flattering arithmetic this
 * surface exists to remove.
 *
 * Three states are excluded from BOTH numerator and denominator, for three
 * different reasons, and keeping them straight is the whole job:
 *  - `NOT_APPLICABLE` — the rung is meaningless for this class and is graded
 *    elsewhere, so counting it would manufacture a gap nobody has.
 *  - `UNAUTHORED` — nobody has declared a cell for it, so there is no question
 *    to be right or wrong about. This is a fact about the DENOMINATOR, and it
 *    is why the space reports PARTIAL confidence rather than a clean ratio.
 *  - `SOURCE_DOWN` — the source did not answer, so it says nothing about the
 *    plant at all.
 * Excluding rather than zero-counting means an outage or an under-declared
 * space shrinks the denominator, instead of silently depressing the percentage
 * as though the machine itself had got worse.
 */
export function ladderCoverage(
  classes: readonly DeviceClassNode[],
): { covered: number; total: number; ratio: number } | null {
  let covered = 0;
  let total = 0;
  for (const node of classes) {
    for (const rung of RUNG_ORDER) {
      const state = node.rungs[rung].state;
      if (state === "NOT_APPLICABLE" || state === "UNAUTHORED" || state === "SOURCE_DOWN") continue;
      total += 1;
      if (state === "COVERED") covered += 1;
    }
  }
  if (total === 0) return null;
  return { covered, total, ratio: covered / total };
}

/** Classes with no covered rung at all — the holes the board must make obvious. */
export function unseenClasses(
  classes: readonly DeviceClassNode[],
): readonly DeviceClassNode[] {
  return classes.filter(isUnseen);
}

export function isUnseen(node: DeviceClassNode): boolean {
  const gradeable = RUNG_ORDER.filter(
    (rung) => node.rungs[rung].state !== "NOT_APPLICABLE" && node.rungs[rung].state !== "UNAUTHORED",
  );
  // A class with no gradeable rung at all is either graded elsewhere or not yet
  // declared — neither is blindness. Counting it would inflate the one figure
  // on this board that must never inflate.
  if (gradeable.length === 0) return false;
  return gradeable.every((rung) => node.rungs[rung].state !== "COVERED");
}

/** The longest-standing blindness on a class, in days, or `null` if undated. */
export function worstBlindDays(node: DeviceClassNode): number | null {
  const ages = RUNG_ORDER.map((rung) => node.rungs[rung].blindDays).filter(
    (value): value is number => typeof value === "number" && value > 0,
  );
  return ages.length === 0 ? null : Math.max(...ages);
}

/** The rung states of a class, flattened for the ring and the grid. */
export function rungStates(node: DeviceClassNode): Record<Rung, SignalState> {
  return RUNG_ORDER.reduce(
    (acc, rung) => {
      acc[rung] = node.rungs[rung].state;
      return acc;
    },
    {} as Record<Rung, SignalState>,
  );
}

/** Per-rung reasons, for the lamp's accessible name. */
export function rungReasons(node: DeviceClassNode): Partial<Record<Rung, string>> {
  return RUNG_ORDER.reduce<Partial<Record<Rung, string>>>((acc, rung) => {
    const reason = node.rungs[rung].reason;
    if (reason) acc[rung] = reason;
    return acc;
  }, {});
}

/** Per-rung blindness ages, for the lamp's dated label. */
export function rungBlindDays(node: DeviceClassNode): Partial<Record<Rung, number>> {
  return RUNG_ORDER.reduce<Partial<Record<Rung, number>>>((acc, rung) => {
    const days = node.rungs[rung].blindDays;
    if (typeof days === "number" && days > 0) acc[rung] = days;
    return acc;
  }, {});
}

/**
 * How many cells the instrument could not grade, and therefore how much of the
 * board is a statement about the SETPOINT rather than about the machine.
 */
export function ungradedCells(classes: readonly DeviceClassNode[]): number {
  let count = 0;
  for (const node of classes) {
    for (const rung of RUNG_ORDER) {
      const detail = node.rungs[rung];
      if (!detail.graded && detail.state !== "NOT_APPLICABLE" && detail.state !== "UNAUTHORED") {
        count += 1;
      }
    }
  }
  return count;
}
