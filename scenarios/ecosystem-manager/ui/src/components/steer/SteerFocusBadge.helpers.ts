import { formatSkillSetLabel, formatSkillSetTooltip } from '@/lib/utils';
import type { AutoSteerProfile, ExecutionHistory } from '@/types/api';

export interface SteerFocusInfo {
  autoSteerProfileName?: string;
  skillSetLabel?: string;
  skillTooltip?: string;
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
  skillNames: Array<{ id: string; name: string }> = [],
): SteerFocusInfo {
  const manualSet = execution.steer_skill_ids ?? [];
  const manualSetLabel =
    manualSet.length > 0
      ? formatSkillSetLabel(manualSet, skillNames, { maxVisible: 1, emptyLabel: '' })
      : undefined;
  const manualSetTooltip =
    manualSet.length > 0
      ? formatSkillSetTooltip(manualSet, skillNames)
      : undefined;

  const profileId = execution.auto_steer_profile_id;
  const profile = profileId ? profilesById[profileId] : undefined;
  const autoSteerProfileName = profileId ? profile?.name ?? profileId : undefined;

  if (autoSteerProfileName) {
    const skillIds = execution.steer_skill_ids ?? [];
    const iteration =
      typeof execution.steer_phase_iteration === 'number' ? execution.steer_phase_iteration : undefined;
    const skillSetLabel = formatSkillSetLabel(skillIds, skillNames, {
      maxVisible: 1,
      emptyLabel: autoSteerProfileName,
    });
    const tooltipBody = formatSkillSetTooltip(skillIds, skillNames) ?? skillSetLabel;
    const skillTooltip =
      typeof iteration === 'number' && iteration > 0
        ? `Iteration ${iteration}: ${tooltipBody}`
        : tooltipBody;

    return {
      autoSteerProfileName,
      skillSetLabel,
      skillTooltip,
      manualSetLabel,
      manualSetTooltip,
    };
  }

  const queueIndex =
    typeof execution.steer_phase_index === 'number' ? execution.steer_phase_index : undefined;
  const queueSetLabel = execution.steer_set_label;
  if (queueSetLabel || queueIndex !== undefined) {
    return {
      queueSetLabel,
      queueIndex,
      queueTooltip: queueSetLabel ? `Queue set: ${queueSetLabel}` : undefined,
      manualSetLabel,
      manualSetTooltip,
    };
  }

  return {
    manualSetLabel,
    manualSetTooltip,
  };
}
