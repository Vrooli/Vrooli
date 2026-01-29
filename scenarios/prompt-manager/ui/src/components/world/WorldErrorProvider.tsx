/**
 * WorldErrorProvider - Centralized error tracking for 3D world components.
 * Collects and manages errors from all error boundaries in the 3D scene.
 */

import { useCallback, useState, type ReactNode } from 'react'
import { WorldErrorContext, type WorldError, type WorldErrorContextValue } from './WorldErrorContext'

interface WorldErrorProviderProps {
  children: ReactNode
  /** Maximum number of errors to keep in history */
  maxErrors?: number
}

let errorIdCounter = 0

/**
 * Provider for centralized 3D world error tracking.
 */
export function WorldErrorProvider({ children, maxErrors = 50 }: WorldErrorProviderProps) {
  const [errors, setErrors] = useState<WorldError[]>([])

  const recordError = useCallback((componentName: string, error: Error): string => {
    const id = `world-error-${++errorIdCounter}`
    const worldError: WorldError = {
      id,
      componentName,
      error,
      timestamp: Date.now(),
      recovered: false,
    }

    setErrors((prev) => {
      const updated = [worldError, ...prev]
      // Keep only recent errors
      return updated.slice(0, maxErrors)
    })

    return id
  }, [maxErrors])

  const markRecovered = useCallback((errorId: string) => {
    setErrors((prev) =>
      prev.map((e) => (e.id === errorId ? { ...e, recovered: true } : e))
    )
  }, [])

  const clearErrors = useCallback(() => {
    setErrors([])
  }, [])

  const hasActiveErrors = errors.some((e) => !e.recovered)

  const getComponentErrors = useCallback(
    (componentName: string) => errors.filter((e) => e.componentName === componentName),
    [errors]
  )

  const value: WorldErrorContextValue = {
    errors,
    recordError,
    markRecovered,
    clearErrors,
    hasActiveErrors,
    getComponentErrors,
  }

  return (
    <WorldErrorContext.Provider value={value}>
      {children}
    </WorldErrorContext.Provider>
  )
}
