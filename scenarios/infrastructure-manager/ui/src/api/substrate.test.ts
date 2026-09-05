import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  coverageClient: { getCoverage: vi.fn(), listCells: vi.fn() },
  fetchLadder: vi.fn(),
  fetchPortabilityGrid: vi.fn(),
}));
vi.mock("./reliability", () => mocks);

import { fetchSubstrateBoard } from "./substrate";

/**
 * These exercise the board adapter's projection rules — the point where wire
 * values become lamp states, and therefore the single place where this
 * instrument could most easily start lying.
 *
 * The ORDER of the projection matters as much as the mapping: an unread source
 * outranks everything, an inapplicable rung must not read as a gap, and only a
 * measured, in-band, graded reading may light.
 */

// Wire enum values, written out so a change to the generated enum breaks these
// tests loudly instead of silently re-mapping a state.
const RUNG_IDENTITY = 1;
const OBS_MEASURED = 1;
const OBS_UNMEASURABLE = 2;
const OBS_UNAVAILABLE = 3;
const OBS_NOT_APPLICABLE = 4;
const OBS_BLOCKED = 5;
const OBS_UNREAD = 6;
const STATUS_NOW = 1;
const STATUS_IN_REACH = 2;
const STATUS_MISSING = 3;
const TRUST_VALID = 1;
const TRUST_UNAVAILABLE = 6;
const BAND_OUT_OF_BAND = 2;

function cell(overrides: Record<string, unknown> = {}) {
  return {
    deviceClass: "block-device",
    rung: RUNG_IDENTITY,
    hostOs: "linux",
    key: "block-device/identity/linux",
    cellRef: "substrate/SB9",
    question: "Is every attached device identified?",
    status: STATUS_NOW,
    statusSource: "space",
    observation: OBS_MEASURED,
    reason: "",
    mechanism: "",
    remediation: "",
    blockedBy: 0,
    trust: TRUST_VALID,
    unavailableReason: "",
    deviceCount: 2,
    blindDevices: 0,
    barId: "bar",
    graded: true,
    ungradedReason: "",
    band: 1,
    provisional: false,
    capability: "",
    capabilityStatus: "",
    capabilityReason: "",
    ...overrides,
  };
}

function ladderResponse(cells: ReturnType<typeof cell>[]) {
  return {
    ladder: {
      cells,
      sources: [{ id: "device-graph", available: true, reason: "" }],
      findings: [],
      hostOs: "linux",
      coverageAvailable: true,
      coverageReason: "",
      checkPlatforms: [],
    },
  };
}

function primeCoverage() {
  mocks.coverageClient.getCoverage.mockResolvedValue({ projections: [] });
  mocks.coverageClient.listCells.mockResolvedValue({ cells: [] });
  mocks.fetchPortabilityGrid.mockResolvedValue({ grid: undefined });
}

async function stateOf(overrides: Record<string, unknown>) {
  primeCoverage();
  mocks.fetchLadder.mockResolvedValue(ladderResponse([cell(overrides)]));
  const board = await fetchSubstrateBoard();
  return board.classes[0]?.rungs.IDENTITY.state;
}

afterEach(() => vi.clearAllMocks());

describe("cell state projection", () => {
  it("lights only a measured, in-band reading", async () => {
    expect(await stateOf({})).toBe("COVERED");
  });

  it("treats an unread observation as a source outage above everything else", async () => {
    // Even with an authored MISSING status, an unread source means the
    // instrument has nothing to say about the plant.
    expect(await stateOf({ observation: OBS_UNREAD, status: STATUS_MISSING })).toBe("SOURCE_DOWN");
  });

  it("treats an unavailable trust verdict as a source outage, not a plant fault", async () => {
    expect(await stateOf({ trust: TRUST_UNAVAILABLE })).toBe("SOURCE_DOWN");
  });

  it("does not render an inapplicable rung as a gap", async () => {
    expect(await stateOf({ observation: OBS_NOT_APPLICABLE, status: STATUS_MISSING })).toBe(
      "NOT_APPLICABLE",
    );
  });

  it("renders an authored MISSING status as declared blindness", async () => {
    expect(await stateOf({ status: STATUS_MISSING, observation: OBS_UNMEASURABLE })).toBe("BLIND");
  });

  it("keeps unmeasurable and mechanism-absent distinct", async () => {
    expect(await stateOf({ observation: OBS_UNMEASURABLE })).toBe("UNMEASURABLE");
    expect(await stateOf({ observation: OBS_UNAVAILABLE })).toBe("UNAVAILABLE");
  });

  it("demotes a rung blocked by a blind foundation to partial, never covered", async () => {
    expect(await stateOf({ observation: OBS_BLOCKED })).toBe("PARTIAL");
  });

  it("renders an out-of-band measured reading as an excursion", async () => {
    expect(await stateOf({ band: BAND_OUT_OF_BAND })).toBe("EXCURSION");
  });

  it("does not call an ungraded reading an excursion, because it has no bar to be out of", async () => {
    expect(await stateOf({ band: BAND_OUT_OF_BAND, graded: false })).toBe("COVERED");
  });

  it("renders an IN-REACH measured reading as partial", async () => {
    expect(await stateOf({ status: STATUS_IN_REACH })).toBe("PARTIAL");
  });

  it("renders an unrecognised observation as an excursion so it is visible", async () => {
    // Never as the most flattering state on the list.
    expect(await stateOf({ observation: 99 })).toBe("EXCURSION");
  });
});

