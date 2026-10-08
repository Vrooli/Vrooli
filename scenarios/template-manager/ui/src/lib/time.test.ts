import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";

import { formatDuration, formatTimestamp } from "./time";

const at = (iso: string) => timestampFromDate(new Date(iso));

describe("formatTimestamp", () => {
  it("returns an em dash when absent", () => {
    expect(formatTimestamp(undefined)).toBe("—");
  });

  it("formats a timestamp to a non-empty string", () => {
    const formatted = formatTimestamp(at("2026-07-14T12:00:00.000Z"));
    expect(formatted).not.toBe("—");
    expect(formatted.length).toBeGreaterThan(0);
  });
});

describe("formatDuration", () => {
  it("returns an em dash when an endpoint is missing", () => {
    expect(formatDuration(undefined, at("2026-07-14T12:00:00Z"))).toBe("—");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), undefined)).toBe("—");
  });

  it("returns an em dash for negative spans", () => {
    expect(formatDuration(at("2026-07-14T12:00:05Z"), at("2026-07-14T12:00:00Z"))).toBe("—");
  });

  it("formats sub-second, seconds, minutes, and hours", () => {
    expect(formatDuration(at("2026-07-14T12:00:00.000Z"), at("2026-07-14T12:00:00.250Z"))).toBe("250ms");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), at("2026-07-14T12:00:05Z"))).toBe("5s");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), at("2026-07-14T12:01:30Z"))).toBe("1m 30s");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), at("2026-07-14T12:02:00Z"))).toBe("2m");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), at("2026-07-14T14:30:00Z"))).toBe("2h 30m");
    expect(formatDuration(at("2026-07-14T12:00:00Z"), at("2026-07-14T14:00:00Z"))).toBe("2h");
  });
});
