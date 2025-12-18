/**
 * ReplayEmptyState component
 *
 * Renders a placeholder when there are no frames to display.
 */

import clsx from 'clsx';
import type { BackgroundDecor } from '../types';

export interface ReplayEmptyStateProps {
  backgroundDecor: BackgroundDecor;
}

export function ReplayEmptyState({ backgroundDecor }: ReplayEmptyStateProps) {
  return (
    <div
      className={clsx(
        'relative flex h-64 items-center justify-center overflow-hidden rounded-3xl text-sm text-slate-200/80 transition-all duration-500',
        backgroundDecor.containerClass,
      )}
      style={backgroundDecor.containerStyle}
    >
      {backgroundDecor.baseLayer}
      {backgroundDecor.overlay}
      <div
        className={clsx(
          'relative z-[1] flex h-full w-full items-center justify-center text-center',
          backgroundDecor.contentClass,
        )}
      >
        <span className="max-w-sm text-xs text-slate-200/80 sm:text-sm">
          Replay timeline will appear once executions capture timeline artifacts.
        </span>
      </div>
    </div>
  );
}
