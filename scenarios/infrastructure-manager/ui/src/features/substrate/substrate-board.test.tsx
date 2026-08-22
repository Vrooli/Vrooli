import { describe, expect, it } from "vitest";

import { describeConstellation } from "./DeviceConstellation";
import {
  isUnseen,
  ladderCoverage,
  rungBlindDays,
  rungReasons,
  rungStates,
  ungradedCells,
  unseenClasses,
  worstBlindDays,
  type DeviceClassNode,
  type RungDetail,
} from "./model";
import { RUNG_ORDER, SIGNAL_STATES, type Rung, type SignalState } from "../../theme/instrument";

/**
 * These assert the Substrate Board's HONESTY INVARIANTS.
 *
 * Every one corresponds to a way a monitoring surface normally lies: printing
 * zero when it read nothing, counting a partial reading as coverage, letting an
 * instrument outage look like a coverage collapse, or counting a rung that does
 * not apply as a gap. The board exists to not do those things, so these are the
 * tests that matter most in this feature.
 */

function detail(state: SignalState, overrides: Partial<RungDetail> = {}): RungDetail {
  return {
    state,
    cellRef: null,
    question: null,
    reason: null,
    mechanism: null,
    remediation: null,
    blockedBy: null,
    trust: null,
    graded: true,
    ungradedReason: null,
    provisional: false,
    blindDays: null,
    ...overrides,
  };
}

function node(
  deviceClass: string,
  states: SignalState[],
  overrides: Partial<DeviceClassNode> = {},
): DeviceClassNode {
  const rungs = RUNG_ORDER.reduce(
    (acc, rung, index) => {
      acc[rung] = detail(states[index] ?? "BLIND");
      return acc;
    },
    {} as Record<Rung, RungDetail>,
  );
  return { deviceClass, rungs, deviceCount: 1, blindDevices: 0, ...overrides };
}

const COVERED: SignalState[] = ["COVERED", "COVERED", "COVERED", "COVERED", "COVERED"];

