import { RUNG_ORDER, type Rung, type SignalState } from "../../theme/instrument";

/**
 * The Substrate Board's view model.
 *
 * This is deliberately a separate layer from the wire types. The board renders
 * a JOIN across three sources — the device graph in `system-monitor`, the
 * portability grid in this scenario's `portability` domain, and the substrate
 * coverage cells authored by `vrooli-autoheal` — and a presentational component
 * that had to understand three proto packages would be impossible to reason
 * about. `api/substrate.ts` owns the translation; everything under
 * `features/substrate/` speaks only the types below.
 *
 * One rule governs every type here: an absent value is never `0` and never
 * `false`. It is `null` with a reason, or an explicit state. The board exists to
 * make blindness visible, so its view model must be incapable of hiding it.
 */

/** A device class as the board groups it. Free-form: the collector owns the set. */
export type DeviceClass = string;

export interface DeviceNode {
  /** Platform-durable identity, e.g. `pci:0000:01:00.0`. Stable across reboots. */
  id: string;
  /** Display name, e.g. "AD103 [GeForce RTX 4070 Ti SUPER]". */
  name: string;
  /** Device class, e.g. "graphics-device". */
  deviceClass: DeviceClass;
  /** Identity of the parent bus or controller; `null` at a tree root. */
  parent: string | null;
  vendor: string | null;
  driver: string | null;
  /** One state per ladder rung. */
  rungs: Readonly<Record<Rung, SignalState>>;
  /**
   * Per-rung reason. REQUIRED wherever a rung is `UNMEASURABLE` or
   * `UNAVAILABLE` — those states are only honest when they carry their reason.
   */
  reasons: Readonly<Partial<Record<Rung, string>>>;
  /** Per-rung remediation, where the collector names one. */
  remediation: Readonly<Partial<Record<Rung, string>>>;
  /** Per-rung blindness age in days, from the coverage model's `gap_open_days`. */
  blindDays: Readonly<Partial<Record<Rung, number>>>;
  /** How this device was discovered. A vendor tool here would be a defect. */
  discoveredBy: string | null;
  /** Probes that enriched an already-discovered device. */
  enrichedBy: readonly string[];
  /** Kernel-visible nodes, e.g. `card1`, `renderD128`. */
  nodes: readonly string[];
}

/** One row of the portability matrix: a capability across operating systems. */
export interface PortabilityRow {
  capability: string;
  /** Keyed by host OS: linux, macos, windows. */
  platforms: Readonly<Record<string, PortabilityCell>>;
}

export interface PortabilityCell {
  /** implemented | degraded | ineligible | unwired | peerless | status_invalid */
  status: string;
  /** qualified | build-verified | unqualified | degraded | ineligible | undeclared */
  qualification: string;
  implementer: string | null;
  mechanism: string | null;
  reason: string;
}

/**
 * A source the board read from, and whether it answered.
 *
 * This is instrument state, not plant state, and the board renders it on the
 * chrome rather than in the data. An owner outage must never read as a
 * coverage collapse — the distinction is the scenario's whole trust model.
 */
export interface SourceStatus {
  name: string;
  /** VALID | UNAVAILABLE | UNTRUSTED | ... from the trust vocabulary. */
  verdict: string;
  reason: string | null;
}

export interface SubstrateBoard {
  /** The machine this board describes. */
  host: {
    name: string;
    os: string;
    arch: string;
  };
  devices: readonly DeviceNode[];
  portability: readonly PortabilityRow[];
  sources: readonly SourceStatus[];
  /**
   * The authored denominator's confidence. A ratio printed without this is
   * exactly the unqualified number this instrument refuses to produce.
   */
  denominator: {
    confidence: "AUTHORITATIVE" | "PARTIAL" | "SKETCH";
    rationale: string;
  };
}

/**
 * Counts covered rungs over total rungs across every device.
 *
 * Returns `null` — never `0` — when there is nothing to count, because a board
 * that prints "0%" when it read nothing is indistinguishable from a board that
 * read everything and found nothing, and those are opposite facts.
 *
 * Only `COVERED` counts toward the numerator. `PARTIAL` deliberately does not:
 * a headline figure that counts partial coverage as coverage is the kind of
 * flattering arithmetic this surface exists to remove. Rungs whose state
 * describes the instrument rather than the plant (`UNAVAILABLE`) are excluded
 * from BOTH numerator and denominator, so an outage moves the figure to a
 * smaller denominator rather than silently depressing the percentage.
 */
