import { describe, expect, it } from "vitest";
import { computeNextSelection, type SelectionEntry } from "./AppSelection";

const selectionKey = (entry: SelectionEntry) => `${entry.staged ? "1" : "0"}:${entry.path}`;

function selectionFixture() {
  const first: SelectionEntry = { path: "src/a.ts", staged: false };
  const second: SelectionEntry = { path: "src/b.ts", staged: false };
  const third: SelectionEntry = { path: "src/c.ts", staged: true };
  const fourth: SelectionEntry = { path: "src/d.ts", staged: false };
  const entries = [first, second, third, fourth];
  const orderedKeys = entries.map(selectionKey);
  const orderedIndexMap = new Map(orderedKeys.map((key, index) => [key, index]));
  const orderedKeyToEntry = new Map(entries.map((entry) => [selectionKey(entry), entry]));

  return {
    entries,
    first,
    second,
    third,
    fourth,
    orderedKeys,
    orderedIndexMap,
    orderedKeyToEntry,
  };
}

describe("computeNextSelection", () => {
  it("replaces the selection in single-select mode", () => {
    const fixture = selectionFixture();

    expect(
      computeNextSelection(
        "0:src/b.ts",
        "0:src/a.ts",
        "single",
        [fixture.first],
        fixture.orderedIndexMap,
        fixture.orderedKeys,
        fixture.orderedKeyToEntry,
        selectionKey,
      ),
    ).toEqual([{ path: "src/b.ts", staged: false }]);
  });

  it("adds and removes entries in toggle mode while preserving list order", () => {
    const fixture = selectionFixture();

    const added = computeNextSelection(
      "1:src/c.ts",
      "0:src/a.ts",
      "toggle",
      [fixture.second],
      fixture.orderedIndexMap,
      fixture.orderedKeys,
      fixture.orderedKeyToEntry,
      selectionKey,
    );

    expect(added).toEqual([
      { path: "src/b.ts", staged: false },
      { path: "src/c.ts", staged: true },
    ]);

    expect(
      computeNextSelection(
        "0:src/b.ts",
        "0:src/a.ts",
        "toggle",
        added,
        fixture.orderedIndexMap,
        fixture.orderedKeys,
        fixture.orderedKeyToEntry,
        selectionKey,
      ),
    ).toEqual([{ path: "src/c.ts", staged: true }]);
  });

  it("selects the ordered range between the last and next keys", () => {
    const fixture = selectionFixture();

    expect(
      computeNextSelection(
        "0:src/d.ts",
        "0:src/b.ts",
        "range",
        [fixture.first],
        fixture.orderedIndexMap,
        fixture.orderedKeys,
        fixture.orderedKeyToEntry,
        selectionKey,
      ),
    ).toEqual([
      { path: "src/b.ts", staged: false },
      { path: "src/c.ts", staged: true },
      { path: "src/d.ts", staged: false },
    ]);
  });

  it("falls back to single selection when a range anchor is missing", () => {
    const fixture = selectionFixture();

    expect(
      computeNextSelection(
        "0:src/unknown.ts",
        null,
        "range",
        [fixture.first],
        fixture.orderedIndexMap,
        fixture.orderedKeys,
        fixture.orderedKeyToEntry,
        selectionKey,
      ),
    ).toEqual([{ path: "src/unknown.ts", staged: false }]);
  });
});
