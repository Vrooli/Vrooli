import { describe, expect, it } from "vitest";

import {
  buildContextChecks,
  type ContextCheckRow,
  type ContextFacts,
  type Translate,
} from "./contextChecks";
import { REFERENCE_WIDTHS } from "./viewports";

const t: Translate = (_key, options) => {
  const raw = options?.defaultValue;
  const template = typeof raw === "string" ? raw : _key;
  return template.replace(/\{\{(\w+)\}\}/g, (_match, key: string) => {
    const value = options?.[key];
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
  });
};

function facts(overrides: Partial<ContextFacts> = {}): ContextFacts {
  return {
    slot: {
      label: "KPI card",
      role: "metric",
      accepts: ["metric", "chart"],
      compatible: true,
      roleMatch: "exact",
      box: { w: 555, h: 40 },
    },
    component: { kind: "metric", size: { w: 248, h: 124 } },
    overflow: REFERENCE_WIDTHS.map((width) => ({ width, overflow: false, scrollWidth: width })),
    type: { fontFamily: "Inter", fontSize: "16px" },
    touch: { on: true, small: 0, total: 6 },
    contrast: { on: true, ratio: 7.2, label: "Signal", background: "#0c0e12" },
    states: { total: 4, covered: 4, status: "covered" },
    safeArea: {
      on: true,
      top: 54,
      gap: 22,
      height: 844,
      safeTop: 54,
      safeBottom: 22,
    },
    squint: { on: false },
    ...overrides,
  };
}

const rowsFor = (overrides: Partial<ContextFacts> = {}) => buildContextChecks(facts(overrides), t);

const row = (rows: ContextCheckRow[], id: string): ContextCheckRow => {
  const found = rows.find((entry) => entry.id === id);
  if (!found) throw new Error(`missing row ${id}`);
  return found;
};

const SCORED_IDS = ["slot", "overflow", "type", "touch", "contrast", "states", "safeArea"];

