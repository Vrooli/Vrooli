import { describe, it, expect } from "vitest";

import { diffLines, diffStats } from "./diff";

describe("diffLines", () => {
  it("marks an empty before as a pure addition", () => {
    const lines = diffLines("", "a\nb");
    expect(lines.every((l) => l.op === "add")).toBe(true);
    expect(diffStats(lines)).toEqual({ added: 2, removed: 0 });
  });

  it("marks an empty after as a pure removal", () => {
    const lines = diffLines("a\nb", "");
    expect(lines.every((l) => l.op === "remove")).toBe(true);
    expect(diffStats(lines)).toEqual({ added: 0, removed: 2 });
  });

  it("keeps unchanged lines as context and reports zero stats", () => {
    const lines = diffLines("a\nb\nc", "a\nb\nc");
    expect(lines.every((l) => l.op === "context")).toBe(true);
    expect(diffStats(lines)).toEqual({ added: 0, removed: 0 });
  });

  it("detects a single changed line as one remove + one add", () => {
    const lines = diffLines("a\nb\nc", "a\nB\nc");
    expect(diffStats(lines)).toEqual({ added: 1, removed: 1 });
    expect(lines.find((l) => l.op === "add")?.text).toBe("B");
    expect(lines.find((l) => l.op === "remove")?.text).toBe("b");
  });

  it("normalizes CRLF so line endings alone are not a diff", () => {
    expect(diffStats(diffLines("a\r\nb", "a\nb"))).toEqual({ added: 0, removed: 0 });
  });
});
