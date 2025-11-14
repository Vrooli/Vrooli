import { useEffect, useState } from 'react'
import { buildApiUrl } from '../utils/apiClient'
import type { Draft, OperationalTargetsResponse } from '../types'

interface UseDraftTargetsOptions {
  selectedDraft: Draft | null
}

interface UseDraftTargetsReturn {
  targetsData: OperationalTargetsResponse | null
  targetsLoading: boolean
  targetsError: string | null
}

export function useDraftTargets({ selectedDraft }: UseDraftTargetsOptions): UseDraftTargetsReturn {
  const [targetsData, setTargetsData] = useState<OperationalTargetsResponse | null>(null)
  const [targetsLoading, setTargetsLoading] = useState(false)
  const [targetsError, setTargetsError] = useState<string | null>(null)

  useEffect(() => {
    if (!selectedDraft) {
      setTargetsData(null)
      setTargetsError(null)
      setTargetsLoading(false)
      return
    }

    const controller = new AbortController()

    const loadTargets = async () => {
      setTargetsLoading(true)
      setTargetsError(null)
      try {
        const encodedName = encodeURIComponent(selectedDraft.entity_name)
        const response = await fetch(
          buildApiUrl(`/catalog/${selectedDraft.entity_type}/${encodedName}/targets`),
          { signal: controller.signal },
        )
        if (response.status === 404) {
          setTargetsData({
            entity_type: selectedDraft.entity_type,
            entity_name: selectedDraft.entity_name,
            targets: [],
            unmatched_requirements: [],
          })
          return
        }
        if (!response.ok) {
          throw new Error(await response.text())
        }
        const data: OperationalTargetsResponse = await response.json()
        setTargetsData(data)
      } catch (err) {
        if (controller.signal.aborted) {
          return
        }
        setTargetsError(err instanceof Error ? err.message : 'Failed to load operational targets')
      } finally {
        if (!controller.signal.aborted) {
          setTargetsLoading(false)
        }
      }
    }

    loadTargets()
    return () => controller.abort()
  }, [selectedDraft?.id]) // Only re-run when draft ID changes

  return {
    targetsData,
    targetsLoading,
    targetsError,
  }
}
