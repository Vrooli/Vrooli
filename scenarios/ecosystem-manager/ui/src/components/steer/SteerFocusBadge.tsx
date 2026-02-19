import { cn, formatSkillSetLabel, formatSkillSetTooltip } from '@/lib/utils';
import type { AutoSteerProfile, ExecutionHistory } from '@/types/api';
import { Compass, ListOrdered, Zap } from 'lucide-react';

export interface SteerFocusBadgeProps {
  autoSteerProfileName?: string;
  phaseSetLabel?: string;
  phaseTooltip?: string;
  manualSetLabel?: string;
  manualSetTooltip?: string;
  queueSetLabel?: string;
  queueTooltip?: string;
  queueIndex?: number;
  queueTotal?: number;
  queueExhausted?: boolean;
  className?: string;
}

export interface SteerFocusInfo {
  autoSteerProfileName?: string;
  phaseSetLabel?: string;
  phaseTooltip?: string;
  manualSetLabel?: string;
  manualSetTooltip?: string;
  queueSetLabel?: string;
  queueTooltip?: string;
  queueIndex?: number;
  queueTotal?: number;
  queueExhausted?: boolean;
}

export function getExecutionSteerFocus(
  execution: ExecutionHistory,
  profilesById: Record<string, AutoSteerProfile | undefined> = {},
  phaseNames: Array<{ id: string; name: string }> = [],
): SteerFocusInfo {
  const manualSet = execution.steer_skill_ids ?? [];
  const manualSetLabel =
    manualSet.length > 0
      ? formatSkillSetLabel(manualSet, phaseNames, { maxVisible: 1, emptyLabel: '' })
      : undefined;
  const manualSetTooltip =
    manualSet.length > 0
      ? formatSkillSetTooltip(manualSet, phaseNames)
      : undefined;

  const profileId = execution.auto_steer_profile_id;
  const profile = profileId ? profilesById[profileId] : undefined;
  const autoSteerProfileName = profileId ? profile?.name ?? profileId : undefined;

  if (autoSteerProfileName) {
    const phaseIndexRaw =
      typeof execution.steer_phase_index === 'number' ? execution.steer_phase_index : undefined;
    const phaseArrayIndex =
      typeof phaseIndexRaw === 'number' && phaseIndexRaw > 0 ? phaseIndexRaw - 1 : undefined;
    const phaseSkillIds =
      typeof phaseArrayIndex === 'number' && profile?.phases?.[phaseArrayIndex]
        ? profile.phases[phaseArrayIndex].skill_ids
        : execution.steer_skill_ids;
    const phaseSetLabel = formatSkillSetLabel(phaseSkillIds, phaseNames, {
      maxVisible: 1,
      emptyLabel: autoSteerProfileName,
    });
    const tooltipBody = formatSkillSetTooltip(phaseSkillIds, phaseNames) ?? phaseSetLabel;
    const phaseTooltip =
      typeof phaseIndexRaw === 'number'
        ? `Phase ${phaseIndexRaw}: ${tooltipBody}`
        : tooltipBody;

    return {
      autoSteerProfileName,
      phaseSetLabel,
      phaseTooltip,
    };
  }

  if (manualSetLabel) {
    return { manualSetLabel, manualSetTooltip };
  }

  return {};
}

export function SteerFocusBadge({
  autoSteerProfileName,
  phaseSetLabel,
  phaseTooltip,
  manualSetLabel,
  manualSetTooltip,
  queueSetLabel,
  queueTooltip,
  queueIndex,
  queueTotal,
  queueExhausted,
  className,
}: SteerFocusBadgeProps) {
  if (!autoSteerProfileName && !manualSetLabel && queueSetLabel === undefined && queueIndex === undefined) {
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
            {phaseSetLabel && (
              <>
                <span className="text-[10px] text-indigo-800/70 dark:text-indigo-100/70">•</span>
                <span className="font-normal text-[11px] text-indigo-800/90 dark:text-indigo-100/90">
                  {phaseSetLabel}
                </span>
              </>
            )}
          </span>
        </div>
      </div>
    );
  }

  if (queueSetLabel !== undefined || queueIndex !== undefined) {
    const position = queueIndex !== undefined && queueTotal !== undefined
      ? `${queueIndex + 1}/${queueTotal}`
      : undefined;
    const tooltip = queueExhausted
      ? 'Queue exhausted'
      : position
        ? `Queue position ${position}: ${queueTooltip || queueSetLabel || 'N/A'}`
        : queueTooltip || queueSetLabel;

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
            <span className={cn('font-normal text-[11px]', queueExhausted && 'line-through')}>
              {queueExhausted ? 'Done' : queueSetLabel || 'Queue'}
            </span>
          </span>
        </div>
      </div>
    );
  }

  if (!manualSetLabel) {
    return null;
  }

  return (
    <div
      title={manualSetTooltip}
      className={cn(
        'flex items-center gap-1.5 px-2 py-1 rounded bg-amber-100 text-amber-900 border border-amber-200 dark:bg-amber-500/10 dark:text-amber-50 dark:border-amber-500/30',
        className,
      )}
    >
      <Compass className="h-3.5 w-3.5" />
      <div className="leading-tight text-xs font-semibold">{manualSetLabel}</div>
    </div>
  );
}
