import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { test } from "vitest";
import { useRunsPageState } from "../../src/hooks/useRunsPageState.js";

test("useRunsPageState selects individual and shift-ranged runs, then clears them on exit", () => {
  const { result } = renderHook(() => useRunsPageState());
  const runs = [{ id: "one" }, { id: "two" }, { id: "three" }];
  act(() => result.current.toggleSelectionMode());
  assert.equal(result.current.selectionMode, true);
  act(() => result.current.handleRunCheckboxChange("one", 0, false, runs));
  assert.deepEqual([...result.current.selectedRunIds], ["one"]);
  act(() => result.current.handleRunCheckboxChange("three", 2, true, runs));
  assert.deepEqual([...result.current.selectedRunIds], ["one", "two", "three"]);
  act(() => result.current.clearSelection());
  assert.equal(result.current.selectedRunIds.size, 0);
  act(() => result.current.handleRunCheckboxChange("two", 1, false, runs));
  act(() => result.current.toggleSelectionMode());
  assert.equal(result.current.selectionMode, false);
  assert.equal(result.current.selectedRunIds.size, 0);
});
