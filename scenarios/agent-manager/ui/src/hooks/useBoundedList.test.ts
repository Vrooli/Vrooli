import { act, renderHook } from "@testing-library/react";
import { expect, test } from "vitest";
import { useBoundedList } from "./useBoundedList";

test("shows a page at a time and follows the list's length", () => {
  const items = Array.from({ length: 45 }, (_, i) => i);
  const { result, rerender } = renderHook(({ list }) => useBoundedList(list, 20), { initialProps: { list: items } });
  expect(result.current.visibleItems).toHaveLength(20);
  expect(result.current.hiddenCount).toBe(25);
  act(() => result.current.showMore());
  expect(result.current.visibleItems).toHaveLength(40);
  act(() => result.current.showMore());
  expect(result.current.visibleItems).toHaveLength(45);
  expect(result.current.hiddenCount).toBe(0);
  rerender({ list: items.slice(0, 5) });
  expect(result.current.visibleItems).toHaveLength(5);
});
