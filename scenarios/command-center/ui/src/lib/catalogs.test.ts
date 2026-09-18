import { describe, expect, it } from "vitest";
import { validateRoomBinding } from "./catalogs";

describe("catalog bind validation", () => {
  it("rejects a rows signal missing a required column", () => {
    const catalogs = { signals: [{ id: "funnel", shape: "rows", columns: { key: { type: "string" } } }], compositions: [{ id: "demo", slots: { stages: { shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } } } }] };
    expect(validateRoomBinding({ composition: "demo", bind: { stages: "funnel" } }, catalogs)).toMatch(/share/);
  });
  it("accepts a typed binding with all required columns", () => {
    const catalogs = { signals: [{ id: "funnel", shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } }], compositions: [{ id: "demo", slots: { stages: { shape: "rows", columns: { key: { type: "string" }, share: { type: "number" } } } } }] };
    expect(validateRoomBinding({ composition: "demo", bind: { stages: "funnel" } }, catalogs)).toBeNull();
  });
});
