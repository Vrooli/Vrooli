import { cn } from '../../lib/utils'

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {}

/**
 * Skeleton loading component with smooth pulsing animation.
 * Use for content placeholders during async operations.
 */
export function Skeleton({ className, ...props }: SkeletonProps) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-slate-200/70', className)}
      {...props}
    />
  )
}

/**
 * Card skeleton for catalog/list views
 */
export function CardSkeleton({ className, style }: { className?: string; style?: React.CSSProperties }) {
  return (
    <div className={cn('rounded-2xl border border-slate-100 bg-white p-5 sm:p-6', className)} style={style}>
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 space-y-2">
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-2/3" />
          </div>
          <Skeleton className="h-6 w-16 shrink-0" />
        </div>
        <Skeleton className="h-6 w-24" />
      </div>
      <div className="mt-4 border-t border-dashed pt-4">
        <div className="space-y-3">
          <Skeleton className="h-2 w-full rounded-full" />
          <div className="flex gap-2">
            <Skeleton className="h-9 flex-1" />
            <Skeleton className="h-9 flex-1" />
          </div>
        </div>
      </div>
    </div>
  )
}

/**
 * Text skeleton variants
 */
export function TextSkeleton({
  lines = 1,
  className
}: {
  lines?: number
  className?: string
}) {
  return (
    <div className={cn('space-y-2', className)}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={cn(
            'h-4',
            i === lines - 1 && lines > 1 ? 'w-2/3' : 'w-full'
          )}
        />
      ))}
    </div>
  )
}

/**
 * Button skeleton
 */
export function ButtonSkeleton({ className }: { className?: string }) {
  return <Skeleton className={cn('h-10 w-24 rounded-lg', className)} />
}
