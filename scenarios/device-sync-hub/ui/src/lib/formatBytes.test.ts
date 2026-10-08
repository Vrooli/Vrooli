import { describe, expect, it } from "vitest";

import { formatBytes } from "./formatBytes";

describe("formatBytes", () => {
  it("renders bytes with no decimals", () => {
    expect(formatBytes(512)).toBe("512 B");
  });

  it("scales to KB/MB with one decimal", () => {
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatBytes(1_572_864)).toBe("1.5 MB");
  });

  it("accepts the proto int64 bigint shape", () => {
    expect(formatBytes(2n * 1024n * 1024n * 1024n)).toBe("2 GB");
  });

  it("clamps negatives / non-finite to 0 B", () => {
    expect(formatBytes(-5)).toBe("0 B");
    expect(formatBytes(Number.NaN)).toBe("0 B");
  });
});
