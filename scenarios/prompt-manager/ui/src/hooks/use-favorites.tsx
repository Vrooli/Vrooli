/**
 * Local favorites management using localStorage.
 *
 * Favorites are not stored in the API - this is client-side state only.
 * This follows the boundary-of-responsibility principle: the API handles
 * prompt storage and metrics, while favorites are a UI preference.
 */

import { useState, useCallback, useEffect } from 'react'

const FAVORITES_KEY = 'prompt-manager-favorites'

/**
 * Hook for managing prompt favorites in localStorage
 */
export function useFavorites() {
  const [favorites, setFavorites] = useState<Set<string>>(() => {
    if (typeof window === 'undefined') return new Set()
    const stored = localStorage.getItem(FAVORITES_KEY)
    if (!stored) return new Set()
    try {
      const parsed = JSON.parse(stored) as string[]
      return new Set(parsed)
    } catch {
      return new Set()
    }
  })

  // Persist to localStorage when favorites change
  useEffect(() => {
    localStorage.setItem(FAVORITES_KEY, JSON.stringify([...favorites]))
  }, [favorites])

  const isFavorite = useCallback((id: string) => favorites.has(id), [favorites])

  const toggleFavorite = useCallback((id: string) => {
    setFavorites(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const addFavorite = useCallback((id: string) => {
    setFavorites(prev => new Set([...prev, id]))
  }, [])

  const removeFavorite = useCallback((id: string) => {
    setFavorites(prev => {
      const next = new Set(prev)
      next.delete(id)
      return next
    })
  }, [])

  const getFavoriteIds = useCallback(() => [...favorites], [favorites])

  return {
    favorites,
    isFavorite,
    toggleFavorite,
    addFavorite,
    removeFavorite,
    getFavoriteIds,
    count: favorites.size,
  }
}
