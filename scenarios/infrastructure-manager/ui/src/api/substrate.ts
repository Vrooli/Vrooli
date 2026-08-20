import {
  ConfidenceLevel,
  Projection,
  type CellStatus,
} from "@vrooli/proto-types/infrastructure-manager/v1/coverage/coverage_pb";
// A Cell carries the SHARED projection enum, which is a different generated
// type from the coverage domain's own Projection even though the two share
// their wire numbers. Importing both under distinct names keeps the comparison
// type-safe instead of casting the difference away.
import { Projection as CellProjection } from "@vrooli/proto-types/infrastructure-manager/v1/shared/cell_pb";

import {
  HostOS,
  Qualification,
  ResolutionStatus,
  type PlatformEntry,
} from "@vrooli/proto-types/infrastructure-manager/v1/portability/portability_pb";

import { coverageClient, fetchPortabilityGrid } from "./reliability";
import {
  type DeviceNode,
  type PortabilityRow,
  type SourceStatus,
  type SubstrateBoard,
} from "../features/substrate/model";

/**
 * Assembles the Substrate Board from the instrument's read surfaces.
 *
 * The board is a JOIN across three sources with different owners:
 *
 *  1. the substrate COVERAGE cells — authored by `vrooli-autoheal` in
 *     `docs/spaces/substrate-space.md`, read here through this scenario's
 *     coverage domain. This is where the denominator, its confidence, and the
 *     `gap_open_days` that date every declared blindness come from.
 *  2. the DEVICE GRAPH — collected by `system-monitor`, read through the
 *     substrate device source.
 *  3. the PORTABILITY grid — this scenario's `portability` domain.
 *
 * THE CONTRACT THIS FILE ENFORCES: a source that does not answer produces a
 * `SourceStatus` with a non-VALID verdict and a reason. It NEVER produces an
 * empty device list that the board would then render as "nothing attached", and
 * it never substitutes a default. That distinction is the scenario's entire
 * trust model — an owner outage must not read as a coverage collapse — so it is
 * enforced here at the boundary rather than left to each component.
 */

/** Trust verdicts this adapter can assign. Mirrors `docs/concepts/TRUST-MODEL.md`. */
const VERDICT_VALID = "VALID";
const VERDICT_UNAVAILABLE = "UNAVAILABLE";

const CONFIDENCE_NAMES: Record<number, SubstrateBoard["denominator"]["confidence"]> = {
  [ConfidenceLevel.AUTHORITATIVE]: "AUTHORITATIVE",
  [ConfidenceLevel.PARTIAL]: "PARTIAL",
};

function confidenceName(level: number | undefined): SubstrateBoard["denominator"]["confidence"] {
  // Anything the enum does not name resolves to SKETCH rather than to the most
  // flattering value. An unrecognised confidence is not evidence of a good
  // denominator; it is evidence that nobody stated one.
  return (level !== undefined && CONFIDENCE_NAMES[level]) || "SKETCH";
}

export async function fetchSubstrateBoard(): Promise<SubstrateBoard> {
  const sources: SourceStatus[] = [];

  const coverage = await readSubstrateCoverage(sources);
  const devices = readDeviceGraph(sources);
  const portability = await readPortability(sources);

  return {
    host: coverage.host,
    devices,
    portability,
    sources,
    denominator: coverage.denominator,
  };
}

interface CoverageRead {
  host: SubstrateBoard["host"];
  denominator: SubstrateBoard["denominator"];
  /** `gap_open_days` keyed by cell id, so a blind region can be dated. */
  gapDaysByCell: Readonly<Record<string, number>>;
}

