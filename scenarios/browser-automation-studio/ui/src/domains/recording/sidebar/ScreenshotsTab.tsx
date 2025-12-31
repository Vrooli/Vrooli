/**
 * ScreenshotsTab Component
 *
 * Displays execution screenshots in the sidebar during execution mode.
 * Shows a thumbnail strip with the selected screenshot preview.
 */

import { useEffect, useRef } from 'react';
import { format } from 'date-fns';
import { Image, Loader2 } from 'lucide-react';
import clsx from 'clsx';
import type { Screenshot } from '@/domains/executions/store';
import type { Execution } from '@/domains/executions';

interface ScreenshotsTabProps {
  /** Array of captured screenshots */
  screenshots: Screenshot[];
  /** Index of currently selected screenshot */
  selectedIndex?: number;
  /** Callback when a screenshot is selected */
  onSelectScreenshot?: (index: number) => void;
  /** Current execution status */
  executionStatus?: Execution['status'];
  /** Additional CSS classes */
  className?: string;
}

export function ScreenshotsTab({
  screenshots,
  selectedIndex = 0,
  onSelectScreenshot,
  executionStatus,
  className,
}: ScreenshotsTabProps) {
  const thumbnailContainerRef = useRef<HTMLDivElement>(null);
  const selectedThumbnailRef = useRef<HTMLButtonElement>(null);

  // Auto-scroll to selected thumbnail
  useEffect(() => {
    if (selectedThumbnailRef.current && thumbnailContainerRef.current) {
      selectedThumbnailRef.current.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
        inline: 'center',
      });
    }
  }, [selectedIndex]);

  const selectedScreenshot = screenshots[selectedIndex];
  const isLoading = executionStatus === 'pending' || executionStatus === 'running';
  const isFailed = executionStatus === 'failed';
  const isCancelled = executionStatus === 'cancelled';

  // Empty state
  if (screenshots.length === 0) {
    return (
      <div className={clsx('flex flex-col h-full', className)}>
        <div className="flex-1 flex items-center justify-center p-6 text-center">
          <div>
            {isLoading ? (
              <>
                <Loader2 size={32} className="mx-auto mb-3 text-flow-accent animate-spin" />
                <div className="text-sm text-gray-400 mb-1">
                  Waiting for screenshots...
                </div>
                <div className="text-xs text-gray-500">
                  Screenshots will appear as the workflow executes
                </div>
              </>
            ) : (
              <>
                <Image size={32} className="mx-auto mb-3 text-gray-600" />
                <div className="text-sm text-gray-400 mb-1">No screenshots captured</div>
                {isFailed && (
                  <div className="text-xs text-gray-500">
                    Execution failed before screenshots could be captured
                  </div>
                )}
                {isCancelled && (
                  <div className="text-xs text-gray-500">
                    Execution was cancelled before screenshots could be captured
                  </div>
                )}
                {executionStatus === 'completed' && (
                  <div className="text-xs text-gray-500">
                    This workflow did not capture any screenshots
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={clsx('flex flex-col h-full', className)}>
      {/* Main preview area */}
      <div className="flex-1 p-3 overflow-auto">
        {selectedScreenshot && (
          <div className="rounded-xl border border-gray-700 overflow-hidden bg-gray-900">
            {/* Screenshot header */}
            <div className="bg-gray-800/80 px-3 py-2 flex items-center justify-between text-xs border-b border-gray-700">
              <span className="truncate font-medium text-gray-300">
                {selectedScreenshot.stepName}
              </span>
              <span className="text-gray-500 flex-shrink-0 ml-2">
                {format(selectedScreenshot.timestamp, 'HH:mm:ss.SSS')}
              </span>
            </div>

            {/* Screenshot image */}
            <img
              src={selectedScreenshot.url}
              alt={selectedScreenshot.stepName}
              className="block w-full"
              loading="lazy"
            />
          </div>
        )}
      </div>

      {/* Thumbnail strip */}
      {screenshots.length > 1 && (
        <div className="border-t border-gray-700 p-2">
          <div
            ref={thumbnailContainerRef}
            className="flex gap-2 overflow-x-auto pb-1"
          >
            {screenshots.map((screenshot, index) => {
              const isSelected = index === selectedIndex;
              return (
                <button
                  key={screenshot.id}
                  ref={isSelected ? selectedThumbnailRef : undefined}
                  type="button"
                  onClick={() => onSelectScreenshot?.(index)}
                  className={clsx(
                    'flex-shrink-0 w-16 h-12 rounded-lg overflow-hidden border-2 transition-all duration-150',
                    isSelected
                      ? 'border-flow-accent shadow-lg shadow-flow-accent/20'
                      : 'border-gray-700 hover:border-gray-600'
                  )}
                  title={`${screenshot.stepName} - ${format(screenshot.timestamp, 'HH:mm:ss')}`}
                >
                  <img
                    src={screenshot.url}
                    alt={screenshot.stepName}
                    loading="lazy"
                    className="w-full h-full object-cover"
                  />
                </button>
              );
            })}
          </div>

          {/* Counter */}
          <div className="text-xs text-gray-500 text-center mt-1">
            {selectedIndex + 1} of {screenshots.length}
          </div>
        </div>
      )}
    </div>
  );
}
