import { describe, it, expect } from "vitest";
import { computeLineDiff } from "./diff";

const expectDefined = <T>(value: T | undefined, label: string): T => {
  expect(value, `${label} should be defined`).toBeDefined();
  return value as T;
};

describe("computeLineDiff", () => {
  it("marks all lines as added when before is empty", () => {
    const result = computeLineDiff("", "line1\nline2");
    const added = result.filter((l) => l.type === "added");
    expect(added.length).toBe(2);
    expect(expectDefined(added[0], "first added line").content).toBe("line1");
    expect(expectDefined(added[1], "second added line").content).toBe("line2");
  });

  it("marks all lines as removed when after is empty", () => {
    const result = computeLineDiff("line1\nline2", "");
    const removed = result.filter((l) => l.type === "removed");
    expect(removed.length).toBe(2);
    expect(expectDefined(removed[0], "first removed line").content).toBe("line1");
    expect(expectDefined(removed[1], "second removed line").content).toBe("line2");
  });

  it("marks all lines as unchanged when before and after are identical", () => {
    const text = "line1\nline2\nline3";
    const result = computeLineDiff(text, text);
    expect(result.every((l) => l.type === "unchanged")).toBe(true);
    expect(result.length).toBe(3);
  });

  it("handles mixed changes correctly", () => {
    const before = "a\nb\nc\nd";
    const after = "a\nx\nc\nd\ne";
    const result = computeLineDiff(before, after);

    const types = result.map((l) => l.type);
    // "a" unchanged, "b" removed, "x" added, "c" unchanged, "d" unchanged, "e" added
    expect(types).toEqual(["unchanged", "removed", "added", "unchanged", "unchanged", "added"]);
  });

  it("handles both empty strings", () => {
    const result = computeLineDiff("", "");
    // Single empty line, unchanged
    expect(result.length).toBe(1);
    expect(expectDefined(result[0], "empty diff line").type).toBe("unchanged");
    expect(expectDefined(result[0], "empty diff line").content).toBe("");
  });

  it("assigns correct line numbers", () => {
    const result = computeLineDiff("a\nb", "a\nc");
    const unchanged = result.find((l) => l.type === "unchanged")!;
    expect(unchanged.oldLineNo).toBe(1);
    expect(unchanged.newLineNo).toBe(1);

    const removed = result.find((l) => l.type === "removed")!;
    expect(removed.oldLineNo).toBe(2);
    expect(removed.newLineNo).toBeUndefined();

    const added = result.find((l) => l.type === "added")!;
    expect(added.newLineNo).toBe(2);
    expect(added.oldLineNo).toBeUndefined();
  });
});
