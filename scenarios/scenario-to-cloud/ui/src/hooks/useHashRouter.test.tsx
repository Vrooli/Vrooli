import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { buildDocUrl, useHashRouter } from "./useHashRouter";

afterEach(() => {
  window.location.hash = "";
  vi.restoreAllMocks();
});

describe("useHashRouter", () => {
  it.each([
    ["", "dashboard", null],
    ["#wizard", "wizard", null],
    ["#docs/guides/vps-setup", "docs", "guides/vps-setup"],
    ["#deployments/demo?tab=files", "deployments", null],
    ["#unknown", "dashboard", null],
  ] as const)("reads %s into %s", (hash, view, docPath) => {
    window.location.hash = hash;

    const { result } = renderHook(() => useHashRouter());

    expect(result.current.view).toBe(view);
    expect(result.current.docPath).toBe(docPath);
  });

  it("navigates to views and keeps the browser hash synchronized", () => {
    const { result } = renderHook(() => useHashRouter());

    act(() => result.current.navigate("docs", "guides/bridge"));
    expect(result.current.view).toBe("docs");
    expect(result.current.docPath).toBe("guides/bridge");
    expect(window.location.hash).toBe("#docs/guides/bridge");

    act(() => result.current.navigate("deployments"));
    expect(result.current.view).toBe("deployments");
    expect(result.current.deploymentState?.deploymentId).toBeNull();
    expect(window.location.hash).toBe("#deployments");

    act(() => result.current.navigate("wizard"));
    expect(window.location.hash).toBe("#wizard");
  });

  it("handles external hash changes and direct document navigation", () => {
    const { result } = renderHook(() => useHashRouter());

    act(() => {
      window.location.hash = "#docs/reference/security";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(result.current.view).toBe("docs");
    expect(result.current.docPath).toBe("reference/security");

    act(() => result.current.navigateToDoc("guides/onboarding"));
    expect(result.current.view).toBe("docs");
    expect(result.current.docPath).toBe("guides/onboarding");
    expect(window.location.hash).toBe("#docs/guides/onboarding");
  });
});

describe("buildDocUrl", () => {
  it("builds a docs hash link", () => {
    expect(buildDocUrl("guides/onboarding")).toBe("#docs/guides/onboarding");
  });
});
