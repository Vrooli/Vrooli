import { useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { worldPath } from './route-paths'

interface BrowserHistoryState {
  idx?: number
}

export function useAppBack(fallbackPath = worldPath()) {
  const navigate = useNavigate()

  return useCallback(() => {
    const state = window.history.state as BrowserHistoryState | null
    if (typeof state?.idx === 'number' && state.idx > 0) {
      navigate(-1)
      return
    }
    navigate(fallbackPath, { replace: true })
  }, [fallbackPath, navigate])
}
