/**
 * TaskCardBody Component
 * Displays task title, target (for improver tasks), notes preview, and steering indicators
 */

import { FileText } from 'lucide-react';
import type { Task, AutoSteerProfile } from '../../types/api';
import { SteerFocusBadge } from '@/components/steer/SteerFocusBadge';
import { useAppState } from '../../contexts/AppStateContext';

interface TaskCardBodyProps {
  task: Task;
  autoSteerProfile?: AutoSteerProfile;
  autoSteerPhaseIndex?: number;
}

export function TaskCardBody({ task, autoSteerProfile, autoSteerPhaseIndex }: TaskCardBodyProps) {
  const { cachedSettings } = useAppState();
  const condensedMode = cachedSettings?.display?.condensed_mode ?? false;

  const hasNotes = task.notes && task.notes.trim().length > 0;
  const hasAutoSteer = !!task.auto_steer_profile_id || !!autoSteerProfile;
  const manualSteerMode = !hasAutoSteer && task.steer_mode ? task.steer_mode.toUpperCase() : '';
  const phaseIndex = typeof autoSteerPhaseIndex === 'number'
    ? autoSteerPhaseIndex
    : typeof task.auto_steer_phase_index === 'number'
      ? task.auto_steer_phase_index
      : undefined;
  const phaseLabel = autoSteerProfile?.phases?.[phaseIndex ?? -1]?.mode;
  const phaseMode = phaseLabel || task.auto_steer_mode;
  const phaseTooltip = phaseIndex !== undefined && phaseMode
    ? `Phase ${phaseIndex + 1}: ${phaseMode}`
    : phaseMode || autoSteerProfile?.name || 'Auto Steer';
  const primaryTarget = (task.target && task.target[0]) || '';
  const derivedTitle = `${task.operation === 'improver' ? 'Improve' : 'Generate'} ${primaryTarget || task.type}`;
  const displayTitle = derivedTitle.trim() || task.title;

  // Truncate notes to first 150 characters
  const truncatedNotes = hasNotes ? task.notes!.slice(0, 150) + (task.notes!.length > 150 ? '...' : '') : '';
  const spacingClass = condensedMode ? 'space-y-1.5' : 'space-y-2';
  const showNotesPreview = hasNotes && !condensedMode;

  return (
    <div className={spacingClass}>
      {/* Title */}
      <h3 className="text-sm font-medium text-foreground line-clamp-2">
        {displayTitle}
      </h3>

      {/* Notes preview */}
      {showNotesPreview && (
        <div className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <FileText className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <p className="flex-1 min-w-0 line-clamp-2">{truncatedNotes}</p>
        </div>
      )}

      <SteerFocusBadge
        autoSteerProfileName={hasAutoSteer ? autoSteerProfile?.name ?? 'Auto Steer' : undefined}
        phaseMode={phaseMode}
        phaseTooltip={phaseTooltip}
        manualSteerMode={!hasAutoSteer && manualSteerMode ? manualSteerMode : undefined}
      />
    </div>
  );
}
