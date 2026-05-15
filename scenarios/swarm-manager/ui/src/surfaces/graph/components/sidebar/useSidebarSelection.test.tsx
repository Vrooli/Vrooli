import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { SidebarTab } from "./types";
import { useSidebarSelection } from "./useSidebarSelection";

describe("useSidebarSelection", () => {
  it("toggles selection mode only for selectable tabs", () => {
    const { result, rerender } = renderHook(({ tab }: { tab: SidebarTab }) => useSidebarSelection(tab), {
      initialProps: { tab: "backlog" },
    });

    expect(result.current.selectionMode).toBe(false);

    act(() => result.current.toggleMode());

    expect(result.current.selectionMode).toBe(true);

    rerender({ tab: "activity" as const });

    expect(result.current.selectionMode).toBe(false);
    expect(result.current.selectedCount).toBe(0);

    act(() => result.current.toggleMode());

    expect(result.current.selectionMode).toBe(false);
  });

  it("tracks selected ids, selects visible ids, and prunes hidden selections", () => {
    const { result } = renderHook(() => useSidebarSelection("backlog"));

    act(() => result.current.toggleMode());
    act(() => result.current.pruneToVisible(["backlog:idea/a", "backlog:fix/b"]));
    act(() => result.current.selectAllVisible());

    expect(result.current.selectedIds).toEqual(new Set(["backlog:idea/a", "backlog:fix/b"]));

    act(() => result.current.toggleItem("backlog:idea/a"));

    expect(result.current.selectedIds).toEqual(new Set(["backlog:fix/b"]));

    act(() => result.current.pruneToVisible(["backlog:idea/a"]));

    expect(result.current.selectedCount).toBe(0);
  });

  it("clears selection when canceled", () => {
    const { result } = renderHook(() => useSidebarSelection("sessions"));

    act(() => result.current.toggleMode());
    act(() => result.current.selectVisible(["session:a", "session:b"]));
    act(() => result.current.cancelSelection());

    expect(result.current.selectionMode).toBe(false);
    expect(result.current.selectedCount).toBe(0);
  });
});
