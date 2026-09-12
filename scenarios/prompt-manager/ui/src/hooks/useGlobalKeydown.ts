import { useEffect } from 'react'

type KeydownTarget = 'document' | 'window'

interface UseGlobalKeydownOptions {
  enabled?: boolean
  target?: KeydownTarget
}

export function useGlobalKeydown(
  handler: (event: KeyboardEvent) => void,
  { enabled = true, target = 'window' }: UseGlobalKeydownOptions = {}
) {
  useEffect(() => {
    if (!enabled) return

    if (target === 'document') {
      document.addEventListener('keydown', handler)
      return () => {
        document.removeEventListener('keydown', handler)
      }
    }

    window.addEventListener('keydown', handler)
    return () => {
      window.removeEventListener('keydown', handler)
    }
  }, [enabled, handler, target])
}
