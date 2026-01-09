/**
 * ReplayMetadataPanel component
 *
 * Renders frame metadata including step type, status, URL, errors,
 * assertions, extracted data, telemetry, and resiliency information.
 */

import { useMemo } from 'react';
import clsx from 'clsx';
import { ChevronDown } from 'lucide-react';
import type { ReplayFrame } from '../types';
import { formatDurationSeconds, formatValue } from '../utils/formatting';

export interface ReplayMetadataPanelProps {
  frame: ReplayFrame;
  isCollapsed: boolean;
  onToggle: () => void;
  extractedPreview?: string;
  domSnapshotDisplay?: string;
  cursorBehaviorPanel?: React.ReactNode;
}

export function ReplayMetadataPanel({
  frame,
  isCollapsed,
  onToggle,
  extractedPreview,
  domSnapshotDisplay,
  cursorBehaviorPanel,
}: ReplayMetadataPanelProps) {
  const displayDurationMs = frame.totalDurationMs ?? frame.durationMs;
  const retryConfigured =
    typeof frame.retryConfigured === 'number'
      ? frame.retryConfigured > 0
      : frame.retryConfigured === true;

  const hasResiliencyInfo = useMemo(() => {
    return Boolean(
      (displayDurationMs && frame.durationMs && Math.round(displayDurationMs) !== Math.round(frame.durationMs)) ||
        (typeof frame.retryAttempt === 'number' && frame.retryAttempt > 1) ||
        (typeof frame.retryMaxAttempts === 'number' && frame.retryMaxAttempts > 1) ||
        retryConfigured ||
        (typeof frame.retryDelayMs === 'number' && frame.retryDelayMs > 0) ||
        (typeof frame.retryBackoffFactor === 'number' && frame.retryBackoffFactor !== 1 && frame.retryBackoffFactor !== 0) ||
        (frame.retryHistory && frame.retryHistory.length > 0),
    );
  }, [displayDurationMs, frame, retryConfigured]);

  return (
    <div className="rounded-3xl border border-white/10 bg-slate-950/70 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-400">
        <div className="flex flex-wrap items-center gap-3">
          <span className="uppercase tracking-[0.18em] text-slate-500">Frame Metadata</span>
          {frame.stepType && (
            <span className="inline-flex items-center rounded-full border border-blue-500/40 bg-blue-500/10 px-3 py-1 text-[11px] uppercase tracking-[0.2em] text-blue-200">
              {frame.stepType}
            </span>
          )}
          {frame.status && (
            <span
              className={clsx('inline-flex items-center rounded-full px-3 py-1 text-[11px] uppercase tracking-[0.2em]', {
                'border border-emerald-400/40 bg-emerald-500/10 text-emerald-200': frame.status === 'completed',
                'border border-amber-400/40 bg-amber-500/10 text-amber-200': frame.status === 'running',
                'border border-rose-400/40 bg-rose-500/10 text-rose-200': frame.status === 'failed',
              })}
            >
              {frame.status}
            </span>
          )}
        </div>
        <button
          type="button"
          onClick={onToggle}
          className="inline-flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-slate-200 transition-colors hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-flow-accent/60 focus:ring-offset-2 focus:ring-offset-slate-950"
          aria-expanded={!isCollapsed}
          aria-label={isCollapsed ? 'Expand frame metadata' : 'Collapse frame metadata'}
        >
          <ChevronDown
            size={14}
            className={clsx('transition-transform duration-200', {
              '-rotate-180': !isCollapsed,
            })}
          />
        </button>
      </div>

      <div
        className={clsx(
          'grid gap-4 transition-all duration-300 md:grid-cols-2',
          isCollapsed
            ? 'pointer-events-none max-h-0 overflow-hidden opacity-0'
            : 'mt-3 max-h-[1200px] opacity-100',
        )}
      >
        <div className="space-y-2 text-sm text-slate-300">
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.2em] text-slate-500">
            <span>Node</span>
            <span>#{frame.nodeId || frame.stepIndex + 1}</span>
          </div>
          {frame.finalUrl && (
            <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
              {frame.finalUrl}
            </div>
          )}
          {frame.error && (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-xs text-rose-100">
              {frame.error}
            </div>
          )}
          {frame.assertion && (
            <div
              className={clsx(
                'rounded-xl border p-3 text-xs transition-colors',
                frame.assertion.success !== false
                  ? 'border-emerald-400/40 bg-emerald-500/5 text-emerald-100'
                  : 'border-rose-400/40 bg-rose-500/10 text-rose-100',
              )}
            >
              <div className="mb-1 flex items-center justify-between uppercase tracking-[0.2em]">
                <span>Assertion</span>
                <span>
                  {frame.assertion.success === false ? 'Failed' : 'Passed'}
                </span>
              </div>
              <div className="space-y-1 text-[11px] text-slate-200">
                {frame.assertion.selector && (
                  <div>Selector: {frame.assertion.selector}</div>
                )}
                {frame.assertion.mode && (
                  <div>Mode: {frame.assertion.mode}</div>
                )}
                {frame.assertion.expected !== undefined && frame.assertion.expected !== null && (
                  <div>
                    Expected: {formatValue(frame.assertion.expected)}
                  </div>
                )}
                {frame.assertion.actual !== undefined && frame.assertion.actual !== null && (
                  <div>
                    Actual: {formatValue(frame.assertion.actual)}
                  </div>
                )}
                {frame.assertion.message && (
                  <div className="italic text-slate-300">
                    {frame.assertion.message}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        <div className="space-y-2 text-sm text-slate-300">
          {extractedPreview && (
            <div className="rounded-xl border border-emerald-400/30 bg-emerald-500/5 p-3 text-xs text-emerald-100">
              <div className="mb-1 uppercase tracking-[0.2em] text-emerald-300">Extracted Data</div>
              <pre className="max-h-48 overflow-auto whitespace-pre-wrap break-words text-[11px] text-emerald-100/90">
                {extractedPreview}
              </pre>
            </div>
          )}

          {cursorBehaviorPanel}

          {(frame.consoleLogCount || frame.networkEventCount) && (
            <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
              <div className="uppercase tracking-[0.2em] text-slate-400">Telemetry Snapshot</div>
              <div className="mt-2 flex flex-wrap gap-3 text-[11px] text-slate-300">
                {typeof frame.consoleLogCount === 'number' && (
                  <span>{frame.consoleLogCount} console events</span>
                )}
                {typeof frame.networkEventCount === 'number' && (
                  <span>{frame.networkEventCount} network events</span>
                )}
              </div>
            </div>
          )}

          {domSnapshotDisplay && (
            <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
              <div className="uppercase tracking-[0.2em] text-slate-400">DOM Snapshot</div>
              <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-[11px] text-slate-300">
                {domSnapshotDisplay}
              </pre>
            </div>
          )}

          {hasResiliencyInfo && (
            <div className="rounded-xl border border-amber-400/30 bg-amber-500/5 p-3 text-xs text-amber-100">
              <div className="uppercase tracking-[0.2em] text-amber-300">Resiliency</div>
              <div className="mt-2 space-y-1 text-[11px] text-amber-100/90">
                {displayDurationMs && (
                  <div>Total duration: {formatDurationSeconds(displayDurationMs)}</div>
                )}
                {frame.durationMs && displayDurationMs && Math.round(displayDurationMs) !== Math.round(frame.durationMs) && (
                  <div>Final attempt: {formatDurationSeconds(frame.durationMs)}</div>
                )}
                {typeof frame.retryAttempt === 'number' && frame.retryAttempt > 0 && (
                  <div>
                    Attempt: {frame.retryAttempt}
                    {typeof frame.retryMaxAttempts === 'number' && frame.retryMaxAttempts > 0
                      ? ` / ${frame.retryMaxAttempts}`
                      : ''}
                  </div>
                )}
                {retryConfigured && (
                  <div>
                    Configured retries
                    {typeof frame.retryConfigured === 'number' ? `: ${frame.retryConfigured}` : ''}
                  </div>
                )}
                {typeof frame.retryDelayMs === 'number' && frame.retryDelayMs > 0 && (
                  <div>Initial delay: {formatDurationSeconds(frame.retryDelayMs)}</div>
                )}
                {typeof frame.retryBackoffFactor === 'number' && frame.retryBackoffFactor !== 1 && frame.retryBackoffFactor !== 0 && (
                  <div>Backoff factor: ×{frame.retryBackoffFactor.toFixed(2)}</div>
                )}
                {frame.retryHistory && frame.retryHistory.length > 0 && (
                  <div className="mt-2 space-y-1">
                    {frame.retryHistory.map((entry, index) => (
                      <div
                        key={`${entry.attempt ?? index}`}
                        className={clsx('flex items-center justify-between rounded-lg border px-2 py-1', {
                          'border-emerald-400/20 bg-emerald-400/10 text-emerald-100': entry.success,
                          'border-rose-400/30 bg-rose-500/10 text-rose-100': entry.success === false,
                          'border-white/10 bg-white/5 text-slate-200': entry.success === undefined,
                        })}
                      >
                        <span>
                          Attempt {entry.attempt ?? index + 1}
                          {typeof entry.durationMs === 'number'
                            ? ` • ${formatDurationSeconds(entry.durationMs)}`
                            : ''}
                          {typeof entry.callDurationMs === 'number' && entry.callDurationMs !== entry.durationMs
                            ? ` (call ${formatDurationSeconds(entry.callDurationMs)})`
                            : ''}
                        </span>
                        <span className="ml-3 text-right text-[10px] uppercase tracking-[0.2em]">
                          {entry.success === false ? 'Failed' : entry.success ? 'Passed' : 'Result'}
                        </span>
                        {entry.error && (
                          <span className="ml-3 truncate text-[10px] text-amber-200/90" title={entry.error}>
                            {entry.error}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
