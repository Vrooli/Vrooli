/**
 * WorldErrorContext - Centralized error tracking for 3D world components.
 * Collects and manages errors from all error boundaries in the 3D scene.
 */

import { createContext, useContext } from 'react'

/** Recorded error with metadata */
export interface WorldError {
  id: string
  componentName: string
  error: Error
  timestamp: number
  recovered: boolean
}

export interface WorldErrorContextValue {
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

export const WorldErrorContext = createContext<WorldErrorContextValue | null>(null)

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
