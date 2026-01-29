/**
 * EditorErrorBoundary - Error boundary for the WYSIWYG editor.
 *
 * Catches errors in the editor components and displays a fallback UI
 * instead of crashing the entire application.
 */

import { Component, type ReactNode, type ErrorInfo } from 'react'
import { EditorErrorFallback } from './EditorErrorFallback'

interface EditorErrorBoundaryProps {
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
        <EditorErrorFallback
          error={this.state.error}
          onReset={this.handleReset}
          className={this.props.className}
        />
      )
    }

    return this.props.children
  }
}
