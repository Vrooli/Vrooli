import { describe, it, expect, beforeEach } from "vitest";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";

describe("useWorkspaceStore", () => {
  beforeEach(() => {
    // Reset store state between tests
    useWorkspaceStore.setState({
      panes: [],
      columnFractions: [],
      rowFractions: [],
      activePane: null,
      settingsModalOpen: false,
      defaultHeaderColor: "transparent",
      defaultThemeId: "slate-ocean",
      defaultFontSize: 14,
    });
  });

  it("starts with empty state", () => {
    const state = useWorkspaceStore.getState();
    expect(state.panes).toEqual([]);
    expect(state.columnFractions).toEqual([]);
    expect(state.rowFractions).toEqual([]);
    expect(state.activePane).toBeNull();
    expect(state.settingsModalOpen).toBe(false);
  });

  describe("addPane", () => {
    it("adds a pane with default color", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      const state = useWorkspaceStore.getState();
      expect(state.panes).toHaveLength(1);
      expect(state.panes[0]).toEqual({
        sessionId: "sess-1",
        name: "bash",
        headerColor: "transparent",
        themeId: "slate-ocean",
        fontSize: 14,
        groupId: null,
      });
    });

    it("does not add duplicate session IDs", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      expect(useWorkspaceStore.getState().panes).toHaveLength(1);
    });
  });

  describe("removePane", () => {
    it("removes a pane by session ID", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().addPane("sess-2", "zsh");
      useWorkspaceStore.getState().removePane("sess-1");
      const state = useWorkspaceStore.getState();
      expect(state.panes).toHaveLength(1);
      expect(state.panes[0]?.sessionId).toBe("sess-2");
    });

    it("clears activePane if removing active session", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().setActivePane("sess-1");
      useWorkspaceStore.getState().removePane("sess-1");
      expect(useWorkspaceStore.getState().activePane).toBeNull();
    });

    it("preserves activePane if removing different session", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().addPane("sess-2", "zsh");
      useWorkspaceStore.getState().setActivePane("sess-1");
      useWorkspaceStore.getState().removePane("sess-2");
      expect(useWorkspaceStore.getState().activePane).toBe("sess-1");
    });
  });

  describe("renamePaneById", () => {
    it("renames a pane", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().renamePaneById("sess-1", "my-server");
      expect(useWorkspaceStore.getState().panes[0]?.name).toBe("my-server");
    });
  });

  describe("setPaneColor", () => {
    it("sets a pane's header color", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().setPaneColor("sess-1", "#ff7a7a");
      expect(useWorkspaceStore.getState().panes[0]?.headerColor).toBe(
        "#ff7a7a",
      );
    });
  });

  describe("movePaneToIndex", () => {
    it("moves a pane forward", () => {
      useWorkspaceStore.getState().addPane("a", "a");
      useWorkspaceStore.getState().addPane("b", "b");
      useWorkspaceStore.getState().addPane("c", "c");
      useWorkspaceStore.getState().movePaneToIndex("a", 2);
      const ids = useWorkspaceStore.getState().panes.map((p) => p.sessionId);
      expect(ids).toEqual(["b", "c", "a"]);
    });

    it("moves a pane backward", () => {
      useWorkspaceStore.getState().addPane("a", "a");
      useWorkspaceStore.getState().addPane("b", "b");
      useWorkspaceStore.getState().addPane("c", "c");
      useWorkspaceStore.getState().movePaneToIndex("c", 0);
      const ids = useWorkspaceStore.getState().panes.map((p) => p.sessionId);
      expect(ids).toEqual(["c", "a", "b"]);
    });

    it("clamps to valid range", () => {
      useWorkspaceStore.getState().addPane("a", "a");
      useWorkspaceStore.getState().addPane("b", "b");
      useWorkspaceStore.getState().movePaneToIndex("a", 100);
      const ids = useWorkspaceStore.getState().panes.map((p) => p.sessionId);
      expect(ids).toEqual(["b", "a"]);
    });

    it("no-ops when moving to same position", () => {
      useWorkspaceStore.getState().addPane("a", "a");
      useWorkspaceStore.getState().addPane("b", "b");
      const before = useWorkspaceStore.getState().panes;
      useWorkspaceStore.getState().movePaneToIndex("a", 0);
      expect(useWorkspaceStore.getState().panes).toBe(before);
    });
  });

  describe("setColumnFractions / setRowFractions", () => {
    it("sets column fractions", () => {
      useWorkspaceStore.getState().setColumnFractions([0.6, 0.4]);
      expect(useWorkspaceStore.getState().columnFractions).toEqual([0.6, 0.4]);
    });

    it("sets row fractions", () => {
      useWorkspaceStore.getState().setRowFractions([0.3, 0.7]);
      expect(useWorkspaceStore.getState().rowFractions).toEqual([0.3, 0.7]);
    });
  });

  describe("setActivePane", () => {
    it("sets the active pane", () => {
      useWorkspaceStore.getState().setActivePane("sess-1");
      expect(useWorkspaceStore.getState().activePane).toBe("sess-1");
    });

    it("clears the active pane", () => {
      useWorkspaceStore.getState().setActivePane("sess-1");
      useWorkspaceStore.getState().setActivePane(null);
      expect(useWorkspaceStore.getState().activePane).toBeNull();
    });
  });

  describe("setSettingsModalOpen", () => {
    it("toggles settings modal", () => {
      useWorkspaceStore.getState().setSettingsModalOpen(true);
      expect(useWorkspaceStore.getState().settingsModalOpen).toBe(true);
      useWorkspaceStore.getState().setSettingsModalOpen(false);
      expect(useWorkspaceStore.getState().settingsModalOpen).toBe(false);
    });
  });

  describe("default appearance", () => {
    it("starts with default values", () => {
      const state = useWorkspaceStore.getState();
      expect(state.defaultHeaderColor).toBe("transparent");
      expect(state.defaultThemeId).toBe("slate-ocean");
      expect(state.defaultFontSize).toBe(14);
    });

    it("setDefaultHeaderColor updates the value", () => {
      useWorkspaceStore.getState().setDefaultHeaderColor("#ff7a7a");
      expect(useWorkspaceStore.getState().defaultHeaderColor).toBe("#ff7a7a");
    });

    it("setDefaultThemeId updates the value", () => {
      useWorkspaceStore.getState().setDefaultThemeId("dracula");
      expect(useWorkspaceStore.getState().defaultThemeId).toBe("dracula");
    });

    it("setDefaultFontSize updates and clamps the value", () => {
      useWorkspaceStore.getState().setDefaultFontSize(20);
      expect(useWorkspaceStore.getState().defaultFontSize).toBe(20);

      useWorkspaceStore.getState().setDefaultFontSize(2);
      expect(useWorkspaceStore.getState().defaultFontSize).toBe(8);

      useWorkspaceStore.getState().setDefaultFontSize(100);
      expect(useWorkspaceStore.getState().defaultFontSize).toBe(24);
    });

    it("addPane uses current defaults", () => {
      useWorkspaceStore.getState().setDefaultHeaderColor("#7aa0ff");
      useWorkspaceStore.getState().setDefaultThemeId("dracula");
      useWorkspaceStore.getState().setDefaultFontSize(18);

      useWorkspaceStore.getState().addPane("sess-new", "zsh");
      const pane = useWorkspaceStore.getState().panes[0];
      expect(pane?.headerColor).toBe("#7aa0ff");
      expect(pane?.themeId).toBe("dracula");
      expect(pane?.fontSize).toBe(18);
    });
  });

  describe("resetLayout", () => {
    it("clears column and row fractions", () => {
      useWorkspaceStore.getState().setColumnFractions([0.6, 0.4]);
      useWorkspaceStore.getState().setRowFractions([0.3, 0.7]);
      useWorkspaceStore.getState().resetLayout();
      expect(useWorkspaceStore.getState().columnFractions).toEqual([]);
      expect(useWorkspaceStore.getState().rowFractions).toEqual([]);
    });
  });
});
