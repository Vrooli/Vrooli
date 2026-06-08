import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

const BUILT_IN_SKILL_LABELS: Record<string, string> = {
  progress: 'Progress',
  ux: 'UX',
  refactor: 'Refactor',
  test: 'Test',
  explore: 'Explore',
  polish: 'Polish',
  performance: 'Performance',
  security: 'Security',
};

/**
 * Formats a skill id (e.g., "progress-mode" -> "Progress Mode")
 */
export function formatSkillName(name: string): string {
  return name
    .split(/[-_]/)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

export function normalizeSkillId(skillId?: string): string {
  return (skillId ?? '').trim().toLowerCase();
}

export function getSkillDisplayName(
  skillId?: string,
  skills: Array<{ id: string; name: string }> = []
): string | undefined {
  if (!skillId) return undefined;
  const normalized = normalizeSkillId(skillId);
  const match = skills.find((skill) => normalizeSkillId(skill.id) === normalized);
  return match?.name ?? BUILT_IN_SKILL_LABELS[normalized] ?? formatSkillName(skillId);
}

export function formatSkillSetLabel(
  skillIds: string[] = [],
  skills: Array<{ id: string; name: string }> = [],
  options: { maxVisible?: number; emptyLabel?: string } = {}
): string {
  const maxVisible = options.maxVisible ?? 1;
  const emptyLabel = options.emptyLabel ?? 'Default';
  if (!Array.isArray(skillIds) || skillIds.length === 0) return emptyLabel;

  const labels = skillIds.map((id) => getSkillDisplayName(id, skills) ?? id);
  const visible = labels.slice(0, Math.max(1, maxVisible));
  const hiddenCount = Math.max(0, labels.length - visible.length);

  return hiddenCount > 0 ? `${visible.join(', ')} +${hiddenCount} more` : visible.join(', ');
}

export function formatSkillSetTooltip(
  skillIds: string[] = [],
  skills: Array<{ id: string; name: string }> = []
): string | undefined {
  if (!Array.isArray(skillIds) || skillIds.length === 0) return undefined;
  return skillIds.map((id) => getSkillDisplayName(id, skills) ?? id).join(', ');
}

export function getQueueStepDisplay(
  step: string[] = [],
  skills: Array<{ id: string; name: string }> = []
): { label: string; tooltip?: string } {
  return {
    label: formatSkillSetLabel(step, skills, { maxVisible: 1, emptyLabel: 'Empty set' }),
    tooltip: formatSkillSetTooltip(step, skills),
  };
}

export function getApiErrorMessage(error: unknown): string {
  if (!error) return 'Unknown error';

  if (error instanceof Error) {
    try {
      const match = error.message.match(/API Error \(\d+\): (.+)/);
      if (match?.[1]) {
        const parsed = JSON.parse(match[1]);
        if (typeof parsed === 'string') {
          return parsed;
        }
        if (parsed && typeof parsed === 'object') {
          return parsed.message || parsed.error || error.message;
        }
      }
    } catch {
      // Fall through to raw message
    }
    return error.message;
  }

  if (typeof error === 'object' && error !== null) {
    const maybeError = error as { message?: string; error?: string };
    return maybeError.message || maybeError.error || 'Unknown error';
  }

  return String(error);
}
