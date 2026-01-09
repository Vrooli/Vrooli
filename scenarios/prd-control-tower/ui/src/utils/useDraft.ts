import { useCallback, useRef, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import toast from 'react-hot-toast'
import { buildApiUrl } from './apiClient'

interface UsePrepareDraftOptions {
  onSuccess?: (entityType: string, entityName: string) => void
  onError?: (error: Error) => void
}

export function usePrepareDraft(options?: UsePrepareDraftOptions) {
  const navigate = useNavigate()
  const [preparing, setPreparing] = useState(false)
  const [preparingId, setPreparingId] = useState<string | null>(null)

  // Use refs for callbacks to avoid recreating prepareDraft when options change
  const onSuccessRef = useRef(options?.onSuccess)
  const onErrorRef = useRef(options?.onError)

  useEffect(() => {
    onSuccessRef.current = options?.onSuccess
    onErrorRef.current = options?.onError
  }, [options?.onSuccess, options?.onError])

  const prepareDraft = useCallback(
    async (entityType: string, entityName: string) => {
      const draftKey = `${entityType}:${entityName}`
      setPreparing(true)
      setPreparingId(draftKey)

      try {
        const encodedName = encodeURIComponent(entityName)
        const response = await fetch(buildApiUrl(`/catalog/${entityType}/${encodedName}/draft`), {
          method: 'POST',
        })

        if (!response.ok) {
          const errorMessage = await response.text()
          throw new Error(errorMessage || 'Failed to prepare draft')
        }

        await response.json()

        if (onSuccessRef.current) {
          onSuccessRef.current(entityType, entityName)
        }

        navigate(`/draft/${entityType}/${encodeURIComponent(entityName)}`)
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Unknown error')

        if (onErrorRef.current) {
          onErrorRef.current(error)
        } else {
          toast.error(`Failed to prepare draft for ${entityName}: ${error.message}`)
        }
      } finally {
        setPreparing(false)
        setPreparingId(null)
      }
    },
    [navigate]
  )

  return {
    prepareDraft,
    preparing,
    preparingId,
  }
}
