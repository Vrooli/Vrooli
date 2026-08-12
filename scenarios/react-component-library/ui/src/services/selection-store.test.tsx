import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import {
  createScopedStore,
  useSelectionStore,
} from "./SelectionStore/versions/1.0.0/SelectionStore";

describe("selection store", () => {
  afterEach(cleanup);

  it("updates subscribers for direct and functional writes", () => {
    const store = createScopedStore(["one"]);
    const listener = () => undefined;
    const unsubscribe = store.subscribe(listener);
    expect(store.get()).toEqual(["one"]);
    store.set(["two"]);
    expect(store.get()).toEqual(["two"]);
    store.set((previous) => [...previous, "three"]);
    expect(store.get()).toEqual(["two", "three"]);
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
