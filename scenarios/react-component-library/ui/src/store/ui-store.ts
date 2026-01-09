// Zustand store for UI state management

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { EmulatorFrame, ElementSelection, ViewportPreset, AIMessage } from "../types";
import { VIEWPORT_PRESETS } from "../types";

interface UIState {
  // Editor state
  activeComponentId: string | null;
  setActiveComponentId: (id: string | null) => void;

  // Emulator state
  emulatorFrames: EmulatorFrame[];
  addEmulatorFrame: (preset: ViewportPreset) => void;
  removeEmulatorFrame: (frameId: string) => void;
  updateFrameProps: (frameId: string, props: Record<string, unknown>) => void;

  // Element selection state
  selectionMode: boolean;
  selectedElements: ElementSelection[];
  toggleSelectionMode: () => void;
  addSelectedElement: (element: ElementSelection) => void;
  clearSelectedElements: () => void;

  // AI Chat state
  aiMessages: AIMessage[];
  addAIMessage: (message: AIMessage) => void;
  clearAIMessages: () => void;
  aiPanelOpen: boolean;
  toggleAIPanel: () => void;

  // Search and filter state
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  selectedCategory: string | null;
  setSelectedCategory: (category: string | null) => void;

  // Layout state
  editorPanelSize: number;
  previewPanelSize: number;
  setEditorPanelSize: (size: number) => void;
  setPreviewPanelSize: (size: number) => void;

  // Theme
  theme: "light" | "dark";
  toggleTheme: () => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      // Editor
      activeComponentId: null,
      setActiveComponentId: (id) => set({ activeComponentId: id }),

      // Emulator
      emulatorFrames: [
        {
          id: "default-desktop",
          preset: VIEWPORT_PRESETS[0],
          props: {},
        },
      ],
      addEmulatorFrame: (preset) =>
        set((state) => ({
          emulatorFrames: [
            ...state.emulatorFrames,
            {
              id: `frame-${Date.now()}`,
              preset,
              props: {},
            },
          ],
        })),
      removeEmulatorFrame: (frameId) =>
        set((state) => ({
          emulatorFrames: state.emulatorFrames.filter((f) => f.id !== frameId),
        })),
      updateFrameProps: (frameId, props) =>
        set((state) => ({
          emulatorFrames: state.emulatorFrames.map((f) =>
            f.id === frameId ? { ...f, props: { ...f.props, ...props } } : f
          ),
        })),

      // Selection
      selectionMode: false,
      selectedElements: [],
      toggleSelectionMode: () => set((state) => ({ selectionMode: !state.selectionMode })),
      addSelectedElement: (element) =>
        set((state) => ({
          selectedElements: [...state.selectedElements, element],
        })),
      clearSelectedElements: () => set({ selectedElements: [] }),

      // AI Chat
      aiMessages: [],
      addAIMessage: (message) =>
        set((state) => ({ aiMessages: [...state.aiMessages, message] })),
      clearAIMessages: () => set({ aiMessages: [] }),
      aiPanelOpen: false,
      toggleAIPanel: () => set((state) => ({ aiPanelOpen: !state.aiPanelOpen })),

      // Search/Filter
      searchQuery: "",
      setSearchQuery: (query) => set({ searchQuery: query }),
      selectedCategory: null,
      setSelectedCategory: (category) => set({ selectedCategory: category }),

      // Layout
      editorPanelSize: 50,
      previewPanelSize: 50,
      setEditorPanelSize: (size) => set({ editorPanelSize: size }),
      setPreviewPanelSize: (size) => set({ previewPanelSize: size }),

      // Theme
      theme: "dark",
      toggleTheme: () =>
        set((state) => ({ theme: state.theme === "dark" ? "light" : "dark" })),
    }),
    {
      name: "react-component-library-ui",
      partialize: (state) => ({
        theme: state.theme,
        editorPanelSize: state.editorPanelSize,
        previewPanelSize: state.previewPanelSize,
      }),
    }
  )
);
