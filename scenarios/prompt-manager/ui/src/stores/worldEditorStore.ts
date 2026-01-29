/**
 * World Editor Store - Manages edit mode and object manipulation.
 * Handles placement, selection, and deletion of world objects.
 */

import { create } from 'zustand'
import type { FurnitureType } from '@/types/furniture'
import type { DecorationType } from '@/types/decoration'

/** Types of objects that can be selected/placed */
export type WorldObjectType = 'member' | 'furniture' | 'decoration'

/** Currently selected object */
export interface SelectedObject {
  id: string
  type: WorldObjectType
}

/** Object being placed (from palette) */
export interface PlacingObject {
  type: WorldObjectType
  subtype: FurnitureType | DecorationType
}

interface WorldEditorState {
  /** Whether edit mode is active */
  isEditMode: boolean
  /** Currently selected object (if any) */
  selectedObject: SelectedObject | null
  /** Object currently being placed */
  placingObject: PlacingObject | null
  /** Whether the palette is open */
  isPaletteOpen: boolean
  /** Current palette tab */
  paletteTab: 'furniture' | 'decorations' | 'members'
  /** History of actions for undo */
  actionHistory: EditorAction[]
  /** Redo stack */
  redoStack: EditorAction[]
}

/** Actions for undo/redo */
type EditorAction =
  | { type: 'add-furniture'; id: string; furnitureType: FurnitureType; position: [number, number, number] }
  | { type: 'add-decoration'; id: string; decorationType: DecorationType; position: [number, number, number] }
  | { type: 'remove-furniture'; id: string; data: unknown }
  | { type: 'remove-decoration'; id: string; data: unknown }
  | { type: 'move-furniture'; id: string; from: [number, number, number]; to: [number, number, number] }
  | { type: 'move-decoration'; id: string; from: [number, number, number]; to: [number, number, number] }

interface WorldEditorActions {
  /** Toggle edit mode */
  setEditMode: (enabled: boolean) => void
  toggleEditMode: () => void
  /** Select an object */
  selectObject: (object: SelectedObject | null) => void
  /** Start placing an object from palette */
  startPlacing: (object: PlacingObject) => void
  /** Cancel placement */
  cancelPlacing: () => void
  /** Confirm placement at position */
  confirmPlacement: (position: [number, number, number]) => void
  /** Toggle palette */
  setPaletteOpen: (open: boolean) => void
  togglePalette: () => void
  /** Set palette tab */
  setPaletteTab: (tab: 'furniture' | 'decorations' | 'members') => void
  /** Delete selected object */
  deleteSelected: () => void
  /** Record action for undo */
  recordAction: (action: EditorAction) => void
  /** Undo last action */
  undo: () => void
  /** Redo last undone action */
  redo: () => void
  /** Reset editor state */
  reset: () => void
}

type WorldEditorStore = WorldEditorState & WorldEditorActions

const initialState: WorldEditorState = {
  isEditMode: false,
  selectedObject: null,
  placingObject: null,
  isPaletteOpen: false,
  paletteTab: 'furniture',
  actionHistory: [],
  redoStack: [],
}

const MAX_HISTORY = 50

export const useWorldEditorStore = create<WorldEditorStore>((set, get) => ({
  ...initialState,

  setEditMode: (enabled) => {
    set({
      isEditMode: enabled,
      // Clear selection when exiting edit mode
      selectedObject: enabled ? get().selectedObject : null,
      placingObject: enabled ? get().placingObject : null,
    })
  },

  toggleEditMode: () => {
    get().setEditMode(!get().isEditMode)
  },

  selectObject: (object) => {
    set({
      selectedObject: object,
      // Cancel placing when selecting existing object
      placingObject: object ? null : get().placingObject,
    })
  },

  startPlacing: (object) => {
    set({
      placingObject: object,
      selectedObject: null, // Deselect when starting to place
    })
  },

  cancelPlacing: () => {
    set({ placingObject: null })
  },

  confirmPlacement: (_position) => {
    const { placingObject } = get()
    if (!placingObject) return

    // Note: Actual furniture/decoration creation is handled by the component
    // that calls this, using the furniture/decoration stores
    set({ placingObject: null })
  },

  setPaletteOpen: (open) => {
    set({ isPaletteOpen: open })
  },

  togglePalette: () => {
    set({ isPaletteOpen: !get().isPaletteOpen })
  },

  setPaletteTab: (tab) => {
    set({ paletteTab: tab })
  },

  deleteSelected: () => {
    const { selectedObject } = get()
    if (!selectedObject) return

    // Note: Actual deletion is handled by the component that observes this store
    set({ selectedObject: null })
  },

  recordAction: (action) => {
    set((state) => ({
      actionHistory: [...state.actionHistory, action].slice(-MAX_HISTORY),
      redoStack: [], // Clear redo on new action
    }))
  },

  undo: () => {
    const { actionHistory } = get()
    if (actionHistory.length === 0) return

    const lastAction = actionHistory[actionHistory.length - 1]
    if (!lastAction) return

    set((state) => ({
      actionHistory: state.actionHistory.slice(0, -1),
      redoStack: [...state.redoStack, lastAction],
    }))

    // Note: Actual undo logic depends on action type and would be handled
    // by components observing this store
  },

  redo: () => {
    const { redoStack } = get()
    if (redoStack.length === 0) return

    const action = redoStack[redoStack.length - 1]
    if (!action) return

    set((state) => ({
      redoStack: state.redoStack.slice(0, -1),
      actionHistory: [...state.actionHistory, action],
    }))
  },

  reset: () => set(initialState),
}))

/**
 * Hook to check if currently in edit mode
 */
export function useIsEditMode(): boolean {
  return useWorldEditorStore((state) => state.isEditMode)
}

/**
 * Hook to get the selected object
 */
export function useSelectedObject(): SelectedObject | null {
  return useWorldEditorStore((state) => state.selectedObject)
}

/**
 * Hook to check if placing an object
 */
export function useIsPlacing(): boolean {
  return useWorldEditorStore((state) => state.placingObject !== null)
}
