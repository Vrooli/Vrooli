import { create } from "zustand";
import { persist } from "zustand/middleware";
import { clampFontSize } from "../lib/fontSizeUtils";
import { TERMINAL_FONT_SIZE } from "../consts/config";

export interface PaneMetadata {
  sessionId: string;
  name: string;
  headerColor: string;
}

interface WorkspaceState {
  panes: PaneMetadata[];
  columnFractions: number[];
  rowFractions: number[];
  activePane: string | null;
  terminalFontSize: number;
  isMinimapVisible: boolean;
  settingsModalOpen: boolean;
  sessionsModalOpen: boolean;
  aiModalOpen: boolean;
}

interface WorkspaceActions {
  addPane: (sessionId: string, name: string) => void;
  removePane: (sessionId: string) => void;
  renamePaneById: (sessionId: string, name: string) => void;
  setPaneColor: (sessionId: string, color: string) => void;
  movePaneToIndex: (sessionId: string, newIndex: number) => void;
  setColumnFractions: (fractions: number[]) => void;
  setRowFractions: (fractions: number[]) => void;
  setActivePane: (sessionId: string | null) => void;
  setTerminalFontSize: (size: number) => void;
  setMinimapVisible: (visible: boolean) => void;
  setSettingsModalOpen: (open: boolean) => void;
  setSessionsModalOpen: (open: boolean) => void;
  setAiModalOpen: (open: boolean) => void;
  resetLayout: () => void;
}

export type WorkspaceStore = WorkspaceState & WorkspaceActions;

export const useWorkspaceStore = create<WorkspaceStore>()(
  persist(
    (set) => ({
      panes: [],
      columnFractions: [],
      rowFractions: [],
      activePane: null,
      terminalFontSize: TERMINAL_FONT_SIZE,
      isMinimapVisible: true,
      settingsModalOpen: false,
      sessionsModalOpen: false,
      aiModalOpen: false,

      addPane: (sessionId, name) =>
        set((state) => {
          if (state.panes.some((p) => p.sessionId === sessionId)) return state;
          return {
            panes: [
              ...state.panes,
              { sessionId, name, headerColor: "transparent" },
            ],
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
      setTerminalFontSize: (size) => set({ terminalFontSize: clampFontSize(size) }),
      setMinimapVisible: (visible) => set({ isMinimapVisible: visible }),
      setSettingsModalOpen: (open) => set({ settingsModalOpen: open }),
      setSessionsModalOpen: (open) => set({ sessionsModalOpen: open }),
      setAiModalOpen: (open) => set({ aiModalOpen: open }),

      resetLayout: () =>
        set({ columnFractions: [], rowFractions: [] }),
    }),
    {
      name: "wc-workspace",
      partialize: (state) => ({
        panes: state.panes,
        columnFractions: state.columnFractions,
        rowFractions: state.rowFractions,
        terminalFontSize: state.terminalFontSize,
        isMinimapVisible: state.isMinimapVisible,
      }),
    },
  ),
);
