import type { ReactNode, Ref } from 'react';

interface ReplayCanvasProps {
  aspectRatio: number;
  zoom: number;
  anchorStyle: string;
  screenshotRef?: Ref<HTMLDivElement>;
  screenshotUrl?: string;
  screenshotAlt?: string;
  transition?: string;
  emptyState?: ReactNode;
  children?: ReactNode;
}

export function ReplayCanvas({
  aspectRatio,
  zoom,
  anchorStyle,
  screenshotRef,
  screenshotUrl,
  screenshotAlt,
  transition,
  emptyState,
  children,
}: ReplayCanvasProps) {
  return (
    <div className="relative" style={{ paddingTop: `${aspectRatio}%` }}>
      <div
        ref={screenshotRef}
        className="absolute inset-0"
        style={{ transform: `scale(${zoom})`, transformOrigin: anchorStyle, transition }}
      >
        {screenshotUrl ? (
          <img
            src={screenshotUrl}
            alt={screenshotAlt ?? 'Replay screenshot'}
            loading="lazy"
            className="h-full w-full object-cover"
          />
        ) : (
          emptyState ?? (
            <div className="flex h-full w-full items-center justify-center bg-slate-900 text-slate-500">
              Screenshot unavailable
            </div>
          )
        )}
        <div className="absolute inset-0">{children}</div>
      </div>
    </div>
  );
}

export default ReplayCanvas;
