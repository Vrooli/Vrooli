/**
 * EditorErrorBoundary - Error boundary for the WYSIWYG editor.
 *
 * Catches errors in the editor components and displays a fallback UI
 * instead of crashing the entire application.
 */

import { Component, type ReactNode, type ErrorInfo } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface EditorErrorBoundaryProps {
  /** Child components to render */
  children: ReactNode
  /** Optional fallback render function */
  fallback?: (error: Error, reset: () => void) => ReactNode
  /** Callback when an error is caught */
  onError?: (error: Error, errorInfo: ErrorInfo) => void
  /** Additional CSS classes */
  className?: string
}

interface EditorErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

/**
 * Error boundary that catches errors in the editor and provides a recovery option.
 */
export class EditorErrorBoundary extends Component<
  EditorErrorBoundaryProps,
  EditorErrorBoundaryState
> {
  constructor(props: EditorErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): EditorErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    console.error('EditorErrorBoundary caught an error:', error, errorInfo)
    this.props.onError?.(error, errorInfo)
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: null })
  }

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.handleReset)
      }

      return (
        <DefaultErrorFallback
          error={this.state.error}
          onReset={this.handleReset}
          className={this.props.className}
        />
      )
    }

    return this.props.children
  }
}

interface DefaultErrorFallbackProps {
  error: Error
  onReset: () => void
  className?: string
}

/**
 * Default error fallback UI.
 */
function DefaultErrorFallback({
  error,
  onReset,
  className,
}: DefaultErrorFallbackProps) {
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
