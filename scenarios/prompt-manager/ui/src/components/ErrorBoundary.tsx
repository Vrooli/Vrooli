import React from 'react'
import { motion } from 'framer-motion'
import { AlertTriangle, RefreshCw, Bug, Copy, CheckCircle2 } from 'lucide-react'
import { Button } from './ui/button'
import { Card, CardContent, CardHeader, CardTitle } from './ui/card'
import { cn } from '@/lib/utils'

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
  errorInfo: React.ErrorInfo | null
  copied: boolean
}

interface ErrorBoundaryProps {
  children: React.ReactNode
  fallback?: React.ComponentType<{ error: Error; resetError: () => void }>
}

export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  private copyTimeout: NodeJS.Timeout | null = null

  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      copied: false
    }
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return {
      hasError: true,
      error
    }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    this.setState({
      error,
      errorInfo
    })

    // Log to console for debugging
    console.error('🚨 Error Boundary Caught:', error)
    console.error('📍 Component Stack:', errorInfo.componentStack)
    
    // In a real app, you might want to send this to an error reporting service
    // Example: sendErrorToService(error, errorInfo)
  }

  handleReset = () => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      copied: false
    })
  }

  handleCopyError = () => {
    if (!this.state.error || !this.state.errorInfo) return

    const errorReport = {
      timestamp: new Date().toISOString(),
      error: {
        name: this.state.error.name,
        message: this.state.error.message,
        stack: this.state.error.stack
      },
      componentStack: this.state.errorInfo.componentStack,
      userAgent: navigator.userAgent,
      url: window.location.href
    }

    navigator.clipboard.writeText(JSON.stringify(errorReport, null, 2))
      .then(() => {
        this.setState({ copied: true })
        if (this.copyTimeout) clearTimeout(this.copyTimeout)
        this.copyTimeout = setTimeout(() => {
          this.setState({ copied: false })
        }, 2000)
      })
      .catch(err => {
        console.error('Failed to copy error report:', err)
      })
  }

  componentWillUnmount() {
    if (this.copyTimeout) {
      clearTimeout(this.copyTimeout)
    }
  }

  render() {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        const FallbackComponent = this.props.fallback
        return <FallbackComponent error={this.state.error!} resetError={this.handleReset} />
      }

      // Default error UI
      return (
        <div className="min-h-screen bg-gradient-to-br from-red-50 to-orange-50 dark:from-red-950 dark:to-orange-950 flex items-center justify-center p-6">
          <motion.div
            initial={{ opacity: 0, scale: 0.9, y: 20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            transition={{ duration: 0.5, ease: "easeOut" }}
            className="w-full max-w-2xl"
          >
            <Card className="border-red-200 dark:border-red-800 shadow-2xl">
              <CardHeader className="text-center pb-4">
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ delay: 0.2, type: "spring", stiffness: 300, damping: 20 }}
                  className="mx-auto w-16 h-16 bg-red-100 dark:bg-red-900 rounded-full flex items-center justify-center mb-4"
                >
                  <AlertTriangle className="w-8 h-8 text-red-600 dark:text-red-400" />
                </motion.div>
                
                <CardTitle className="text-2xl text-red-900 dark:text-red-100 mb-2">
                  Oops! Something went wrong
                </CardTitle>
                
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.3 }}
                  className="text-red-600 dark:text-red-300"
                >
                  The application encountered an unexpected error. Don't worry, your data is safe.
                </motion.p>
              </CardHeader>

              <CardContent className="space-y-6">
                {/* Error Message */}
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.4 }}
                >
                  <h3 className="font-semibold text-sm text-muted-foreground mb-2 flex items-center gap-2">
                    <Bug className="w-4 h-4" />
                    Error Details
                  </h3>
                  <div className="bg-red-50 dark:bg-red-950/50 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <p className="text-sm font-mono text-red-800 dark:text-red-200 break-all">
                      {this.state.error?.name}: {this.state.error?.message}
                    </p>
                  </div>
                </motion.div>

                {/* Stack Trace (collapsible) */}
                {this.state.error?.stack && (
                  <motion.details
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.5 }}
                    className="group"
                  >
                    <summary className="cursor-pointer text-sm font-semibold text-muted-foreground mb-2 hover:text-foreground transition-colors">
                      📋 Stack Trace (click to expand)
                    </summary>
                    <div className="bg-muted rounded-lg p-4 mt-2 max-h-40 overflow-auto">
                      <pre className="text-xs font-mono text-muted-foreground whitespace-pre-wrap break-all">
                        {this.state.error.stack}
                      </pre>
                    </div>
                  </motion.details>
                )}

                {/* Action Buttons */}
                <motion.div
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.6 }}
                  className="flex flex-col sm:flex-row gap-3 pt-4 border-t"
                >
                  <Button
                    onClick={this.handleReset}
                    className="flex-1 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800"
                  >
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Try Again
                  </Button>

                  <Button
                    variant="outline"
                    onClick={this.handleCopyError}
                    className={cn(
                      "flex-1 transition-all duration-200",
                      this.state.copied && "border-green-500 text-green-600 bg-green-50 dark:bg-green-950"
                    )}
                  >
                    {this.state.copied ? (
                      <>
                        <CheckCircle2 className="w-4 h-4 mr-2" />
                        Copied!
                      </>
                    ) : (
                      <>
                        <Copy className="w-4 h-4 mr-2" />
                        Copy Error Report
                      </>
                    )}
                  </Button>

                  <Button
                    variant="outline"
                    onClick={() => window.location.reload()}
                    className="flex-1"
                  >
                    <RefreshCw className="w-4 h-4 mr-2" />
                    Refresh Page
                  </Button>
                </motion.div>

                {/* Help Text */}
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.7 }}
                  className="text-center"
                >
                  <p className="text-xs text-muted-foreground">
                    If the problem persists, try refreshing the page or contact support with the error report above.
                  </p>
                </motion.div>
              </CardContent>
            </Card>
          </motion.div>
        </div>
      )
    }

    return this.props.children
  }
}

// Hook version for functional components
export function useErrorHandler() {
  return (error: Error, errorInfo?: React.ErrorInfo) => {
    console.error('🚨 Error caught by hook:', error)
    if (errorInfo) {
      console.error('📍 Error info:', errorInfo)
    }
    
    // You could trigger a state update here to show an error state
    // or integrate with your global error handling system
  }
}

// Higher-order component for wrapping components with error boundaries
export function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>,
  fallback?: React.ComponentType<{ error: Error; resetError: () => void }>
) {
  const WrappedComponent = (props: P) => (
    <ErrorBoundary fallback={fallback}>
      <Component {...props} />
    </ErrorBoundary>
  )
  
  WrappedComponent.displayName = `withErrorBoundary(${Component.displayName || Component.name})`
  
  return WrappedComponent
}