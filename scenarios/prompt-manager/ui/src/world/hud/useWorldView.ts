import { useSyncExternalStore } from 'react'
import type { WorldStore, WorldView } from '../sim'

/** Subscribe React to the sim's read model; re-renders only on discrete changes. */
export function useWorldView(store: WorldStore): WorldView {
  const subscribe = (listener: () => void) => store.subscribe(listener)
  const getView = () => store.getView()
  return useSyncExternalStore(subscribe, getView, getView)
}
