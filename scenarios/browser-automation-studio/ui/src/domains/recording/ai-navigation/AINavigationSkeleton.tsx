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
 * Skeleton for the AI Navigation Panel.
 */
export function AINavigationPanelSkeleton() {
  return (
    <div className="flex flex-col h-full bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          <Skeleton className="w-5 h-5 rounded-full" />
          <Skeleton className="w-24 h-4" />
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 p-4 space-y-4">
        {/* Prompt Input */}
        <div>
          <Skeleton className="w-40 h-4 mb-2" />
          <Skeleton className="w-full h-20 rounded-lg" />
          <Skeleton className="w-32 h-3 mt-1" />
        </div>

        {/* Model Selector */}
        <div>
          <Skeleton className="w-24 h-4 mb-2" />
          <div className="space-y-2">
            <Skeleton className="w-full h-14 rounded-lg" />
            <Skeleton className="w-full h-14 rounded-lg" />
            <Skeleton className="w-full h-14 rounded-lg" />
          </div>
          <Skeleton className="w-20 h-3 mt-2" />
        </div>

        {/* Max Steps */}
        <div>
          <div className="flex justify-between mb-2">
            <Skeleton className="w-20 h-4" />
            <Skeleton className="w-16 h-4" />
          </div>
          <Skeleton className="w-full h-2 rounded-lg" />
        </div>

        {/* Cost Estimate */}
        <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div className="flex justify-between">
            <Skeleton className="w-24 h-4" />
            <Skeleton className="w-12 h-4" />
          </div>
          <Skeleton className="w-48 h-3 mt-2" />
        </div>
      </div>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-gray-200 dark:border-gray-700">
        <Skeleton className="w-full h-10 rounded-lg" />
      </div>
    </div>
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
 * Full AI Navigation view skeleton.
 */
export function AINavigationViewSkeleton() {
  return (
    <div className="flex h-full">
      {/* Left: Browser Preview */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Browser URL Bar */}
        <div className="border-b border-gray-200 dark:border-gray-700 px-4 py-2.5">
          <div className="flex items-center gap-2">
            <Skeleton className="w-8 h-8 rounded" />
            <Skeleton className="w-8 h-8 rounded" />
            <Skeleton className="flex-1 h-8 rounded-lg" />
          </div>
        </div>

        {/* Browser View */}
        <div className="flex-1 overflow-hidden bg-gray-100 dark:bg-gray-800">
          <div className="h-full flex items-center justify-center">
            <div className="animate-pulse flex flex-col items-center">
              <Skeleton className="w-16 h-16 rounded-lg mb-4" />
              <Skeleton className="w-32 h-4 mb-2" />
              <Skeleton className="w-48 h-3" />
            </div>
          </div>
        </div>
      </div>

      {/* Right: AI Navigation Panel & Steps */}
      <div className="w-96 flex flex-col border-l border-gray-200 dark:border-gray-700">
        <div className="flex-shrink-0" style={{ maxHeight: '60%' }}>
          <AINavigationPanelSkeleton />
        </div>

        <div className="flex-1 min-h-0 overflow-hidden flex flex-col border-t border-gray-200 dark:border-gray-700">
          <div className="px-4 py-2 border-b border-gray-200 dark:border-gray-700">
            <Skeleton className="w-16 h-4" />
          </div>
          <div className="flex-1 overflow-y-auto">
            <AINavigationTimelineSkeleton count={3} />
          </div>
        </div>
      </div>
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
