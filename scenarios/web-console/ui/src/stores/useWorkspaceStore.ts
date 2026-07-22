import { create } from "zustand";
import { persist } from "zustand/middleware";
import { clampFontSize } from "../lib/fontSizeUtils";
import { DEFAULT_THEME_ID, TERMINAL_FONT_SIZE } from "../consts/config";
import { DEFAULT_WAKE_WORD_THRESHOLD } from "../audio-integration/hooks/voice/wakeword/types";
// Auto-stop / segment silence defaults come from the audio-integration package
// so the store can never carry a value that disagrees with the client VAD
// fallback (or, transitively, the audio-tools server). See vad.ts.
import { VAD_FALLBACK_SILENCE_TIMEOUT_MS, VAD_FALLBACK_SEGMENT_SILENCE_MS } from "../audio-integration/hooks/voice/vad";
import type { ModifierState } from "../consts/toolbar-keys";

export interface PaneMetadata {
  sessionId: string;
  name: string;
  headerColor: string;
  themeId: string;
  fontSize: number;
  groupId: string | null;
  supportsMessagesView: boolean;
}

export type DisplayMode = "grid" | "tabs" | "sidebar";
export type ToolbarLayout = "compact" | "expanded";
export type PlusButtonBehavior = "launcher" | "new-terminal";
/** Sidebar ordering: "manual" honors backend sort_order (drag-reorderable);
 *  the rest are view-only sorts that never write sort_order. */
export type SidebarSortMode = "manual" | "name" | "activity" | "unread";

/** Which origin bucket the sidebar tab strip currently shows. Mirrors the
 *  session-origin buckets (see workspaceNavigation.originBucket): a session with
 *  origin "unspecified" folds into "programmatic", so the tab set is exactly
 *  these three. */
export type SidebarOriginTab = "ui" | "programmatic" | "remote";

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

/** The pane appearance properties that can be propagated to other sessions. */
export type AppearanceProperty = "headerColor" | "themeId" | "fontSize";

export interface ApplyAppearanceOptions {
  /** Which properties of the source pane to propagate. */
  properties: AppearanceProperty[];
  /** Copy the selected properties onto every currently-open pane. */
  toExistingPanes: boolean;
  /** Save the selected properties as the defaults seeded into new panes. */
  asNewPaneDefault: boolean;
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
  /** One-shot request to open the settings modal on a specific tab (e.g. a
   *  deep link from the appearance modal). Consumed and cleared by
   *  SettingsModal. Not persisted. */
  settingsInitialTab: string | null;
  aiModalOpen: boolean;
  /** Whether the inline AI suggestion bar is active (mobile only). Not persisted. */
  aiSuggestActive: boolean;
  voiceEnabled: boolean;
  voiceShortcut: string;
  vadAutoStop: boolean;
  /** Silence duration (ms) before VAD auto-stops recording. */
  vadSilenceTimeoutMs: number;
  voiceLanguage: string;
  /** Persistent voice mode — mic stays active until tapped again. */
  persistentMode: boolean;
  /** Whether audio-level wake word detection is enabled. */
  wakeWordEnabled: boolean;
  /** Similarity threshold for wake word matching (0.1-0.95). */
  wakeWordThreshold: number;
  /** Silence duration (ms) that triggers a segment boundary in persistent mode. */
  segmentSilenceMs: number;
  ttsVoice: string;
  ttsRate: number;
  ttsPitch: number;
  autoTtsEnabled: boolean;
  /** Whether the audio bar starts muted on app load. Default true; tap the
   *  speaker icon to unmute. Project-owner preference for greenfield install. */
  startMutedOnLoad: boolean;
  ttsBackendPreference: "auto" | "kokoro" | "browser";
  kokoroVoice: string;
  kokoroSpeed: number;
  defaultHeaderColor: string;
  defaultThemeId: string;
  defaultFontSize: number;
  /** What a quick tap on the + button does: open the launcher or create an empty terminal. */
  plusButtonBehavior: PlusButtonBehavior;
  /** Recently used key combo IDs for the combo picker. Max 5, most recent first. */
  recentCombos: string[];
  /** Recently picked header colors (individual hex values, not pairs). Max 6,
   *  most recent first. Recorded only on explicit user pick. */
  recentHeaderColors: string[];
  /** Sidebar session ordering mode. View-only except "manual". */
  sidebarSortMode: SidebarSortMode;
  /** Active origin tab in the sidebar. Only meaningful while the tab strip is
   *  mounted (i.e. at least one non-UI-origin session exists); when the active
   *  bucket has no sessions the sidebar falls back to the first present bucket
   *  without mutating this, so the choice survives a bucket emptying and
   *  refilling. */
  sidebarOriginTab: SidebarOriginTab;
  /** Tint the app chrome (status bar, top bar, toolbar, sidebar) to match the
   *  focused terminal's background in single-focus (tabs/sidebar) modes. */
  adaptiveChrome: boolean;
  /** Mobile toolbar modifier key toggles (Ctrl/Alt/Shift). Not persisted. */
  modifiers: ModifierState;
  groups: TabGroupMeta[];
  tabContextMenu: TabContextMenuState | null;
  /** Open state of the Manage Groups drawer. Non-null = open; `sessionId`
   *  carries the optional session context (opened from a tab's menu) that
   *  enables the per-group assign/remove toggle. Ephemeral — not persisted. */
  manageGroupsTarget: { sessionId: string | null } | null;
  /** Keep the device screen awake to support hands-free voice interaction. */
  keepScreenAwake: boolean;
  /** Unsent terminal input keyed by session, snapshotted when a pane unmounts
   *  (offscreen sessions are unmounted to keep cost flat in N) and re-injected
   *  on remount. Ephemeral — not persisted. */
  pendingInputDrafts: Record<string, string>;
}

