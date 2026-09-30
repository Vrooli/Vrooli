import { describe, it, expect } from "vitest";
import { buildLineDiff, diffSequence, diffStat, diffWords, tokenizeWords } from "./word-diff";

describe("tokenizeWords", () => {
  it("round-trips losslessly", () => {
    const text = "Remove  swarm-manager's boot call\nand its tests.";
    expect(tokenizeWords(text).join("")).toBe(text);
  });

  it("returns nothing for an empty string", () => {
    expect(tokenizeWords("")).toEqual([]);
  });
});

describe("diffSequence", () => {
  it("marks an unchanged sequence entirely equal", () => {
    expect(diffSequence(["a", "b"], ["a", "b"]).every((change) => change.kind === "equal")).toBe(true);
  });

  it("finds an insertion in the middle without disturbing the ends", () => {
    const changes = diffSequence(["a", "d"], ["a", "b", "c", "d"]);
    expect(changes.filter((change) => change.kind === "insert").map((change) => change.value)).toEqual(["b", "c"]);
    expect(changes.filter((change) => change.kind === "delete")).toHaveLength(0);
  });

  it("finds a deletion", () => {
    const changes = diffSequence(["a", "b", "c"], ["a", "c"]);
    expect(changes.filter((change) => change.kind === "delete").map((change) => change.value)).toEqual(["b"]);
  });

  it("handles an empty before side", () => {
    expect(diffSequence([], ["a"])).toEqual([{ kind: "insert", value: "a" }]);
  });

  it("handles an empty after side", () => {
    expect(diffSequence(["a"], [])).toEqual([{ kind: "delete", value: "a" }]);
  });

  it("emits every element when the two sides share nothing", () => {
    const changes = diffSequence(["a", "b"], ["c", "d"]);
    expect(changes.filter((change) => change.kind === "delete")).toHaveLength(2);
    expect(changes.filter((change) => change.kind === "insert")).toHaveLength(2);
  });

  it("degrades to replace-the-block past the cell budget without hanging", () => {
    // 600x600 tokens with no shared prefix/suffix exceeds MAX_LCS_CELLS.
    const before = Array.from({ length: 600 }, (_, index) => `old${index}`);
    const after = Array.from({ length: 600 }, (_, index) => `new${index}`);
    const started = performance.now();
    const changes = diffSequence(before, after);
    expect(performance.now() - started).toBeLessThan(1000);
    expect(changes.filter((change) => change.kind === "delete")).toHaveLength(600);
    expect(changes.filter((change) => change.kind === "insert")).toHaveLength(600);
    expect(changes.filter((change) => change.kind === "equal")).toHaveLength(0);
  });
});

describe("diffWords", () => {
  it("isolates changed words and keeps the shared ones equal", () => {
    const segments = diffWords("the quick brown fox", "the slow green fox");
    expect(segments[0]).toEqual({ kind: "equal", text: "the " });
    expect(segments[segments.length - 1]).toEqual({ kind: "equal", text: " fox" });
    expect(segments.filter((segment) => segment.kind === "delete").map((segment) => segment.text)).toEqual(["quick", "brown"]);
    expect(segments.filter((segment) => segment.kind === "insert").map((segment) => segment.text)).toEqual(["slow", "green"]);
  });

  it("coalesces a run of same-kind tokens into one segment", () => {
    const segments = diffWords("keep", "keep two three");
    expect(segments.filter((segment) => segment.kind === "insert")).toEqual([{ kind: "insert", text: " two three" }]);
  });

  it("reconstructs the before side from equal+delete segments", () => {
    const before = "Remove the boot call and its tests";
    const after = "Remove the package and its tests";
    const rebuilt = diffWords(before, after)
      .filter((segment) => segment.kind !== "insert")
      .map((segment) => segment.text)
      .join("");
    expect(rebuilt).toBe(before);
  });

  it("reconstructs the after side from equal+insert segments", () => {
    const before = "Remove the boot call and its tests";
    const after = "Remove the package and its tests";
    const rebuilt = diffWords(before, after)
      .filter((segment) => segment.kind !== "delete")
      .map((segment) => segment.text)
      .join("");
    expect(rebuilt).toBe(after);
  });
});

describe("buildLineDiff", () => {
  it("pairs similar lines and diffs them inline", () => {
    const rows = buildLineDiff("Remove the boot call from startup", "Remove the package from startup");
    expect(rows.map((row) => row.kind)).toEqual(["delete", "insert"]);
    // A paired row carries word segments, not one opaque block.
    expect(rows[0]?.segments.length ?? 0).toBeGreaterThan(1);
  });

  it("does not pair dissimilar lines", () => {
    const rows = buildLineDiff("alpha beta gamma", "wholly unrelated sentence");
    expect(rows.map((row) => row.kind)).toEqual(["delete", "insert"]);
    expect(rows[0]?.segments ?? []).toHaveLength(1);
    expect(rows[1]?.segments ?? []).toHaveLength(1);
  });

  it("keeps untouched lines as context", () => {
    const rows = buildLineDiff("keep\nold line\nkeep two", "keep\nkeep two");
    expect(rows.filter((row) => row.kind === "context").map((row) => row.segments[0]?.text ?? "")).toEqual(["keep", "keep two"]);
    expect(rows.filter((row) => row.kind === "delete")).toHaveLength(1);
  });

  it("treats a pure addition as inserts only", () => {
    const rows = buildLineDiff("one", "one\ntwo");
    expect(rows.filter((row) => row.kind === "delete")).toHaveLength(0);
    expect(rows.filter((row) => row.kind === "insert")).toHaveLength(1);
  });
});

describe("diffStat", () => {
  it("counts characters removed and added", () => {
    expect(diffStat("keep this word", "keep this word too")).toEqual({ removed: 0, added: 4 });
  });

  it("replaces whole tokens, so a one-word edit counts the whole word", () => {
    // Word-level granularity is deliberate: character-level diffs of prose
    // produce unreadable confetti.
    expect(diffStat("abc", "abcd")).toEqual({ removed: 3, added: 4 });
  });

  it("reports nothing for identical text", () => {
    expect(diffStat("same text", "same text")).toEqual({ removed: 0, added: 0 });
  });
});
