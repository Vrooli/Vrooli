import type { CSSProperties, ReactNode } from 'react';
import clsx from 'clsx';
import type { CursorDecor } from '../catalog';

interface ReplayCursorOverlayProps {
  cursorDecor: CursorDecor;
  trailPoints: Array<{ x: number; y: number }>;
  trailStrokeWidth: number;
  showTrail: boolean;
  overlayBounds?: { width: number; height: number } | null;
  ghostStyle?: CSSProperties;
  pointerStyle?: CSSProperties;
  pointerClassName: string;
  pointerEventProps?: React.HTMLAttributes<HTMLDivElement>;
  clickEffect?: ReactNode;
}

export function ReplayCursorOverlay({
  cursorDecor,
  trailPoints,
  trailStrokeWidth,
  showTrail,
  overlayBounds,
  ghostStyle,
  pointerStyle,
  pointerClassName,
  pointerEventProps,
  clickEffect,
}: ReplayCursorOverlayProps) {
  const overlayWidth = overlayBounds?.width ?? 0;
  const overlayHeight = overlayBounds?.height ?? 0;
  const shouldRenderTrail = showTrail && overlayWidth > 0 && overlayHeight > 0 && trailPoints.length >= 2;

  return (
    <>
      {shouldRenderTrail && (
        <svg
          className="absolute inset-0 h-full w-full"
          viewBox={`0 0 ${overlayWidth} ${overlayHeight}`}
          preserveAspectRatio="none"
        >
          <polyline
            points={trailPoints.map((p) => `${p.x},${p.y}`).join(' ')}
            stroke={cursorDecor.trailColor}
            strokeWidth={trailStrokeWidth}
            strokeLinecap="round"
            strokeLinejoin="round"
            fill="none"
          />
        </svg>
      )}

      {ghostStyle && cursorDecor.renderBase && (
        <div
          role="presentation"
          className={clsx('absolute pointer-events-none select-none transition-all duration-500 ease-out', cursorDecor.wrapperClass)}
          style={ghostStyle}
        >
          {cursorDecor.renderBase}
        </div>
      )}

      {pointerStyle && cursorDecor.renderBase && (
        <div role="presentation" className={pointerClassName} style={pointerStyle} {...pointerEventProps}>
          {clickEffect}
          {cursorDecor.renderBase}
        </div>
      )}
    </>
  );
}

export default ReplayCursorOverlay;
