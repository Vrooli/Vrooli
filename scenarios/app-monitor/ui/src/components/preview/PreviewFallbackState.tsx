import clsx from 'clsx';
import { AlertCircle, Loader2 } from 'lucide-react';
import type { PreviewFallbackState } from '@/hooks/usePreviewOverlay';
import './PreviewFallbackState.css';

interface PreviewFallbackStateProps {
  state: PreviewFallbackState;
  variant: 'overlay' | 'panel';
  className?: string;
}

export default function PreviewFallbackState({ state, variant, className }: PreviewFallbackStateProps) {
  const isError = state.type === 'error';

  return (
    <div
      className={clsx(
        'preview-fallback',
        `preview-fallback--${variant}`,
        `preview-fallback--${state.type}`,
        state.showSkeleton && 'preview-fallback--with-skeleton',
        className,
      )}
      role="status"
      aria-live="polite"
      aria-busy={state.isBlocking ? true : undefined}
    >
      {state.showSkeleton && (
        <div className="preview-fallback__skeleton" aria-hidden>
          <div className="preview-fallback__pulse preview-fallback__pulse--title" />
          <div className="preview-fallback__pulse preview-fallback__pulse--line" />
          <div className="preview-fallback__pulse preview-fallback__pulse--line preview-fallback__pulse--line-short" />
          <div className="preview-fallback__pulse preview-fallback__pulse--surface" />
        </div>
      )}
      <div className="preview-fallback__content">
        {state.showSpinner ? (
          <Loader2 size={18} aria-hidden className="preview-fallback__icon preview-fallback__icon--spin" />
        ) : isError ? (
          <AlertCircle size={18} aria-hidden className="preview-fallback__icon" />
        ) : null}
        <span>{state.message}</span>
      </div>
    </div>
  );
}
