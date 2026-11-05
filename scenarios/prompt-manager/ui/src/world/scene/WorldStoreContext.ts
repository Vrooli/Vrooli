import { createContext, useContext } from 'react'
import type { WorldStore } from '../sim'

/** The live sim store, read by scene components inside useFrame through getState(). */
export const WorldStoreContext = createContext<WorldStore | null>(null)

export function useWorldStore(): WorldStore {
  const store = useContext(WorldStoreContext)
  if (!store) throw new Error('WorldStoreContext is missing; mount scene components under WorldView')
  return store
}
