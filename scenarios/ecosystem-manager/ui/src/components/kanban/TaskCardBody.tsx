/**
 * TaskCardBody Component
 * Displays task title, target (for improver tasks), notes preview, and auto-steer indicator
 */

import { Target, FileText, Zap } from 'lucide-react';
import type { Task, AutoSteerProfile } from '../../types/api';
import { useAppState } from '../../contexts/AppStateContext';

interface TaskCardBodyProps {
  task: Task;
  autoSteerProfile?: AutoSteerProfile;
  autoSteerPhaseIndex?: number;
}

export function TaskCardBody({ task, autoSteerProfile, autoSteerPhaseIndex }: TaskCardBodyProps) {
  const { cachedSettings } = useAppState();
  const condensedMode = cachedSettings?.display?.condensed_mode ?? false;

  const hasTarget = task.target && task.target.length > 0;
  const hasNotes = task.notes && task.notes.trim().length > 0;
  const hasAutoSteer = !!task.auto_steer_profile_id || !!autoSteerProfile;
  const phaseIndex = typeof autoSteerPhaseIndex === 'number'
    ? autoSteerPhaseIndex
    : typeof task.auto_steer_phase_index === 'number'
      ? task.auto_steer_phase_index
      : undefined;
  const phaseLabel = autoSteerProfile?.phases?.[phaseIndex ?? -1]?.mode;

  // Truncate notes to first 150 characters
  const truncatedNotes = hasNotes ? task.notes!.slice(0, 150) + (task.notes!.length > 150 ? '...' : '') : '';
  const spacingClass = condensedMode ? 'space-y-1.5' : 'space-y-2';
  const showNotesPreview = hasNotes && !condensedMode;

  return (
    <div className={spacingClass}>
      {/* Title */}
      <h3 className="text-sm font-medium text-foreground line-clamp-2">
        {task.title}
      </h3>

      {/* Target (for improver tasks) */}
      {hasTarget && (
        <div className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <Target className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <div className="flex-1 min-w-0">
            {task.target!.map((t, i) => (
              <span key={i} className="inline-block">
                {t}
                {i < task.target!.length - 1 && ', '}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Notes preview */}
      {showNotesPreview && (
        <div className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <FileText className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <p className="flex-1 min-w-0 line-clamp-2">{truncatedNotes}</p>
        </div>
      )}

      {/* Auto Steer indicator */}
      {hasAutoSteer && (
        <div className="flex items-center gap-1.5 px-2 py-1 rounded bg-indigo-100 text-indigo-900 border border-indigo-200 dark:bg-indigo-500/10 dark:text-indigo-100 dark:border-indigo-500/30">
          <Zap className="h-3.5 w-3.5" />
          <div className="flex flex-col leading-tight">
            <span className="text-xs font-semibold">
              {autoSteerProfile?.name ?? 'Auto Steer'}
            </span>
            {phaseIndex !== undefined && (
              <span className="text-[11px] text-indigo-800/80 dark:text-indigo-100/80">
                Phase {phaseIndex + 1}{phaseLabel ? `: ${phaseLabel}` : ''}
              </span>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
