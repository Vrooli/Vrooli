import { describe, expect, it } from "vitest";
import { resolveSlotBindings, validateRoomBinding } from "./catalogs";

describe("catalog bind validation", () => {
  it("rejects a rows signal missing a required column", () => {
    const catalogs = { signals: [{ id: "funnel", shape: "rows", columns: { key: { type: "string" } } }], compositions: [{ id: "demo", slots: { stages: { shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } } } }] };
    expect(validateRoomBinding({ composition: "demo", bind: { stages: "funnel" } }, catalogs)).toMatch(/share/);
  });
  it("accepts a typed binding with all required columns", () => {
    const catalogs = { signals: [{ id: "funnel", shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } }], compositions: [{ id: "demo", slots: { stages: { shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } } } }] };
    expect(validateRoomBinding({ composition: "demo", bind: { stages: "funnel" } }, catalogs)).toBeNull();
  });
  it("accepts numbered bindings for a variadic scalar slot", () => {
    const catalogs = { signals: [{ id: "a", shape: "scalar" }, { id: "b", shape: "scalar" }], compositions: [{ id: "demo", slots: { secondary: { shape: "scalar", variadic: true, maxItems: 3 } } }] };
    expect(validateRoomBinding({ composition: "demo", bind: { "secondary.0": "a", "secondary.1": "b" } }, catalogs)).toBeNull();
    expect(resolveSlotBindings("hive-lattice", ["a", "b"], { "secondary.0": "a", "secondary.1": "b" })).toMatchObject({ "secondary.0": "a", "secondary.1": "b" });
  });
});
