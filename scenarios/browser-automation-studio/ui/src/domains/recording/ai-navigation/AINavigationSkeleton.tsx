/**
 * AINavigationSkeleton Component
 *
 * Skeleton loading states for AI navigation components.
 * Provides visual feedback while content is loading.
 */

interface SkeletonProps {
  className?: string;
}

/** Base skeleton element with pulse animation */
function Skeleton({ className = '' }: SkeletonProps) {
  return (
    <div
      className={`animate-pulse bg-gray-200 dark:bg-gray-700 rounded ${className}`}
    />
  );
}

/**
 * Skeleton for a single AI navigation step.
 */
export function AINavigationStepSkeleton() {
  return (
    <div className="py-2 px-3 border-b border-gray-200 dark:border-gray-700">
      <div className="flex items-center gap-3">
        <Skeleton className="w-6 h-6 rounded" />
        <Skeleton className="w-4 h-4 rounded" />
        <Skeleton className="flex-1 h-4" />
        <Skeleton className="w-6 h-4 rounded" />
        <Skeleton className="w-12 h-4" />
        <Skeleton className="w-4 h-4" />
      </div>
    </div>
  );
}

/**
 * Skeleton for the AI navigation timeline.
 */
export function AINavigationTimelineSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="divide-y divide-gray-200 dark:divide-gray-700">
      {Array.from({ length: count }).map((_, i) => (
        <AINavigationStepSkeleton key={i} />
      ))}
    </div>
  );
}

/**
 * Loading indicator for when AI is processing.
 */
export function AIProcessingIndicator({ message = 'AI is processing...' }: { message?: string }) {
  return (
    <div className="flex items-center gap-3 p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg border border-purple-200 dark:border-purple-800">
      <div className="relative">
        <div className="w-8 h-8 border-2 border-purple-200 dark:border-purple-800 rounded-full" />
        <div className="absolute top-0 left-0 w-8 h-8 border-2 border-transparent border-t-purple-500 rounded-full animate-spin" />
      </div>
      <div>
        <p className="text-sm font-medium text-purple-700 dark:text-purple-300">{message}</p>
        <p className="text-xs text-purple-500 dark:text-purple-400">This may take a moment...</p>
      </div>
    </div>
  );
}

/**
 * Waiting for WebSocket connection indicator.
 */
export function WebSocketConnectingIndicator() {
  return (
    <div className="flex items-center gap-2 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
      <div className="flex space-x-1">
        <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
        <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
        <div className="w-2 h-2 bg-blue-500 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
      </div>
      <span className="text-sm text-blue-700 dark:text-blue-300">
        Connecting to server...
      </span>
    </div>
  );
}
