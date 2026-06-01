import { describe, expect, it } from 'vitest';
import { prioritizeSelectedSkills } from './SkillPickerDialog';
import type { SkillInfo } from '@/types/api';

describe('SkillPickerDialog ordering', () => {
  it('moves preselected skills to the front while preserving relative order', () => {
    const skills: SkillInfo[] = [
      { id: 'progress', name: 'Progress', modes: [], source: 'builtin' },
      { id: 'refactor', name: 'Refactor', modes: [], source: 'builtin' },
      { id: 'test', name: 'Test', modes: [], source: 'builtin' },
      { id: 'ux', name: 'UX', modes: [], source: 'builtin' },
    ];

    const ordered = prioritizeSelectedSkills(skills, ['test', 'progress']);

    expect(ordered.map((p) => p.id)).toEqual(['progress', 'test', 'refactor', 'ux']);
  });

  it('is case-insensitive for selected skill ids', () => {
    const skills: SkillInfo[] = [
      { id: 'progress', name: 'Progress', modes: [], source: 'builtin' },
      { id: 'react-coherence', name: 'React Coherence', modes: [], source: 'prompt-manager' },
      { id: 'api-steer', name: 'API Steer', modes: [], source: 'prompt-manager' },
    ];

    const ordered = prioritizeSelectedSkills(skills, ['REACT-COHERENCE']);

    expect(ordered.map((p) => p.id)).toEqual(['react-coherence', 'progress', 'api-steer']);
  });
});
