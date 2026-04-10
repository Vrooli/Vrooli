/**
 * Tests for useDetailNavigation hook.
 *
 * Verifies sidebar state preservation across detail open/close cycles,
 * drill-to-lens behavior, and sidebar opening from detail pages.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { useDetailNavigation } from "./useDetailNavigation";
import { useDetailSelectionStore } from "../stores/detail-selection-store";
import { useGraphUIStore } from "../surfaces/graph/stores/graph-ui-store";

// Reset stores between tests.
beforeEach(() => {
  useDetailSelectionStore.setState({ selection: null });
  useGraphUIStore.setState({
    sidebarCollapsed: false,
    sidebarWasOpenBeforeDetail: false,
    focusNodeLabel: null,
  });

  // Reset localStorage mocks.
  vi.spyOn(Storage.prototype, "getItem").mockReturnValue(null);
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => undefined);
});

function wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe("useDetailNavigation", () => {
  describe("openDetail", () => {
    it("opens a backlog detail page", () => {
      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "backlog", kind: "execute", name: "test-item" });
      });

      const selection = useDetailSelectionStore.getState().selection;
      expect(selection).toEqual({ entityType: "backlog", kind: "execute", name: "test-item", tab: undefined });
    });

    it("opens a scenario detail page", () => {
      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "scenario", name: "my-scenario" });
      });

      const selection = useDetailSelectionStore.getState().selection;
      expect(selection?.entityType).toBe("scenario");
      expect(selection?.name).toBe("my-scenario");
    });

    it("opens an execution detail page", () => {
      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "execution", identifier: "exec-123" });
      });

      const selection = useDetailSelectionStore.getState().selection;
      expect(selection?.entityType).toBe("execution");
      expect(selection?.identifier).toBe("exec-123");
    });

    it("opens an initiative detail page", () => {
      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "initiative", name: "init-1" });
      });

      const selection = useDetailSelectionStore.getState().selection;
      expect(selection?.entityType).toBe("initiative");
      expect(selection?.name).toBe("init-1");
    });

    it("saves sidebar state when opened from sidebar", () => {
      useGraphUIStore.setState({ sidebarCollapsed: false });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail(
          { entityType: "backlog", kind: "fix", name: "bug-1" },
          { fromSidebar: true },
        );
      });

      expect(useGraphUIStore.getState().sidebarWasOpenBeforeDetail).toBe(true);
    });

    it("does not save sidebar state when opened from graph", () => {
      useGraphUIStore.setState({ sidebarCollapsed: false });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "backlog", kind: "fix", name: "bug-1" });
      });

      expect(useGraphUIStore.getState().sidebarWasOpenBeforeDetail).toBe(false);
    });

    it("collapses sidebar on mobile viewport", () => {
      // Simulate mobile viewport.
      Object.defineProperty(window, "innerWidth", { value: 400, writable: true });
      useGraphUIStore.setState({ sidebarCollapsed: false });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "scenario", name: "test" });
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);

      // Restore.
      Object.defineProperty(window, "innerWidth", { value: 1024, writable: true });
    });
  });

  describe("closeDetail", () => {
    it("clears the selection", () => {
      useDetailSelectionStore.setState({
        selection: { entityType: "backlog", kind: "execute", name: "test" },
      });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.closeDetail();
      });

      expect(useDetailSelectionStore.getState().selection).toBeNull();
    });

    it("restores sidebar when it was open before detail", () => {
      useGraphUIStore.setState({
        sidebarCollapsed: true,
        sidebarWasOpenBeforeDetail: true,
      });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.closeDetail();
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
      expect(useGraphUIStore.getState().sidebarWasOpenBeforeDetail).toBe(false);
    });

    it("keeps sidebar closed when it was closed before detail", () => {
      useGraphUIStore.setState({
        sidebarCollapsed: true,
        sidebarWasOpenBeforeDetail: false,
      });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.closeDetail();
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
    });

    it("round-trips: open from sidebar → close → sidebar restored", () => {
      useGraphUIStore.setState({ sidebarCollapsed: false });
      Object.defineProperty(window, "innerWidth", { value: 400, writable: true });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      // Open from sidebar.
      act(() => {
        result.current.openDetail(
          { entityType: "backlog", kind: "fix", name: "bug" },
          { fromSidebar: true },
        );
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
      expect(useGraphUIStore.getState().sidebarWasOpenBeforeDetail).toBe(true);

      // Close detail.
      act(() => {
        result.current.closeDetail();
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
      expect(useDetailSelectionStore.getState().selection).toBeNull();

      Object.defineProperty(window, "innerWidth", { value: 1024, writable: true });
    });

    it("round-trips: open from graph → close → sidebar stays closed", () => {
      useGraphUIStore.setState({ sidebarCollapsed: true });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openDetail({ entityType: "scenario", name: "test" });
      });

      act(() => {
        result.current.closeDetail();
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
    });
  });

  describe("openSidebar", () => {
    it("opens the sidebar", () => {
      useGraphUIStore.setState({ sidebarCollapsed: true });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.openSidebar();
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
    });
  });

  describe("drillToLens", () => {
    it("clears detail selection", () => {
      useDetailSelectionStore.setState({
        selection: { entityType: "backlog", kind: "execute", name: "test" },
      });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.drillToLens("node-1", "operations");
      });

      expect(useDetailSelectionStore.getState().selection).toBeNull();
    });

    it("collapses the sidebar", () => {
      useGraphUIStore.setState({ sidebarCollapsed: false });

      const { result } = renderHook(() => useDetailNavigation(), { wrapper });

      act(() => {
        result.current.drillToLens("node-1", "operations");
      });

      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
    });
  });
});
