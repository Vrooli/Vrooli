import { describe, expect, it } from "vitest";

import { describeConstellation } from "./DeviceConstellation";
import {
  groupByClass,
  ladderCoverage,
  unseenDevices,
  worstBlindDays,
  type DeviceNode,
} from "./model";
import { RUNG_ORDER, type Rung, type SignalState } from "../../theme/instrument";

/**
 * These assert the Substrate Board's HONESTY INVARIANTS.
 *
 * Every one of them corresponds to a way a monitoring surface normally lies:
 * printing zero when it read nothing, counting a partial reading as coverage,
 * letting an instrument outage look like a coverage collapse, or counting a
 * rung that does not apply as a gap. The board exists to not do those things,
 * so these are the tests that matter most in this feature.
 */

const ring = (...states: SignalState[]): Record<Rung, SignalState> =>
  RUNG_ORDER.reduce(
    (acc, rung, index) => {
      acc[rung] = states[index] ?? "BLIND";
      return acc;
    },
    {} as Record<Rung, SignalState>,
  );

function device(overrides: Partial<DeviceNode> & Pick<DeviceNode, "id">): DeviceNode {
  return {
    name: overrides.id,
    deviceClass: "block-device",
    parent: null,
    vendor: null,
    driver: null,
    rungs: ring("COVERED", "COVERED", "COVERED", "COVERED", "COVERED"),
    reasons: {},
    remediation: {},
    blindDays: {},
    discoveredBy: "linux-sysfs-device-tree",
    enrichedBy: [],
    nodes: [],
    ...overrides,
  };
}

describe("ladder coverage", () => {
  it("returns null rather than zero when there is nothing to count", () => {
    // A board that prints 0% when it read nothing is indistinguishable from a
    // board that read everything and found nothing. Opposite facts.
    expect(ladderCoverage([])).toBeNull();
  });

  it("does not count a partial reading as coverage", () => {
    const result = ladderCoverage([
      device({ id: "a", rungs: ring("COVERED", "PARTIAL", "PARTIAL", "PARTIAL", "PARTIAL") }),
    ]);
    expect(result).toEqual({ covered: 1, total: 5, ratio: 1 / 5 });
  });

  it("excludes an unreachable source from both numerator and denominator", () => {
    // An outage must move the figure to a smaller denominator, never silently
    // depress the percentage as though the plant had got worse.
    const result = ladderCoverage([
      device({ id: "a", rungs: ring("COVERED", "COVERED", "SOURCE_DOWN", "SOURCE_DOWN", "SOURCE_DOWN") }),
    ]);
    expect(result).toEqual({ covered: 2, total: 2, ratio: 1 });
  });

  it("excludes a rung that cannot apply to the device class", () => {
    const result = ladderCoverage([
      device({ id: "a", rungs: ring("COVERED", "NOT_APPLICABLE", "NOT_APPLICABLE", "COVERED", "BLIND") }),
    ]);
    expect(result).toEqual({ covered: 2, total: 3, ratio: 2 / 3 });
  });

  it("counts an unmeasurable rung as uncovered, not as excluded", () => {
    // The host COULD produce this value and refused. That is a real gap and it
    // must depress the figure, unlike a rung that does not apply.
    const result = ladderCoverage([
      device({ id: "a", rungs: ring("COVERED", "COVERED", "COVERED", "COVERED", "UNMEASURABLE") }),
    ]);
    expect(result).toEqual({ covered: 4, total: 5, ratio: 4 / 5 });
  });
});

describe("unseen devices", () => {
  it("finds a device with no covered rung", () => {
    const blind = device({ id: "igpu", rungs: ring("BLIND", "BLIND", "BLIND", "BLIND", "BLIND") });
    expect(unseenDevices([device({ id: "ok" }), blind])).toEqual([blind]);
  });

  it("does not count a device whose every rung is graded elsewhere", () => {
    const elsewhere = device({
      id: "pci-function",
      rungs: ring(
        "NOT_APPLICABLE",
        "NOT_APPLICABLE",
        "NOT_APPLICABLE",
        "NOT_APPLICABLE",
        "NOT_APPLICABLE",
      ),
    });
    expect(unseenDevices([elsewhere])).toEqual([]);
  });
});

describe("class roll-up", () => {
  it("takes the worst member state, so one healthy device cannot average away a blind one", () => {
    const groups = groupByClass([
      device({ id: "good", deviceClass: "block-device" }),
      device({
        id: "bad",
        deviceClass: "block-device",
        rungs: ring("COVERED", "COVERED", "COVERED", "COVERED", "UNMEASURABLE"),
      }),
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0]?.rungs.ANTICIPATION).toBe("UNMEASURABLE");
  });

  it("does not let an inapplicable rung win the worst-of roll-up", () => {
    const groups = groupByClass([
      device({
        id: "a",
        rungs: ring("COVERED", "COVERED", "COVERED", "COVERED", "NOT_APPLICABLE"),
      }),
      device({ id: "b" }),
    ]);
    expect(groups[0]?.rungs.ANTICIPATION).toBe("COVERED");
  });
});

describe("blindness dating", () => {
  it("reports the longest-standing blindness on a device", () => {
    expect(worstBlindDays(device({ id: "a", blindDays: { EVIDENCE: 7, ANTICIPATION: 114 } }))).toBe(114);
  });

  it("returns null rather than zero when nothing is dated", () => {
    expect(worstBlindDays(device({ id: "a" }))).toBeNull();
  });
});

describe("constellation text alternative", () => {
  it("states the finding rather than describing a picture", () => {
    const groups = groupByClass([
      device({ id: "gpu", deviceClass: "graphics-device", rungs: ring("BLIND", "BLIND", "BLIND", "BLIND", "BLIND") }),
      device({ id: "disk", deviceClass: "block-device" }),
    ]);
    const description = describeConstellation("swarminator", groups);
    expect(description).toContain("graphics-device");
    expect(description).toMatch(/no covered rung/);
  });

  it("says it read nothing rather than implying the machine is empty", () => {
    const description = describeConstellation("swarminator", []);
    expect(description).toContain("read nothing");
    expect(description).toContain("not the same as the machine having nothing attached");
  });
});
