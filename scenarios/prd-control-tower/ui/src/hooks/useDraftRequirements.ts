import { useEffect, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import type { Draft, RequirementGroup, RequirementsResponse } from '../types'

interface UseDraftRequirementsOptions {
  selectedDraft: Draft | null
}

interface UseDraftRequirementsReturn {
  requirementsData: RequirementGroup[] | null
  requirementsLoading: boolean
  requirementsError: string | null
}

export function useDraftRequirements({
  selectedDraft,
}: UseDraftRequirementsOptions): UseDraftRequirementsReturn {
  const [requirementsData, setRequirementsData] = useState<RequirementGroup[] | null>(null)
  const [requirementsLoading, setRequirementsLoading] = useState(false)
  const [requirementsError, setRequirementsError] = useState<string | null>(null)

  useEffect(() => {
    if (!selectedDraft) {
      setRequirementsData(null)
      setRequirementsError(null)
      setRequirementsLoading(false)
      return
    }

    const controller = new AbortController()

    const loadRequirements = async () => {
      setRequirementsLoading(true)
      setRequirementsError(null)
      try {
        const encodedName = encodeURIComponent(selectedDraft.entity_name)
        const response = await fetch(
          buildApiUrl(`/catalog/${selectedDraft.entity_type}/${encodedName}/requirements`),
          { signal: controller.signal },
        )
        if (response.status === 404) {
          setRequirementsData([])
          return
        }
        if (!response.ok) {
          throw new Error(await response.text())
        }
        const data: RequirementsResponse = await response.json()
        setRequirementsData(data.groups)
      } catch (err) {
        if (controller.signal.aborted) {
          return
        }
        setRequirementsError(err instanceof Error ? err.message : 'Failed to load requirements')
      } finally {
        if (!controller.signal.aborted) {
          setRequirementsLoading(false)
        }
      }
    }

    loadRequirements()
    return () => controller.abort()
  }, [selectedDraft?.id]) // Only re-run when draft ID changes

  return {
    requirementsData,
    requirementsLoading,
    requirementsError,
  }
}
