import { describe, expect, it } from "vitest";

import { CaptureTier } from "../../api/perf";
import {
  TIER_LABEL_KEY,
  fleetTierKey,
  formatBytes,
  formatMs,
  formatMsFloat,
  formatTimestamp,
  tierChipClass,
  tierKey,
} from "./format";

describe("tier helpers", () => {
  it("maps the typed CaptureTier enum to label keys", () => {
    expect(tierKey(CaptureTier.CAPTURE_TIER_NONE)).toBe("none");
    expect(tierKey(CaptureTier.CAPTURE_TIER_0)).toBe("tier0");
    expect(tierKey(CaptureTier.CAPTURE_TIER_1)).toBe("tier1");
    expect(tierKey(999 as CaptureTier)).toBe("unknown");
  });

  it("normalizes the fleet free-text tier into the same key space", () => {
    expect(fleetTierKey("none")).toBe("none");
    expect(fleetTierKey(" 0 ")).toBe("tier0");
    expect(fleetTierKey("1")).toBe("tier1");
    expect(fleetTierKey("garbage")).toBe("unknown");
  });

  it("returns a token-driven chip class for every key", () => {
    expect(tierChipClass("tier1")).toContain("app-success");
    expect(tierChipClass("tier0")).toContain("app-info");
    expect(tierChipClass("none")).toContain("app-surface-muted");
    expect(tierChipClass("unknown")).toContain("app-surface-muted");
  });

  it("exposes a label key for each tier", () => {
    expect(TIER_LABEL_KEY.tier1).toBe("tier.tier1");
  });
});

describe("formatMs", () => {
  it("renders an em-dash for missing / non-positive values", () => {
    expect(formatMs(undefined)).toBe("—");
    expect(formatMs(0)).toBe("—");
    expect(formatMs(-5)).toBe("—");
  });
  it("renders ms below 1s and seconds at/above 1s", () => {
    expect(formatMs(250)).toBe("250 ms");
    expect(formatMs(1500)).toBe("1.50 s");
    expect(formatMs(2000n)).toBe("2.00 s");
  });
});

describe("formatMsFloat", () => {
  it("handles missing and fractional values", () => {
    expect(formatMsFloat(undefined)).toBe("—");
    expect(formatMsFloat(0)).toBe("—");
    expect(formatMsFloat(4.25)).toBe("4.3 ms");
    expect(formatMsFloat(1250)).toBe("1.25 s");
  });
});

describe("formatBytes", () => {
  it("renders B / KB / MB and degrades gracefully", () => {
    expect(formatBytes(undefined)).toBe("—");
    expect(formatBytes(0)).toBe("—");
    expect(formatBytes(512)).toBe("512 B");
    expect(formatBytes(2048)).toBe("2.0 KB");
    expect(formatBytes(3_145_728n)).toBe("3.00 MB");
  });
});

describe("formatTimestamp", () => {
  it("renders a local datetime and passes through invalid input", () => {
    expect(formatTimestamp("")).toBe("—");
    expect(formatTimestamp("not-a-date")).toBe("not-a-date");
    expect(formatTimestamp("2026-06-21T00:00:00Z")).not.toBe("—");
  });
});
