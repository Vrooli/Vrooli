/**
 * TaskCardBody Component
 * Displays task title, target (for improver tasks), notes preview, and steering indicators
 */

import { FileText } from 'lucide-react';
import type { Task, AutoSteerProfile } from '../../types/api';
import { SteerFocusBadge } from '@/components/steer/SteerFocusBadge';
import { useMergedPhaseNames } from '@/hooks/usePromptFiles';
import { formatSkillSetLabel, formatSkillSetTooltip, getQueueStepDisplay } from '@/lib/utils';
import { useAppState } from '../../contexts/AppStateContext';

interface TaskCardBodyProps {
  task: Task;
  autoSteerProfile?: AutoSteerProfile;
  autoSteerPhaseIndex?: number;
}

export function TaskCardBody({ task, autoSteerProfile, autoSteerPhaseIndex }: TaskCardBodyProps) {
  const { cachedSettings } = useAppState();
  const condensedMode = cachedSettings?.display?.condensed_mode ?? false;
  const { data: phaseNames = [] } = useMergedPhaseNames();

  const hasNotes = task.notes && task.notes.trim().length > 0;
  const hasAutoSteer = !!task.auto_steer_profile_id || !!autoSteerProfile;
  const hasQueueSteering = !hasAutoSteer && task.steering_queue && task.steering_queue.length > 0;
  const manualSet = !hasAutoSteer && !hasQueueSteering ? task.steer_set ?? [] : [];
  const manualSetLabel =
    manualSet.length > 0
      ? formatSkillSetLabel(manualSet, phaseNames, { maxVisible: 1, emptyLabel: '' })
      : '';

  const phaseIndex = typeof autoSteerPhaseIndex === 'number'
    ? autoSteerPhaseIndex
    : typeof task.auto_steer_phase_index === 'number'
      ? task.auto_steer_phase_index
      : undefined;
  const phaseSkillIds =
    autoSteerProfile?.phases?.[phaseIndex ?? -1]?.skill_ids ?? [];
  const phaseSetLabel =
    phaseSkillIds.length > 0
      ? formatSkillSetLabel(phaseSkillIds, phaseNames, { maxVisible: 1, emptyLabel: '' })
      : '';
  const phaseTooltip =
    phaseIndex !== undefined && phaseSkillIds.length > 0
      ? `Phase ${phaseIndex + 1}: ${formatSkillSetTooltip(phaseSkillIds, phaseNames)}`
      : phaseSetLabel || autoSteerProfile?.name || 'Auto Steer';

  const primaryTarget = (task.target && task.target[0]) || '';
  const derivedTitle = `${task.operation === 'improver' ? 'Improve' : 'Generate'} ${primaryTarget || task.type}`;
  const displayTitle = derivedTitle.trim() || task.title;

  const truncatedNotes = hasNotes ? task.notes!.slice(0, 150) + (task.notes!.length > 150 ? '...' : '') : '';
  const spacingClass = condensedMode ? 'space-y-1.5' : 'space-y-2';
  const showNotesPreview = hasNotes && !condensedMode;

  const queueLength = task.steering_queue?.length ?? 0;
  const effectiveQueueIndex = hasQueueSteering
    ? Math.min(
      Math.max(typeof task.steering_queue_index === 'number' ? task.steering_queue_index : 0, 0),
      Math.max(queueLength - 1, 0),
    )
    : undefined;
  const queueStep = hasQueueSteering && typeof effectiveQueueIndex === 'number'
    ? task.steering_queue?.[effectiveQueueIndex]
    : undefined;
  const queueDisplay = getQueueStepDisplay(queueStep ?? [], phaseNames);

  return (
    <div className={spacingClass}>
      <h3 className="text-sm font-medium text-foreground line-clamp-2">{displayTitle}</h3>

      {showNotesPreview && (
        <div className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <FileText className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <p className="flex-1 min-w-0 line-clamp-2">{truncatedNotes}</p>
        </div>
      )}

      <SteerFocusBadge
        autoSteerProfileName={hasAutoSteer ? autoSteerProfile?.name ?? 'Auto Steer' : undefined}
        phaseSetLabel={phaseSetLabel || undefined}
        phaseTooltip={phaseTooltip}
        manualSetLabel={!hasAutoSteer && !hasQueueSteering && manualSetLabel ? manualSetLabel : undefined}
        manualSetTooltip={!hasAutoSteer && !hasQueueSteering ? formatSkillSetTooltip(manualSet, phaseNames) : undefined}
        queueSetLabel={hasQueueSteering ? (task.steering_queue_set_label || queueDisplay.label) : undefined}
        queueTooltip={hasQueueSteering ? (queueDisplay.tooltip || undefined) : undefined}
        queueIndex={hasQueueSteering ? effectiveQueueIndex : undefined}
        queueTotal={hasQueueSteering ? (task.steering_queue_total ?? task.steering_queue?.length) : undefined}
        queueExhausted={hasQueueSteering ? task.steering_queue_exhausted : undefined}
      />
    </div>
  );
}
