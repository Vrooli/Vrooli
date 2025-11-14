import { useCallback, useEffect, useMemo, useState } from 'react'
import toast from 'react-hot-toast'
import { buildApiUrl } from '../utils/apiClient'
import type { RequirementGroup, RequirementRecord, RequirementsResponse } from '../types'

interface UseRequirementsExplorerOptions {
  entityType?: string
  entityName?: string
}

interface RequirementsExplorerState {
  groups: RequirementGroup[]
  loading: boolean
  error: string | null
  selectedRequirement: RequirementRecord | null
  filter: string
  setFilter: (value: string) => void
  filteredGroups: RequirementGroup[]
  handleSelectRequirement: (req: RequirementRecord) => void
  refresh: () => Promise<void>
}

function filterRequirementGroups(groups: RequirementGroup[], query: string): RequirementGroup[] {
  if (!query.trim()) {
    return groups
  }

  const lowerQuery = query.toLowerCase()

  function filterGroup(group: RequirementGroup): RequirementGroup | null {
    const matchingReqs = group.requirements.filter(
      (req) =>
        req.title.toLowerCase().includes(lowerQuery) ||
        req.id.toLowerCase().includes(lowerQuery) ||
        req.description?.toLowerCase().includes(lowerQuery) ||
        req.category?.toLowerCase().includes(lowerQuery),
    )

    const matchingChildren = group.children?.map(filterGroup).filter((g): g is RequirementGroup => g !== null) ?? []

    if (matchingReqs.length === 0 && matchingChildren.length === 0) {
      return null
    }

    return {
      ...group,
      requirements: matchingReqs,
      children: matchingChildren,
    }
  }

  return groups.map(filterGroup).filter((g): g is RequirementGroup => g !== null)
}

export function useRequirementsExplorer({
  entityType,
  entityName,
}: UseRequirementsExplorerOptions): RequirementsExplorerState {
  const [groups, setGroups] = useState<RequirementGroup[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedRequirement, setSelectedRequirement] = useState<RequirementRecord | null>(null)
  const [filter, setFilter] = useState('')

  const fetchRequirements = useCallback(async () => {
    if (!entityType || !entityName) {
      setGroups([])
      return
    }

    setLoading(true)
    setError(null)

    try {
      const response = await fetch(buildApiUrl(`/catalog/${entityType}/${entityName}/requirements`))
      if (!response.ok) {
        throw new Error(`Failed to fetch requirements: ${response.statusText}`)
      }

      const data = (await response.json()) as RequirementsResponse
      setGroups(data.groups || [])
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error loading requirements'
      setError(message)
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }, [entityType, entityName])

  useEffect(() => {
    void fetchRequirements()
  }, [fetchRequirements])

  const filteredGroups = useMemo(() => filterRequirementGroups(groups, filter), [groups, filter])

  const handleSelectRequirement = useCallback((req: RequirementRecord) => {
    setSelectedRequirement(req)
  }, [])

  return {
    groups,
    loading,
    error,
    selectedRequirement,
    filter,
    setFilter,
    filteredGroups,
    handleSelectRequirement,
    refresh: fetchRequirements,
  }
}
