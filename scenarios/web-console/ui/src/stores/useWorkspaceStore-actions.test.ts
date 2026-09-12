import { beforeEach, describe, expect, it } from "vitest";
import { act, renderHook } from "@testing-library/react";
import { useWorkspaceStore } from "./useWorkspaceStore";
import { useEffectiveFontSize } from "./useWorkspaceStore";

describe("useWorkspaceStore action surface", () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ panes: [], groups: [], pendingInputDrafts: {}, recentCombos: [], recentHeaderColors: [], modifiers: { ctrl: false, alt: false, shift: false } });
  });

  it("keeps the preference and ephemeral action setters observable", () => {
    const s = useWorkspaceStore.getState();
    s.addPane("a", "A");
    s.addPane("b", "B");
    s.setPaneTheme("a", "dark");
    s.setPaneFontSize("a", 18);
    s.setPaneManuallyUnread("a", true);
    expect(s.setActivePane("a")).toBe(true);
    expect(s.setActivePane("a")).toBe(false);
    s.setAppearanceModalPane("a");
    s.setMinimapVisible(false);
    s.setDisplayMode("tabs");
    s.setToolbarPreset("dense");
    s.updateToolbarPrefs({ maxRows: 1 });
    s.setToolbarControlEnabled("ai", true);
    s.setSettingsModalOpen(true);
    s.setSettingsInitialTab("voice");
    s.setAiModalOpen(true);
    s.setAiSuggestActive(true);
    s.setVoiceEnabled(false);
    s.setVoiceShortcut("Alt+V");
    s.setVadAutoStop(false);
    s.setVadSilenceTimeoutMs(900);
    s.setVoiceLanguage("fr");
    s.setPersistentMode(true);
    s.setWakeWordEnabled(true);
    s.setWakeWordThreshold(0.8);
    s.setSegmentSilenceMs(700);
    s.setTtsVoice("voice");
    s.setTtsRate(1.2);
    s.setTtsPitch(0.9);
    s.setAutoTtsEnabled(true);
    s.setStartMutedOnLoad(false);
    s.setTtsBackendPreference("kokoro");
    s.setKokoroVoice("af");
    s.setKokoroSpeed(1.1);
    s.setPlusButtonBehavior("new-terminal");
    s.setSidebarSortMode("unread");
    s.setSidebarView("archive");
    s.setSidebarOriginTab("remote");
    s.setAdaptiveChrome(false);
		s.setTmuxMouseMode(true);
    s.setKeepScreenAwake(false);
    s.setDeviceFontSize("a", 20);
    s.setViewerCount("a", 2);
    s.setTabContextMenu({ sessionId: "a", position: { x: 1, y: 2 } });
    s.setManageGroupsOpen(true);
    s.setPendingInputDraft("a", "queued");
    expect(s.consumePendingInputDraft("a")).toBe("queued");
    expect(s.consumePendingInputDraft("a")).toBeUndefined();
    s.setPendingInputDraft("a", "");
    s.addRecentCombo("x");
    s.addRecentCombo("y");
    s.addRecentCombo("x");
    s.addRecentHeaderColor("#fff");
    s.addRecentHeaderColor("transparent");
    s.addRecentHeaderColor("#fff|#000");
    s.toggleModifier("ctrl");
    s.clearModifiers();
    s.setColumnFractions([0.5, 0.5]);
    s.setRowFractions([1]);
    s.resetLayout();
    s.setGroups([{ id: "g", name: "G", color: "#123456", isCollapsed: false }]);
    s.setPaneGroup("a", "g");
    s.toggleGroupCollapsed("g");
    s.updateGroup("g", { name: "Renamed" });
    s.removeGroup("g");
    s.addGroup({ id: "g2", name: "G2", color: "#000", isCollapsed: false });
    s.removePane("b");
    const result = useWorkspaceStore.getState();
    expect(result.displayMode).toBe("tabs");
    expect(result.deviceFontSize.a).toBe(20);
    expect(result.viewerCounts.a).toBe(2);
    expect(result.modifiers).toEqual({ ctrl: false, alt: false, shift: false });
  });

  it("keeps terminal input buffers and device-local scroll/font preferences bounded", () => {
    const s = useWorkspaceStore.getState();
    s.setTouchScrollSensitivity(0);
    s.setWheelScrollSensitivity(9);
    expect(useWorkspaceStore.getState().touchScrollSensitivity).toBe(0.1);
    expect(useWorkspaceStore.getState().wheelScrollSensitivity).toBe(4);
		s.resetScrollSensitivities();
		expect(useWorkspaceStore.getState().touchScrollSensitivity).toBe(1);
		expect(useWorkspaceStore.getState().wheelScrollSensitivity).toBe(1);
		expect(useWorkspaceStore.getState().tmuxMouseMode).toBe(true);
		s.setPredictionLatencyThresholdMs(-10);
		expect(useWorkspaceStore.getState().predictionLatencyThresholdMs).toBe(0);
		s.setPredictionLatencyThresholdMs(2000);
		expect(useWorkspaceStore.getState().predictionLatencyThresholdMs).toBe(1000);

    s.setPendingInputBuffer("a", [{ data: "draft", intent: "typing" }]);
    const entries = s.consumePendingInputBuffer("a");
    expect(entries).toEqual([{ data: "draft", intent: "typing" }]);
    expect(s.consumePendingInputBuffer("a")).toBeUndefined();
    s.setPendingInputBuffer("a", []);

    s.setDeviceFontSize("a", 99);
    expect(useWorkspaceStore.getState().deviceFontSize.a).toBe(24);
    s.clearDeviceFontSize("a");
    expect(useWorkspaceStore.getState().deviceFontSize.a).toBeUndefined();

    act(() => {
      useWorkspaceStore.setState({
        panes: [{ sessionId: "a", name: "A", headerColor: "transparent", themeId: "slate-ocean", fontSize: 16, groupId: null, supportsMessagesView: false, manuallyUnread: false }],
        deviceFontSize: { a: 20 },
      });
    });
    const { result } = renderHook(() => useEffectiveFontSize("a"));
    expect(result.current).toBe(20);
  });

  // A terminal rewrites its title on every prompt, so the overwhelmingly common
  // call is a rename to the name the pane already has. That must not produce a
  // new `panes` array: every subscriber re-renders off that identity, and the
  // pane was being renamed roughly twenty times a second by an idle agent.
  it("does not touch pane identity when a rename changes nothing", () => {
    const s = useWorkspaceStore.getState();
    s.addPane("a", "A");
    s.renamePaneById("a", "claude — working");
    const renamed = useWorkspaceStore.getState().panes;

    s.renamePaneById("a", "claude — working");
    expect(useWorkspaceStore.getState().panes).toBe(renamed);

    s.renamePaneById("a", "claude — done");
    expect(useWorkspaceStore.getState().panes).not.toBe(renamed);
    expect(useWorkspaceStore.getState().panes[0]?.name).toBe("claude — done");
  });

  it("ignores a rename for a pane that does not exist", () => {
    const s = useWorkspaceStore.getState();
    s.addPane("a", "A");
    const before = useWorkspaceStore.getState().panes;
    s.renamePaneById("missing", "whatever");
    expect(useWorkspaceStore.getState().panes).toBe(before);
  });
});
