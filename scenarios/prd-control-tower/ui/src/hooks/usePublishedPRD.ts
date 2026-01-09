import { useCallback, useEffect, useState } from 'react'

import { buildApiUrl } from '../utils/apiClient'
import type { PublishedPRDResponse } from '../types'

interface UsePublishedPRDOptions {
  treatNotFoundAsEmpty?: boolean
}

export function usePublishedPRD(entityType?: string, entityName?: string, options: UsePublishedPRDOptions = {}) {
  const { treatNotFoundAsEmpty = false } = options
  const [data, setData] = useState<PublishedPRDResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    if (!entityType || !entityName) {
      setData(null)
      setError(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}`))
      if (!response.ok) {
        if (response.status === 404 && treatNotFoundAsEmpty) {
          setData(null)
          setError(null)
          setLoading(false)
          return
        }
        throw new Error(`Failed to fetch PRD (status ${response.status})`)
      }

      const payload: PublishedPRDResponse = await response.json()
      setData(payload)
    } catch (err) {
      setData(null)
      setError(err instanceof Error ? err.message : 'Failed to load PRD')
    } finally {
      setLoading(false)
    }
  }, [entityType, entityName, treatNotFoundAsEmpty])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, loading, error, refresh: fetchData }
}
