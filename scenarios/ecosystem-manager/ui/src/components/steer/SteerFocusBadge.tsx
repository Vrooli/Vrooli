import { cn } from '@/lib/utils';
import type { AutoSteerProfile, ExecutionHistory } from '@/types/api';
import { Compass, ListOrdered, Zap } from 'lucide-react';

export interface SteerFocusBadgeProps {
  autoSteerProfileName?: string;
  phaseMode?: string;
  phaseTooltip?: string;
  manualSteerMode?: string;
  queueMode?: string;
  queueIndex?: number;
  queueTotal?: number;
  queueExhausted?: boolean;
  className?: string;
}

export interface SteerFocusInfo {
  autoSteerProfileName?: string;
  phaseMode?: string;
  phaseTooltip?: string;
  manualSteerMode?: string;
  queueMode?: string;
  queueIndex?: number;
  queueTotal?: number;
  queueExhausted?: boolean;
}

export function getExecutionSteerFocus(
  execution: ExecutionHistory,
  profilesById: Record<string, AutoSteerProfile | undefined> = {},
): SteerFocusInfo {
  const manualSteerMode = execution.steer_mode ? execution.steer_mode.toUpperCase() : undefined;
  const profileId = execution.auto_steer_profile_id;
  const profile = profileId ? profilesById[profileId] : undefined;
  const autoSteerProfileName = profileId ? profile?.name ?? profileId : undefined;

  if (autoSteerProfileName) {
    const phaseIndex =
      typeof execution.steer_phase_index === 'number' ? execution.steer_phase_index : undefined;
    const phaseMode =
      typeof phaseIndex === 'number' && profile?.phases?.[phaseIndex]
        ? profile.phases[phaseIndex].mode
        : undefined;
    const phaseTooltip =
      phaseMode && typeof phaseIndex === 'number'
        ? `Phase ${phaseIndex + 1}: ${phaseMode}`
        : phaseMode ?? autoSteerProfileName;

    return {
      autoSteerProfileName,
      phaseMode,
      phaseTooltip,
    };
  }

  if (manualSteerMode) {
    return { manualSteerMode };
  }

  return {};
}

export function SteerFocusBadge({
  autoSteerProfileName,
  phaseMode,
  phaseTooltip,
  manualSteerMode,
  queueMode,
  queueIndex,
  queueTotal,
  queueExhausted,
  className,
}: SteerFocusBadgeProps) {
  if (!autoSteerProfileName && !manualSteerMode && queueMode === undefined && queueIndex === undefined) {
    return null;
  }

  // Auto Steer Profile badge (indigo)
  if (autoSteerProfileName) {
    return (
      <div
        title={phaseTooltip}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 rounded bg-indigo-100 text-indigo-900 border border-indigo-200 dark:bg-indigo-500/10 dark:text-indigo-100 dark:border-indigo-500/30',
          className,
        )}
      >
        <Zap className="h-3.5 w-3.5" />
        <div className="flex flex-col leading-tight">
          <span className="text-xs font-semibold flex items-center gap-1">
            <span>{autoSteerProfileName}</span>
            {phaseMode && (
              <>
                <span className="text-[10px] text-indigo-800/70 dark:text-indigo-100/70">•</span>
                <span className="font-normal text-[11px] text-indigo-800/90 dark:text-indigo-100/90 uppercase">
                  {phaseMode}
                </span>
              </>
            )}
          </span>
        </div>
      </div>
    );
  }

  // Queue steering badge (cyan)
  if (queueMode !== undefined || queueIndex !== undefined) {
    const position = queueIndex !== undefined && queueTotal !== undefined
      ? `${queueIndex + 1}/${queueTotal}`
      : undefined;
    const displayMode = queueMode?.toUpperCase?.() ?? queueMode;
    const tooltip = queueExhausted
      ? 'Queue exhausted'
      : position
        ? `Queue position ${position}: ${displayMode || 'N/A'}`
        : displayMode;

    return (
      <div
        title={tooltip}
        className={cn(
          'flex items-center gap-1.5 px-2 py-1 rounded bg-cyan-100 text-cyan-900 border border-cyan-200 dark:bg-cyan-500/10 dark:text-cyan-100 dark:border-cyan-500/30',
          queueExhausted && 'opacity-60',
          className,
        )}
      >
        <ListOrdered className="h-3.5 w-3.5" />
        <div className="flex flex-col leading-tight">
          <span className="text-xs font-semibold flex items-center gap-1">
            {position && (
              <>
                <span className="font-mono text-[10px] text-cyan-700 dark:text-cyan-200/70">{position}</span>
                <span className="text-[10px] text-cyan-800/70 dark:text-cyan-100/70">•</span>
              </>
            )}
            <span className={cn('font-normal text-[11px] uppercase', queueExhausted && 'line-through')}>
              {queueExhausted ? 'Done' : displayMode || 'Queue'}
            </span>
          </span>
        </div>
      </div>
    );
  }

  // Manual steering badge (amber)
  const manualLabel = manualSteerMode?.toUpperCase?.() ?? manualSteerMode;
  if (!manualLabel) {
    return null;
  }

  return (
    <div
      className={cn(
        'flex items-center gap-1.5 px-2 py-1 rounded bg-amber-100 text-amber-900 border border-amber-200 dark:bg-amber-500/10 dark:text-amber-50 dark:border-amber-500/30',
        className,
      )}
    >
      <Compass className="h-3.5 w-3.5" />
      <div className="leading-tight text-xs font-semibold">{manualLabel}</div>
    </div>
  );
}