interface WorkspaceActions {
  addPane: (sessionId: string, name: string, activate?: boolean, supportsMessagesView?: boolean) => void;
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
  setSettingsInitialTab: (tab: string | null) => void;
  setAiModalOpen: (open: boolean) => void;
  setAiSuggestActive: (active: boolean) => void;
  setVoiceEnabled: (enabled: boolean) => void;
  setVoiceShortcut: (shortcut: string) => void;
  setVadAutoStop: (enabled: boolean) => void;
  setVadSilenceTimeoutMs: (ms: number) => void;
  setVoiceLanguage: (lang: string) => void;
  setPersistentMode: (enabled: boolean) => void;
  setWakeWordEnabled: (enabled: boolean) => void;
  setWakeWordThreshold: (threshold: number) => void;
  setSegmentSilenceMs: (ms: number) => void;
  setTtsVoice: (voice: string) => void;
  setTtsRate: (rate: number) => void;
  setTtsPitch: (pitch: number) => void;
  setAutoTtsEnabled: (enabled: boolean) => void;
  setStartMutedOnLoad: (enabled: boolean) => void;
  setTtsBackendPreference: (pref: "auto" | "kokoro" | "browser") => void;
  setKokoroVoice: (voice: string) => void;
  setKokoroSpeed: (speed: number) => void;
  setDefaultHeaderColor: (color: string) => void;
  setDefaultThemeId: (themeId: string) => void;
  setDefaultFontSize: (size: number) => void;
  setPlusButtonBehavior: (behavior: PlusButtonBehavior) => void;
  resetLayout: () => void;
  addRecentCombo: (comboId: string) => void;
  /** Record an explicitly-picked header color into recents (dedup, cap 6). */
  addRecentHeaderColor: (color: string) => void;
  setSidebarSortMode: (mode: SidebarSortMode) => void;
  setSidebarOriginTab: (tab: SidebarOriginTab) => void;
  setAdaptiveChrome: (enabled: boolean) => void;
  toggleModifier: (key: keyof ModifierState) => void;
  clearModifiers: () => void;
  setGroups: (groups: TabGroupMeta[]) => void;
  addGroup: (group: TabGroupMeta) => void;
  removeGroup: (groupId: string) => void;
  updateGroup: (groupId: string, update: Partial<Omit<TabGroupMeta, "id">>) => void;
  setPaneGroup: (sessionId: string, groupId: string | null) => void;
  /** Assign a pane to a group AND move it adjacent to the group's last member
   *  so groups stay contiguous. Returns nothing; caller syncs order + update. */
  addPaneToGroup: (sessionId: string, groupId: string) => void;
  toggleGroupCollapsed: (groupId: string) => void;
  /** Propagate a subset of the source pane's appearance to existing panes
   *  and/or the new-pane defaults. Never mutates defaults implicitly — the
   *  caller opts into each target. */
  applyAppearance: (sessionId: string, options: ApplyAppearanceOptions) => void;
  setTabContextMenu: (menu: TabContextMenuState | null) => void;
  setManageGroupsTarget: (target: { sessionId: string | null } | null) => void;
  setKeepScreenAwake: (enabled: boolean) => void;
  /** Stash a session's unsent terminal input before its pane unmounts. */
  setPendingInputDraft: (sessionId: string, draft: string) => void;
  /** Read and clear a session's stashed input (returns undefined if none). */
  consumePendingInputDraft: (sessionId: string) => string | undefined;
}

export type WorkspaceStore = WorkspaceState & WorkspaceActions;

