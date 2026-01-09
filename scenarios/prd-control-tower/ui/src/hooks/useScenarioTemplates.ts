import { useCallback, useEffect, useState } from 'react'

import { buildApiUrl } from '../utils/apiClient'
import type { ScenarioTemplate, ScenarioTemplateListResponse } from '../types'

interface UseScenarioTemplatesOptions {
  enabled?: boolean
}

export function useScenarioTemplates(options: UseScenarioTemplatesOptions = {}) {
  const { enabled = true } = options
  const [templates, setTemplates] = useState<ScenarioTemplate[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchTemplates = useCallback(async () => {
    if (!enabled) {
      return
    }

    setLoading(true)
    setError(null)
    try {
      const response = await fetch(buildApiUrl('/scenario-templates'))
      if (!response.ok) {
        throw new Error(`Unable to load templates (${response.statusText})`)
      }

      const data: ScenarioTemplateListResponse = await response.json()
      setTemplates(data.templates)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load templates')
      setTemplates([])
    } finally {
      setLoading(false)
    }
  }, [enabled])

  useEffect(() => {
    fetchTemplates()
  }, [fetchTemplates])

  return { templates, loading, error, refresh: fetchTemplates }
}
