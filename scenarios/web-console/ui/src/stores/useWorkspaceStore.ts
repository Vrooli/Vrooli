import { create } from "zustand";
import { persist } from "zustand/middleware";
import { clampFontSize } from "../lib/fontSizeUtils";
import { DEFAULT_THEME_ID, TERMINAL_FONT_SIZE } from "../consts/config";
import type { ModifierState } from "../consts/toolbar-keys";

export interface PaneMetadata {
  sessionId: string;
  name: string;
  headerColor: string;
  themeId: string;
  fontSize: number;
  groupId: string | null;
}

export type DisplayMode = "grid" | "tabs";
export type ToolbarLayout = "compact" | "expanded";

export interface TabGroupMeta {
  id: string;
  name: string;
  color: string;
  isCollapsed: boolean;
}

export interface TabContextMenuState {
  sessionId: string;
  position: { x: number; y: number };
}

interface WorkspaceState {
  panes: PaneMetadata[];
  columnFractions: number[];
  rowFractions: number[];
  activePane: string | null;
  appearanceModalPane: string | null;
  isMinimapVisible: boolean;
  displayMode: DisplayMode;
  /** Mobile toolbar key layout: "compact" (single row) or "expanded" (two rows with D-pad). */
  toolbarLayout: ToolbarLayout;
  settingsModalOpen: boolean;
  sessionsModalOpen: boolean;
  aiModalOpen: boolean;
  voiceEnabled: boolean;
  voiceShortcut: string;
  vadAutoStop: boolean;
  /** Silence duration (ms) before VAD auto-stops recording. */
  vadSilenceTimeoutMs: number;
  voiceLanguage: string;
  ttsVoice: string;
  ttsRate: number;
  ttsPitch: number;
  autoTtsEnabled: boolean;
  ttsBackendPreference: "auto" | "kokoro" | "browser";
  kokoroVoice: string;
  kokoroSpeed: number;
  defaultHeaderColor: string;
  defaultThemeId: string;
  defaultFontSize: number;
  /** Recently used key combo IDs for the combo picker. Max 5, most recent first. */
  recentCombos: string[];
  /** Mobile toolbar modifier key toggles (Ctrl/Alt/Shift). Not persisted. */
  modifiers: ModifierState;
  groups: TabGroupMeta[];
  tabContextMenu: TabContextMenuState | null;
}

interface WorkspaceActions {
  addPane: (sessionId: string, name: string, activate?: boolean) => void;
  removePane: (sessionId: string) => void;
  renamePaneById: (sessionId: string, name: string) => void;
  setPaneColor: (sessionId: string, color: string) => void;
  setPaneTheme: (sessionId: string, themeId: string) => void;
  setPaneFontSize: (sessionId: string, size: number) => void;
  movePaneToIndex: (sessionId: string, newIndex: number) => void;
  setColumnFractions: (fractions: number[]) => void;
  setRowFractions: (fractions: number[]) => void;
  setActivePane: (sessionId: string | null) => void;
  setAppearanceModalPane: (sessionId: string | null) => void;
  setMinimapVisible: (visible: boolean) => void;
  setDisplayMode: (mode: DisplayMode) => void;
  setToolbarLayout: (layout: ToolbarLayout) => void;
  setSettingsModalOpen: (open: boolean) => void;
  setSessionsModalOpen: (open: boolean) => void;
  setAiModalOpen: (open: boolean) => void;
  setVoiceEnabled: (enabled: boolean) => void;
  setVoiceShortcut: (shortcut: string) => void;
  setVadAutoStop: (enabled: boolean) => void;
  setVadSilenceTimeoutMs: (ms: number) => void;
  setVoiceLanguage: (lang: string) => void;
  setTtsVoice: (voice: string) => void;
  setTtsRate: (rate: number) => void;
  setTtsPitch: (pitch: number) => void;
  setAutoTtsEnabled: (enabled: boolean) => void;
  setTtsBackendPreference: (pref: "auto" | "kokoro" | "browser") => void;
  setKokoroVoice: (voice: string) => void;
  setKokoroSpeed: (speed: number) => void;
  setDefaultHeaderColor: (color: string) => void;
  setDefaultThemeId: (themeId: string) => void;
  setDefaultFontSize: (size: number) => void;
  resetLayout: () => void;
  addRecentCombo: (comboId: string) => void;
  toggleModifier: (key: keyof ModifierState) => void;
  clearModifiers: () => void;
  setGroups: (groups: TabGroupMeta[]) => void;
  addGroup: (group: TabGroupMeta) => void;
  removeGroup: (groupId: string) => void;
  updateGroup: (groupId: string, update: Partial<Omit<TabGroupMeta, "id">>) => void;
  setPaneGroup: (sessionId: string, groupId: string | null) => void;
  toggleGroupCollapsed: (groupId: string) => void;
  setTabContextMenu: (menu: TabContextMenuState | null) => void;
}

