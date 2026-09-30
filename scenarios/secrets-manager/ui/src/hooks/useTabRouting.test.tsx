import { act, renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { useTabRouting } from "./useTabRouting";

function routerAt(path: string) {
  return function RouterWrapper({ children }: { children: React.ReactNode }) {
    return <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>;
  };
}

describe("useTabRouting", () => {
  it("defaults invalid paths to the dashboard and supplies the default resource subtab", () => {
    const { result } = renderHook(() => useTabRouting(), { wrapper: routerAt("/unrecognized/route") });

    expect(result.current.activeTab).toBe("dashboard");
    expect(result.current.resourceTab).toBe("tier");
  });

  it("preserves supported resource routes and navigates through the canonical paths", () => {
    const { result } = renderHook(() => useTabRouting(), { wrapper: routerAt("/resources/resource") });

    expect(result.current).toMatchObject({ activeTab: "resources", resourceTab: "resource" });

    act(() => result.current.setActiveTab("deployment"));
    expect(result.current).toMatchObject({ activeTab: "deployment", resourceTab: "tier" });

    act(() => result.current.setActiveTab("resources"));
    expect(result.current).toMatchObject({ activeTab: "resources", resourceTab: "tier" });

    act(() => result.current.setResourceTab("resource"));
    expect(result.current).toMatchObject({ activeTab: "resources", resourceTab: "resource" });
  });
});
