import { beforeEach, describe, expect, it } from "vitest";

import { setLocale } from "../i18n";
import { formatAgo, formatDueIn, formatExact } from "./ago";

const NOW = Date.parse("2026-11-14T12:00:00Z");
const minutes = (count: number) => new Date(NOW - count * 60_000).toISOString();

describe("formatAgo", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("reads recent times as a count of the largest whole unit", () => {
    expect(formatAgo(minutes(0.2), NOW)).toBe("now");
    expect(formatAgo(minutes(12), NOW)).toBe("12 minutes ago");
    expect(formatAgo(minutes(60 * 3 + 20), NOW)).toBe("3 hours ago");
    expect(formatAgo(minutes(60 * 24 * 2), NOW)).toBe("2 days ago");
  });

  it("never reports a time in the future when the clock runs ahead", () => {
    expect(formatAgo(new Date(NOW + 90_000).toISOString(), NOW)).toBe("now");
  });

  it("switches to the calendar date after a week, adding the year only when it differs", () => {
    expect(formatAgo(minutes(60 * 24 * 20), NOW)).toBe("Oct 25");
    expect(formatAgo("2025-03-02T10:00:00Z", NOW)).toBe("Mar 2, 2025");
  });

  it("returns null for a missing or unreadable time so the caller can say so", () => {
    expect(formatAgo(undefined, NOW)).toBeNull();
    expect(formatAgo("", NOW)).toBeNull();
    expect(formatAgo("not a time", NOW)).toBeNull();
  });

  it("follows the chosen language", async () => {
    await setLocale("ja");
    expect(formatAgo(minutes(12), NOW)).toBe("12 分前");
  });
});

describe("formatDueIn", () => {
  beforeEach(async () => { await setLocale("en"); });

  it("counts toward a deadline in minutes, hours then days, and past one in the other direction", () => {
    expect(formatDueIn(new Date(NOW + 20 * 60_000).toISOString(), NOW)).toBe("in 20 minutes");
    expect(formatDueIn(new Date(NOW + 5 * 3_600_000).toISOString(), NOW)).toBe("in 5 hours");
    expect(formatDueIn(new Date(NOW + 4 * 86_400_000).toISOString(), NOW)).toBe("in 4 days");
    expect(formatDueIn(minutes(30), NOW)).toBe("30 minutes ago");
  });

  it("never rounds a deadline seconds away down to zero", () => {
    expect(formatDueIn(new Date(NOW + 10_000).toISOString(), NOW)).toBe("in 1 minute");
  });

  it("returns null for a missing deadline", () => {
    expect(formatDueIn("", NOW)).toBeNull();
  });
});

describe("formatExact", () => {
  it("gives the full local time with a zone for records and tooltips", () => {
    expect(formatExact("2026-11-14T12:00:00Z")).toMatch(/2026/);
    expect(formatExact("junk")).toBeNull();
  });
});
