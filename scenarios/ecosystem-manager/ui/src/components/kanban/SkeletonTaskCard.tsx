/**
 * SkeletonTaskCard Component
 * Loading placeholder for task cards
 */

import { Skeleton } from '../ui/skeleton';

export function SkeletonTaskCard() {
  return (
    <div className="bg-slate-800 border border-white/10 rounded-lg p-3 space-y-3">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="space-y-2 flex-1">
          <Skeleton className="h-4 w-20" />
          <div className="flex gap-2">
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-5 w-20" />
          </div>
        </div>
        <Skeleton className="h-2 w-2 rounded-full" />
      </div>

      {/* Body */}
      <div className="space-y-2">
        <Skeleton className="h-5 w-full" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-1/2" />
      </div>

      {/* Footer */}
      <div className="flex items-center justify-between pt-2 border-t border-white/5">
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-7 w-7 rounded" />
      </div>
    </div>
  );
}
