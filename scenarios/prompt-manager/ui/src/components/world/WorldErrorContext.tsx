/**
 * WorldErrorContext - Centralized error tracking for 3D world components.
 * Collects and manages errors from all error boundaries in the 3D scene.
 */

import { createContext, useContext, useCallback, useState, type ReactNode } from 'react'

/** Recorded error with metadata */
export interface WorldError {
  id: string
  componentName: string
  error: Error
  timestamp: number
  recovered: boolean
}

interface WorldErrorContextValue {
  /** Recent errors from world components */
  errors: WorldError[]
  /** Record a new error */
  recordError: (componentName: string, error: Error) => string
  /** Mark an error as recovered */
  markRecovered: (errorId: string) => void
  /** Clear all errors */
  clearErrors: () => void
  /** Check if any errors are active (not recovered) */
  hasActiveErrors: boolean
  /** Get errors for a specific component */
  getComponentErrors: (componentName: string) => WorldError[]
}

const WorldErrorContext = createContext<WorldErrorContextValue | null>(null)

let errorIdCounter = 0

interface WorldErrorProviderProps {
  children: ReactNode
  /** Maximum number of errors to keep in history */
  maxErrors?: number
}

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

/**
 * Hook to access world error context.
 * Returns null if used outside of WorldErrorProvider.
 */
export function useWorldErrors(): WorldErrorContextValue | null {
  return useContext(WorldErrorContext)
}

/**
 * Hook that throws if used outside of WorldErrorProvider.
 */
export function useWorldErrorsRequired(): WorldErrorContextValue {
  const context = useContext(WorldErrorContext)
  if (!context) {
    throw new Error('useWorldErrorsRequired must be used within a WorldErrorProvider')
  }
  return context
}
