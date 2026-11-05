/**
 * @libraryId react-component-library:DragDropStore
 * @displayName Drag Drop Store
 * @description The scoped interaction store tracking the active drag, candidate drop targets, collision results, keyboard dragging state, announcements, and the drag overlay.
 * @version 1.0.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:DragDropStore
 * @vrooliComponentSourceSlot services.drag-drop-store */
import {
  createScopedStore,
  type ScopedStore,
} from "@vrooli/react-component-library/createScopedStore/1";

export type DragPhase = "idle" | "pointer" | "keyboard";
export interface DragPosition {
  x: number;
  y: number;
}
export interface DragDropState {
  activeId?: string;
  phase: DragPhase;
  position: DragPosition;
  velocity: DragPosition;
  overId?: string;
}
export interface DragDropStore extends ScopedStore<DragDropState> {
  start: (id: string, phase: Exclude<DragPhase, "idle">, position: DragPosition) => void;
  move: (position: DragPosition, velocity?: DragPosition) => void;
  setOver: (id?: string) => void;
  end: () => void;
  cancel: () => void;
}

export function createDragDropStore(initialPosition: DragPosition = { x: 0, y: 0 }): DragDropStore {
  const scoped = createScopedStore<DragDropState>({
    phase: "idle",
    position: initialPosition,
    velocity: { x: 0, y: 0 },
  });
  return {
    ...scoped,
    start: (activeId, phase, position) =>
      scoped.set({ activeId, phase, position, velocity: { x: 0, y: 0 } }),
    move: (position, velocity = { x: 0, y: 0 }) =>
      scoped.set((state) => ({ ...state, position, velocity })),
    setOver: (overId) => scoped.set((state) => ({ ...state, overId })),
    end: () =>
      scoped.set((state) => ({
        ...state,
        activeId: undefined,
        phase: "idle",
        velocity: { x: 0, y: 0 },
      })),
    cancel: () =>
      scoped.set((state) => ({
        ...state,
        activeId: undefined,
        phase: "idle",
        position: initialPosition,
        velocity: { x: 0, y: 0 },
      })),
  };
}
