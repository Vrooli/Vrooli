import { useState } from 'react';
import { Circle, Compass, ListOrdered, Zap, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAutoSteerProfiles } from '@/hooks/useAutoSteer';
import { usePhaseNames } from '@/hooks/usePromptFiles';
import { SteeringConfigDialog } from './SteeringConfigDialog';
import type { SteeringConfig, SteeringStrategy, AutoSteerProfile } from '@/types/api';

interface SteeringConfigPickerProps {
  value: SteeringConfig;
  onChange: (config: SteeringConfig) => void;
  disabled?: boolean;
  className?: string;
}

interface StrategyDisplay {
  label: string;
  sublabel?: string;
  icon: React.ElementType;
  colorClasses: string;
}

function formatPhaseName(name: string): string {
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function getStrategyDisplay(
  config: SteeringConfig,
  profiles: AutoSteerProfile[]
): StrategyDisplay {
  switch (config.strategy) {
    case 'profile': {
      const profile = profiles.find((p) => p.id === config.profileId);
      const phasesCount = profile?.phases?.length || 0;
      return {
        label: profile?.name || 'Unknown Profile',
        sublabel: phasesCount > 0 ? `${phasesCount} phases` : undefined,
        icon: Zap,
        colorClasses:
          'bg-indigo-500/10 text-indigo-100 border-indigo-500/30 hover:bg-indigo-500/20',
      };
    }
    case 'queue': {
      const items = config.queue || [];
      if (items.length === 0) {
        return {
          label: 'Queue',
          sublabel: 'Empty',
          icon: ListOrdered,
          colorClasses: 'bg-cyan-500/10 text-cyan-100 border-cyan-500/30 hover:bg-cyan-500/20',
        };
      }
      const preview = items.slice(0, 3).map(formatPhaseName).join(' → ');
      const more = items.length > 3 ? ` +${items.length - 3}` : '';
      return {
        label: preview + more,
        sublabel: `${items.length} item${items.length === 1 ? '' : 's'}`,
        icon: ListOrdered,
        colorClasses: 'bg-cyan-500/10 text-cyan-100 border-cyan-500/30 hover:bg-cyan-500/20',
      };
    }
    case 'manual': {
      const mode = config.manualMode;
      return {
        label: mode ? formatPhaseName(mode) : 'Manual',
        sublabel: mode ? 'Manual focus' : 'Select a mode',
        icon: Compass,
        colorClasses: 'bg-amber-500/10 text-amber-50 border-amber-500/30 hover:bg-amber-500/20',
      };
    }
    case 'none':
    default:
      return {
        label: 'Default',
        sublabel: 'Progress mode',
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
}: SteeringConfigPickerProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { data: profiles = [], isLoading: isLoadingProfiles } = useAutoSteerProfiles();
  const { data: phaseNames = [], isLoading: isLoadingPhases } = usePhaseNames();

  const display = getStrategyDisplay(value, profiles);
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
        profiles={profiles}
        phaseNames={phaseNames}
        isLoadingProfiles={isLoadingProfiles}
        isLoadingPhases={isLoadingPhases}
      />
    </>
  );
}

// Helper function to derive SteeringConfig from task data
export function deriveSteeringConfig(task: {
  auto_steer_profile_id?: string;
  steering_queue?: string[];
  steer_mode?: string;
}): SteeringConfig {
  // Priority: Profile > Queue > Manual > None (matches backend)
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
  if (task.steer_mode) {
    return {
      strategy: 'manual',
      manualMode: task.steer_mode,
    };
  }
  return {
    strategy: 'none',
  };
}

// Helper function to extract task fields from SteeringConfig
export function extractSteeringFields(config: SteeringConfig): {
  steer_mode?: string;
  auto_steer_profile_id?: string;
  steering_queue?: string[];
} {
  switch (config.strategy) {
    case 'profile':
      return {
        auto_steer_profile_id: config.profileId,
        steer_mode: undefined,
        steering_queue: undefined,
      };
    case 'queue':
      return {
        steering_queue: config.queue,
        steer_mode: undefined,
        auto_steer_profile_id: undefined,
      };
    case 'manual':
      return {
        steer_mode: config.manualMode,
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
    case 'none':
    default:
      return {
        steer_mode: undefined,
        auto_steer_profile_id: undefined,
        steering_queue: undefined,
      };
  }
}
