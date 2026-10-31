import assert from "node:assert/strict";
import { act, renderHook } from "@testing-library/react";
import { test } from "vitest";
import { compareSortValues, useTableSort } from "./SortableHeader.js";

type Field = "model" | "price";

test("a newly chosen text column starts ascending and a numeric column descending; the same column flips", () => {
  const { result } = renderHook(() => useTableSort<Field>("model", { direction: "asc", textFields: ["model"] }));
  assert.deepEqual([result.current.field, result.current.direction], ["model", "asc"]);
  act(() => result.current.toggle("price"));
  assert.deepEqual([result.current.field, result.current.direction], ["price", "desc"]);
  act(() => result.current.toggle("price"));
  assert.equal(result.current.direction, "asc");
  act(() => result.current.toggle("model"));
  assert.deepEqual([result.current.field, result.current.direction], ["model", "asc"]);
});

test("strings compare by locale and numbers numerically in the sort direction", () => {
  assert.ok(compareSortValues("alpha", "beta", "asc") < 0);
  assert.ok(compareSortValues("alpha", "beta", "desc") > 0);
  assert.ok(compareSortValues(2, 10, "asc") < 0);
});
