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
    expect(result.current.collectionName).toBe("");

    act(() => {
      window.location.hash = "#/collections/knowledge_chunks_v1";
      window.dispatchEvent(new Event("hashchange"));
    });

    expect(result.current.route).toBe("collection");
    expect(result.current.collectionName).toBe("knowledge_chunks_v1");
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

  it("navigateToCollection updates hash and collection state", () => {
    window.location.hash = "#/";
    const { result } = renderHook(() => useHashRoute());

    act(() => {
      result.current.navigateToCollection("knowledge_chunks_v1");
    });

    expect(window.location.hash).toBe("#/collections/knowledge_chunks_v1");
    expect(result.current.route).toBe("collection");
    expect(result.current.collectionName).toBe("knowledge_chunks_v1");
  });
});
