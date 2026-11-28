import { describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";
import {
  createSelectionStore,
  useSelectionStore,
} from "@vrooli/react-component-library/SelectionStore/2.0.0";

describe("SelectionStore", () => {
  it("updates subscribers for keyed writes and range selection", () => {
    const store = createSelectionStore(["one"], "multi");
    const unsubscribe = store.subscribe(() => undefined);
    expect([...store.getSnapshot().keys]).toEqual(["one"]);
    store.setSelected(["two"]);
    store.extendTo("three", ["two", "three"]);
    expect([...store.getSnapshot().keys]).toEqual(["two", "three"]);
    unsubscribe();
  });
  it("toggles items and publishes selected state through the hook", () => {
    const { result } = renderHook(() => useSelectionStore<string>());
    act(() => result.current.toggle("one"));
    expect(result.current.selected).toEqual(["one"]);
    act(() => result.current.toggle("one"));
    expect(result.current.selected).toEqual([]);
    act(() => result.current.setSelected(["two"]));
    expect(result.current.selected).toEqual(["two"]);
  });
  it("supports keyed anchor ranges in either direction", () => {
    const store = createSelectionStore([], "multi");
    store.toggle("b");
    store.extendTo("d", ["a", "b", "c", "d"]);
    expect([...store.getSnapshot().keys]).toEqual(["b", "c", "d"]);
    store.extendTo("a", ["a", "b", "c", "d"]);
    expect([...store.getSnapshot().keys]).toEqual(["a", "b"]);
    expect(store.getSnapshot().anchorKey).toBe("b");
  });
  it("supports select-all, invert, and retention policies", () => {
    const store = createSelectionStore([], "multi");
    store.selectAll(["a", "b", "c"]);
    store.invert(["a", "b", "c"]);
    expect(store.size()).toBe(0);
    store.select("b");
    store.toggle("c");
    store.retain(["b"], "prune");
    expect([...store.getSnapshot().keys]).toEqual(["b"]);
    store.toggle("d");
    store.retain(["b"], "keep");
    expect([...store.getSnapshot().keys]).toEqual(["b", "d"]);
  });
  it("clears when mode is none and notifies once per mutation", () => {
    const store = createSelectionStore([], "multi");
    let count = 0;
    store.subscribe(() => {
      count += 1;
    });
    store.toggle("a");
    store.setMode("none");
    expect(count).toBe(2);
    expect(store.size()).toBe(0);
  });
});
