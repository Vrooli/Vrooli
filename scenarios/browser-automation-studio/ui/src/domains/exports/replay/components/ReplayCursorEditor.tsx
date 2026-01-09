/**
 * ReplayCursorEditor component
 *
 * Renders the cursor behavior panel with speed profile and path style
 * dropdowns, allowing users to customize cursor animation per frame.
 */

import type {
  ReplayPoint,
  CursorPlan,
  CursorAnimationOverride,
  CursorSpeedProfile,
  CursorPathStyle,
} from '../types';
import { SPEED_PROFILE_OPTIONS, CURSOR_PATH_STYLE_OPTIONS } from '../constants';
import { formatCoordinate } from '../utils/formatting';
import { toAbsolutePoint, toNormalizedPoint } from '../utils/geometry';

export interface ReplayCursorEditorProps {
  cursorPlan: CursorPlan;
  currentOverride: CursorAnimationOverride | undefined;
  cursorPosition: ReplayPoint | undefined;
  baseSpeedProfile: CursorSpeedProfile;
  basePathStyle: CursorPathStyle;
  onUpdateOverride: (
    frameId: string,
    mutator: (previous: CursorAnimationOverride | undefined) => CursorAnimationOverride | null | undefined,
  ) => void;
  onResetAll: () => void;
}

export function ReplayCursorEditor({
  cursorPlan,
  currentOverride,
  cursorPosition,
  baseSpeedProfile,
  basePathStyle,
  onUpdateOverride,
  onResetAll,
}: ReplayCursorEditorProps) {
  const effectiveSpeedProfile = cursorPlan.speedProfile ?? baseSpeedProfile;
  const currentTargetPoint = toAbsolutePoint(cursorPlan.targetNormalized, cursorPlan.dims);
  const currentCursorNormalized = cursorPosition
    ? toNormalizedPoint(cursorPosition, cursorPlan.dims)
    : undefined;
  const selectedPathStyle = currentOverride?.pathStyle ?? basePathStyle;
  const usingRecordedTrail = cursorPlan.hasRecordedTrail;
  const pathStyleOption = CURSOR_PATH_STYLE_OPTIONS.find((option) => option.id === selectedPathStyle) ?? CURSOR_PATH_STYLE_OPTIONS[0];

  const handleResetSpeedProfile = () => {
    onUpdateOverride(cursorPlan.frameId, (previous) => {
      if (!previous) {
        return undefined;
      }
      const next = { ...previous };
      delete next.speedProfile;
      return next;
    });
  };

  const handleResetPathStyle = () => {
    onUpdateOverride(cursorPlan.frameId, (previous) => {
      if (!previous) {
        return undefined;
      }
      const next = { ...previous };
      delete next.pathStyle;
      return next;
    });
  };

  return (
    <div className="rounded-xl border border-white/5 bg-white/5 p-3 text-xs text-slate-200">
      <div className="flex flex-wrap items-center justify-between gap-2 uppercase tracking-[0.2em] text-slate-400">
        <span>Cursor Behavior</span>
        {currentOverride && (
          <button
            type="button"
            className="rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
            onClick={onResetAll}
          >
            Reset
          </button>
        )}
      </div>
      <div className="mt-2 space-y-1 text-[11px] text-slate-300">
        <div className="flex items-center justify-between">
          <span>Current position</span>
          <span className="font-mono text-slate-200">
            {formatCoordinate(cursorPosition, cursorPlan.dims)}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span>Target placement</span>
          <span className="font-mono text-slate-200">
            {formatCoordinate(currentTargetPoint, cursorPlan.dims)}
          </span>
        </div>
        {currentCursorNormalized && (
          <div className="flex items-center justify-between">
            <span>Live cursor</span>
            <span className="font-mono text-slate-200">
              {`${Math.round(currentCursorNormalized.x * 100)}%, ${Math.round(currentCursorNormalized.y * 100)}%`}
            </span>
          </div>
        )}
        <div className="flex items-center justify-between">
          <span>Mode</span>
          <span className="text-slate-400">
            {usingRecordedTrail ? 'Recorded capture' : pathStyleOption.label}
          </span>
        </div>
      </div>
      <div className="mt-3 space-y-1">
        <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.2em] text-slate-400">
          <label htmlFor="cursor-speed-profile" className="cursor-pointer">
            Speed profile
          </label>
          {currentOverride?.speedProfile && currentOverride.speedProfile !== baseSpeedProfile && (
            <button
              type="button"
              className="rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
              onClick={handleResetSpeedProfile}
            >
              Default
            </button>
          )}
        </div>
        <select
          id="cursor-speed-profile"
          className="w-full rounded-lg border border-slate-700/60 bg-slate-900/70 px-2.5 py-1.5 text-[11px] text-slate-200 focus:border-flow-accent/60 focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
          value={effectiveSpeedProfile}
          onChange={(event) => {
            const selected = event.target.value as CursorSpeedProfile;
            onUpdateOverride(cursorPlan.frameId, (previous) => ({
              ...(previous ?? {}),
              speedProfile: selected,
            }));
          }}
        >
          {SPEED_PROFILE_OPTIONS.map((option) => (
            <option key={option.id} value={option.id}>
              {option.label} — {option.description}
            </option>
          ))}
        </select>
      </div>
      <div className="mt-3 space-y-1">
        <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.2em] text-slate-400">
          <label htmlFor="cursor-path-style" className="cursor-pointer">
            Path shape
          </label>
          {currentOverride?.pathStyle && currentOverride.pathStyle !== basePathStyle && (
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-full border border-slate-600/60 bg-slate-900/60 px-2 py-1 text-[10px] uppercase tracking-[0.2em] text-slate-300 hover:border-flow-accent/40 hover:text-white"
              onClick={handleResetPathStyle}
            >
              Default
            </button>
          )}
        </div>
        <select
          id="cursor-path-style"
          className="w-full rounded-lg border border-slate-700/60 bg-slate-900/70 px-2.5 py-1.5 text-[11px] text-slate-200 focus:border-flow-accent/60 focus:outline-none focus:ring-2 focus:ring-flow-accent/40"
          value={selectedPathStyle}
          onChange={(event) => {
            const selected = event.target.value as CursorPathStyle;
            onUpdateOverride(cursorPlan.frameId, (previous) => ({
              ...(previous ?? {}),
              pathStyle: selected,
            }));
          }}
        >
          {CURSOR_PATH_STYLE_OPTIONS.map((option) => (
            <option key={option.id} value={option.id}>
              {option.label} — {option.description}
            </option>
          ))}
        </select>
        <p className="text-[10px] text-slate-500">
          {usingRecordedTrail
            ? 'Using captured cursor trail from the execution. Choose a shape to override it.'
            : pathStyleOption.description}
        </p>
      </div>
      <p className="mt-3 text-[10px] text-slate-500">
        Tip: drag the cursor directly on the replay canvas to fine-tune the target placement.
      </p>
    </div>
  );
}
