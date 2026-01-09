import { useCallback, useEffect, useMemo, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import type { Draft } from '../types'

interface UseDraftsOptions {
  routeEntityType?: string
  routeEntityName?: string
  decodedRouteName: string
}

interface UseDraftsReturn {
  drafts: Draft[]
  loading: boolean
  refreshing: boolean
  error: string | null
  filter: string
  setFilter: (value: string) => void
  filteredDrafts: Draft[]
  selectedDraft: Draft | null
  fetchDrafts: (options?: { silent?: boolean }) => Promise<void>
}

export function useDrafts({
  routeEntityType,
  routeEntityName,
  decodedRouteName,
}: UseDraftsOptions): UseDraftsReturn {
  const normalizedRouteType = routeEntityType?.toLowerCase()
  const normalizedRouteName = decodedRouteName?.toLowerCase?.() ?? ''

  const [drafts, setDrafts] = useState<Draft[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState(decodedRouteName)

  const fetchDrafts = useCallback(async (options?: { silent?: boolean }) => {
    if (options?.silent) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }
    setError(null)

    try {
      const response = await fetch(buildApiUrl('/drafts'))
      if (!response.ok) {
        throw new Error(`Failed to fetch drafts: ${response.statusText}`)
      }
      const data: { drafts: Draft[] } = await response.json()
      setDrafts(data.drafts || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      if (options?.silent) {
        setRefreshing(false)
      } else {
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    fetchDrafts()
  }, [fetchDrafts])

  useEffect(() => {
    if (decodedRouteName) {
      setFilter(decodedRouteName)
      return
    }

    if (routeEntityName === undefined) {
      setFilter('')
    }
  }, [decodedRouteName, routeEntityName])

  const selectedDraft = useMemo(() => {
    if (!normalizedRouteName) {
      return null
    }

    return (
      drafts.find((draft) => {
        const sameName = draft.entity_name.toLowerCase() === normalizedRouteName
        if (normalizedRouteType) {
          return sameName && draft.entity_type.toLowerCase() === normalizedRouteType
        }
        return sameName
      }) || null
    )
  }, [drafts, normalizedRouteName, normalizedRouteType])

  const filteredDrafts = useMemo(() => {
    const searchLower = filter.toLowerCase()
    const routeFilterActive = Boolean(routeEntityName && decodedRouteName)

    return drafts.filter((draft) => {
      const matchesSearch =
        draft.entity_name.toLowerCase().includes(searchLower) ||
        draft.entity_type.toLowerCase().includes(searchLower) ||
        (draft.owner && draft.owner.toLowerCase().includes(searchLower))
      const matchesRouteType =
        !normalizedRouteType || !routeFilterActive || draft.entity_type.toLowerCase() === normalizedRouteType
      const matchesRouteName =
        !normalizedRouteName || !routeFilterActive || draft.entity_name.toLowerCase() === normalizedRouteName

      return matchesSearch && matchesRouteType && matchesRouteName
    })
  }, [drafts, filter, normalizedRouteName, normalizedRouteType, routeEntityName, decodedRouteName])

  return {
    drafts,
    loading,
    refreshing,
    error,
    filter,
    setFilter,
    filteredDrafts,
    selectedDraft,
    fetchDrafts,
  }
}
