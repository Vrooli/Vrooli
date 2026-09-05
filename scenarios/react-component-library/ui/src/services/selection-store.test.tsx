import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  createSelectionStore,
  useSelectionStore,
} from "@vrooli/react-component-library/SelectionStore/2";

describe("selection store", () => {
  afterEach(cleanup);

  it("updates subscribers for keyed writes and range selection", () => {
    const store = createSelectionStore(["one"], "multi");
    const listener = () => undefined;
    const unsubscribe = store.subscribe(listener);
    expect([...store.getSnapshot().keys]).toEqual(["one"]);
    store.setSelected(["two"]);
    store.extendTo("three", ["two", "three"]);
    expect([...store.getSnapshot().keys]).toEqual(["two", "three"]);
    unsubscribe();
  });

  it("toggles items and publishes selected state", () => {
    const { result } = renderHook(() => useSelectionStore<string>());
    act(() => result.current.toggle("one"));
    expect(result.current.selected).toEqual(["one"]);
    act(() => result.current.toggle("one"));
    expect(result.current.selected).toEqual([]);
    act(() => result.current.setSelected(["two"]));
    expect(result.current.selected).toEqual(["two"]);
  });
});
