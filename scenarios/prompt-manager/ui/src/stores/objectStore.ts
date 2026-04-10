/**
 * Object store for managing scene objects.
 * Handles CRUD operations for furniture, decorations, and interactive objects.
 */

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  SceneObject,
  FurnitureObject,
  DecorationObject,
  InteractiveObject,
  ObjectType,
} from '@/types/objects'

/** Union type for all scene objects */
type AnySceneObject = SceneObject | FurnitureObject | DecorationObject | InteractiveObject

interface ObjectState {
  /** All objects in the scene by ID */
  objects: Record<string, AnySceneObject>
  /** Object ordering for z-index */
  objectOrder: string[]
}

interface ObjectActions {
  /** Add a new object to the scene */
  addObject: (object: AnySceneObject) => void
  /** Update an existing object */
  updateObject: (id: string, updates: Partial<AnySceneObject>) => void
  /** Remove an object from the scene */
  removeObject: (id: string) => void
  /** Get an object by ID */
  getObject: (id: string) => AnySceneObject | undefined
  /** Get all objects of a specific type */
  getObjectsByType: (type: ObjectType) => AnySceneObject[]
  /** Move object to front (highest z-index) */
  bringToFront: (id: string) => void
  /** Move object to back (lowest z-index) */
  sendToBack: (id: string) => void
  /** Update object position */
  setObjectPosition: (id: string, position: [number, number, number]) => void
  /** Update object rotation */
  setObjectRotation: (id: string, rotation: [number, number, number]) => void
  /** Update object scale */
  setObjectScale: (id: string, scale: [number, number, number]) => void
  /** Toggle object visibility */
  setObjectVisible: (id: string, visible: boolean) => void
  /** Clear all objects */
  clearAll: () => void
  /** Reset to initial state */
  reset: () => void
}

type ObjectStore = ObjectState & ObjectActions

const initialState: ObjectState = {
  objects: {},
  objectOrder: [],
}

/**
 * Zustand store for scene objects with persistence
 */
export const useObjectStore = create<ObjectStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      addObject: (object) => {
        set({
          objects: { ...get().objects, [object.id]: object },
          objectOrder: [...get().objectOrder, object.id],
        })
      },

      updateObject: (id, updates) => {
        const current = get().objects[id]
        if (!current) return

        set({
          objects: {
            ...get().objects,
            [id]: { ...current, ...updates } as AnySceneObject,
          },
        })
      },

      removeObject: (id) => {
        const { objects, objectOrder } = get()
        const { [id]: _, ...rest } = objects
        void _
        set({
          objects: rest,
          objectOrder: objectOrder.filter((oid) => oid !== id),
        })
      },

      getObject: (id) => get().objects[id],

      getObjectsByType: (type) =>
        Object.values(get().objects).filter((obj) => obj.type === type),

      bringToFront: (id) => {
        const { objectOrder } = get()
        if (!objectOrder.includes(id)) return

        set({
          objectOrder: [...objectOrder.filter((oid) => oid !== id), id],
        })
      },

      sendToBack: (id) => {
        const { objectOrder } = get()
        if (!objectOrder.includes(id)) return

        set({
          objectOrder: [id, ...objectOrder.filter((oid) => oid !== id)],
        })
      },

      setObjectPosition: (id, position) => {
        const current = get().objects[id]
        if (!current) return

        set({
          objects: {
            ...get().objects,
            [id]: { ...current, position },
          },
        })
      },

      setObjectRotation: (id, rotation) => {
        const current = get().objects[id]
        if (!current) return

        set({
          objects: {
            ...get().objects,
            [id]: { ...current, rotation },
          },
        })
      },

      setObjectScale: (id, scale) => {
        const current = get().objects[id]
        if (!current) return

        set({
          objects: {
            ...get().objects,
            [id]: { ...current, scale },
          },
        })
      },

      setObjectVisible: (id, visible) => {
        const current = get().objects[id]
        if (!current) return

        set({
          objects: {
            ...get().objects,
            [id]: { ...current, visible },
          },
        })
      },

      clearAll: () => set({ objects: {}, objectOrder: [] }),

      reset: () => set(initialState),
    }),
    {
      name: 'scene-objects',
      partialize: (state) => ({
        objects: state.objects,
        objectOrder: state.objectOrder,
      }),
    }
  )
)

/**
 * Selector for getting objects in render order
 */
export const selectObjectsInOrder = (state: ObjectStore) =>
  state.objectOrder.map((id) => state.objects[id]).filter(Boolean)

/**
 * Selector for visible objects only
 */
export const selectVisibleObjects = (state: ObjectStore) =>
  Object.values(state.objects).filter((obj) => obj.visible !== false)
