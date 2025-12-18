import { useRef, useEffect } from 'react';
import { format } from 'date-fns';
import { Image } from 'lucide-react';
import clsx from 'clsx';
import type { Screenshot } from '@stores/executionEventProcessor';
import { selectors } from '@constants/selectors';

interface ScreenshotsPanelProps {
  screenshots: Screenshot[];
  selectedScreenshot: Screenshot | null;
  onSelectScreenshot: (screenshot: Screenshot) => void;
  executionStatus: string;
  isFailed: boolean;
  isCancelled: boolean;
}

export function ScreenshotsPanel({
  screenshots,
  selectedScreenshot,
  onSelectScreenshot,
  executionStatus,
  isFailed,
  isCancelled,
}: ScreenshotsPanelProps) {
  const screenshotRefs = useRef<Record<string, HTMLDivElement | null>>({});

  useEffect(() => {
    if (!selectedScreenshot) return;
    const element = screenshotRefs.current[selectedScreenshot.id];
    if (element) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'start',
        inline: 'nearest',
      });
    }
  }, [selectedScreenshot]);

  if (screenshots.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center p-6 text-center">
        <div>
          <Image size={32} className="mx-auto mb-3 text-gray-600" />
          <div className="text-sm text-gray-400 mb-1">No screenshots captured</div>
          {isFailed && (
            <div className="text-xs text-gray-500">
              Execution failed before screenshot steps could run
            </div>
          )}
          {isCancelled && (
            <div className="text-xs text-gray-500">
              Execution was cancelled before screenshot steps could run
            </div>
          )}
          {executionStatus === 'completed' && (
            <div className="text-xs text-gray-500">
              This workflow does not include screenshot steps
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="flex-1 p-3 overflow-auto">
        <div className="space-y-4" data-testid={selectors.executions.viewer.screenshots}>
          {screenshots.map((screenshot) => (
            <div
              key={screenshot.id}
              ref={(node) => {
                if (node) {
                  screenshotRefs.current[screenshot.id] = node;
                } else {
                  delete screenshotRefs.current[screenshot.id];
                }
              }}
              onClick={() => onSelectScreenshot(screenshot)}
              className={clsx(
                'cursor-pointer overflow-hidden rounded-xl border transition-all duration-200',
                selectedScreenshot?.id === screenshot.id
                  ? 'border-flow-accent/80 shadow-[0_22px_50px_rgba(59,130,246,0.35)]'
                  : 'border-gray-800 hover:border-flow-accent/50 hover:shadow-[0_15px_40px_rgba(59,130,246,0.2)]'
              )}
              data-testid={selectors.timeline.frame}
            >
              <div className="bg-flow-node/80 px-3 py-2 flex items-center justify-between text-xs text-flow-text-secondary">
                <span className="truncate font-medium">{screenshot.stepName}</span>
                <span className="text-flow-text-muted">
                  {format(screenshot.timestamp, 'HH:mm:ss.SSS')}
                </span>
              </div>
              <img
                src={screenshot.url}
                alt={screenshot.stepName}
                loading="lazy"
                className="block w-full"
                data-testid={selectors.executions.viewer.screenshot}
              />
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-gray-800 p-2 overflow-x-auto">
        <div className="flex gap-2">
          {screenshots.map((screenshot) => (
            <div
              key={screenshot.id}
              onClick={() => onSelectScreenshot(screenshot)}
              className={clsx(
                'flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden cursor-pointer border-2 transition-all duration-150',
                selectedScreenshot?.id === screenshot.id
                  ? 'border-flow-accent shadow-[0_12px_30px_rgba(59,130,246,0.35)]'
                  : 'border-gray-700 hover:border-flow-accent/60'
              )}
            >
              <img
                src={screenshot.url}
                alt={screenshot.stepName}
                loading="lazy"
                className="w-full h-full object-cover"
              />
            </div>
          ))}
        </div>
      </div>
    </>
  );
}
