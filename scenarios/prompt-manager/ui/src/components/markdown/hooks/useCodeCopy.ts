import { useCallback, useState } from 'react'
import { copyToClipboard } from '@/lib/clipboard'
import { toast } from '@/hooks/use-toast'

interface UseCodeCopyReturn {
  /** Whether the code was recently copied */
  copied: boolean
  /** Copy the code to clipboard */
  copyCode: () => void
}

/**
 * Hook for copying code to clipboard with toast feedback.
 */
export function useCodeCopy(code: string): UseCodeCopyReturn {
  const [copied, setCopied] = useState(false)

  const copyCode = useCallback(() => {
    void copyToClipboard(code)
      .then(() => {
        setCopied(true)
        toast({
          title: 'Copied to clipboard',
          variant: 'success',
        })

        // Reset copied state after 2 seconds
        setTimeout(() => setCopied(false), 2000)
      })
      .catch(() => {
        toast({
          title: 'Failed to copy',
          variant: 'destructive',
        })
      })
  }, [code])

  return { copied, copyCode }
}
