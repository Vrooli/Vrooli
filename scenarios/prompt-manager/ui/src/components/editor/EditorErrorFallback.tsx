/**
 * EditorErrorFallback - Default error UI for the editor boundary.
 */

import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'

interface EditorErrorFallbackProps {
  error: Error
  onReset: () => void
  className?: string
}

export function EditorErrorFallback({
  error,
  onReset,
  className,
}: EditorErrorFallbackProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center p-8',
        'bg-card rounded-lg border border-destructive/30',
        'text-center',
        className
      )}
    >
      <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
      <h3 className="text-lg font-semibold text-foreground mb-2">
        Editor Error
      </h3>
      <p className="text-sm text-muted-foreground mb-4 max-w-md">
        Something went wrong with the editor. This might be due to invalid
        content or a temporary issue.
      </p>
      <details className="text-xs text-muted-foreground mb-4 max-w-md">
        <summary className="cursor-pointer hover:text-foreground">
          Technical details
        </summary>
        <pre className="mt-2 p-2 bg-muted rounded text-left overflow-auto">
          {error.message}
        </pre>
      </details>
      <button
        type="button"
        onClick={onReset}
        className={cn(
          'flex items-center gap-2 px-4 py-2',
          'bg-primary hover:bg-primary/90 text-primary-foreground',
          'rounded-lg transition-colors'
        )}
      >
        <RefreshCw className="h-4 w-4" />
        Try Again
      </button>
    </div>
  )
}