describe("context checks", () => {
  it("reports measured passes with their evidence", () => {
    const rows = rowsFor();
    expect(row(rows, "slot")).toMatchObject({ verdict: "pass" });
    expect(row(rows, "overflow")).toMatchObject({ verdict: "pass" });
    expect(row(rows, "type")).toMatchObject({ verdict: "unmeasured", scored: false, group: "info" });
    expect(row(rows, "touch")).toMatchObject({ verdict: "pass" });
    expect(row(rows, "contrast")).toMatchObject({ verdict: "pass" });
    expect(row(rows, "states").verdict).toBe("pass");
    expect(row(rows, "safeArea").verdict).toBe("pass");
    expect(rows.filter((row) => row.scored && row.verdict === "pass")).toHaveLength(6);
    expect(rows.filter((row) => row.scored)).toHaveLength(6);
  });

  it("never shows a pass for an instrument that has not run", () => {
    const rows = rowsFor({
      overflow: [],
      type: null,
      touch: { on: false, small: 0, total: 0 },
      contrast: { on: false, ratio: null, label: "", background: "" },
      safeArea: null,
      states: null,
    });
    for (const id of ["overflow", "type", "touch", "contrast", "states", "safeArea"]) {
      expect(row(rows, id).verdict).toBe("unmeasured");
    }
    expect(rows.filter((row) => row.scored && row.verdict === "pass")).toHaveLength(1);
  });

  /**
   * The P3 acceptance rule: a pass is only earned by a present, plausible
   * measurement. Each case below removes the backing measurement while leaving
   * every other fact healthy, and asserts the row drops out of "pass".
   */
  it("withholds every pass whose backing measurement is missing", () => {
    const cases: Array<{ id: string; overrides: Partial<ContextFacts> }> = [
      { id: "overflow", overrides: { overflow: [{ width: 390, overflow: false, scrollWidth: 390 }] } },
      {
        id: "overflow",
        overrides: {
          overflow: [
            { width: 390, overflow: false, scrollWidth: 390 },
            { width: 834, overflow: false, scrollWidth: 834 },
          ],
        },
      },
      { id: "type", overrides: { type: null } },
      {
        id: "type",
        overrides: { type: { fontFamily: "", fontSize: "16px" } },
      },
      { id: "touch", overrides: { touch: { on: true, small: 0, total: 0 } } },
      { id: "touch", overrides: { touch: { on: false, small: 0, total: 6 } } },
      { id: "touch", overrides: { touch: { on: true, small: 9, total: 3 } } },
      {
        id: "contrast",
        overrides: {
          contrast: { on: true, ratio: null, label: "Body", background: "#fff" },
        },
      },
      {
        id: "contrast",
        overrides: {
          contrast: { on: false, ratio: 12, label: "Body", background: "#fff" },
        },
      },
      {
        id: "contrast",
        overrides: {
          contrast: { on: true, ratio: 0, label: "Body", background: "#fff" },
        },
      },
      {
        id: "contrast",
        overrides: {
          contrast: { on: true, ratio: Number.NaN, label: "Body", background: "#fff" },
        },
      },
      {
        id: "contrast",
        overrides: {
          contrast: { on: true, ratio: 7.2, label: "Body", background: "" },
        },
      },
      { id: "states", overrides: { states: { total: 4, covered: 0, status: "covered" } } },
      { id: "states", overrides: { states: { total: 4, covered: 5, status: "covered" } } },
      { id: "states", overrides: { states: null } },
      { id: "safeArea", overrides: { safeArea: null } },
      {
        id: "safeArea",
        overrides: {
          safeArea: { on: true, top: 54, gap: 22, height: 0, safeTop: 54, safeBottom: 22 },
        },
      },
      {
        id: "safeArea",
        overrides: {
          safeArea: { on: false, top: 54, gap: 22, height: 844, safeTop: 54, safeBottom: 22 },
        },
      },
      {
        id: "slot",
        overrides: {
          slot: { label: "Open region", role: undefined, accepts: [], compatible: true, roleMatch: "undeclared", box: { w: 400, h: 40 } },
        },
      },
    ];

    for (const testCase of cases) {
      const rows = rowsFor(testCase.overrides);
      expect(row(rows, testCase.id).verdict, `${testCase.id} ${JSON.stringify(testCase.overrides)}`).not.toBe(
        "pass",
      );
    }
  });

  it("derives overflow from the measured scroll width, not a supplied flag", () => {
    const rows = rowsFor({
      overflow: REFERENCE_WIDTHS.map((width) => ({
        width,
        overflow: false,
        scrollWidth: width === 390 ? width + 8 : width,
      })),
    });
    expect(row(rows, "overflow").verdict).toBe("fail");
    expect(row(rows, "overflow").detail).toContain("390 ✗");

    const unmeasured = rowsFor({
      overflow: REFERENCE_WIDTHS.map((width) => ({ width, overflow: false })),
    });
    expect(row(unmeasured, "overflow").verdict).toBe("unmeasured");
  });

  it("warns instead of passing an open slot that declares no accepted kinds", () => {
    const open = rowsFor({
      slot: {
        label: "Open region",
        role: "metric",
        accepts: [],
        compatible: true,
        roleMatch: "role-only",
        box: { w: 400, h: 40 },
      },
    });
    expect(row(open, "slot")).toMatchObject({ verdict: "warn" });
    expect(row(open, "slot").detail).toContain("anything");

    const unroled = rowsFor({
      slot: {
        label: "Untyped",
        role: undefined,
        accepts: ["metric"],
        compatible: true,
        roleMatch: "undeclared",
        box: { w: 400, h: 40 },
      },
    });
    expect(row(unroled, "slot").verdict).toBe("warn");
  });

  it("fails when the page declares no slot at all", () => {
    const rows = rowsFor({ slot: undefined });
    expect(row(rows, "slot")).toMatchObject({ verdict: "fail" });
    expect(row(rows, "slot").detail).toContain("no component slot");
  });

  it("warns when the slot does not declare the component's kind", () => {
    const rows = rowsFor({
      slot: {
        label: "Header action",
        role: "action",
        accepts: ["button"],
        compatible: false,
        roleMatch: "mismatch",
        box: { w: 140, h: 44 },
      },
    });
    expect(row(rows, "slot").verdict).toBe("warn");
    expect(row(rows, "slot").detail).toContain("button");
  });

  it("warns when a slot accepts the kind but does not name it as its role", () => {
    const rows = rowsFor({
      slot: {
        label: "Live component",
        role: "chart",
        accepts: ["card"],
        compatible: true,
        roleMatch: "accepted",
        box: { w: 1128, h: 112 },
      },
    });
    expect(row(rows, "slot")).toMatchObject({ verdict: "warn" });
    expect(row(rows, "slot").detail).toContain("accepts metric by kind");
  });

  it("groups component fit separately from page health", () => {
    const rows = rowsFor();
    expect(row(rows, "slot").group).toBe("fit");
    expect(row(rows, "type").group).toBe("info");
    expect(row(rows, "type").scored).toBe(false);
    expect(row(rows, "overflow").group).toBe("page");
    expect(row(rows, "states").group).toBe("evidence");
    expect(row(rows, "squint").group).toBe("human");
  });

  it("fails overflow when any reference width overflows", () => {
    const rows = rowsFor({
      overflow: REFERENCE_WIDTHS.map((width) => ({
        width,
        overflow: width === 390,
        scrollWidth: width === 390 ? width + 4 : width,
      })),
    });
    expect(row(rows, "overflow").verdict).toBe("fail");
    expect(row(rows, "overflow").detail).toContain("390 ✗");
  });

  it("bands contrast and counts small touch targets", () => {
    const low = rowsFor({
      contrast: { on: true, ratio: 2.1, label: "Muted", background: "#999" },
      touch: { on: true, small: 3, total: 8 },
    });
    expect(row(low, "contrast").verdict).toBe("fail");
    expect(row(low, "touch").verdict).toBe("warn");

    const warn = rowsFor({
      contrast: { on: true, ratio: 3.6, label: "Body", background: "#fff" },
    });
    expect(row(warn, "contrast").verdict).toBe("warn");
  });

  it("passes only complete declared-state coverage", () => {
    const complete = rowsFor({ states: { total: 4, covered: 4, status: "covered" } });
    expect(row(complete, "states").verdict).toBe("pass");

    const partial = rowsFor({ states: { total: 4, covered: 2, status: "partial" } });
    expect(row(partial, "states").verdict).toBe("warn");

    const none = rowsFor({ states: null });
    expect(row(none, "states").verdict).toBe("unmeasured");
  });

  it("keeps the squint preview unmeasured whether blur is on or off", () => {
    for (const on of [false, true]) {
      const rows = rowsFor({ squint: { on } });
      expect(row(rows, "squint")).toMatchObject({ verdict: "unmeasured", scored: false });
      expect(row(rows, "squint").detail.length).toBeGreaterThan(0);
    }
    expect(rowsFor({ squint: { on: true } }).filter((row) => row.scored).some((entry) => entry.id === "squint")).toBe(false);
    expect(rowsFor({ squint: { on: true } }).filter((row) => row.scored && row.verdict === "pass")).toHaveLength(6);
  });

  it("only ever emits the four known verdicts", () => {
    const rows = rowsFor();
    for (const entry of rows) {
      expect(["pass", "fail", "warn", "unmeasured"]).toContain(entry.verdict);
    }
    expect(rows.map((entry) => entry.id)).toEqual([...SCORED_IDS, "squint"]);
  });
});
