import { cn } from '@/lib/utils';
import type { AutoSteerProfile, ExecutionHistory } from '@/types/api';
import { Compass, Zap } from 'lucide-react';

export interface SteerFocusBadgeProps {
  autoSteerProfileName?: string;
  phaseMode?: string;
  phaseTooltip?: string;
  manualSteerMode?: string;
  className?: string;
}

export interface SteerFocusInfo {
  autoSteerProfileName?: string;
  phaseMode?: string;
  phaseTooltip?: string;
  manualSteerMode?: string;
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
  className,
}: SteerFocusBadgeProps) {
  if (!autoSteerProfileName && !manualSteerMode) {
    return null;
  }

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
