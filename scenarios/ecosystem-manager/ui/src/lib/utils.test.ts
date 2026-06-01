import { describe, expect, it } from 'vitest';
import {
  formatSkillSetLabel,
  formatSkillSetTooltip,
  getSkillDisplayName,
  getQueueStepDisplay,
} from './utils';

const SKILLS = [
  { id: 'progress', name: 'Progress' },
  { id: 'react-coherence', name: 'React Coherence' },
  { id: 'api-steer', name: 'API Steer' },
];

describe('steering label utils', () => {
  it('formats set labels with +k more', () => {
    expect(
      formatSkillSetLabel(['react-coherence', 'api-steer'], SKILLS, { maxVisible: 1 })
    ).toBe('React Coherence +1 more');
  });

  it('formats set labels without truncation when visible count covers all', () => {
    expect(
      formatSkillSetLabel(['react-coherence', 'api-steer'], SKILLS, { maxVisible: 2 })
    ).toBe('React Coherence, API Steer');
  });

  it('returns empty label when set is empty', () => {
    expect(formatSkillSetLabel([], SKILLS, { emptyLabel: 'None selected' })).toBe('None selected');
  });

  it('builds full tooltip labels', () => {
    expect(formatSkillSetTooltip(['react-coherence', 'api-steer'], SKILLS)).toBe(
      'React Coherence, API Steer'
    );
  });

  it('returns queue step summary and tooltip', () => {
    const result = getQueueStepDisplay(['react-coherence', 'api-steer'], SKILLS);
    expect(result.label).toBe('React Coherence +1 more');
    expect(result.tooltip).toBe('React Coherence, API Steer');
  });

  it('falls back to skill formatting for unknown ids', () => {
    expect(getSkillDisplayName('custom-skill', SKILLS)).toBe('Custom Skill');
  });
});
