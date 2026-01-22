/**
 * PanelErrorBoundary - Reusable error boundary for UI panels.
 *
 * Provides a less intrusive error UI suitable for individual sections,
 * allowing other parts of the app to continue functioning when one
 * panel encounters an error.
 */

import React from 'react'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'

interface PanelErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

interface PanelErrorBoundaryProps {
  children: React.ReactNode
  /**
   * Panel name for display in error message
   */
  panelName?: string
  /**
   * Optional custom className for the error container
   */
  className?: string
}

/**
 * Error boundary component for individual panels.
 *
 * Usage:
 * ```tsx
 * <PanelErrorBoundary panelName="Editor">
 *   <EditorPanel />
 * </PanelErrorBoundary>
 * ```
 */
export class PanelErrorBoundary extends React.Component<
  PanelErrorBoundaryProps,
  PanelErrorBoundaryState
> {
  constructor(props: PanelErrorBoundaryProps) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
    }
  }

  static getDerivedStateFromError(error: Error): Partial<PanelErrorBoundaryState> {
    return {
      hasError: true,
      error,
    }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    // Log error with panel context
    console.error(
      `🚨 [${this.props.panelName ?? 'Panel'}] Error:`,
      error
    )
    console.error('📍 Component Stack:', errorInfo.componentStack)
  }

  handleRetry = () => {
    this.setState({
      hasError: false,
      error: null,
    })
  }

  /**
   * Render the error fallback UI.
   */
  private renderErrorFallback() {
    const { panelName, className } = this.props
    const { error } = this.state

    return (
      <div
        className={cn(
          'flex flex-col items-center justify-center p-6 h-full min-h-[200px]',
          'bg-red-950/20 border border-red-900/30 rounded-lg',
          className
        )}
      >
        <div className="flex items-center justify-center w-12 h-12 mb-4 bg-red-900/30 rounded-full">
          <AlertTriangle className="w-6 h-6 text-red-400" />
        </div>

        <h3 className="text-sm font-medium text-red-300 mb-1">
          {panelName ? `${panelName} Error` : 'Something went wrong'}
        </h3>

        <p className="text-xs text-red-400/80 mb-4 text-center max-w-xs">
          {error?.message ?? 'An unexpected error occurred'}
        </p>

        <button
          type="button"
          onClick={this.handleRetry}
          className={cn(
            'flex items-center gap-2 px-3 py-1.5 text-xs font-medium',
            'text-red-300 bg-red-900/30 hover:bg-red-900/50',
            'border border-red-800/50 rounded-md transition-colors'
          )}
        >
          <RefreshCw className="w-3.5 h-3.5" />
          Try Again
        </button>
      </div>
    )
  }

  render() {
    if (this.state.hasError && this.state.error) {
      return this.renderErrorFallback()
    }

    return this.props.children
  }
}
