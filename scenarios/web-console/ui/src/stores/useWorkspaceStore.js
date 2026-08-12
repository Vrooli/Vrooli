import { create } from "zustand";
import { persist } from "zustand/middleware";
import { clampFontSize } from "../lib/fontSizeUtils";
import { parsePaneColor } from "../lib/paneColor";
import { groupIdForDropPosition, orderPanesByGroupBlocks } from "../lib/workspaceNavigation";
import { DEFAULT_THEME_ID, TERMINAL_FONT_SIZE } from "../consts/config";
import { DEFAULT_WAKE_WORD_THRESHOLD } from "../audio-integration/hooks/voice/wakeword/types";
// Auto-stop / segment silence defaults come from the audio-integration package
// so the store can never carry a value that disagrees with the client VAD
// fallback (or, transitively, the audio-tools server). See vad.ts.
import { VAD_FALLBACK_SILENCE_TIMEOUT_MS, VAD_FALLBACK_SEGMENT_SILENCE_MS } from "../audio-integration/hooks/voice/vad";
/**
 * Apply a group-membership change to one pane and restore the block invariant
 * (see orderPanesByGroupBlocks). Every membership write goes through here so
 * "contiguous" is a property of the state itself rather than something each
 * caller has to remember to re-establish.
 *
 * Joining a group also SEEDS the pane's color from the group when the pane has
 * none of its own. The group color used to be a render-time fallback
 * implemented in the sidebar row and nowhere else, so the same session looked
 * grouped in the sidebar and uncolored in the tab strip, the grid pane header,
 * and the appearance modal. Seeding makes it a real default: persisted,
 * editable, and identical on every surface and device.
 *
 * A pane the user has deliberately colored is never overwritten, and leaving a
 * group keeps the color — "remove from group" should not silently restyle a
 * session the user can see.
 */
