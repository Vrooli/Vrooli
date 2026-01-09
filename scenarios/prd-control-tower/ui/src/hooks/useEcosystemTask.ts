import { useCallback, useEffect, useState } from 'react'

import { buildApiUrl } from '../utils/apiClient'
import type { EcosystemTaskStatusResponse } from '../types'

export function useEcosystemTask(entityType?: string, entityName?: string) {
  const [status, setStatus] = useState<EcosystemTaskStatusResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const fetchStatus = useCallback(async () => {
    if (!entityType || !entityName) {
      setStatus(null)
      return
    }

    setLoading(true)
    setError(null)
    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/ecosystem-task`))
      if (!response.ok) {
        throw new Error(`Failed to load task status (${response.status})`)
      }
      const payload: EcosystemTaskStatusResponse = await response.json()
      setStatus(payload)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load task status')
      setStatus(null)
    } finally {
      setLoading(false)
    }
  }, [entityType, entityName])

  const createTask = useCallback(async () => {
    if (!entityType || !entityName) {
      return
    }

    setCreating(true)
    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/ecosystem-task`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      })
      if (!response.ok) {
        const text = await response.text()
        throw new Error(text || 'Failed to create task')
      }
      const payload: EcosystemTaskStatusResponse = await response.json()
      setStatus(payload)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create task')
    } finally {
      setCreating(false)
    }
  }, [entityType, entityName])

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  return { status, loading, error, refresh: fetchStatus, createTask, creating }
}