export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set, get) => ({
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
      groups: [],
      tabContextMenu: null,
      manageGroupsTarget: null,
      keepScreenAwake: true,

      addRecentCombo: (comboId) =>
        set((state) => {
          const filtered = state.recentCombos.filter((id) => id !== comboId);
          return { recentCombos: [comboId, ...filtered].slice(0, 5) };
        }),

      addRecentHeaderColor: (color) =>
        set((state) => {
          // Recents track individual colors only; ignore transparent and pairs.
          if (color === "transparent" || color.includes("|")) return state;
          const filtered = state.recentHeaderColors.filter((c) => c !== color);
          return { recentHeaderColors: [color, ...filtered].slice(0, 6) };
        }),

      setSidebarSortMode: (mode) => set({ sidebarSortMode: mode }),
      setSidebarOriginTab: (tab) => set({ sidebarOriginTab: tab }),
      setAdaptiveChrome: (enabled) => set({ adaptiveChrome: enabled }),

      addPane: (sessionId, name, activate, supportsMessagesView = false) =>
        set((state) => {
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
              },
            ],
            ...(activate ? { activePane: sessionId } : {}),
          };
        }),

      removePane: (sessionId) =>
        set((state) => {
          const pendingInputDrafts = { ...state.pendingInputDrafts };
          delete pendingInputDrafts[sessionId];
          return {
            panes: state.panes.filter((p) => p.sessionId !== sessionId),
            activePane:
              state.activePane === sessionId ? null : state.activePane,
            pendingInputDrafts,
          };
        }),

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

      resetLayout: () =>
        set({ columnFractions: [], rowFractions: [] }),
      toggleModifier: (key) =>
        set((state) => ({ modifiers: { ...state.modifiers, [key]: !state.modifiers[key] } })),
      clearModifiers: () =>
        set({ modifiers: { ctrl: false, alt: false, shift: false } }),
      setPendingInputDraft: (sessionId, draft) =>
        set((state) => {
          const next = { ...state.pendingInputDrafts };
          if (draft) next[sessionId] = draft;
          else delete next[sessionId];
          return { pendingInputDrafts: next };
        }),
      consumePendingInputDraft: (sessionId) => {
        const draft = get().pendingInputDrafts[sessionId];
        if (draft === undefined) return undefined;
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
        panes: state.panes.map((p) => p.sessionId === sessionId ? { ...p, groupId } : p),
      })),
      addPaneToGroup: (sessionId, groupId) => set((state) => {
        const tagged = state.panes.map((p) =>
          p.sessionId === sessionId ? { ...p, groupId } : p,
        );
        // Index of the group's last *other* member (the pane being added is
        // excluded so it doesn't anchor on itself).
        let lastMember = -1;
        tagged.forEach((p, i) => {
          if (p.groupId === groupId && p.sessionId !== sessionId) lastMember = i;
        });
        // First member of a (possibly new) group: nothing to be contiguous with.
        if (lastMember === -1) return { panes: tagged };
        const fromIdx = tagged.findIndex((p) => p.sessionId === sessionId);
        const next = [...tagged];
        const [item] = next.splice(fromIdx, 1);
        if (!item) return { panes: tagged };
        // After removal the last-member index shifts left by one if the moved
        // pane was before it; insert just after the last member either way.
        const insertAt = fromIdx < lastMember ? lastMember : lastMember + 1;
        next.splice(insertAt, 0, item);
        return { panes: next };
      }),
      toggleGroupCollapsed: (groupId) => set((state) => ({
        groups: state.groups.map((g) =>
          g.id === groupId ? { ...g, isCollapsed: !g.isCollapsed } : g
        ),
      })),
      applyAppearance: (sessionId, options) =>
        set((state) => {
          const source = state.panes.find((p) => p.sessionId === sessionId);
          if (!source || options.properties.length === 0) return state;
          const patch: Partial<Pick<PaneMetadata, "headerColor" | "themeId" | "fontSize">> = {};
          if (options.properties.includes("headerColor")) patch.headerColor = source.headerColor;
          if (options.properties.includes("themeId")) patch.themeId = source.themeId;
          if (options.properties.includes("fontSize")) patch.fontSize = source.fontSize;
          const next: Partial<WorkspaceState> = {};
          if (options.toExistingPanes) {
            next.panes = state.panes.map((p) => ({ ...p, ...patch }));
          }
          if (options.asNewPaneDefault) {
            if (patch.headerColor !== undefined) next.defaultHeaderColor = patch.headerColor;
            if (patch.themeId !== undefined) next.defaultThemeId = patch.themeId;
            if (patch.fontSize !== undefined) next.defaultFontSize = patch.fontSize;
          }
          return next;
        }),
      setTabContextMenu: (menu) => set({ tabContextMenu: menu }),
      setManageGroupsTarget: (target) => set({ manageGroupsTarget: target }),
      setKeepScreenAwake: (enabled) => set({ keepScreenAwake: enabled }),
    }),
    {
      name: "wc-workspace",
      version: 17,
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
        if (version < 9) {
          state.plusButtonBehavior ??= "launcher";
        }
        if (version < 10) {
          state.persistentMode ??= false;
          // v12: Replace commandPrefix with wake word settings
          delete (state as Record<string, unknown>).commandPrefix;
          state.wakeWordEnabled ??= false;
          state.wakeWordThreshold ??= DEFAULT_WAKE_WORD_THRESHOLD;
          state.segmentSilenceMs ??= 1500;
        }
        if (version < 11) {
          state.keepScreenAwake ??= true;
        }
        // v12 introduced lowLatencyVoice, later removed — no migration needed
        // (any persisted value is simply ignored on hydrate).
        if (version < 13) {
          state.startMutedOnLoad ??= true;
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
          state.recentHeaderColors ??= [];
          state.sidebarSortMode ??= "manual";
        }
        if (version < 16) {
          state.adaptiveChrome ??= true;
        }
        if (version < 17) {
          state.sidebarOriginTab ??= "ui";
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
    },
  ),
);
