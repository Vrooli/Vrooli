/**
 * PresentationWrapper - Conditionally wraps content in ReplayPresentation
 *
 * When showReplayStyle is true, wraps children in ReplayPresentation with
 * watermark overlay and proper layout. Otherwise renders children directly.
 *
 * This component is used by both RecordPreviewPanel and ExecutionPreviewPanel
 * to provide consistent presentation styling across recording and execution modes.
 */

import type { Ref, ReactNode } from 'react';
import clsx from 'clsx';
import ReplayPresentation from '@/domains/exports/replay/ReplayPresentation';
import { WatermarkOverlay } from '@/domains/exports/replay/WatermarkOverlay';
import type { ReplayPresentationModel } from '@/domains/exports/replay/useReplayPresentationModel';
import type { ReplayRect } from '@/domains/replay-layout';
import type { WatermarkSettings } from '@/domains/exports/replay/types';

export interface PresentationWrapperProps {
  /** Whether to show the replay style presentation */
  showReplayStyle: boolean;
  /** Children to render inside the wrapper */
  children: ReactNode;
  /** Presentation model for ReplayPresentation (required when showReplayStyle is true) */
  presentationModel?: ReplayPresentationModel;
  /** Content rectangle for coordinate mapping */
  previewContentRect?: ReplayRect | null;
  /** Ref to the viewport element */
  viewportRef?: Ref<HTMLDivElement>;
  /** Watermark settings (rendered as overlay when present) */
  watermark?: WatermarkSettings | null;
  /** Additional class name for the outer container */
  className?: string;
}

export function PresentationWrapper({
  showReplayStyle,
  children,
  presentationModel,
  previewContentRect,
  viewportRef,
  watermark,
  className,
}: PresentationWrapperProps) {
  const previewLayout = showReplayStyle && presentationModel ? presentationModel.layout : null;

  return (
    <div
      className={clsx(
        'h-full w-full flex',
        showReplayStyle ? 'bg-slate-950/95' : 'bg-transparent',
        showReplayStyle ? 'items-center justify-center p-4' : 'items-stretch justify-stretch',
        className,
      )}
    >
      <div
        style={
          showReplayStyle && previewLayout
            ? {
                width: previewLayout.display.width + previewLayout.contentInset.x * 2,
                height: previewLayout.display.height + previewLayout.contentInset.y * 2,
              }
            : { width: '100%', height: '100%' }
        }
        className={clsx('relative', !showReplayStyle && 'h-full w-full')}
      >
        {showReplayStyle && previewLayout && presentationModel ? (
          <div data-theme="dark" className="relative h-full w-full">
            <ReplayPresentation
              model={presentationModel}
              viewportContentRect={previewContentRect ?? undefined}
              viewportRef={viewportRef}
              overlayNode={watermark ? <WatermarkOverlay settings={watermark} /> : null}
              containerClassName="h-full w-full"
              contentClassName="h-full w-full"
            >
              <div className="flex h-full w-full items-center justify-center">
                {children}
              </div>
            </ReplayPresentation>
          </div>
        ) : (
          <div className="h-full w-full">
            {children}
          </div>
        )}
      </div>
    </div>
  );
}