export type WorkspaceStore = WorkspaceState & WorkspaceActions;

export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set) => ({
      panes: [],
      columnFractions: [],
      rowFractions: [],
      activePane: null,
      appearanceModalPane: null,
      isMinimapVisible: true,
      displayMode: "grid",
      toolbarLayout: "expanded",
      settingsModalOpen: false,
      sessionsModalOpen: false,
      aiModalOpen: false,
      voiceEnabled: true,
      voiceShortcut: "Ctrl+Shift+Space",
      vadAutoStop: true,
      vadSilenceTimeoutMs: 2000,
      voiceLanguage: "en-US",
      ttsVoice: "",
      ttsRate: 1.0,
      ttsPitch: 1.0,
      autoTtsEnabled: false,
      ttsBackendPreference: "auto",
      kokoroVoice: "af_heart",
      kokoroSpeed: 1.0,
      defaultHeaderColor: "transparent",
      defaultThemeId: DEFAULT_THEME_ID,
      defaultFontSize: TERMINAL_FONT_SIZE,
      recentCombos: [],
      modifiers: { ctrl: false, alt: false, shift: false },
      groups: [],
      tabContextMenu: null,

      addRecentCombo: (comboId) =>
        set((state) => {
          const filtered = state.recentCombos.filter((id) => id !== comboId);
          return { recentCombos: [comboId, ...filtered].slice(0, 5) };
        }),

      addPane: (sessionId, name, activate) =>
        set((state) => {
          if (state.panes.some((p) => p.sessionId === sessionId)) {
            return activate ? { activePane: sessionId } : state;
          }
          return {
            panes: [
              ...state.panes,
              { sessionId, name, headerColor: state.defaultHeaderColor, themeId: state.defaultThemeId, fontSize: state.defaultFontSize, groupId: null },
            ],
            ...(activate ? { activePane: sessionId } : {}),
          };
        }),

      removePane: (sessionId) =>
        set((state) => ({
          panes: state.panes.filter((p) => p.sessionId !== sessionId),
          activePane:
            state.activePane === sessionId ? null : state.activePane,
        })),

      renamePaneById: (sessionId, name) =>
        set((state) => ({
          panes: state.panes.map((p) =>
            p.sessionId === sessionId ? { ...p, name } : p,
          ),
        })),

      setPaneColor: (sessionId, color) =>
        set((state) => ({
          panes: state.panes.map((p) =>
            p.sessionId === sessionId ? { ...p, headerColor: color } : p,
          ),
        })),

      setPaneTheme: (sessionId, themeId) =>
        set((state) => ({
          panes: state.panes.map((p) =>
            p.sessionId === sessionId ? { ...p, themeId } : p,
          ),
        })),

      setPaneFontSize: (sessionId, size) =>
        set((state) => ({
          panes: state.panes.map((p) =>
            p.sessionId === sessionId ? { ...p, fontSize: clampFontSize(size) } : p,
          ),
        })),

      movePaneToIndex: (sessionId, newIndex) =>
        set((state) => {
          const idx = state.panes.findIndex((p) => p.sessionId === sessionId);
          if (idx === -1) return state;
          const clamped = Math.max(0, Math.min(newIndex, state.panes.length - 1));
          if (idx === clamped) return state;
          const next = [...state.panes];
          const removed = next.splice(idx, 1);
          const item = removed[0];
          if (item) next.splice(clamped, 0, item);
          return { panes: next };
        }),

      setColumnFractions: (fractions) => set({ columnFractions: fractions }),
      setRowFractions: (fractions) => set({ rowFractions: fractions }),
      setActivePane: (sessionId) => set({ activePane: sessionId }),
      setAppearanceModalPane: (sessionId) => set({ appearanceModalPane: sessionId }),
      setMinimapVisible: (visible) => set({ isMinimapVisible: visible }),
      setDisplayMode: (mode) => set({ displayMode: mode }),
      setToolbarLayout: (layout) => set({ toolbarLayout: layout }),
      setSettingsModalOpen: (open) => set({ settingsModalOpen: open }),
      setSessionsModalOpen: (open) => set({ sessionsModalOpen: open }),
      setAiModalOpen: (open) => set({ aiModalOpen: open }),
      setVoiceEnabled: (enabled) => set({ voiceEnabled: enabled }),
      setVoiceShortcut: (shortcut) => set({ voiceShortcut: shortcut }),
      setVadAutoStop: (enabled) => set({ vadAutoStop: enabled }),
      setVadSilenceTimeoutMs: (ms) => set({ vadSilenceTimeoutMs: ms }),
      setVoiceLanguage: (lang) => set({ voiceLanguage: lang }),
      setTtsVoice: (voice) => set({ ttsVoice: voice }),
      setTtsRate: (rate) => set({ ttsRate: rate }),
      setTtsPitch: (pitch) => set({ ttsPitch: pitch }),
      setAutoTtsEnabled: (enabled) => set({ autoTtsEnabled: enabled }),
      setTtsBackendPreference: (pref) => set({ ttsBackendPreference: pref }),
      setKokoroVoice: (voice) => set({ kokoroVoice: voice }),
      setKokoroSpeed: (speed) => set({ kokoroSpeed: speed }),
      setDefaultHeaderColor: (color) => set({ defaultHeaderColor: color }),
      setDefaultThemeId: (themeId) => set({ defaultThemeId: themeId }),
      setDefaultFontSize: (size) => set({ defaultFontSize: clampFontSize(size) }),

      resetLayout: () =>
        set({ columnFractions: [], rowFractions: [] }),
      toggleModifier: (key) =>
        set((state) => ({ modifiers: { ...state.modifiers, [key]: !state.modifiers[key] } })),
      clearModifiers: () =>
        set({ modifiers: { ctrl: false, alt: false, shift: false } }),
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
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, groupId } : p),
      })),
      toggleGroupCollapsed: (groupId) => set((state) => ({
        groups: state.groups.map((g) =>
          g.id === groupId ? { ...g, isCollapsed: !g.isCollapsed } : g
        ),
      })),
      setTabContextMenu: (menu) => set({ tabContextMenu: menu }),
    }),
    {
      name: "wc-workspace",
      version: 8,
      migrate: (persisted, version) => {
        const state = persisted as Record<string, unknown>;
        if (version < 1) {
          // Alt+Space is intercepted by Linux window managers before
          // reaching the browser — migrate to a shortcut that works.
          if (state.voiceShortcut === "Alt+Space") {
            state.voiceShortcut = "Ctrl+Shift+Space";
          }
        }
        if (version < 2) {
          state.ttsVoice ??= "";
          state.ttsRate ??= 1.0;
          state.ttsPitch ??= 1.0;
        }
        if (version < 3) {
          state.vadSilenceTimeoutMs ??= 2000;
        }
        if (version < 4) {
          state.toolbarLayout ??= "expanded";
        }
        if (version < 5) {
          state.recentCombos ??= [];
        }
        if (version < 6) {
          // Panes and activePane are now backend-persisted; clear stale localStorage data
          delete state.panes;
          delete state.activePane;
        }
        if (version < 7) {
          state.autoTtsEnabled ??= false;
        }
        if (version < 8) {
          state.ttsBackendPreference ??= "auto";
          state.kokoroVoice ??= "af_heart";
          state.kokoroSpeed ??= 1.0;
        }
        return state as unknown as WorkspaceState & WorkspaceActions;
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
        vadSilenceTimeoutMs: state.vadSilenceTimeoutMs,
        voiceLanguage: state.voiceLanguage,
        ttsVoice: state.ttsVoice,
        ttsRate: state.ttsRate,
        ttsPitch: state.ttsPitch,
        autoTtsEnabled: state.autoTtsEnabled,
        ttsBackendPreference: state.ttsBackendPreference,
        kokoroVoice: state.kokoroVoice,
        kokoroSpeed: state.kokoroSpeed,
        defaultHeaderColor: state.defaultHeaderColor,
        defaultThemeId: state.defaultThemeId,
        defaultFontSize: state.defaultFontSize,
        recentCombos: state.recentCombos,
      }),
    },
  ),
);