function withGroupAssigned(panes, groups, sessionId, groupId) {
    const groupColor = groupId ? groups.find((g) => g.id === groupId)?.color : undefined;
    const tagged = panes.map((pane) => {
        if (pane.sessionId !== sessionId)
            return pane;
        const seedColor = groupColor && parsePaneColor(pane.headerColor).isTransparent;
        return seedColor ? { ...pane, groupId, headerColor: groupColor } : { ...pane, groupId };
    });
    return orderPanesByGroupBlocks(tagged);
}
export const useWorkspaceStore = create()(persist((set, get) => ({
    panes: [],
    columnFractions: [],
    rowFractions: [],
    activePane: null,
    appearanceModalPane: null,
    isMinimapVisible: true,
    displayMode: "grid",
    toolbarLayout: "expanded",
    settingsModalOpen: false,
    settingsInitialTab: null,
    aiModalOpen: false,
    aiSuggestActive: false,
    voiceEnabled: true,
    voiceShortcut: "Ctrl+Shift+Space",
    vadAutoStop: true,
    vadSilenceTimeoutMs: VAD_FALLBACK_SILENCE_TIMEOUT_MS,
    voiceLanguage: "en-US",
    persistentMode: false,
    wakeWordEnabled: false,
    wakeWordThreshold: DEFAULT_WAKE_WORD_THRESHOLD,
    segmentSilenceMs: VAD_FALLBACK_SEGMENT_SILENCE_MS,
    ttsVoice: "",
    ttsRate: 1.0,
    ttsPitch: 1.0,
    autoTtsEnabled: false,
    startMutedOnLoad: true,
    ttsBackendPreference: "auto",
    kokoroVoice: "af_heart",
    kokoroSpeed: 1.0,
    defaultHeaderColor: "transparent",
    defaultThemeId: DEFAULT_THEME_ID,
    defaultFontSize: TERMINAL_FONT_SIZE,
    plusButtonBehavior: "launcher",
    recentCombos: [],
    recentHeaderColors: [],
    sidebarSortMode: "manual",
    sidebarOriginTab: "ui",
    adaptiveChrome: true,
    modifiers: { ctrl: false, alt: false, shift: false },
    pendingInputDrafts: {},
    deviceFontSize: {},
    viewerCounts: {},
    groups: [],
    tabContextMenu: null,
    manageGroupsTarget: null,
    keepScreenAwake: true,
    addRecentCombo: (comboId) => set((state) => {
        const filtered = state.recentCombos.filter((id) => id !== comboId);
        return { recentCombos: [comboId, ...filtered].slice(0, 5) };
    }),
    addRecentHeaderColor: (color) => set((state) => {
        // Recents track individual colors only; ignore transparent and pairs.
        if (color === "transparent" || color.includes("|"))
            return state;
        const filtered = state.recentHeaderColors.filter((c) => c !== color);
        return { recentHeaderColors: [color, ...filtered].slice(0, 6) };
    }),
    setSidebarSortMode: (mode) => set({ sidebarSortMode: mode }),
    setSidebarOriginTab: (tab) => set({ sidebarOriginTab: tab }),
    setAdaptiveChrome: (enabled) => set({ adaptiveChrome: enabled }),
    addPane: (sessionId, name, activate, supportsMessagesView = false) => set((state) => {
        if (state.panes.some((p) => p.sessionId === sessionId)) {
            return activate ? { activePane: sessionId } : state;
        }
        return {
            panes: [
                ...state.panes,
                {
                    sessionId,
                    name,
                    headerColor: state.defaultHeaderColor,
                    themeId: state.defaultThemeId,
                    fontSize: state.defaultFontSize,
                    groupId: null,
                    supportsMessagesView,
                    manuallyUnread: false,
                },
            ],
            ...(activate ? { activePane: sessionId } : {}),
        };
    }),
    removePane: (sessionId) => set((state) => {
        const pendingInputDrafts = { ...state.pendingInputDrafts };
        delete pendingInputDrafts[sessionId];
        return {
            panes: state.panes.filter((p) => p.sessionId !== sessionId),
            activePane: state.activePane === sessionId ? null : state.activePane,
            pendingInputDrafts,
        };
    }),
    renamePaneById: (sessionId, name) => set((state) => ({
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, name } : p),
    })),
    setPaneColor: (sessionId, color) => set((state) => ({
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, headerColor: color } : p),
    })),
    setPaneTheme: (sessionId, themeId) => set((state) => ({
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, themeId } : p),
    })),
    setPaneFontSize: (sessionId, size) => set((state) => ({
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, fontSize: clampFontSize(size) } : p),
    })),
    setDeviceFontSize: (sessionId, size) => set((state) => ({
        deviceFontSize: { ...state.deviceFontSize, [sessionId]: clampFontSize(size) },
    })),
    setViewerCount: (sessionId, count) => set((state) => ({ viewerCounts: { ...state.viewerCounts, [sessionId]: count } })),
    movePaneToIndex: (sessionId, newIndex) => set((state) => {
        const idx = state.panes.findIndex((p) => p.sessionId === sessionId);
        if (idx === -1)
            return state;
        const clamped = Math.max(0, Math.min(newIndex, state.panes.length - 1));
        if (idx === clamped)
            return state;
        const moving = state.panes[idx];
        if (!moving)
            return state;
        const next = [...state.panes];
        next.splice(idx, 1);
        next.splice(clamped, 0, moving);
        // Where a pane lands decides what it belongs to. Without this the
        // block invariant would quietly undo any drop inside a group, which
        // reads to the user as the drag having done nothing at all.
        const groupId = groupIdForDropPosition(next, clamped, moving.groupId);
        return {
            panes: groupId === moving.groupId
                ? orderPanesByGroupBlocks(next)
                : withGroupAssigned(next, state.groups, sessionId, groupId),
        };
    }),
    setColumnFractions: (fractions) => set({ columnFractions: fractions }),
    setRowFractions: (fractions) => set({ rowFractions: fractions }),
    setActivePane: (sessionId) => {
        const previous = get().activePane;
        if (previous === sessionId)
            return false;
        let cleared = false;
        set((state) => {
            const target = sessionId
                ? state.panes.find((p) => p.sessionId === sessionId)
                : undefined;
            cleared = target?.manuallyUnread === true;
            return {
                activePane: sessionId,
                ...(cleared
                    ? {
                        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, manuallyUnread: false } : p),
                    }
                    : {}),
            };
        });
        return cleared;
    },
    setPaneManuallyUnread: (sessionId, manuallyUnread) => set((state) => ({
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, manuallyUnread } : p),
    })),
    setAppearanceModalPane: (sessionId) => set({ appearanceModalPane: sessionId }),
    setMinimapVisible: (visible) => set({ isMinimapVisible: visible }),
    setDisplayMode: (mode) => set({ displayMode: mode }),
    setToolbarLayout: (layout) => set({ toolbarLayout: layout }),
    setSettingsModalOpen: (open) => set({ settingsModalOpen: open }),
    setSettingsInitialTab: (tab) => set({ settingsInitialTab: tab }),
    setAiModalOpen: (open) => set({ aiModalOpen: open }),
    setAiSuggestActive: (active) => set({ aiSuggestActive: active }),
    setVoiceEnabled: (enabled) => set({ voiceEnabled: enabled }),
    setVoiceShortcut: (shortcut) => set({ voiceShortcut: shortcut }),
    setVadAutoStop: (enabled) => set({ vadAutoStop: enabled }),
    setVadSilenceTimeoutMs: (ms) => set({ vadSilenceTimeoutMs: ms }),
    setVoiceLanguage: (lang) => set({ voiceLanguage: lang }),
    setPersistentMode: (enabled) => set({ persistentMode: enabled }),
    setWakeWordEnabled: (enabled) => set({ wakeWordEnabled: enabled }),
    setWakeWordThreshold: (threshold) => set({ wakeWordThreshold: threshold }),
    setSegmentSilenceMs: (ms) => set({ segmentSilenceMs: ms }),
    setTtsVoice: (voice) => set({ ttsVoice: voice }),
    setTtsRate: (rate) => set({ ttsRate: rate }),
    setTtsPitch: (pitch) => set({ ttsPitch: pitch }),
    setAutoTtsEnabled: (enabled) => set({ autoTtsEnabled: enabled }),
    setStartMutedOnLoad: (enabled) => set({ startMutedOnLoad: enabled }),
    setTtsBackendPreference: (pref) => set({ ttsBackendPreference: pref }),
    setKokoroVoice: (voice) => set({ kokoroVoice: voice }),
    setKokoroSpeed: (speed) => set({ kokoroSpeed: speed }),
    setDefaultHeaderColor: (color) => set({ defaultHeaderColor: color }),
    setDefaultThemeId: (themeId) => set({ defaultThemeId: themeId }),
    setDefaultFontSize: (size) => set({ defaultFontSize: clampFontSize(size) }),
    setPlusButtonBehavior: (behavior) => set({ plusButtonBehavior: behavior }),
    resetLayout: () => set({ columnFractions: [], rowFractions: [] }),
    toggleModifier: (key) => set((state) => ({ modifiers: { ...state.modifiers, [key]: !state.modifiers[key] } })),
    clearModifiers: () => set({ modifiers: { ctrl: false, alt: false, shift: false } }),
    setPendingInputDraft: (sessionId, draft) => set((state) => {
        const next = { ...state.pendingInputDrafts };
        if (draft)
            next[sessionId] = draft;
        else
            delete next[sessionId];
        return { pendingInputDrafts: next };
    }),
    consumePendingInputDraft: (sessionId) => {
        const draft = get().pendingInputDrafts[sessionId];
        if (draft === undefined)
            return undefined;
        set((state) => {
            const next = { ...state.pendingInputDrafts };
            delete next[sessionId];
            return { pendingInputDrafts: next };
        });
        return draft;
    },
    setGroups: (groups) => set({ groups }),
    addGroup: (group) => set((state) => ({ groups: [...state.groups, group] })),
    removeGroup: (groupId) => set((state) => ({
        groups: state.groups.filter((g) => g.id !== groupId),
        panes: state.panes.map((p) => p.groupId === groupId ? { ...p, groupId: null } : p),
    })),
    updateGroup: (groupId, update) => set((state) => ({
        groups: state.groups.map((g) => g.id === groupId ? { ...g, ...update } : g),
    })),
    setPaneGroup: (sessionId, groupId) => set((state) => ({
        panes: withGroupAssigned(state.panes, state.groups, sessionId, groupId),
    })),
    toggleGroupCollapsed: (groupId) => set((state) => ({
        groups: state.groups.map((g) => g.id === groupId ? { ...g, isCollapsed: !g.isCollapsed } : g),
    })),
    applyAppearance: (sessionId, options) => set((state) => {
        const source = state.panes.find((p) => p.sessionId === sessionId);
        if (!source || options.properties.length === 0)
            return state;
        const patch = {};
        if (options.properties.includes("headerColor"))
            patch.headerColor = source.headerColor;
        if (options.properties.includes("themeId"))
            patch.themeId = source.themeId;
        if (options.properties.includes("fontSize"))
            patch.fontSize = source.fontSize;
        const next = {};
        if (options.toExistingPanes) {
            next.panes = state.panes.map((p) => ({ ...p, ...patch }));
        }
        if (options.asNewPaneDefault) {
            if (patch.headerColor !== undefined)
                next.defaultHeaderColor = patch.headerColor;
            if (patch.themeId !== undefined)
                next.defaultThemeId = patch.themeId;
            if (patch.fontSize !== undefined)
                next.defaultFontSize = patch.fontSize;
        }
        return next;
    }),
    setTabContextMenu: (menu) => set({ tabContextMenu: menu }),
    setManageGroupsTarget: (target) => set({ manageGroupsTarget: target }),
    setKeepScreenAwake: (enabled) => set({ keepScreenAwake: enabled }),
}), {
    name: "wc-workspace",
    version: 17,
    migrate: (persisted, version) => {
        const state = persisted;
        if (version < 1) {
            // Alt+Space is intercepted by Linux window managers before
            // reaching the browser — migrate to a shortcut that works.
            if (state.voiceShortcut === "Alt+Space") {
                state.voiceShortcut = "Ctrl+Shift+Space";
            }
        }
        if (version < 2) {
            state.ttsVoice ?? (state.ttsVoice = "");
            state.ttsRate ?? (state.ttsRate = 1.0);
            state.ttsPitch ?? (state.ttsPitch = 1.0);
        }
        if (version < 3) {
            state.vadSilenceTimeoutMs ?? (state.vadSilenceTimeoutMs = 2000);
        }
        if (version < 4) {
            state.toolbarLayout ?? (state.toolbarLayout = "expanded");
        }
        if (version < 5) {
            state.recentCombos ?? (state.recentCombos = []);
        }
        if (version < 6) {
            // Panes and activePane are now backend-persisted; clear stale localStorage data
            delete state.panes;
            delete state.activePane;
        }
        if (version < 7) {
            state.autoTtsEnabled ?? (state.autoTtsEnabled = false);
        }
        if (version < 8) {
            state.ttsBackendPreference ?? (state.ttsBackendPreference = "auto");
            state.kokoroVoice ?? (state.kokoroVoice = "af_heart");
            state.kokoroSpeed ?? (state.kokoroSpeed = 1.0);
        }
        if (version < 9) {
            state.plusButtonBehavior ?? (state.plusButtonBehavior = "launcher");
        }
        if (version < 10) {
            state.persistentMode ?? (state.persistentMode = false);
            // v12: Replace commandPrefix with wake word settings
            delete state.commandPrefix;
            state.wakeWordEnabled ?? (state.wakeWordEnabled = false);
            state.wakeWordThreshold ?? (state.wakeWordThreshold = DEFAULT_WAKE_WORD_THRESHOLD);
            state.segmentSilenceMs ?? (state.segmentSilenceMs = 1500);
        }
        if (version < 11) {
            state.keepScreenAwake ?? (state.keepScreenAwake = true);
        }
        // v12 introduced lowLatencyVoice, later removed — no migration needed
        // (any persisted value is simply ignored on hydrate).
        if (version < 13) {
            state.startMutedOnLoad ?? (state.startMutedOnLoad = true);
        }
        if (version < 14) {
            // VAD timing is now server-sourced from audio-tools'
            // stt_stream_config.vad_silence_ms. Drop any per-browser
            // overrides so the next hydrate wins on first paint after
            // upgrade — see useVoiceInput.ts hydration loop.
            delete state.vadSilenceTimeoutMs;
            delete state.segmentSilenceMs;
        }
        if (version < 15) {
            state.recentHeaderColors ?? (state.recentHeaderColors = []);
            state.sidebarSortMode ?? (state.sidebarSortMode = "manual");
        }
        if (version < 16) {
            state.adaptiveChrome ?? (state.adaptiveChrome = true);
        }
        if (version < 17) {
            state.sidebarOriginTab ?? (state.sidebarOriginTab = "ui");
        }
        return state;
    },
    partialize: (state) => ({
        columnFractions: state.columnFractions,
        rowFractions: state.rowFractions,
        isMinimapVisible: state.isMinimapVisible,
        displayMode: state.displayMode,
        toolbarLayout: state.toolbarLayout,
        voiceEnabled: state.voiceEnabled,
        voiceShortcut: state.voiceShortcut,
        vadAutoStop: state.vadAutoStop,
        // vadSilenceTimeoutMs intentionally NOT persisted —
        // hydrated from audio-tools stt_stream_config on each mount.
        voiceLanguage: state.voiceLanguage,
        persistentMode: state.persistentMode,
        wakeWordEnabled: state.wakeWordEnabled,
        wakeWordThreshold: state.wakeWordThreshold,
        // segmentSilenceMs intentionally NOT persisted (same reason).
        ttsVoice: state.ttsVoice,
        ttsRate: state.ttsRate,
        ttsPitch: state.ttsPitch,
        autoTtsEnabled: state.autoTtsEnabled,
        startMutedOnLoad: state.startMutedOnLoad,
        ttsBackendPreference: state.ttsBackendPreference,
        kokoroVoice: state.kokoroVoice,
        kokoroSpeed: state.kokoroSpeed,
        defaultHeaderColor: state.defaultHeaderColor,
        defaultThemeId: state.defaultThemeId,
        defaultFontSize: state.defaultFontSize,
        plusButtonBehavior: state.plusButtonBehavior,
        recentCombos: state.recentCombos,
        recentHeaderColors: state.recentHeaderColors,
        sidebarSortMode: state.sidebarSortMode,
        sidebarOriginTab: state.sidebarOriginTab,
        adaptiveChrome: state.adaptiveChrome,
        keepScreenAwake: state.keepScreenAwake,
    }),
}));
