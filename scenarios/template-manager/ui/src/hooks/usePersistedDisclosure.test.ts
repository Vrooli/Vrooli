import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { usePersistedDisclosure } from "./usePersistedDisclosure";

describe("usePersistedDisclosure", () => {
  afterEach(() => {
    window.localStorage.clear();
  });

  it("defaults to the provided open state", () => {
    const { result } = renderHook(() => usePersistedDisclosure("section-a", false));
    expect(result.current[0]).toBe(false);
  });

  it("toggles and persists to localStorage", () => {
    const { result } = renderHook(() => usePersistedDisclosure("section-b", true));
    act(() => result.current[1]());
    expect(result.current[0]).toBe(false);
    expect(window.localStorage.getItem("vrooli.tm.disclosure.section-b")).toBe("closed");
  });

  it("rehydrates a persisted choice over the default", () => {
    window.localStorage.setItem("vrooli.tm.disclosure.section-c", "closed");
    const { result } = renderHook(() => usePersistedDisclosure("section-c", true));
    expect(result.current[0]).toBe(false);
  });
});
