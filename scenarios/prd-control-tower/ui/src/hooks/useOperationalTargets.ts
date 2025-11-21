import { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { buildApiUrl } from '../utils/apiClient'
import type { OperationalTarget, OperationalTargetsResponse, RequirementRecord } from '../types'

interface UseOperationalTargetsOptions {
  entityType?: string
  entityName?: string
  autoSelectTargetId?: string | null
  autoSelectTargetSearch?: string | null
}

interface OperationalTargetsState {
  targets: OperationalTarget[]
  unmatchedRequirements: RequirementRecord[]
  loading: boolean
  error: string | null
  selectedTarget: OperationalTarget | null
  filter: string
  setFilter: (value: string) => void
  categoryFilter: string
  setCategoryFilter: (value: string) => void
  statusFilter: 'all' | 'complete' | 'pending'
  setStatusFilter: (value: 'all' | 'complete' | 'pending') => void
  filteredTargets: OperationalTarget[]
  handleSelectTarget: (target: OperationalTarget) => void
  refresh: () => Promise<void>
}

export function useOperationalTargets({ entityType, entityName, autoSelectTargetId, autoSelectTargetSearch }: UseOperationalTargetsOptions): OperationalTargetsState {
  const [targets, setTargets] = useState<OperationalTarget[]>([])
  const [unmatchedRequirements, setUnmatchedRequirements] = useState<RequirementRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedTarget, setSelectedTarget] = useState<OperationalTarget | null>(null)
  const [filter, setFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState<'all' | 'complete' | 'pending'>('all')

  const fetchTargets = useCallback(async () => {
    if (!entityType || !entityName) {
      setTargets([])
      setUnmatchedRequirements([])
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/targets`))
      if (!response.ok) {
        throw new Error(`Failed to fetch operational targets: ${response.statusText}`)
      }

      const data = (await response.json()) as OperationalTargetsResponse
      setTargets(data.targets || [])
      setUnmatchedRequirements(data.unmatched_requirements || [])
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error loading operational targets'
      setError(message)
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }, [entityType, entityName])

  useEffect(() => {
    void fetchTargets()
  }, [fetchTargets])

  // Auto-select target based on URL params
  useEffect(() => {
    if (!targets.length) return

    // Priority 1: Select by exact target ID
    if (autoSelectTargetId) {
      const target = targets.find((t) => t.id === autoSelectTargetId)
      if (target) {
        setSelectedTarget(target)
        return
      }
    }

    // Priority 2: Select by search term (matches against target ID)
    if (autoSelectTargetSearch) {
      const target = targets.find((t) => t.id === autoSelectTargetSearch)
      if (target) {
        setSelectedTarget(target)
        setFilter(autoSelectTargetSearch)
        return
      }
    }
  }, [targets, autoSelectTargetId, autoSelectTargetSearch])

  const filteredTargets = useMemo(() => {
    return targets.filter((target) => {
      const matchesSearch =
        !filter.trim() ||
        target.title.toLowerCase().includes(filter.toLowerCase()) ||
        target.id.toLowerCase().includes(filter.toLowerCase()) ||
        target.notes?.toLowerCase().includes(filter.toLowerCase())

      const matchesCategory = categoryFilter === 'all' || target.category === categoryFilter

      const matchesStatus =
        statusFilter === 'all' || (statusFilter === 'complete' ? target.status === 'complete' : target.status === 'pending')

      return matchesSearch && matchesCategory && matchesStatus
    })
  }, [targets, filter, categoryFilter, statusFilter])

  const handleSelectTarget = useCallback((target: OperationalTarget) => {
    setSelectedTarget(target)
  }, [])

  return {
    targets,
    unmatchedRequirements,
    loading,
    error,
    selectedTarget,
    filter,
    setFilter,
    categoryFilter,
    setCategoryFilter,
    statusFilter,
    setStatusFilter,
    filteredTargets,
    handleSelectTarget,
    refresh: fetchTargets,
  }
}
