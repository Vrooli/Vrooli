/**
 * Interaction store for managing 3D scene interactions.
 * Handles hover, drag, selection modes, and input state.
 */
// AI_CHECK: R3F_INTERACTION_CHURN=1 | LAST: 2026-02-17

import { create } from 'zustand'
import { usePerformanceStore } from '@/stores/performanceStore'

/** Available interaction modes */
export type InteractionMode = 'navigate' | 'select' | 'drag' | 'place'

/** Drag state information */
export interface DragState {
  objectId: string
  startPosition: [number, number, number]
  currentPosition: [number, number, number]
  offset: [number, number, number]
}

interface InteractionState {
  /** Current interaction mode */
  mode: InteractionMode
  /** ID of currently hovered object (null if none) */
  hoveredObjectId: string | null
  /** Whether a drag operation is in progress */
  isDragging: boolean
  /** Dragged object information */
  draggedObjectId: string | null
  /** Full drag state for position tracking */
  dragState: DragState | null
  /** Currently selected object IDs */
  selectedObjectIds: string[]
  /** Whether multi-select is active (shift held) */
  isMultiSelectActive: boolean
  /** Last click position in world coordinates */
  lastClickPosition: [number, number, number] | null
}

interface InteractionActions {
  /** Set the interaction mode */
  setMode: (mode: InteractionMode) => void
  /** Set hovered object */
  setHovered: (id: string | null) => void
  /** Start dragging an object */
  startDrag: (objectId: string, startPosition: [number, number, number]) => void
  /** Update drag position */
  updateDrag: (currentPosition: [number, number, number]) => void
  /** End drag operation */
  endDrag: () => void
  /** Cancel drag operation */
  cancelDrag: () => void
  /** Select an object */
  selectObject: (id: string) => void
  /** Toggle object selection */
  toggleSelection: (id: string) => void
  /** Add to selection (multi-select) */
  addToSelection: (id: string) => void
  /** Clear all selections */
  clearSelection: () => void
  /** Set multi-select active state */
  setMultiSelectActive: (active: boolean) => void
  /** Record click position */
  setLastClickPosition: (position: [number, number, number] | null) => void
  /** Reset all interaction state */
  reset: () => void
}

type InteractionStore = InteractionState & InteractionActions

const initialState: InteractionState = {
  mode: 'navigate',
  hoveredObjectId: null,
  isDragging: false,
  draggedObjectId: null,
  dragState: null,
  selectedObjectIds: [],
  isMultiSelectActive: false,
  lastClickPosition: null,
}

/**
 * Zustand store for interaction state
 */
export const useInteractionStore = create<InteractionStore>((set, get) => ({
  ...initialState,

  setMode: (mode) => {
    set({ mode })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setHovered: (id) => {
    const state = get()
    if (state.isDragging || state.hoveredObjectId === id) return // Don't change hover during drag or no-op updates
    set({ hoveredObjectId: id })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  startDrag: (objectId, startPosition) => {
    set({
      isDragging: true,
      draggedObjectId: objectId,
      dragState: {
        objectId,
        startPosition,
        currentPosition: startPosition,
        offset: [0, 0, 0],
      },
      mode: 'drag',
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  updateDrag: (currentPosition) => {
    const { dragState } = get()
    if (!dragState) return

    if (
      dragState.currentPosition[0] === currentPosition[0] &&
      dragState.currentPosition[1] === currentPosition[1] &&
      dragState.currentPosition[2] === currentPosition[2]
    ) {
      return
    }

    set({
      dragState: {
        ...dragState,
        currentPosition,
        offset: [
          currentPosition[0] - dragState.startPosition[0],
          currentPosition[1] - dragState.startPosition[1],
          currentPosition[2] - dragState.startPosition[2],
        ],
      },
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  endDrag: () => {
    set({
      isDragging: false,
      draggedObjectId: null,
      dragState: null,
      mode: 'navigate',
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  cancelDrag: () => {
    set({
      isDragging: false,
      draggedObjectId: null,
      dragState: null,
      mode: 'navigate',
    })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  selectObject: (id) => {
    set({ selectedObjectIds: [id] })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  toggleSelection: (id) => {
    const { selectedObjectIds } = get()
    if (selectedObjectIds.includes(id)) {
      set({ selectedObjectIds: selectedObjectIds.filter((sid) => sid !== id) })
    } else {
      set({ selectedObjectIds: [...selectedObjectIds, id] })
    }
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  addToSelection: (id) => {
    const { selectedObjectIds } = get()
    if (!selectedObjectIds.includes(id)) {
      set({ selectedObjectIds: [...selectedObjectIds, id] })
      usePerformanceStore.getState().recordInteractionStoreWrite()
    }
  },

  clearSelection: () => {
    set({ selectedObjectIds: [] })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setMultiSelectActive: (active) => {
    set({ isMultiSelectActive: active })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  setLastClickPosition: (position) => {
    set({ lastClickPosition: position })
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },

  reset: () => {
    set(initialState)
    usePerformanceStore.getState().recordInteractionStoreWrite()
  },
}))

/**
 * Selector for checking if a specific object is hovered
 */
export const selectIsHovered = (state: InteractionStore, objectId: string) =>
  state.hoveredObjectId === objectId

/**
 * Selector for checking if a specific object is selected
 */
export const selectIsSelected = (state: InteractionStore, objectId: string) =>
  state.selectedObjectIds.includes(objectId)

/**
 * Selector for checking if a specific object is being dragged
 */
export const selectIsDragged = (state: InteractionStore, objectId: string) =>
  state.draggedObjectId === objectId
