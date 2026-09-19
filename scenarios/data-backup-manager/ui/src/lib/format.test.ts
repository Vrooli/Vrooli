import { describe, expect, it } from "vitest";

import { formatAge, formatBytes, formatDuration, isOlderThan } from "./format";

describe("formatBytes", () => {
  it("renders 0 for non-positive and non-finite inputs", () => {
    expect(formatBytes(0)).toBe("0 B");
    expect(formatBytes(-5)).toBe("0 B");
    expect(formatBytes(Number.NaN)).toBe("0 B");
  });

  it("keeps small byte counts in B with no fraction", () => {
    expect(formatBytes(512)).toBe("512 B");
  });

  it("scales up through KB/MB/GB with a compact mantissa", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(5 * 1024 * 1024)).toBe("5 MB");
    expect(formatBytes(1610612736)).toBe("1.5 GB");
  });

  it("accepts bigint (proto int64 fields)", () => {
    expect(formatBytes(2n * 1024n * 1024n * 1024n)).toBe("2 GB");
  });
});

describe("formatDuration", () => {
  const base = new Date("2026-01-01T00:00:00Z");
  const plus = (ms: number) => new Date(base.getTime() + ms);

  it("returns an em dash when a bound is missing or negative", () => {
    expect(formatDuration(base, undefined)).toBe("—");
    expect(formatDuration(undefined, base)).toBe("—");
    expect(formatDuration(plus(1000), base)).toBe("—");
  });

  it("renders sub-second, seconds, minutes, and hours", () => {
    expect(formatDuration(base, plus(340))).toBe("340ms");
    expect(formatDuration(base, plus(5000))).toBe("5s");
    expect(formatDuration(base, plus(72_000))).toBe("1m 12s");
    expect(formatDuration(base, plus(7_500_000))).toBe("2h 5m");
  });
});

describe("formatAge", () => {
  const now = new Date("2026-05-01T12:00:00Z");

  it("renders the never-label for undefined (a first-class state)", () => {
    expect(formatAge(undefined, "never", now)).toBe("never");
    expect(formatAge(undefined, "not yet verified", now)).toBe("not yet verified");
  });

  it("renders a relative age for a past instant", () => {
    const threeDaysAgo = new Date(now.getTime() - 3 * 86_400_000);
    expect(formatAge(threeDaysAgo, "never", now)).toBe("3 days ago");
    const twoHoursAgo = new Date(now.getTime() - 2 * 3_600_000);
    expect(formatAge(twoHoursAgo, "never", now)).toBe("2 hours ago");
  });
});

describe("isOlderThan", () => {
  const now = new Date("2026-05-01T00:00:00Z");
  const day = 86_400_000;

  it("treats a missing timestamp as older than any window", () => {
    expect(isOlderThan(undefined, 30 * day, now)).toBe(true);
  });

  it("compares against the window", () => {
    expect(isOlderThan(new Date(now.getTime() - 10 * day), 30 * day, now)).toBe(false);
    expect(isOlderThan(new Date(now.getTime() - 40 * day), 30 * day, now)).toBe(true);
  });
});
