import { describe, it, expect, beforeEach } from "vitest";
import { useWorkspaceStore, type TabGroupMeta } from "../stores/useWorkspaceStore";

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
      sidebarOriginTab: "ui",
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
        supportsMessagesView: false,
  manuallyUnread: false,
      });
    });

    it("does not add duplicate session IDs", () => {
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      useWorkspaceStore.getState().addPane("sess-1", "bash");
      expect(useWorkspaceStore.getState().panes).toHaveLength(1);
    });

    it("marks Claude and Codex panes as message-capable when requested", () => {
      useWorkspaceStore.getState().addPane("sess-ai", "bash", false, true);
      expect(useWorkspaceStore.getState().panes[0]?.supportsMessagesView).toBe(true);
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

  describe("pane status", () => {
    it("stores transient status outside the terminal buffer and clears it with the pane", () => {
      useWorkspaceStore.getState().addPane("sess-status", "bash");
      useWorkspaceStore.getState().setPaneStatus("sess-status", { kind: "disconnected" });
      expect(useWorkspaceStore.getState().paneStatuses["sess-status"]).toEqual({ kind: "disconnected" });
      useWorkspaceStore.getState().removePane("sess-status");
      expect(useWorkspaceStore.getState().paneStatuses["sess-status"]).toBeUndefined();
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

  describe("applyAppearance", () => {
    beforeEach(() => {
      const store = useWorkspaceStore.getState();
      store.addPane("sess-1", "bash");
      store.addPane("sess-2", "zsh");
      store.setPaneColor("sess-1", "#ff7a7a");
      store.setPaneTheme("sess-1", "dracula");
      store.setPaneFontSize("sess-1", 18);
    });

    it("copies only the selected properties to existing panes", () => {
      useWorkspaceStore.getState().applyAppearance("sess-1", {
        properties: ["fontSize"],
        toExistingPanes: true,
        asNewPaneDefault: false,
      });
      const other = useWorkspaceStore.getState().panes.find((p) => p.sessionId === "sess-2");
      expect(other?.fontSize).toBe(18);
      expect(other?.headerColor).toBe("transparent");
      expect(other?.themeId).toBe("slate-ocean");
    });

    it("does not touch defaults unless asNewPaneDefault is set", () => {
      useWorkspaceStore.getState().applyAppearance("sess-1", {
        properties: ["headerColor", "themeId", "fontSize"],
        toExistingPanes: true,
        asNewPaneDefault: false,
      });
      const state = useWorkspaceStore.getState();
      expect(state.defaultHeaderColor).toBe("transparent");
      expect(state.defaultThemeId).toBe("slate-ocean");
      expect(state.defaultFontSize).toBe(14);
    });

    it("saves selected properties as new-pane defaults without touching panes", () => {
      useWorkspaceStore.getState().applyAppearance("sess-1", {
        properties: ["headerColor", "themeId"],
        toExistingPanes: false,
        asNewPaneDefault: true,
      });
      const state = useWorkspaceStore.getState();
      expect(state.defaultHeaderColor).toBe("#ff7a7a");
      expect(state.defaultThemeId).toBe("dracula");
      expect(state.defaultFontSize).toBe(14);
      const other = state.panes.find((p) => p.sessionId === "sess-2");
      expect(other?.headerColor).toBe("transparent");
    });

    it("no-ops with an empty property list or unknown session", () => {
      const before = useWorkspaceStore.getState().panes;
      useWorkspaceStore.getState().applyAppearance("sess-1", {
        properties: [],
        toExistingPanes: true,
        asNewPaneDefault: true,
      });
      useWorkspaceStore.getState().applyAppearance("missing", {
        properties: ["fontSize"],
        toExistingPanes: true,
        asNewPaneDefault: true,
      });
      expect(useWorkspaceStore.getState().panes).toEqual(before);
      expect(useWorkspaceStore.getState().defaultFontSize).toBe(14);
    });
  });

  describe("settingsInitialTab", () => {
    it("sets and clears the one-shot deep-link tab", () => {
      useWorkspaceStore.getState().setSettingsInitialTab("new-pane-defaults");
      expect(useWorkspaceStore.getState().settingsInitialTab).toBe("new-pane-defaults");
      useWorkspaceStore.getState().setSettingsInitialTab(null);
      expect(useWorkspaceStore.getState().settingsInitialTab).toBeNull();
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

  describe("startMutedOnLoad", () => {
    it("defaults to true", () => {
      expect(useWorkspaceStore.getState().startMutedOnLoad).toBe(true);
    });

    it("setStartMutedOnLoad updates the value", () => {
      useWorkspaceStore.getState().setStartMutedOnLoad(false);
      expect(useWorkspaceStore.getState().startMutedOnLoad).toBe(false);
      useWorkspaceStore.getState().setStartMutedOnLoad(true);
      expect(useWorkspaceStore.getState().startMutedOnLoad).toBe(true);
    });
  });

  describe("keepScreenAwake", () => {
    it("defaults to true", () => {
      expect(useWorkspaceStore.getState().keepScreenAwake).toBe(true);
    });

    it("can be toggled off", () => {
      useWorkspaceStore.getState().setKeepScreenAwake(false);
      expect(useWorkspaceStore.getState().keepScreenAwake).toBe(false);
    });

    it("can be toggled back on", () => {
      useWorkspaceStore.getState().setKeepScreenAwake(false);
      useWorkspaceStore.getState().setKeepScreenAwake(true);
      expect(useWorkspaceStore.getState().keepScreenAwake).toBe(true);
    });
  });

  describe("addRecentHeaderColor", () => {
    beforeEach(() => {
      useWorkspaceStore.setState({ recentHeaderColors: [] });
    });

    it("prepends most-recent-first", () => {
      useWorkspaceStore.getState().addRecentHeaderColor("#ff6b6b");
      useWorkspaceStore.getState().addRecentHeaderColor("#4dabf7");
      expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#4dabf7", "#ff6b6b"]);
    });

    it("dedups and promotes an existing color", () => {
      useWorkspaceStore.getState().addRecentHeaderColor("#ff6b6b");
      useWorkspaceStore.getState().addRecentHeaderColor("#4dabf7");
      useWorkspaceStore.getState().addRecentHeaderColor("#ff6b6b");
      expect(useWorkspaceStore.getState().recentHeaderColors).toEqual(["#ff6b6b", "#4dabf7"]);
    });

    it("caps at 6 entries", () => {
      for (const c of ["#111111", "#222222", "#333333", "#444444", "#555555", "#666666", "#777777"]) {
        useWorkspaceStore.getState().addRecentHeaderColor(c);
      }
      const recents = useWorkspaceStore.getState().recentHeaderColors;
      expect(recents).toHaveLength(6);
      expect(recents[0]).toBe("#777777");
      expect(recents).not.toContain("#111111");
    });

    it("ignores transparent and pair values", () => {
      useWorkspaceStore.getState().addRecentHeaderColor("transparent");
      useWorkspaceStore.getState().addRecentHeaderColor("#ff6b6b|#4dabf7");
      expect(useWorkspaceStore.getState().recentHeaderColors).toEqual([]);
    });
  });

  describe("setPaneGroup", () => {
    const pane = (sessionId: string, groupId: string | null = null, headerColor = "transparent") => ({
      sessionId,
      name: sessionId,
      headerColor,
      themeId: "default",
      fontSize: 14,
      groupId,
      supportsMessagesView: false,
  manuallyUnread: false,
    });
    const g1: TabGroupMeta = { id: "g1", name: "Work", color: "#4dabf7", isCollapsed: false };

    it("pulls the joining pane into the group's block", () => {
      useWorkspaceStore.setState({
        panes: [pane("a", "g1"), pane("b", "g1"), pane("c"), pane("d")],
        groups: [g1],
      });
      useWorkspaceStore.getState().setPaneGroup("d", "g1");
      expect(useWorkspaceStore.getState().panes.map((p) => p.sessionId)).toEqual(["a", "b", "d", "c"]);
    });

    it("heals a group that was already split", () => {
      // "b" sits between two members. Joining is also the moment to restore
      // the invariant — leaving g1 in two pieces would render two headers.
      useWorkspaceStore.setState({
        panes: [pane("a", "g1"), pane("b"), pane("c", "g1"), pane("d")],
        groups: [g1],
      });
      useWorkspaceStore.getState().setPaneGroup("d", "g1");
      expect(useWorkspaceStore.getState().panes.map((p) => p.sessionId)).toEqual(["a", "c", "d", "b"]);
      expect(useWorkspaceStore.getState().panes.find((p) => p.sessionId === "d")?.groupId).toBe("g1");
    });

    it("leaves order unchanged for the first member of a group", () => {
      useWorkspaceStore.setState({ panes: [pane("a"), pane("b"), pane("c")], groups: [g1] });
      useWorkspaceStore.getState().setPaneGroup("b", "g1");
      expect(useWorkspaceStore.getState().panes.map((p) => p.sessionId)).toEqual(["a", "b", "c"]);
      expect(useWorkspaceStore.getState().panes[1]?.groupId).toBe("g1");
    });

    it("keeps the surviving members contiguous when a middle member leaves", () => {
      useWorkspaceStore.setState({
        panes: [pane("a", "g1"), pane("b", "g1"), pane("c", "g1")],
        groups: [g1],
      });
      useWorkspaceStore.getState().setPaneGroup("b", null);
      expect(useWorkspaceStore.getState().panes.map((p) => p.sessionId)).toEqual(["a", "c", "b"]);
    });

    it("seeds the group's color onto a pane that has none", () => {
      useWorkspaceStore.setState({ panes: [pane("a")], groups: [g1] });
      useWorkspaceStore.getState().setPaneGroup("a", "g1");
      expect(useWorkspaceStore.getState().panes[0]?.headerColor).toBe("#4dabf7");
    });

    it("never overwrites a color the user chose, and keeps it on leaving", () => {
      useWorkspaceStore.setState({ panes: [pane("a", null, "#ff6b6b")], groups: [g1] });
      useWorkspaceStore.getState().setPaneGroup("a", "g1");
      expect(useWorkspaceStore.getState().panes[0]?.headerColor).toBe("#ff6b6b");
      useWorkspaceStore.getState().setPaneGroup("a", null);
      expect(useWorkspaceStore.getState().panes[0]?.headerColor).toBe("#ff6b6b");
    });
  });

  describe("setSidebarSortMode", () => {
    it("updates the sidebar sort mode", () => {
      useWorkspaceStore.getState().setSidebarSortMode("activity");
      expect(useWorkspaceStore.getState().sidebarSortMode).toBe("activity");
      useWorkspaceStore.getState().setSidebarSortMode("manual");
      expect(useWorkspaceStore.getState().sidebarSortMode).toBe("manual");
    });
  });

  describe("setSidebarOriginTab", () => {
    it("defaults to the ui bucket", () => {
      expect(useWorkspaceStore.getState().sidebarOriginTab).toBe("ui");
    });

    it("updates the active origin tab", () => {
      useWorkspaceStore.getState().setSidebarOriginTab("programmatic");
      expect(useWorkspaceStore.getState().sidebarOriginTab).toBe("programmatic");
      useWorkspaceStore.getState().setSidebarOriginTab("remote");
      expect(useWorkspaceStore.getState().sidebarOriginTab).toBe("remote");
    });
  });

  describe("persist migration v14→v15", () => {
    it("seeds new fields without dropping existing persisted state", () => {
      const migrate = useWorkspaceStore.persist.getOptions().migrate;
      expect(migrate).toBeDefined();
      if (!migrate) throw new Error("migrate function missing");
      const prior = {
        voiceShortcut: "Ctrl+Shift+Space",
        defaultHeaderColor: "#ff6b6b",
        recentCombos: ["ctrl-c"],
        displayMode: "sidebar",
      };
      const migrated = migrate(prior, 14) as Record<string, unknown>;
      expect(migrated.recentHeaderColors).toEqual([]);
      expect(migrated.sidebarSortMode).toBe("manual");
      // Existing persisted fields survive untouched.
      expect(migrated.defaultHeaderColor).toBe("#ff6b6b");
      expect(migrated.recentCombos).toEqual(["ctrl-c"]);
      expect(migrated.displayMode).toBe("sidebar");
    });
  });

  describe("persist migration from the original snapshot", () => {
    it("upgrades legacy preferences through every historical migration boundary", () => {
      const migrate = useWorkspaceStore.persist.getOptions().migrate;
      expect(migrate).toBeDefined();
      if (!migrate) throw new Error("migrate function missing");

      const migrated = migrate(
        {
          voiceShortcut: "Alt+Space",
          panes: [{ sessionId: "stale" }],
          activePane: "stale",
          commandPrefix: "/wake",
          vadSilenceTimeoutMs: 900,
          segmentSilenceMs: 700,
        },
        0,
      ) as Record<string, unknown>;

      expect(migrated.voiceShortcut).toBe("Ctrl+Shift+Space");
      expect(migrated.ttsVoice).toBe("");
      expect(migrated.ttsRate).toBe(1);
      expect(migrated.ttsPitch).toBe(1);
      expect(migrated.panes).toBeUndefined();
      expect(migrated.activePane).toBeUndefined();
      expect(migrated.commandPrefix).toBeUndefined();
      expect(migrated.startMutedOnLoad).toBe(true);
      expect(migrated.vadSilenceTimeoutMs).toBeUndefined();
      expect(migrated.segmentSilenceMs).toBeUndefined();
      expect(migrated.predictionLatencyThresholdMs).toBe(20);
    });
  });

  describe("persist migration v16→v17", () => {
    it("seeds sidebarOriginTab on a pre-origin-tab snapshot and leaves prior state intact", () => {
      const migrate = useWorkspaceStore.persist.getOptions().migrate;
      if (!migrate) throw new Error("migrate function missing");
      const prior = {
        displayMode: "sidebar",
        sidebarSortMode: "activity",
        adaptiveChrome: false,
        recentHeaderColors: ["#111111"],
      };
      const migrated = migrate(prior, 16) as Record<string, unknown>;
      expect(migrated.sidebarOriginTab).toBe("ui");
      // Fields from the pre-change snapshot survive the migration untouched.
      expect(migrated.displayMode).toBe("sidebar");
      expect(migrated.sidebarSortMode).toBe("activity");
      expect(migrated.adaptiveChrome).toBe(false);
      expect(migrated.recentHeaderColors).toEqual(["#111111"]);
    });

    it("does not clobber a persisted origin tab from a future-shaped snapshot", () => {
      const migrate = useWorkspaceStore.persist.getOptions().migrate;
      if (!migrate) throw new Error("migrate function missing");
      const migrated = migrate({ sidebarOriginTab: "remote" }, 16) as Record<string, unknown>;
      expect(migrated.sidebarOriginTab).toBe("remote");
    });
  });
});
