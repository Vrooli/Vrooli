import { describe, expect, it } from "vitest";

import { buildFixtureChecks } from "./fixtureChecks";
import type { ComponentProbe } from "./componentProbe";

const t = (key: string, options?: Record<string, unknown>) => {
  const template = typeof options?.defaultValue === "string" ? options.defaultValue : key;
  return template.replace(/\{\{(\w+)\}\}/g, (_match, name: string) => {
    const value = options?.[name];
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
  });
};

const probe: ComponentProbe = {
  measured: true,
  targets: { total: 1, small: 1 },
  typography: {
    fontFamily: "Inter, sans-serif",
    fontSize: "18px",
    color: "rgb(247, 241, 232)",
    backgroundStack: ["rgba(0, 0, 0, 0)", "rgb(20, 24, 33)"],
    label: "Adoption health",
  },
};

describe("fixture fit checks", () => {
  it("measures size, touch, contrast and keeps type informational", () => {
    const rows = buildFixtureChecks({ x: 0, y: 0, width: 360, height: 283 }, probe, t);
    const byId = Object.fromEntries(rows.map((row) => [row.id, row]));

    expect(byId.size!).toMatchObject({ verdict: "pass", group: "fit", scored: true });
    expect(byId.size!.detail).toContain("360 × 283");
    expect(byId.touch!).toMatchObject({ verdict: "warn", group: "fit" });
    expect(byId.touch!.detail).toContain("1 of 1");
    expect(byId.contrast!).toMatchObject({ verdict: "pass", group: "fit" });
    expect(byId.contrast!.detail).toMatch(/\d+\.\d:1/);
    expect(byId.type!).toMatchObject({ group: "info", scored: false });
    expect(byId.type!.detail).toContain("Inter");
  });

  it("reports no interactive elements and no text honestly", () => {
    const rows = buildFixtureChecks(
      { x: 0, y: 0, width: 40, height: 40 },
      { measured: true, targets: { total: 0, small: 0 }, typography: null },
      t,
    );
    const byId = Object.fromEntries(rows.map((row) => [row.id, row]));
    expect(byId.touch!).toMatchObject({ verdict: "unmeasured" });
    expect(byId.touch!.detail).toContain("No interactive elements");
    expect(byId.contrast!).toMatchObject({ verdict: "unmeasured" });
    expect(byId.contrast!.detail).toContain("No readable text");
  });

  it("stays unmeasured until the frame has reported", () => {
    const rows = buildFixtureChecks(null, null, t);
    expect(rows.every((row) => row.id === "type" || row.verdict === "unmeasured")).toBe(true);
  });
});
