import { describe, it, expect, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useHashRoute } from "./useHashRoute";

describe("useHashRoute", () => {
  const originalHash = window.location.hash;

  afterEach(() => {
    window.location.hash = originalHash;
  });

  it("initializes from the current hash and reacts to changes", () => {
    window.location.hash = "#/metrics";

    const { result } = renderHook(() => useHashRoute());
    expect(result.current.route).toBe("metrics");

    act(() => {
      window.location.hash = "#/search";
      window.dispatchEvent(new Event("hashchange"));
    });

    expect(result.current.route).toBe("search");
  });

  it("navigate updates the hash and route state", () => {
    window.location.hash = "#/";

    const { result } = renderHook(() => useHashRoute());

    act(() => {
      result.current.navigate("graph");
    });

    expect(window.location.hash).toBe("#/graph");
    expect(result.current.route).toBe("graph");
  });
});
