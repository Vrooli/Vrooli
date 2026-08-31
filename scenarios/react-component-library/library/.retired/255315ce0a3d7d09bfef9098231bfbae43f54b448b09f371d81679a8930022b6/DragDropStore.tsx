/** @vrooliComponentSource services.drag-drop-store */
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
  start: (
    id: string,
    phase: Exclude<DragPhase, "idle">,
    position: DragPosition,
  ) => void;
  move: (position: DragPosition, velocity?: DragPosition) => void;
  setOver: (id?: string) => void;
  end: () => void;
  cancel: () => void;
}

export function createDragDropStore(
  initialPosition: DragPosition = { x: 0, y: 0 },
): DragDropStore {
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
