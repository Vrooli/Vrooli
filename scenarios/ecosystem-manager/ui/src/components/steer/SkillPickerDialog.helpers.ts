import { normalizeSkillId } from '@/lib/utils';
import type { SkillInfo } from '@/types/api';
import type { SortOption } from '@/hooks/useSkillUsage';

export const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'name', label: 'Name (A-Z)' },
  { value: 'recent', label: 'Recent' },
  { value: 'most-used', label: 'Most Used' },
];

export const BUILT_IN_SKILLS: SkillInfo[] = [
  { id: 'progress', name: 'Progress', description: 'Advance core objectives and operational targets.', modes: [], source: 'builtin' },
  { id: 'ux', name: 'UX', description: 'Improve usability, accessibility, and user flows.', modes: [], source: 'builtin' },
  { id: 'refactor', name: 'Refactor', description: 'Raise code quality without changing behavior.', modes: [], source: 'builtin' },
  { id: 'test', name: 'Test', description: 'Expand coverage and harden edge cases.', modes: [], source: 'builtin' },
  { id: 'explore', name: 'Explore', description: 'Explore options before committing to a path.', modes: [], source: 'builtin' },
  { id: 'polish', name: 'Polish', description: 'Finalize copy, visuals, and small fixes.', modes: [], source: 'builtin' },
  { id: 'performance', name: 'Performance', description: 'Profile and optimize slow paths.', modes: [], source: 'builtin' },
  { id: 'security', name: 'Security', description: 'Reduce vulnerabilities and tighten validation.', modes: [], source: 'builtin' },
];

export function prioritizeSelectedSkills(
  skills: SkillInfo[],
  selectedSkillIds: string[],
): SkillInfo[] {
  if (skills.length === 0 || selectedSkillIds.length === 0) {
    return skills;
  }

  const selected = new Set(selectedSkillIds.map((id) => normalizeSkillId(id)));
  return [...skills].sort((a, b) => {
    const aSelected = selected.has(normalizeSkillId(a.id)) ? 1 : 0;
    const bSelected = selected.has(normalizeSkillId(b.id)) ? 1 : 0;
    return bSelected - aSelected;
  });
}