describe("board assembly", () => {
  it("marks a rung with no authored cell as UNAUTHORED, not as a source outage", async () => {
    primeCoverage();
    mocks.fetchLadder.mockResolvedValue(ladderResponse([cell()]));
    const board = await fetchSubstrateBoard();
    expect(board.classes[0]?.rungs.IDENTITY.state).toBe("COVERED");
    expect(board.classes[0]?.rungs.ANTICIPATION.state).toBe("UNAUTHORED");
  });

  it("excludes cells for other operating systems from the local class read", async () => {
    // Only the host this instrument runs on can be refined by a live device
    // read; blending declaration-only platforms would mix two kinds of evidence.
    primeCoverage();
    mocks.fetchLadder.mockResolvedValue(
      ladderResponse([cell(), cell({ hostOs: "macos", observation: OBS_UNREAD })]),
    );
    const board = await fetchSubstrateBoard();
    expect(board.classes[0]?.rungs.IDENTITY.state).toBe("COVERED");
  });

  it("reports null device counts when every cell for a class was unread", async () => {
    // A `0` from an unread cell is a wire default, not a measurement.
    primeCoverage();
    mocks.fetchLadder.mockResolvedValue(
      ladderResponse([cell({ observation: OBS_UNREAD, deviceCount: 0 })]),
    );
    const board = await fetchSubstrateBoard();
    expect(board.classes[0]?.deviceCount).toBeNull();
  });

  it("attaches a gap age to the rung whose authored cell carries one", async () => {
    mocks.coverageClient.getCoverage.mockResolvedValue({ projections: [] });
    mocks.coverageClient.listCells.mockResolvedValue({
      cells: [{ id: "SB9", gapOpenDays: 114, projection: 11 }],
    });
    mocks.fetchPortabilityGrid.mockResolvedValue({ grid: undefined });
    mocks.fetchLadder.mockResolvedValue(ladderResponse([cell({ status: STATUS_MISSING })]));
    const board = await fetchSubstrateBoard();
    expect(board.classes[0]?.rungs.IDENTITY.blindDays).toBe(114);
  });

  it("names an unreachable ladder rather than returning an empty machine", async () => {
    primeCoverage();
    mocks.fetchLadder.mockRejectedValue(new Error("ladder unreachable"));
    const board = await fetchSubstrateBoard();
    expect(board.classes).toEqual([]);
    expect(board.sources.find((source) => source.name === "ladder")).toMatchObject({
      verdict: "UNAVAILABLE",
      reason: "ladder unreachable",
    });
  });

  it("falls back to SKETCH confidence when the denominator could not be read", async () => {
    mocks.coverageClient.getCoverage.mockRejectedValue(new Error("coverage down"));
    mocks.coverageClient.listCells.mockRejectedValue(new Error("coverage down"));
    mocks.fetchPortabilityGrid.mockResolvedValue({ grid: undefined });
    mocks.fetchLadder.mockResolvedValue(ladderResponse([cell()]));
    const board = await fetchSubstrateBoard();
    expect(board.denominator.confidence).toBe("SKETCH");
    expect(board.denominator.rationale).toMatch(/did not answer/);
  });

  it("surfaces every source the ladder itself read, verbatim", async () => {
    primeCoverage();
    const response = ladderResponse([cell()]);
    response.ladder.sources = [
      { id: "device-graph", available: false, reason: "deadline exceeded" },
    ];
    mocks.fetchLadder.mockResolvedValue(response);
    const board = await fetchSubstrateBoard();
    expect(board.sources).toContainEqual({
      name: "device-graph",
      verdict: "UNAVAILABLE",
      reason: "deadline exceeded",
    });
  });
});