export function ladderCoverage(
  devices: readonly DeviceNode[],
): { covered: number; total: number; ratio: number } | null {
  let covered = 0;
  let total = 0;
  for (const device of devices) {
    for (const rung of RUNG_ORDER) {
      const state = device.rungs[rung];
      // Excluded from BOTH numerator and denominator: a rung that cannot apply
      // to this device class is not a gap, and a source that did not answer
      // tells us nothing about the plant. Counting either would move the
      // headline figure for a reason that has nothing to do with coverage.
      if (state === "NOT_APPLICABLE" || state === "SOURCE_DOWN") continue;
      total += 1;
      if (state === "COVERED") covered += 1;
    }
  }
  if (total === 0) return null;
  return { covered, total, ratio: covered / total };
}

/** Devices with no covered rung at all — the ones the board must make obvious. */
export function unseenDevices(devices: readonly DeviceNode[]): readonly DeviceNode[] {
  return devices.filter((device) => {
    const gradeable = RUNG_ORDER.filter((rung) => device.rungs[rung] !== "NOT_APPLICABLE");
    // A device every one of whose rungs is inapplicable is graded elsewhere,
    // not unseen. Counting it as unseen would inflate the one figure on this
    // board that must never be inflated.
    if (gradeable.length === 0) return false;
    return gradeable.every((rung) => device.rungs[rung] !== "COVERED");
  });
}

/** The longest-standing blindness on a device, in days, or `null` if none is dated. */
export function worstBlindDays(device: DeviceNode): number | null {
  const ages = RUNG_ORDER.map((rung) => device.blindDays[rung]).filter(
    (value): value is number => typeof value === "number",
  );
  return ages.length === 0 ? null : Math.max(...ages);
}

/**
 * Groups devices by class for the constellation.
 *
 * The board arranges CLASSES around the host, not individual devices: this host
 * alone reports 58 PCI functions, and 58 nodes in a ring is a decoration rather
 * than a finding. Each class node carries the worst state per rung across its
 * members, so a single unmeasurable disk cannot be averaged away by a healthy one.
 */
export function groupByClass(devices: readonly DeviceNode[]): readonly DeviceClassGroup[] {
  const byClass = new Map<DeviceClass, DeviceNode[]>();
  for (const device of devices) {
    const bucket = byClass.get(device.deviceClass);
    if (bucket) {
      bucket.push(device);
    } else {
      byClass.set(device.deviceClass, [device]);
    }
  }
  return [...byClass.entries()]
    .map(([deviceClass, members]) => ({
      deviceClass,
      members,
      rungs: worstRungStates(members),
    }))
    .sort((a, b) => a.deviceClass.localeCompare(b.deviceClass));
}

export interface DeviceClassGroup {
  deviceClass: DeviceClass;
  members: readonly DeviceNode[];
  rungs: Readonly<Record<Rung, SignalState>>;
}

/**
 * Ordered worst-first. A class's rung is only as good as its weakest member,
 * because the point of the board is to surface the device that is unwatched,
 * not to report the average of the ones that are.
 */
const STATE_SEVERITY: Record<SignalState, number> = {
  // NOT_APPLICABLE ranks BELOW covered on purpose: a rung that is meaningless
  // for a class must never win a worst-of roll-up, or every class would look
  // worse than it is because one of its rungs is graded elsewhere.
  NOT_APPLICABLE: -1,
  COVERED: 0,
  PARTIAL: 1,
  UNAVAILABLE: 2,
  UNMEASURABLE: 3,
  BLIND: 4,
  // SOURCE_DOWN outranks every plant state because it invalidates them: if the
  // source did not answer, the other readings on that class are stale, and
  // presenting a stale reading as current is the failure the trust model names.
  SOURCE_DOWN: 5,
  EXCURSION: 6,
};

function worstRungStates(devices: readonly DeviceNode[]): Record<Rung, SignalState> {
  return RUNG_ORDER.reduce(
    (acc, rung) => {
      let worst: SignalState = "NOT_APPLICABLE";
      for (const device of devices) {
        const state = device.rungs[rung];
        if (STATE_SEVERITY[state] > STATE_SEVERITY[worst]) {
          worst = state;
        }
      }
      acc[rung] = worst;
      return acc;
    },
    {} as Record<Rung, SignalState>,
  );
}
