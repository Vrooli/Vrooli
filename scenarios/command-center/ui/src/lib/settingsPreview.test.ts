import { describe, expect, it } from "vitest";
import type { CatalogEntry } from "./api";
import { sampleReadings, themeStyle } from "./settingsPreview";

describe("sampleReadings", () => {
  const signals: CatalogEntry[] = [
    { id: "active_scenarios", label: "Apps running", unit: "count", format: "integer", shape: "scalar", coverage: "NOW", sample: { value: 12, series: [10, 12], basis: "authored" } },
    { id: "funnel_30d", label: "Funnel", kind: "funnel", shape: "rows", coverage: "IN-REACH", sample: { value: 3, series: [], basis: "authored", rows: [{ key: "visit", label: "Visit", value: 100, share: 1 }] } },
    { id: "bare_metric", shape: "scalar" },
  ];

  it("values a reading from its authored sample and carries coverage", () => {
    const [running] = sampleReadings(signals);
    expect(running).toMatchObject({ id: "active_scenarios", label: "Apps running", coverage: "NOW", trust: "VALID", empirical: "HIT", value: 12, origin: "sample" });
    expect(running?.kind).toBe("scalar");
  });

  it("maps a rows signal to a panel kind and keeps its sample rows", () => {
    const funnel = sampleReadings(signals)[1];
    expect(funnel?.kind).toBe("funnel");
    expect(funnel?.rows?.[0]).toMatchObject({ key: "visit", value: 100 });
  });

  it("falls back to a null value and the id as label when no sample or label exists", () => {
    const bare = sampleReadings(signals)[2];
    expect(bare).toMatchObject({ id: "bare_metric", label: "bare_metric", value: null, coverage: "NOW" });
  });
});

describe("themeStyle", () => {
  it("keeps only string custom properties from the theme tokens", () => {
    const theme: CatalogEntry = { id: "cosmos", tokens: { "--color-primary": "#8aa8ff", "--color-background": "#06041a", "--bad": 5, "notAVar": "#fff" } };
    expect(themeStyle(theme)).toEqual({ "--color-primary": "#8aa8ff", "--color-background": "#06041a" });
  });

  it("returns an empty style when the theme or its tokens are absent", () => {
    expect(themeStyle(undefined)).toEqual({});
    expect(themeStyle({ id: "blank" })).toEqual({});
  });
});
