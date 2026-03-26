import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useRouter } from "./router";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-UI-LIBRARY]

describe("useRouter", () => {
  beforeEach(() => {
    window.location.hash = "";
  });

  it("defaults to brands page when hash is empty", () => {
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "brands" });
  });

  it("parses /brands hash", () => {
    window.location.hash = "/brands";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "brands" });
  });

  it("parses /brands/new as brand-create", () => {
    window.location.hash = "/brands/new";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "brand-create" });
  });

  it("parses /brands/:id as brand-detail", () => {
    window.location.hash = "/brands/abc-123";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "brand-detail", id: "abc-123" });
  });

  it("parses /brands/:id/edit as brand-edit", () => {
    window.location.hash = "/brands/abc-123/edit";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "brand-edit", id: "abc-123" });
  });

  it("navigate sets hash", () => {
    const { result } = renderHook(() => useRouter());
    act(() => {
      result.current.navigate("/brands/new");
    });
    expect(window.location.hash).toBe("#/brands/new");
  });

  it("responds to hashchange events", async () => {
    const { result } = renderHook(() => useRouter());
    expect(result.current.route.page).toBe("brands");

    act(() => {
      window.location.hash = "/brands/new";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(result.current.route).toEqual({ page: "brand-create" });
  });

  it("parses /scanner as scanner page", () => {
    window.location.hash = "/scanner";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "scanner" });
  });

  it("parses /standards as standards page", () => {
    window.location.hash = "/standards";
    const { result } = renderHook(() => useRouter());
    expect(result.current.route).toEqual({ page: "standards" });
  });
});