async function readSubstrateCoverage(sources: SourceStatus[]): Promise<CoverageRead> {
  try {
    const [coverage, cells] = await Promise.all([
      coverageClient.getCoverage({}),
      coverageClient.listCells({}),
    ]);
    const substrate = coverage.projections.find(
      (projection) => projection.projection === Projection.SUBSTRATE,
    );
    const gapDaysByCell: Record<string, number> = {};
    for (const cell of cells.cells) {
      if (cell.projection !== CellProjection.SUBSTRATE) continue;
      if (cell.gapOpenDays > 0) {
        gapDaysByCell[cell.id] = cell.gapOpenDays;
      }
    }
    sources.push({ name: "coverage", verdict: VERDICT_VALID, reason: null });
    return {
      host: hostIdentity(),
      denominator: {
        confidence: confidenceName(substrate?.confidence?.level),
        rationale:
          substrate?.confidence?.rationale ||
          substrate?.unavailableReason ||
          "No denominator rationale was supplied for the substrate projection.",
      },
      gapDaysByCell,
    };
  } catch (error) {
    sources.push({
      name: "coverage",
      verdict: VERDICT_UNAVAILABLE,
      reason: describeError(error),
    });
    return {
      host: hostIdentity(),
      denominator: {
        // The denominator is UNKNOWN, not good. Reporting SKETCH here is the
        // honest floor: no ratio computed against it should be trusted, and the
        // board prints the confidence beside every figure so the reader knows.
        confidence: "SKETCH",
        rationale:
          "The coverage source did not answer, so the substrate denominator could not be read. No ratio on this board is graded against an authored denominator.",
      },
      gapDaysByCell: {},
    };
  }
}

/**
 * Reads the device graph through the substrate device source.
 *
 * NOT YET WIRED. `system-monitor` collects the device graph on a 30s cached
 * provider (`api/internal/collectors/devicegraph.go`) and publishes it as
 * metrics, but exposes no typed read verb for it yet; the join is tracked as
 * cells `SB9`-`SB13` in `scenarios/vrooli-autoheal/docs/spaces/substrate-space.md`,
 * which are `IN-REACH` precisely because the sensor ships and the JOIN does not.
 *
 * Until that verb exists this reports the source as UNAVAILABLE with that
 * reason. It deliberately does NOT return an empty device list as though the
 * machine had nothing attached, and it deliberately does not carry fixture data
 * — a board that shows plausible devices it did not read would be the exact
 * dishonesty the whole scenario exists to remove.
 */
function readDeviceGraph(sources: SourceStatus[]): readonly DeviceNode[] {
  sources.push({
    name: "device-graph",
    verdict: VERDICT_UNAVAILABLE,
    reason:
      "system-monitor collects the device graph but exposes no typed read verb for it yet; substrate cells SB9-SB13 are IN-REACH pending that join.",
  });
  return [];
}

/** Wire enum -> the qualification rung name the matrix renders. */
const QUALIFICATION_NAMES: Record<number, string> = {
  [Qualification.QUALIFIED]: "qualified",
  [Qualification.BUILD_VERIFIED]: "build-verified",
  [Qualification.UNQUALIFIED]: "unqualified",
  [Qualification.DEGRADED]: "degraded",
  [Qualification.INELIGIBLE]: "ineligible",
  [Qualification.UNDECLARED]: "undeclared",
};

/** Wire enum -> the resolution status name. */
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
 * An UNSPECIFIED qualification is mapped to the literal string "unspecified"
 * rather than to any valid rung. The matrix renders an unrecognised rung with
 * the excursion treatment, so a wire value nobody handled becomes VISIBLE
 * instead of quietly rendering as the most flattering thing on the list.
 */
async function readPortability(sources: SourceStatus[]): Promise<readonly PortabilityRow[]> {
  try {
    const response = await fetchPortabilityGrid();
    const grid = response.grid;
    if (!grid) {
      sources.push({
        name: "portability",
        verdict: VERDICT_UNAVAILABLE,
        reason: "the portability domain answered without a grid.",
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
          .map((platform: PlatformEntry) => [HOST_OS_NAMES[platform.hostOs], toCell(platform)] as const)
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
  return {
    status: STATUS_NAMES[platform.status] ?? "unspecified",
    qualification: QUALIFICATION_NAMES[platform.qualification] ?? "unspecified",
    implementer: platform.implementer || null,
    mechanism: platform.mechanism || null,
    reason: [platform.reason, platform.qualificationReason].filter(Boolean).join(" — "),
  };
}

/**
 * The host this board describes.
 *
 * Identity comes from the coverage read's own context rather than from the
 * browser, which knows nothing about the machine the instrument is watching.
 * Until the device source supplies it, this names the instrument's own host in
 * the only terms the UI can honestly claim.
 */
function hostIdentity(): SubstrateBoard["host"] {
  return { name: "host", os: "", arch: "" };
}

function describeError(error: unknown): string {
  if (error instanceof Error) return error.message;
  return String(error);
}

/** Re-exported for tests that need to assert on cell status without the enum. */
export type { CellStatus };