describe("ladder coverage", () => {
  it("returns null rather than zero when there is nothing to count", () => {
    // A board that prints 0% when it read nothing is indistinguishable from a
    // board that read everything and found nothing. Opposite facts.
    expect(ladderCoverage([])).toBeNull();
  });

  it("does not count a partial reading as coverage", () => {
    const result = ladderCoverage([
      node("storage", ["COVERED", "PARTIAL", "PARTIAL", "PARTIAL", "PARTIAL"]),
    ]);
    expect(result).toEqual({ covered: 1, total: 5, ratio: 1 / 5 });
  });

  it("excludes an unreachable source from both numerator and denominator", () => {
    // An outage must shrink the denominator, never silently depress the
    // percentage as though the plant itself had got worse.
    const result = ladderCoverage([
      node("storage", ["COVERED", "COVERED", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN"]),
    ]);
    expect(result).toEqual({ covered: 2, total: 2, ratio: 1 });
  });

  it("keeps an unsampled host distinct from an unreachable source", () => {
    const unsampled = detail("HOST_NOT_SAMPLED", { reasonCode: "host_not_sampled" });
    const unreachable = detail("SOURCE_DOWN", { reasonCode: "source_unavailable" });
    expect(unsampled.state).not.toBe(unreachable.state);
    expect(ladderCoverage([node("storage", [unsampled.state, unreachable.state, ...COVERED.slice(2)])])).toEqual({ covered: 3, total: 3, ratio: 1 });
  });

  it("excludes a rung that cannot apply to the class", () => {
    const result = ladderCoverage([
      node("bus", ["COVERED", "NOT_APPLICABLE", "NOT_APPLICABLE", "COVERED", "BLIND"]),
    ]);
    expect(result).toEqual({ covered: 2, total: 3, ratio: 2 / 3 });
  });

  it("counts an unmeasurable rung as uncovered, not as excluded", () => {
    // The host COULD produce this value and refused. That is a real gap and it
    // must depress the figure, unlike a rung that does not apply at all.
    const result = ladderCoverage([
      node("storage", ["COVERED", "COVERED", "COVERED", "COVERED", "UNMEASURABLE"]),
    ]);
    expect(result).toEqual({ covered: 4, total: 5, ratio: 4 / 5 });
  });
});

describe("unseen classes", () => {
  it("finds a class with no covered rung", () => {
    const blind = node("graphics-device", ["BLIND", "BLIND", "BLIND", "BLIND", "BLIND"]);
    expect(unseenClasses([node("storage", COVERED), blind])).toEqual([blind]);
  });

  it("does not count a class whose every rung is graded elsewhere", () => {
    const elsewhere = node("pci-device", [
      "NOT_APPLICABLE",
      "NOT_APPLICABLE",
      "NOT_APPLICABLE",
      "NOT_APPLICABLE",
      "NOT_APPLICABLE",
    ]);
    expect(isUnseen(elsewhere)).toBe(false);
    expect(unseenClasses([elsewhere])).toEqual([]);
  });
});

describe("ungraded cells", () => {
  it("counts a measured rung that no setpoint bar graded", () => {
    // Measured-but-ungraded is not passing: nothing judged it. Leaving it out
    // of this count would let an unjudged reading pass as a verdict.
    const ungradedNode = node("thermal", COVERED);
    const rungs = { ...ungradedNode.rungs };
    rungs.ANTICIPATION = detail("COVERED", { graded: false, ungradedReason: "no bar resolves" });
    expect(ungradedCells([{ ...ungradedNode, rungs }])).toBe(1);
  });

  it("does not count an inapplicable rung as ungraded", () => {
    const naNode = node("bus", ["COVERED", "NOT_APPLICABLE", "COVERED", "COVERED", "COVERED"]);
    const rungs = { ...naNode.rungs };
    rungs.TELEMETRY = detail("NOT_APPLICABLE", { graded: false });
    expect(ungradedCells([{ ...naNode, rungs }])).toBe(0);
  });
});

describe("blindness dating", () => {
  it("reports the longest-standing blindness on a class", () => {
    const dated = node("memory", COVERED);
    const rungs = { ...dated.rungs };
    rungs.EVIDENCE = detail("BLIND", { blindDays: 7 });
    rungs.ANTICIPATION = detail("BLIND", { blindDays: 114 });
    expect(worstBlindDays({ ...dated, rungs })).toBe(114);
  });

  it("returns null rather than zero when nothing is dated", () => {
    expect(worstBlindDays(node("storage", COVERED))).toBeNull();
  });

  it("omits a zero-day age rather than reporting it as a dated gap", () => {
    // `gap_open_days` is 0 both for a gap declared today and for one nobody
    // ever dated. Reporting "blind for 0 days" would invent a date.
    const zeroed = node("storage", COVERED);
    const rungs = { ...zeroed.rungs };
    rungs.ANTICIPATION = detail("BLIND", { blindDays: 0 });
    expect(rungBlindDays({ ...zeroed, rungs })).toEqual({});
  });
});

describe("rung projections", () => {
  it("flattens states for the ring and the grid", () => {
    expect(rungStates(node("storage", COVERED)).IDENTITY).toBe("COVERED");
  });

  it("carries only the reasons that exist", () => {
    const withReason = node("storage", COVERED);
    const rungs = { ...withReason.rungs };
    rungs.ANTICIPATION = detail("UNMEASURABLE", { reason: "permission denied" });
    expect(rungReasons({ ...withReason, rungs })).toEqual({ ANTICIPATION: "permission denied" });
  });
});

describe("constellation text alternative", () => {
  it("states the finding rather than describing a picture", () => {
    const description = describeConstellation("swarminator", [
      node("graphics-device", ["BLIND", "BLIND", "BLIND", "BLIND", "BLIND"]),
      node("block-device", COVERED),
    ]);
    expect(description).toContain("graphics-device");
    expect(description).toMatch(/no covered rung/);
  });

  it("says it read nothing rather than implying the machine is empty", () => {
    const description = describeConstellation("swarminator", []);
    expect(description).toContain("read nothing");
    expect(description).toContain("not the same as the machine having nothing attached");
  });
});

describe("unauthored cells", () => {
  it("is excluded from coverage, like an inapplicable rung and for a different reason", () => {
    // Nobody declared a cell here, so there is no question to be right or wrong
    // about. Counting it as a gap would blame the machine for the space
    // document being under-declared.
    const result = ladderCoverage([
      node("usb-device", ["COVERED", "UNAUTHORED", "UNAUTHORED", "UNAUTHORED", "UNAUTHORED"]),
    ]);
    expect(result).toEqual({ covered: 1, total: 1, ratio: 1 });
  });

  it("does not make a class read as unseen", () => {
    // A class whose only graded rung is covered is SEEN, even if the other four
    // were never declared. Reporting it unseen would invent blindness.
    expect(
      isUnseen(node("usb-device", ["COVERED", "UNAUTHORED", "UNAUTHORED", "UNAUTHORED", "UNAUTHORED"])),
    ).toBe(false);
  });

  it("is not counted as an ungraded cell", () => {
    // "No bar grades this reading" and "no cell was ever authored" are
    // different problems with different owners; only the first is a setpoint
    // gap, and the headline figure counts only that.
    const undeclared = node("usb-device", [
      "COVERED",
      "UNAUTHORED",
      "UNAUTHORED",
      "UNAUTHORED",
      "UNAUTHORED",
    ]);
    const rungs = { ...undeclared.rungs };
    for (const rung of ["TELEMETRY", "EVIDENCE", "CONTROL", "ANTICIPATION"] as const) {
      rungs[rung] = detail("UNAUTHORED", { graded: false });
    }
    expect(ungradedCells([{ ...undeclared, rungs }])).toBe(0);
  });

  it("is visually distinct from blind and from a source outage", () => {
    // Three different facts — nobody asked, nobody is watching, and nobody
    // answered — must never share a treatment.
    const tones = new Set(
      (["UNAUTHORED", "BLIND", "SOURCE_DOWN"] as const).map((state) => SIGNAL_STATES[state].tone),
    );
    expect(tones.size).toBe(3);
  });
});
