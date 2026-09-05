import { describe, it, expect } from "vitest";

import { groupMatrixRows, countUnproven, UNLINKED_GROUP_ID } from "./matrixModel";
import { makeMatrixRow } from "./mocks/factories";

describe("groupMatrixRows", () => {
  it("groups requirement rows under their operational target, preserving order", () => {
    const groups = groupMatrixRows([
      makeMatrixRow({ otId: "OT-P0-001", requirementId: "R1" }),
      makeMatrixRow({ otId: "OT-P0-001", requirementId: "R2" }),
      makeMatrixRow({ otId: "OT-P1-002", requirementId: "R3" }),
    ]);
    expect(groups.map((g) => g.otId)).toEqual(["OT-P0-001", "OT-P1-002"]);
    expect(groups[0]!.rows.map((r) => r.requirementId)).toEqual(["R1", "R2"]);
  });

  it("flags an operational target with no requirement as an orphan target", () => {
    const groups = groupMatrixRows([
      makeMatrixRow({ otId: "OT-P0-009", requirementId: "" }),
    ]);
    expect(groups[0]!.isOrphanTarget).toBe(true);
    expect(groups[0]!.rows).toHaveLength(0);
  });

  it("collects rows with no operational target under the unlinked group", () => {
    const groups = groupMatrixRows([
      makeMatrixRow({ otId: "", requirementId: "R-ORPHAN" }),
    ]);
    expect(groups[0]!.otId).toBe(UNLINKED_GROUP_ID);
    expect(groups[0]!.rows.map((r) => r.requirementId)).toEqual(["R-ORPHAN"]);
  });
});

describe("countUnproven", () => {
  it("counts only rows flagged unproven", () => {
    expect(
      countUnproven([
        makeMatrixRow({ unproven: true }),
        makeMatrixRow({ unproven: false }),
        makeMatrixRow({ unproven: true }),
      ]),
    ).toBe(2);
  });
});
