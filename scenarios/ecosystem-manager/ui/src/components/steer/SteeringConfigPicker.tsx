import { useState } from 'react';
import { Circle, Compass, ListOrdered, Zap, ChevronRight } from 'lucide-react';
import { cn, formatSkillSetLabel, getQueueStepDisplay } from '@/lib/utils';
import { useAllAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { useMergedSkillNames } from '@/hooks/usePromptFiles';
import { SteeringConfigDialog } from './SteeringConfigDialog';
import type { SteeringConfig, SteeringStrategy, AutoSteerProfile } from '@/types/api';

interface SteeringConfigPickerProps {
  value: SteeringConfig;
  onChange: (config: SteeringConfig) => void;
  disabled?: boolean;
  className?: string;
  queueIndex?: number;
  queueExhausted?: boolean;
  onQueuePositionChange?: (position: number) => void;
  pendingQueuePosition?: number | null;
}

interface StrategyDisplay {
  label: string;
  sublabel?: string;
  icon: React.ElementType;
  colorClasses: string;
}

function getStrategyDisplay(
  config: SteeringConfig,
  profiles: AutoSteerProfile[],
  skillNames: { id: string; name: string }[]
): StrategyDisplay {
  switch (config.strategy) {
    case 'profile': {
      const profile = profiles.find((p) => p.id === config.profileId);
      const skillCount = profile?.allowed_skills?.length || 0;
      return {
        label: profile?.name || 'Unknown Profile',
        sublabel: skillCount > 0 ? `${skillCount} skill${skillCount === 1 ? '' : 's'}` : undefined,
        icon: Zap,
        colorClasses:
          'bg-indigo-500/10 text-indigo-100 border-indigo-500/30 hover:bg-indigo-500/20',
      };
    }
    case 'queue': {
      const steps = config.queue || [];
      if (steps.length === 0) {
        return {
          label: 'Queue',
          sublabel: 'Empty',
          icon: ListOrdered,
          colorClasses: 'bg-cyan-500/10 text-cyan-100 border-cyan-500/30 hover:bg-cyan-500/20',
        };
      }
      const first = getQueueStepDisplay(steps[0], skillNames).label;
      const more = steps.length > 1 ? ` +${steps.length - 1} more` : '';
      return {
        label: `${first}${more}`,
        sublabel: `${steps.length} step${steps.length === 1 ? '' : 's'}`,
        icon: ListOrdered,
        colorClasses: 'bg-cyan-500/10 text-cyan-100 border-cyan-500/30 hover:bg-cyan-500/20',
      };
    }
    case 'manual': {
      const set = config.manualSet ?? [];
      return {
        label: formatSkillSetLabel(set, skillNames, { maxVisible: 1, emptyLabel: 'Manual' }),
        sublabel: set.length > 0 ? `${set.length} skill${set.length === 1 ? '' : 's'}` : 'Select skills',
        icon: Compass,
        colorClasses: 'bg-amber-500/10 text-amber-50 border-amber-500/30 hover:bg-amber-500/20',
      };
    }
    case 'none':
    default:
      return {
        label: 'Default',
        sublabel: 'Progress set',
        icon: Circle,
        colorClasses: 'bg-slate-500/10 text-slate-300 border-slate-500/30 hover:bg-slate-500/20',
      };
  }
}

export function SteeringConfigPicker({
  value,
  onChange,
  disabled,
  className,
  queueIndex,
  queueExhausted,
  onQueuePositionChange,
  pendingQueuePosition,
}: SteeringConfigPickerProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { data: allProfiles = [], isLoading: isLoadingProfiles } = useAllAutoSteerProfiles();
  const { data: skillNames = [], isLoading: isLoadingSkills } = useMergedSkillNames();

  const display = getStrategyDisplay(value, allProfiles, skillNames);
  const Icon = display.icon;

  return (
    <>
      <button
        type="button"
        onClick={() => !disabled && setDialogOpen(true)}
        disabled={disabled}
        className={cn(
          'flex items-center gap-2 px-3 py-2 rounded-md border transition-colors text-left w-full',
          display.colorClasses,
          disabled && 'opacity-50 cursor-not-allowed',
          className
        )}
      >
        <Icon className="h-4 w-4 shrink-0" />
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium truncate">{display.label}</div>
          {display.sublabel && (
            <div className="text-xs opacity-70 truncate">{display.sublabel}</div>
          )}
        </div>
        <ChevronRight className="h-4 w-4 shrink-0 opacity-50" />
      </button>

      <SteeringConfigDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        value={value}
        onChange={onChange}
        profiles={allProfiles}
        skillNames={skillNames}
        isLoadingProfiles={isLoadingProfiles}
        isLoadingSkills={isLoadingSkills}
        queueIndex={queueIndex}
        queueExhausted={queueExhausted}
        onQueuePositionChange={onQueuePositionChange}
        pendingQueuePosition={pendingQueuePosition}
      />
    </>
  );
}

export function deriveSteeringConfig(task: {
  auto_steer_profile_id?: string;
  steering_queue?: string[][];
  steer_set?: string[];
}): SteeringConfig {
  if (task.auto_steer_profile_id) {
    return {
      strategy: 'profile',
      profileId: task.auto_steer_profile_id,
    };
  }
  if (task.steering_queue && task.steering_queue.length > 0) {
    return {
      strategy: 'queue',
      queue: task.steering_queue,
    };
  }
  if (task.steer_set && task.steer_set.length > 0) {
    return {
      strategy: 'manual',
      manualSet: task.steer_set,
    };
  }
  return {
    strategy: 'none',
  };
}

export function extractSteeringFields(config: SteeringConfig): {
  steer_set?: string[];
  auto_steer_profile_id?: string;
  steering_queue?: string[][];
} {
  switch (config.strategy) {
    case 'profile':
      return {
        auto_steer_profile_id: config.profileId,
        steer_set: undefined,
        steering_queue: undefined,
      };
    case 'queue':
      return {
        steering_queue: config.queue,
        steer_set: undefined,
        auto_steer_profile_id: undefined,
      };
    case 'manual':
      return {
        steer_set: config.manualSet,
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
    case 'none':
    default:
      return {
        steer_set: undefined,
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
  }
}
