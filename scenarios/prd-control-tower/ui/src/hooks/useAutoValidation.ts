import { useCallback, useEffect, useRef, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import type { DraftValidationResult } from '../types'

interface UseAutoValidationOptions {
  draftId: string | null
  content: string
  enabled?: boolean
  debounceMs?: number
}

interface UseAutoValidationReturn {
  validationResult: DraftValidationResult | null
  validating: boolean
  error: string | null
  lastValidatedAt: Date | null
  triggerValidation: () => Promise<void>
}

/**
 * Hook for automatic PRD validation with debouncing.
 * Validates draft content against PRD template structure in real-time.
 *
 * Features:
 * - Debounced validation (default: 3 seconds after typing stops)
 * - Manual trigger option
 * - Caches validation results to avoid redundant API calls
 * - Tracks last validation timestamp
 */
export function useAutoValidation({
  draftId,
  content,
  enabled = true,
  debounceMs = 3000,
}: UseAutoValidationOptions): UseAutoValidationReturn {
  const [validationResult, setValidationResult] = useState<DraftValidationResult | null>(null)
  const [validating, setValidating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastValidatedAt, setLastValidatedAt] = useState<Date | null>(null)
  const [lastValidatedContent, setLastValidatedContent] = useState('')
  const debounceTimerRef = useRef<number | null>(null)

  const triggerValidation = useCallback(async () => {
    if (!draftId) {
      return
    }

    setValidating(true)
    setError(null)

    try {
      const response = await fetch(buildApiUrl(`/drafts/${draftId}/validate`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ use_cache: false }),
      })

      if (!response.ok) {
        throw new Error(`Validation failed: ${response.statusText}`)
      }

      const data = await response.json()
      setValidationResult(data)
      setLastValidatedAt(new Date())
      setLastValidatedContent(content)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Validation error')
    } finally {
      setValidating(false)
    }
  }, [draftId, content])

  // Auto-validation with debouncing
  useEffect(() => {
    if (!enabled || !draftId || validating) {
      return
    }

    // Don't re-validate if content hasn't changed
    if (content === lastValidatedContent) {
      return
    }

    // Clear existing timer
    if (debounceTimerRef.current !== null) {
      window.clearTimeout(debounceTimerRef.current)
    }

    // Set new timer
    debounceTimerRef.current = window.setTimeout(() => {
      triggerValidation()
    }, debounceMs)

    return () => {
      if (debounceTimerRef.current !== null) {
        window.clearTimeout(debounceTimerRef.current)
      }
    }
  }, [content, draftId, enabled, validating, lastValidatedContent, debounceMs, triggerValidation])

  // Reset validation when draft changes
  useEffect(() => {
    setValidationResult(null)
    setLastValidatedContent('')
    setLastValidatedAt(null)
    setError(null)
  }, [draftId])

  return {
    validationResult,
    validating,
    error,
    lastValidatedAt,
    triggerValidation,
  }
}
