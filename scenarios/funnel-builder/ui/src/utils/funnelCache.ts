import { Funnel } from '../types'

const storageKey = (id: string) => `funnel-preview:${id}`

export const saveFunnelToCache = (funnel: Funnel) => {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.setItem(storageKey(funnel.id), JSON.stringify(funnel))
  } catch (error) {
    console.error('Failed to cache funnel for preview', error)
  }
}

export const loadFunnelFromCache = (id: string): Funnel | null => {
  if (typeof window === 'undefined') {
    return null
  }

  try {
    const raw = window.localStorage.getItem(storageKey(id))
    if (!raw) {
      return null
    }
    return JSON.parse(raw) as Funnel
  } catch (error) {
    console.error('Failed to read cached funnel preview', error)
    return null
  }
}

export const removeFunnelFromCache = (id: string) => {
  if (typeof window === 'undefined') {
    return
  }

  try {
    window.localStorage.removeItem(storageKey(id))
  } catch (error) {
    console.error('Failed to remove cached funnel preview', error)
  }
}
