/**
 * ScreenshotSlideshow - Simple screenshot display component
 *
 * A controlled component that displays a single screenshot from an array.
 * The parent manages which screenshot is displayed via currentIndex prop.
 */

import { useMemo } from 'react';
import { ImageOff } from 'lucide-react';
import clsx from 'clsx';

export interface Screenshot {
  /** URL of the screenshot */
  url: string;
  /** Optional timestamp or label */
  timestamp?: number | string;
  /** Optional step label */
  stepLabel?: string;
}

export interface ScreenshotSlideshowProps {
  /** Array of screenshots to display */
  screenshots: Screenshot[];
  /** Current index to display (0-based) */
  currentIndex: number;
  /** Additional class name for the container */
  className?: string;
  /** Whether to show step info overlay */
  showStepInfo?: boolean;
  /** Object-fit style for the image */
  objectFit?: 'contain' | 'cover' | 'fill';
}

export function ScreenshotSlideshow({
  screenshots,
  currentIndex,
  className,
  showStepInfo = true,
  objectFit = 'contain',
}: ScreenshotSlideshowProps) {
  // Get current screenshot with bounds checking
  const currentScreenshot = useMemo(() => {
    if (!screenshots || screenshots.length === 0) return null;
    const index = Math.max(0, Math.min(currentIndex, screenshots.length - 1));
    return screenshots[index];
  }, [screenshots, currentIndex]);

  // Empty state
  if (!currentScreenshot) {
    return (
      <div
        className={clsx(
          'h-full w-full flex flex-col items-center justify-center bg-gray-900 text-gray-400',
          className
        )}
      >
        <ImageOff className="w-12 h-12 mb-3 opacity-50" />
        <p className="text-sm">No screenshots available</p>
      </div>
    );
  }

  return (
    <div className={clsx('h-full w-full relative bg-gray-900', className)}>
      {/* Screenshot image */}
      <img
        src={currentScreenshot.url}
        alt={currentScreenshot.stepLabel || `Screenshot ${currentIndex + 1}`}
        className={clsx(
          'w-full h-full',
          objectFit === 'contain' && 'object-contain',
          objectFit === 'cover' && 'object-cover',
          objectFit === 'fill' && 'object-fill'
        )}
      />

      {/* Step info overlay (optional) */}
      {showStepInfo && currentScreenshot.stepLabel && (
        <div className="absolute bottom-3 left-3 px-2 py-1 bg-black/60 rounded text-white text-xs">
          {currentScreenshot.stepLabel}
        </div>
      )}
    </div>
  );
}
