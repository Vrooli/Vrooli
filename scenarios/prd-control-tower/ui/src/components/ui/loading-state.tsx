import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/utils'

export interface LoadingStateProps {
  message?: string
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

/**
 * LoadingState component - Provides contextual feedback during async operations
 *
 * Design principles:
 * - Clear visual indicator (spinner)
 * - Contextual message explaining what's loading
 * - Consistent sizing and spacing
 * - Neutral, professional appearance
 */
export function LoadingState({ message = 'Loading...', size = 'md', className }: LoadingStateProps) {
  const sizeClasses = {
    sm: 'h-4 w-4',
    md: 'h-6 w-6',
    lg: 'h-8 w-8',
  }

  const textSize = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base',
  }

  const spacing = {
    sm: 'gap-2',
    md: 'gap-3',
    lg: 'gap-4',
  }

  return (
    <div className={cn(
      'flex items-center justify-center text-slate-600',
      spacing[size],
      className
    )}>
      <Loader2 className={cn(sizeClasses[size], 'animate-spin text-violet-500')} />
      <span className={cn(textSize[size], 'font-medium')}>{message}</span>
    </div>
  )
}

/**
 * FullPageLoadingState - Loading state that fills the entire page
 */
export function FullPageLoadingState({ message = 'Loading...', className }: LoadingStateProps) {
  return (
    <div className={cn(
      'flex min-h-screen flex-col items-center justify-center gap-4 bg-gradient-to-br from-violet-50/30 via-white to-slate-50/30',
      className
    )}>
      <Loader2 className="h-8 w-8 animate-spin text-violet-600" />
      <div className="text-center space-y-2">
        <p className="text-lg font-medium text-slate-700">{message}</p>
        <p className="text-sm text-slate-500">Please wait...</p>
      </div>
    </div>
  )
}

/**
 * InlineLoadingState - Compact loading indicator for inline use
 */
export function InlineLoadingState({ message, className }: Pick<LoadingStateProps, 'message' | 'className'>) {
  return (
    <div className={cn('inline-flex items-center gap-2 text-xs text-slate-600', className)}>
      <Loader2 className="h-3 w-3 animate-spin text-violet-500" />
      {message && <span>{message}</span>}
    </div>
  )
}
