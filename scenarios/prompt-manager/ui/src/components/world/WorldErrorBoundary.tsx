/**
 * WorldErrorBoundary - Error boundary specialized for 3D world components.
 *
 * Features:
 * - Catches WebGL/Three.js specific errors
 * - 'minimal' mode for nested boundaries (returns null instead of error UI)
 * - 'componentName' prop for debugging which component failed
 * - 'onError' callback for error tracking
 * - Reset/retry functionality
 */

import { Component, type ReactNode, type ErrorInfo } from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface WorldErrorBoundaryProps {
  /** Child components to render */
  children: ReactNode
  /** Name of the wrapped component for debugging */
  componentName?: string
  /** If true, returns null on error instead of showing error UI */
  minimal?: boolean
  /** Callback when an error is caught */
  onError?: (error: Error, componentName: string) => void
  /** Optional fallback render function */
  fallback?: (error: Error, reset: () => void, componentName: string) => ReactNode
  /** Additional CSS classes */
  className?: string
}

interface WorldErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

/**
 * Error boundary for 3D world components.
 * Use 'minimal' mode for inner components that should fail silently.
 * Use default mode for outer wrappers that should show error UI.
 */
export class WorldErrorBoundary extends Component<
  WorldErrorBoundaryProps,
  WorldErrorBoundaryState
> {
  constructor(props: WorldErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): WorldErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    const componentName = this.props.componentName ?? 'Unknown3DComponent'

    console.error(
      `WorldErrorBoundary caught an error in ${componentName}:`,
      error,
      errorInfo
    )

    // Check if this is a WebGL-specific error
    const errorMessage = error.message || ''
    const isWebGLError =
      errorMessage.includes('WebGL') ||
      errorMessage.includes('context') ||
      errorMessage.includes('THREE') ||
      errorMessage.includes('Cannot read properties of undefined')

    if (isWebGLError) {
      console.warn(
        `WebGL/Three.js error detected in ${componentName}. ` +
        'This may be due to browser compatibility or GPU issues.'
      )
    }

    this.props.onError?.(error, componentName)
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: null })
  }

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      const componentName = this.props.componentName ?? 'Unknown'

      // In minimal mode, just return null - fail silently
      if (this.props.minimal) {
        return null
      }

      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, this.handleReset, componentName)
      }

      // Default error UI
      return (
        <WorldErrorFallback
          error={this.state.error}
          componentName={componentName}
          onReset={this.handleReset}
          className={this.props.className}
        />
      )
    }

    return this.props.children
  }
}

interface WorldErrorFallbackProps {
  error: Error
  componentName: string
  onReset: () => void
  className?: string
}

/**
 * Default error fallback UI for 3D world errors.
 */
function WorldErrorFallback({
  error,
  componentName,
  onReset,
  className,
}: WorldErrorFallbackProps) {
  const errorMessage = error.message || ''
  const isWebGLError =
    errorMessage.includes('WebGL') ||
    errorMessage.includes('context') ||
    errorMessage.includes('THREE')

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center p-8',
        'bg-card/80 backdrop-blur-sm rounded-lg border border-destructive/30',
        'text-center',
        className
      )}
      data-testid="world-error-fallback"
    >
      <AlertTriangle className="h-12 w-12 text-destructive mb-4" />
      <h3 className="text-lg font-semibold text-foreground mb-2">
        3D World Error
      </h3>
      <p className="text-sm text-muted-foreground mb-2">
        Component: <code className="text-xs bg-muted px-1 rounded">{componentName}</code>
      </p>
      <p className="text-sm text-muted-foreground mb-4 max-w-md">
        {isWebGLError
          ? 'There was a WebGL rendering issue. This might be due to browser compatibility or GPU limitations.'
          : 'Something went wrong rendering this part of the 3D world.'}
      </p>
      <details className="text-xs text-muted-foreground mb-4 max-w-md">
        <summary className="cursor-pointer hover:text-foreground">
          Technical details
        </summary>
        <pre className="mt-2 p-2 bg-muted rounded text-left overflow-auto max-h-32">
          {error.message}
          {error.stack && (
            <>
              {'\n\n'}
              {error.stack}
            </>
          )}
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

export default WorldErrorBoundary
