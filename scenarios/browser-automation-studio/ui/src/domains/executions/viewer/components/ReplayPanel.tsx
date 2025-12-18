import { AlertTriangle } from 'lucide-react';
import type { TimelineFrame } from '@stores/executionStore';
import { selectors } from '@constants/selectors';
import { formatCapturedLabel } from '../../utils/exportHelpers';
import { formatSeconds } from '../useExecutionHeartbeat';
import ReplayCustomizationPanel from '../ReplayCustomizationPanel';
import type { ReplayCustomizationController } from '../useReplayCustomization';

interface ReplayPanelProps {
  executionId: string;
  hasTimeline: boolean;
  isFailed: boolean;
  isCancelled: boolean;
  progress: number;
  currentStep: string | null;
  executionError?: string;
  replayFrames: TimelineFrame[];
  replayCustomization: ReplayCustomizationController;
  composerUrl: string;
  composerRef: React.RefObject<HTMLIFrameElement>;
  composerWindowRef: React.MutableRefObject<Window | null>;
  isMovieSpecLoading: boolean;
  isComposerReady: boolean;
  movieSpecError: string | null;
  previewMetrics: {
    capturedFrames: number;
    assetCount: number;
    totalDurationMs: number;
  };
  activeSpecId: string | null;
  selectedDimensions: { width: number; height: number };
}

export function ReplayPanel({
  executionId,
  hasTimeline,
  isFailed,
  isCancelled,
  progress,
  currentStep,
  executionError,
  replayFrames,
  replayCustomization,
  composerUrl,
  composerRef,
  composerWindowRef,
  isMovieSpecLoading,
  isComposerReady,
  movieSpecError,
  previewMetrics,
  activeSpecId,
  selectedDimensions,
}: ReplayPanelProps) {
  return (
    <div
      className="flex-1 overflow-auto p-3 space-y-3"
      data-testid={selectors.replay.player}
    >
      {!hasTimeline && (
        <div className="rounded-lg border border-dashed border-flow-border/70 bg-flow-node/70 px-4 py-3 text-sm text-flow-text-secondary">
          Replay frames stream in as each action runs. Leave this tab open to tailor
          the final cut in real time.
        </div>
      )}
      {(isFailed || isCancelled) && progress < 100 && (
        <div className="rounded-lg border border-rose-500/30 bg-rose-500/10 p-3 flex items-start gap-3">
          <AlertTriangle size={18} className="text-rose-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 text-sm">
            <div className="font-medium text-rose-200 mb-1">
              Execution {isFailed ? 'Failed' : 'Cancelled'} - Replay Incomplete
            </div>
            <div className="text-rose-100/80">
              This replay shows only {replayFrames.length} of the workflow's steps.
              Execution {isFailed ? 'failed' : 'was cancelled'} at{' '}
              {currentStep || 'an unknown step'}.
            </div>
            {isFailed && executionError && (
              <div className="mt-2 text-xs font-mono text-rose-100/70 bg-rose-950/30 px-2 py-1 rounded">
                {executionError}
              </div>
            )}
          </div>
        </div>
      )}
      <ReplayCustomizationPanel controller={replayCustomization} />
      <div className="relative w-full overflow-hidden rounded-2xl border border-flow-border bg-flow-node/70 shadow-[0_25px_70px_rgba(0,0,0,0.35)]">
        <iframe
          key={executionId}
          ref={(node) => {
            if (composerRef && 'current' in composerRef) {
              (composerRef as React.MutableRefObject<HTMLIFrameElement | null>).current = node;
            }
            composerWindowRef.current = node?.contentWindow ?? null;
          }}
          src={composerUrl}
          title="Replay Composer"
          className="w-full border-0"
          style={{
            aspectRatio: `${selectedDimensions.width} / ${selectedDimensions.height}`,
            minHeight: '360px',
          }}
          allow="clipboard-read; clipboard-write"
        />
        {(isMovieSpecLoading || !isComposerReady) && !movieSpecError && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center bg-black/50">
            <span className="text-xs uppercase tracking-[0.3em] text-flow-text-muted">
              {isMovieSpecLoading ? 'Loading replay spec…' : 'Initialising player…'}
            </span>
          </div>
        )}
        {movieSpecError && (
          <div className="absolute inset-0 flex items-center justify-center bg-black/60 p-6 text-center">
            <div className="rounded-xl border border-flow-border bg-flow-node/80 px-6 py-4 text-sm text-flow-text shadow-[0_15px_45px_rgba(0,0,0,0.4)]">
              <div className="mb-1 font-semibold text-flow-text">
                Failed to load replay spec
              </div>
              <div className="text-xs text-flow-text-secondary">{movieSpecError}</div>
              {(previewMetrics.capturedFrames > 0 || previewMetrics.totalDurationMs > 0) && (
                <div className="mt-3 text-[11px] text-flow-text-muted">
                  {previewMetrics.capturedFrames > 0 && (
                    <span>{formatCapturedLabel(previewMetrics.capturedFrames, 'frame')}</span>
                  )}
                  {previewMetrics.capturedFrames > 0 && previewMetrics.totalDurationMs > 0 && (
                    <span> • </span>
                  )}
                  {previewMetrics.totalDurationMs > 0 && (
                    <span>{formatSeconds(previewMetrics.totalDurationMs / 1000)} recorded</span>
                  )}
                </div>
              )}
              {activeSpecId && (
                <div className="mt-1 text-[10px] uppercase tracking-[0.24em] text-flow-text-muted">
                  Spec {activeSpecId.slice(0, 8)}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
